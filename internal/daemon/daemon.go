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

	nextID uint32

	compositor uint32
	shm        uint32
	layerShell uint32

	outputs []*Output
	action  SurfaceAction
}

type Output struct {
	name uint32 // registry name to match global_remove
	id   uint32 // bound wl_output object id

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

	d.nextID = 2
	// get compositor, shm, layerShell
	// create get_registry data
	err := protocol.GetRegistry(d.wlConn)
	if err != nil {
		return err
	}

	var compositorName, shmName, layerShellName uint32

	syncID := d.allocID()
	err = protocol.Sync(d.wlConn, syncID)
	if err != nil {
		return err
	}

	for {
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
			compositorName = global.Name
			d.compositor = d.allocID()
			if err = protocol.Bind(d.wlConn, protocol.RegistryID, compositorName, d.compositor, "wl_compositor", 4); err != nil {
				return err
			}
		case "wl_shm":
			shmName = global.Name
			d.shm = d.allocID()
			if err = protocol.Bind(d.wlConn, protocol.RegistryID, shmName, d.shm, "wl_shm", 1); err != nil {
				return err
			}
		case "zwlr_layer_shell_v1":
			layerShellName = global.Name

			d.layerShell = d.allocID()
			if err = protocol.Bind(d.wlConn, protocol.RegistryID, layerShellName, d.layerShell, "zwlr_layer_shell_v1", 4); err != nil {
				return err
			}
		}
	}

	return nil
}

// add and bind output to the daemons output array
func (d *Daemon) handleOutputAdded(global protocol.Global) error {
	out := &Output{}
	boundID := d.allocID()
	if err := protocol.Bind(d.wlConn, protocol.RegistryID, global.Name, boundID, "wl_output", 3); err != nil {
		return err
	}
	out.id = boundID

	out.surface = d.allocID()
	if err := protocol.CreateSurface(d.wlConn, d.compositor, out.surface); err != nil {
		return err
	}

	d.outputs = append(d.outputs, out)
	return nil
}

// remove output from the daemons output array
func (d *Daemon) handleOutputRemoved(name uint32) {

	for i, out := range d.outputs {
		if out.name == name {
			d.outputs = append(d.outputs[:i], d.outputs[i+1:]...)
			return
		}
	}
}

func (d *Daemon) HandleEvents() {
	for msg := range d.wlConn.Listen() {

		if global, ok := protocol.ParseGlobal(msg); ok {
			if global.Interface == "wl_output" {
				d.handleOutputAdded(global)
			}
			continue
		}

		if name, ok := protocol.ParseGlobalRemove(msg); ok {
			d.handleOutputRemoved(name)
			continue
		}
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

func (d *Daemon) allocID() uint32 {
	d.nextID++
	return d.nextID
}
