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

func CreateSurface(wlc wl.Connection, compositorID, newSurfaceID uint32) error {
	size := 12 // 8 + 4
	buf := make([]byte, size)

	binary.LittleEndian.PutUint32(buf[0:4], compositorID)
	binary.LittleEndian.PutUint32(buf[4:8], (uint32(size)<<16)|wlCompositorCreateSurface)
	binary.LittleEndian.PutUint32(buf[8:12], newSurfaceID)

	return wlc.Write(buf)
}
