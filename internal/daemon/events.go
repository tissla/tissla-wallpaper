package daemon

import (
	"encoding/binary"
	"log"
	"tissla-wallpaper/internal/protocol"
	wl "tissla-wallpaper/internal/wayland"
)

// HandleEvents handles events fired by the wayland server
func (d *Daemon) HandleEvents() {
	for raw := range d.wlConn.Listen() {

		msg := wl.Message(raw)
		switch msg.ObjectID() {

		case protocol.RegistryID:

		case protocol.DisplayID:

		default:
			for _, out := range d.outputs {
				if msg.ObjectID() == out.layerSurf {

					err := d.handleLayerSurfEvent(msg, out)
					if err != nil {
						log.Print(err)
					}
				}
				if msg.ObjectID() == out.id {
					d.handleOutputEvent(msg, out)
				}
			}
		}

	}
}

// collect info about output
func (d *Daemon) handleOutputEvent(msg wl.Message, out *Output) error {
	switch msg.Opcode() {
	case 6:
		strLen := binary.LittleEndian.Uint32(msg.Data()[0:4])
		out.hName = string(msg.Data()[4 : 4+strLen-1])
	}

	return nil
}
func (d *Daemon) handleLayerSurfEvent(msg wl.Message, out *Output) error {

	switch msg.Opcode() {

	case protocol.ZwlrLayerSurfaceV1Configure:
		serial := binary.LittleEndian.Uint32(msg.Data()[0:4])
		width := binary.LittleEndian.Uint32(msg.Data()[4:8])
		height := binary.LittleEndian.Uint32(msg.Data()[8:12])

		out.width = width
		out.height = height

		// sometime after this, use commit
		if err := protocol.AckConfigure(d.wlConn, out.layerSurf, serial); err != nil {
			return err
		}
	}

	return nil
}
