// Package daemon
package daemon

import (
	"errors"
	"log"
	img "tissla-wallpaper/internal/image"
	"tissla-wallpaper/internal/protocol"
	wl "tissla-wallpaper/internal/wayland"
)

type Daemon struct {
	wlConn wl.Connection

	nextID uint32

	compositor uint32
	shm        uint32
	layerShell uint32

	outputs  []*Output
	commands chan command

	// buffers waiting to be destroyed when the compositor sends wl_buffer.release
	// bufferID -> poolID
	buffers map[uint32]uint32

	initialized bool
}

type command struct {
	verb   string
	path   string
	scale  img.ScaleMode
	output string
	reply  chan string
}

type Output struct {
	name  uint32 // registry name to match global_remove
	id    uint32 // bound wl_output object id
	hName string

	surface   uint32
	layerSurf uint32

	// ids of buffer and pool once wallpaper is set, to reuse for changing
	bufferID uint32
	poolID   uint32

	width  uint32
	height uint32
}

type ActionKind int

const (
	ActionStatic ActionKind = iota
	ActionAnimated
)

// SurfaceAction contains the actions intent and path to the resource
type SurfaceAction struct {
	kind  ActionKind
	path  string
	scale img.ScaleMode
}

func New() (*Daemon, error) {

	conn, err := wl.New()
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		wlConn:   conn,
		commands: make(chan command),
		buffers:  make(map[uint32]uint32),
	}

	if err := d.init(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Daemon) init() error {
	d.nextID = 2

	if err := d.send(protocol.GetRegistry()); err != nil {
		return err
	}

	syncID := d.allocID()
	if err := d.send(protocol.Sync(syncID)); err != nil {
		return err
	}

	//collect outputs
	var pendingOutputs []protocol.Global

	for {
		msg, err := d.wlConn.Read()
		if err != nil {
			return err
		}
		if protocol.ParseSyncDone(msg, syncID) {
			break
		}
		global, ok := protocol.ParseGlobal(msg)
		if !ok {
			continue
		}
		switch global.Interface {
		case "wl_compositor":
			d.compositor = d.allocID()
			if err := d.send(protocol.Bind(protocol.RegistryID, global.Name, d.compositor, "wl_compositor", 4)); err != nil {
				return err
			}
		case "wl_shm":
			d.shm = d.allocID()
			if err := d.send(protocol.Bind(protocol.RegistryID, global.Name, d.shm, "wl_shm", 1)); err != nil {
				return err
			}
		case "zwlr_layer_shell_v1":
			d.layerShell = d.allocID()
			if err := d.send(protocol.Bind(protocol.RegistryID, global.Name, d.layerShell, "zwlr_layer_shell_v1", 4)); err != nil {
				return err
			}
		case "wl_output":
			pendingOutputs = append(pendingOutputs, global)
		}
	}

	if d.compositor == 0 {
		return errors.New("wl_compositor not found")
	}
	if d.shm == 0 {
		return errors.New("wl_shm not found")
	}
	if d.layerShell == 0 {
		return errors.New("zwlr_layer_shell_v1 not found")
	}

	// setup output
	for _, g := range pendingOutputs {
		if err := d.setupOutput(g); err != nil {
			return err
		}
	}

	// set to true when finished
	d.initialized = true
	log.Printf("init complete: %d outputs (compositor=%d shm=%d layerShell=%d)",
		len(d.outputs), d.compositor, d.shm, d.layerShell)

	return nil
}

func (d *Daemon) setupOutput(g protocol.Global) error {
	out := &Output{
		name: g.Name,
		id:   d.allocID(),
	}
	if err := d.send(protocol.Bind(protocol.RegistryID, g.Name, out.id, "wl_output", 4)); err != nil {
		return err
	}
	out.surface = d.allocID()
	if err := d.send(protocol.CreateSurface(d.compositor, out.surface)); err != nil {
		return err
	}
	out.layerSurf = d.allocID()
	if err := d.send(protocol.GetLayerSurface(d.layerShell, out.layerSurf, out.surface, out.id, protocol.ZwlrLayerBackground, "wallpaper")); err != nil {
		return err
	}
	if err := d.send(protocol.SetSize(out.layerSurf, 0, 0)); err != nil {
		return err
	}
	if err := d.send(protocol.SetAnchor(out.layerSurf, protocol.AnchorTop|protocol.AnchorBottom|protocol.AnchorLeft|protocol.AnchorRight)); err != nil {
		return err
	}
	if err := d.send(protocol.SetExclusiveZone(out.layerSurf, -1)); err != nil {
		return err
	}
	if err := d.send(protocol.Commit(out.surface)); err != nil {
		return err
	}
	d.outputs = append(d.outputs, out)
	return nil
}

func (d *Daemon) allocID() uint32 {
	d.nextID++
	return d.nextID
}

func (d *Daemon) send(msg wl.Message) error {
	return d.wlConn.Write(msg)
}

func (d *Daemon) sendFd(msg wl.Message, fd int) error {
	return d.wlConn.WriteFd(msg, fd)
}
