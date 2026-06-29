// Package daemon
package daemon

import (
	"errors"
	"time"
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

	initialized bool
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

	if err := protocol.GetRegistry(d.wlConn); err != nil {
		return err
	}

	syncID := d.allocID()
	if err := protocol.Sync(d.wlConn, syncID); err != nil {
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
			if err := protocol.Bind(d.wlConn, protocol.RegistryID, global.Name, d.compositor, "wl_compositor", 4); err != nil {
				return err
			}
		case "wl_shm":
			d.shm = d.allocID()
			if err := protocol.Bind(d.wlConn, protocol.RegistryID, global.Name, d.shm, "wl_shm", 1); err != nil {
				return err
			}
		case "zwlr_layer_shell_v1":
			d.layerShell = d.allocID()
			if err := protocol.Bind(d.wlConn, protocol.RegistryID, global.Name, d.layerShell, "zwlr_layer_shell_v1", 4); err != nil {
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
	return nil
}

func (d *Daemon) setupOutput(g protocol.Global) error {
	out := &Output{
		name: g.Name,
		id:   d.allocID(),
	}
	if err := protocol.Bind(d.wlConn, protocol.RegistryID, g.Name, out.id, "wl_output", 4); err != nil {
		return err
	}
	out.surface = d.allocID()
	if err := protocol.CreateSurface(d.wlConn, d.compositor, out.surface); err != nil {
		return err
	}
	out.layerSurf = d.allocID()
	if err := protocol.GetLayerSurface(d.wlConn, d.layerShell, out.layerSurf, out.surface, out.id, protocol.ZwlrLayerBackground, "wallpaper"); err != nil {
		return err
	}
	if err := protocol.SetSize(d.wlConn, out.layerSurf, 0, 0); err != nil {
		return err
	}
	if err := protocol.SetAnchor(d.wlConn, out.layerSurf, protocol.AnchorTop|protocol.AnchorBottom|protocol.AnchorLeft|protocol.AnchorRight); err != nil {
		return err
	}
	if err := protocol.SetExclusiveZone(d.wlConn, out.layerSurf, -1); err != nil {
		return err
	}
	if err := protocol.Commit(d.wlConn, out.surface); err != nil {
		return err
	}
	d.outputs = append(d.outputs, out)
	return nil
}

func (d *Daemon) allocID() uint32 {
	d.nextID++
	return d.nextID
}
