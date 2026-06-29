// Package protocol contains wayland protocol specifics
package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

// opcodes for compositor
const (
	wlCompositorCreateSurface = 0
	wlCompositorCreateRegion  = 1
	wlCompositorRelease       = 2
)

func CreateSurface(compositorID, newSurfaceID uint32) wl.Message {
	size := 12 // 8 + 4

	msg := wl.NewMessage(size)

	msg.WriteID(compositorID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlCompositorCreateSurface)

	binary.LittleEndian.PutUint32(msg[8:12], newSurfaceID)

	return msg
}
