// Package daemon
package daemon

import (
	"time"
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

	// get screen outputs
	return nil
}
