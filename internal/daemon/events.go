package daemon

import (
	"encoding/binary"
	"log"
	"tissla-wallpaper/internal/protocol"
	wl "tissla-wallpaper/internal/wayland"
)

func (d *Daemon) handleEvent(msg wl.Message) {
	switch msg.ObjectID() {
	case protocol.RegistryID:
		// TODO: hotplug (global / global_remove)
	case protocol.DisplayID:
		// TODO: wl_display.error
	default:

		if _, ok := d.pendingRelease[msg.ObjectID()]; ok {
			if msg.Opcode() == protocol.WlBufferRelease {
				d.releaseBuffer(msg.ObjectID())
			}
			return
		}
		for _, out := range d.outputs {
			if msg.ObjectID() == out.layerSurf {
				if err := d.handleLayerSurfEvent(msg, out); err != nil {
					log.Print(err)
				}
			}
			if msg.ObjectID() == out.id {
				d.handleOutputEvent(msg, out)
			}
		}
	}
}

// collect info about output
func (d *Daemon) handleOutputEvent(msg wl.Message, out *Output) error {
	switch msg.Opcode() {
	case protocol.WlOutputName:
		strLen := binary.LittleEndian.Uint32(msg.Data()[0:4])
		out.hName = string(msg.Data()[4 : 4+strLen-1])
		log.Printf("output name set: %s", out.hName)
	}

	return nil
}
func (d *Daemon) handleLayerSurfEvent(msg wl.Message, out *Output) error {

	switch msg.Opcode() {

	case protocol.ZwlrLayerSurfaceV1Configure:
		serial := binary.LittleEndian.Uint32(msg.Data()[0:4])

		out.width = binary.LittleEndian.Uint32(msg.Data()[4:8])
		out.height = binary.LittleEndian.Uint32(msg.Data()[8:12])

		// sometime after this, use commit
		if err := d.send(protocol.AckConfigure(out.layerSurf, serial)); err != nil {
			return err
		}
	}

	return nil
}
