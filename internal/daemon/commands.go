package daemon

import (
	"errors"
	"fmt"
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
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		conn.Close()

		cmd, err := parseCommand(string(buf[:n]))
		if err != nil {
			log.Printf("parse command: %v", err)
			continue
		}
		d.commands <- cmd
	}
}

func parseCommand(s string) (command, error) {
	parts := strings.SplitN(strings.TrimSpace(s), " ", 2)
	switch parts[0] {
	case "set":
		if len(parts) < 2 {
			return command{}, errors.New("set: missing arguments")
		}
		i := strings.LastIndex(parts[1], " ")
		if i < 0 {
			return command{}, errors.New("set: expected <path> <mode>")
		}
		path := parts[1][:i]
		mode, err := img.ParseScaleMode(parts[1][i+1:])
		if err != nil {
			return command{}, err
		}
		return command{verb: "set", path: path, scale: mode}, nil
	case "clear":
		return command{verb: "clear"}, nil
	case "monitors":
		return command{verb: "monitors"}, nil
	default:
		return command{}, fmt.Errorf("unknown command %q", parts[0])
	}
}

// handleCommand is used by the main loop

func (d *Daemon) handleCommand(cmd command) (string, error) {
	if !d.initialized {
		return "", errors.New("daemon not initialized")
	}
	switch cmd.verb {
	case "set":
		act := SurfaceAction{kind: classify(cmd.path), path: cmd.path, scale: cmd.scale}
		for _, out := range d.outputs {
			if err := d.setWpOnOutput(out, act); err != nil {
				return "", err
			}
		}
		return "wallpaper set: " + cmd.path + "\n", nil
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
