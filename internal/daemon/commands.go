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

// HandleCommands accepts client connections and forwards each request to the
// loop, serving one client to completion before accepting the next.
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

// toCommand translates a wire Request into the daemon's internal command,
// parsing the scale mode once at the boundary.
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

// handleCommand runs in the loop thread and returns the response to send back.
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
		hits := 0
		for _, out := range d.outputs {
			if cmd.output == "" || cmd.output == out.hName {
				if err := d.clearOutput(out); err != nil {
					return "", err
				}
				hits++
			}
		}
		if hits == 0 {
			return "", fmt.Errorf("no output named %q", cmd.output)
		}
		return fmt.Sprintf("cleared %d output(s)\n", hits), nil

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
		return errors.New("animated wallpapers not implemented")
	default:
		return fmt.Errorf("unknown action kind %d", action.kind)
	}
}

func (d *Daemon) setStatic(output *Output, action SurfaceAction) error {
	w := output.width
	h := output.height
	if w == 0 || h == 0 {
		return fmt.Errorf("output %q not configured yet", output.hName)
	}

	size := int(w * h * 4)
	shm, err := wl.NewShmBuffer(size)
	if err != nil {
		return err
	}
	// our mapping is only needed to write the pixels; the compositor keeps the
	// memory alive via the pool, so drop our view when we're done.
	defer syscall.Munmap(shm.Data)

	// render straight into the shm mapping: no intermediate result buffer, no copy
	if err := img.LoadInto(shm.Data, action.path, int(w), int(h), action.scale); err != nil {
		return err
	}

	poolID := d.allocID()
	if err := d.sendFd(protocol.CreatePool(d.shm, poolID, size), shm.Fd); err != nil {
		return err
	}
	// the fd was duplicated into the socket on send; the pool holds the memory now
	syscall.Close(shm.Fd)

	bufferID := d.allocID()
	stride := int32(w * 4)
	if err := d.send(protocol.CreateBuffer(poolID, bufferID, 0, int32(w), int32(h), stride, protocol.WlShmPixelFormatArgb8888)); err != nil {
		return err
	}

	// link the pool and buffer in the d.buffers map
	d.buffers[bufferID] = poolID
	output.bufferID = bufferID
	output.poolID = poolID

	if err := d.send(protocol.Attach(output.surface, bufferID, 0, 0)); err != nil {
		return err
	}
	if err := d.send(protocol.Damage(output.surface, 0, 0, int32(w), int32(h))); err != nil {
		return err
	}
	return d.send(protocol.Commit(output.surface))
}

// clearOutput blanks an output's surface. the buffer is released by the compositor and destroyed by releaseBuffer

func (d *Daemon) clearOutput(output *Output) error {
	if err := d.send(protocol.Attach(output.surface, 0, 0, 0)); err != nil {
		return err
	}
	if err := d.send(protocol.Commit(output.surface)); err != nil {
		return err
	}
	output.bufferID = 0
	output.poolID = 0
	return nil
}

// releaseBuffer destroys a buffer and its pool after the compositor has
// released it, freeing the backing memory.
func (d *Daemon) releaseBuffer(bufferID uint32) {
	poolID, ok := d.buffers[bufferID]
	if !ok {
		return
	}
	log.Printf("releasing buffer %d (pool %d)", bufferID, poolID)
	d.send(protocol.DestroyBuffer(bufferID))
	d.send(protocol.Destroy(poolID))
	delete(d.buffers, bufferID)

	// if this was an output's current buffer, forget it (the wallpaper stays
	// visible via the compositor's own copy)
	for _, out := range d.outputs {
		if out.bufferID == bufferID {
			out.bufferID = 0
			out.poolID = 0
		}
	}
}
