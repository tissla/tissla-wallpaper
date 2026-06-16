// Package daemon
package daemon

import (
	"log"
	"net"
	"os"
	"time"
	"tissla-wallpaper/internal/ipc"
	"tissla-wallpaper/internal/protocol"
	wl "tissla-wallpaper/internal/wayland"
)

type Daemon struct {
	wlConn wl.Connection
	est    time.Time

	compositor uint32
	shm        uint32
	layerShell uint32

	outputs []*Output
	action  SurfaceAction
}

type Output struct {
	id uint32

	surface   uint32
	layerSurf uint32

	width  uint32
	height uint32
}

// SurfaceAction contains the actions intent and path to the resource
type SurfaceAction struct {
	action string
	path   string
}

func New() (*Daemon, error) {

	conn, err := wl.New()
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		wlConn: conn,
		est:    time.Now().UTC(),
	}

	if err := d.init(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Daemon) init() error {

	// get compositor, shm, layerShell
	// create get_registry data
	err := protocol.GetRegistry(d.wlConn)
	if err != nil {
		return err
	}

	for d.compositor == 0 || d.shm == 0 || d.layerShell == 0 {
		msg, err := d.wlConn.Read()
		if err != nil {
			return err
		}

		global, ok := protocol.ParseGlobal(msg)
		if !ok {
			continue
		}

		switch global.Interface {
		case "wl_compositor":
			d.compositor = global.Name
		case "wl_shm":
			d.shm = global.Name
		case "zwlr_layer_shell_v1":
			d.layerShell = global.Name
		}
	}

	// get screen outputs
	return nil
}

func (d *Daemon) HandleEvents() {
	for msg := range d.wlConn.Listen() {

		println("%s", msg)
	}

}

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
		d.handleCommand(string(buf[:n]))
		conn.Close()
	}
}

func (d *Daemon) handleCommand(cmd string) error {

	return nil
}
