package daemon

import (
	"errors"
	"log"
	"net"
	"os"
	"syscall"
	img "tissla-wallpaper/internal/image"
	"tissla-wallpaper/internal/ipc"
	"tissla-wallpaper/internal/protocol"
	wl "tissla-wallpaper/internal/wayland"
)

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

	if !d.initialized {
		return errors.New("daemon not initalized")
	}
	return nil
}

func (d *Daemon) SetWallpaper(outputName string, action SurfaceAction) error {

	// find output
	for _, output := range d.outputs {
		if outputName == output.hName {
			err := d.setWpOnOutput(output, action)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Daemon) setWpOnOutput(output *Output, action SurfaceAction) error {

	h := output.height
	w := output.width
	path := action.path

	wp, err := img.Load(path, int(w), int(h), 1)
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
	if err := protocol.CreatePool(d.wlConn, d.shm, poolID, size, shm.Fd); err != nil {
		return err
	}

	// close fd since mmap now holds a reference
	syscall.Close(shm.Fd)

	// create buffer from the pool
	bufferID := d.allocID()
	stride := int32(w * 4) // 4 bytes per pixel
	if err := protocol.CreateBuffer(d.wlConn, poolID, bufferID, 0, int32(w), int32(h), stride, protocol.WlShmPixelFormatArgb8888); err != nil {
		return err
	}

	// if we got to this point, save poolID and bufferID to the output-struct
	output.bufferID = bufferID
	output.poolID = poolID

	// connect buffer to surface
	if err := protocol.Attach(d.wlConn, output.surface, bufferID, 0, 0); err != nil {
		return err
	}
	if err := protocol.Damage(d.wlConn, output.surface, 0, 0, int32(w), int32(h)); err != nil {
		return err
	}
	if err := protocol.Commit(d.wlConn, output.surface); err != nil {
		return err
	}
	return nil
}
