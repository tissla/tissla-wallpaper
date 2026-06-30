package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	img "tissla-wallpaper/internal/image"
	"tissla-wallpaper/internal/ipc"
	"tissla-wallpaper/internal/protocol"
	wl "tissla-wallpaper/internal/wayland"
)

// HandleCommands parses the commands incoming from the client and puts it in the daemons command-channel

func (d *Daemon) HandleCommands() {
	sockPath := ipc.SocketPath()
	os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		d.serveClient(conn)
	}
}

func (d *Daemon) serveClient(conn net.Conn) {
	defer conn.Close()

	var req ipc.Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		fmt.Fprintf(conn, "error: %v\n", err)
		return
	}

	cmd, err := toCommand(req)
	if err != nil {
		fmt.Fprintf(conn, "error: %v\n", err)
		return
	}

	cmd.reply = make(chan string, 1)
	d.commands <- cmd
	io.WriteString(conn, <-cmd.reply)
}

func toCommand(req ipc.Request) (command, error) {
	cmd := command{verb: req.Verb, path: req.Path, output: req.Output}
	if req.Verb == "set" {
		mode, err := img.ParseScaleMode(req.Mode)
		if err != nil {
			return command{}, err
		}
		cmd.scale = mode
	}
	return cmd, nil
}

func parseCommand(data []byte) (command, error) {
	var req ipc.Request
	if err := json.Unmarshal(data, &req); err != nil {
		return command{}, err
	}
	cmd := command{verb: req.Verb, path: req.Path, output: req.Output}
	if req.Verb == "set" {
		mode, err := img.ParseScaleMode(req.Mode)
		if err != nil {
			return command{}, err
		}
		cmd.scale = mode
	}
	return cmd, nil
} // handleCommand is used by the main loop

func (d *Daemon) handleCommand(cmd command) (string, error) {
	if !d.initialized {
		return "", errors.New("daemon not initialized")
	}

	switch cmd.verb {
	case "set":
		act := SurfaceAction{kind: classify(cmd.path), path: cmd.path, scale: cmd.scale}
		hits := 0
		for _, out := range d.outputs {
			if cmd.output == "" || cmd.output == out.hName {
				if err := d.setWpOnOutput(out, act); err != nil {
					return "", err
				}
				hits++
			}
		}
		if hits == 0 {
			return "", fmt.Errorf("no output named %q", cmd.output)
		}
		return fmt.Sprintf("wallpaper set on %d output(s): %s\n", hits, cmd.path), nil
	case "clear":
		return "", errors.New("clear not implemented")
	case "monitors":
		return d.monitorList(), nil
	default:
		return "", fmt.Errorf("unknown verb %q", cmd.verb)
	}
}

func (d *Daemon) monitorList() string {
	if len(d.outputs) == 0 {
		return "no outputs\n"
	}
	var b strings.Builder
	for _, out := range d.outputs {
		name := out.hName
		if name == "" {
			name = "(unnamed)"
		}
		fmt.Fprintf(&b, "%s - %dx%d\n", name, out.width, out.height)
	}
	return b.String()
}

func classify(path string) ActionKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gif", ".mp4", ".webm":
		return ActionAnimated
	default:
		return ActionStatic
	}
}

func (d *Daemon) setWpOnOutput(output *Output, action SurfaceAction) error {

	switch action.kind {
	case ActionStatic:
		return d.setStatic(output, action)
	case ActionAnimated:
		return errors.New("not implemented")
	default:
		return nil
	}
}

func (d *Daemon) setStatic(output *Output, action SurfaceAction) error {
	h := output.height
	w := output.width
	path := action.path

	wp, err := img.Load(path, int(w), int(h), action.scale)
	if err != nil {
		return err
	}

	size := len(wp.Data)
	shm, err := wl.NewShmBuffer(size)
	if err != nil {
		return err
	}

	// put data in buffer
	copy(shm.Data, wp.Data)

	// send fd to wayland
	poolID := d.allocID()
	if err := d.sendFd(protocol.CreatePool(d.shm, poolID, size), shm.Fd); err != nil {
		return err
	}

	// close fd since mmap now holds a reference
	syscall.Close(shm.Fd)

	// create buffer from the pool
	bufferID := d.allocID()
	stride := int32(w * 4) // 4 bytes per pixel
	if err := d.send(protocol.CreateBuffer(poolID, bufferID, 0, int32(w), int32(h), stride, protocol.WlShmPixelFormatArgb8888)); err != nil {
		return err
	}

	// if we got to this point, save poolID and bufferID to the output-struct
	output.bufferID = bufferID
	output.poolID = poolID

	// connect buffer to surface
	if err := d.send(protocol.Attach(output.surface, bufferID, 0, 0)); err != nil {
		return err
	}
	if err := d.send(protocol.Damage(output.surface, 0, 0, int32(w), int32(h))); err != nil {
		return err
	}
	if err := d.send(protocol.Commit(output.surface)); err != nil {
		return err
	}
	return nil
}
