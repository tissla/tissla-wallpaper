package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

const (
	wlSurfaceAttach          = 1
	wlSurfaceDamage          = 2
	wlSurfaceFrame           = 3
	wlSurfaceSetOpaqueRegion = 4
	wlSurfaceSetInputRegion  = 5
	wlSurfaceCommit          = 6
)

// Attach sets a buffer as the content of this surface
// x and y must be 0 when to bound to wl_surface v5 or higher (offseting here is deprecated and moved to wl_surface.offset)
func Attach(surfaceID, bufferID uint32, x, y int32) wl.Message {

	size := 20

	msg := wl.NewMessage(size)

	msg.WriteID(surfaceID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlSurfaceAttach)

	// args
	binary.LittleEndian.PutUint32(msg[8:12], bufferID)
	binary.LittleEndian.PutUint32(msg[12:16], uint32(x))
	binary.LittleEndian.PutUint32(msg[16:20], uint32(y))
	return msg
}

func Damage(surfaceID uint32, x, y, width, height int32) wl.Message {

	size := 24
	msg := wl.NewMessage(size)

	msg.WriteID(surfaceID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlSurfaceDamage)

	// args
	binary.LittleEndian.PutUint32(msg[8:12], uint32(x))
	binary.LittleEndian.PutUint32(msg[12:16], uint32(y))
	binary.LittleEndian.PutUint32(msg[16:20], uint32(width))
	binary.LittleEndian.PutUint32(msg[20:24], uint32(height))
	return msg
}

func Commit(surfaceID uint32) wl.Message {
	size := 8
	msg := wl.NewMessage(size)

	// header
	msg.WriteID(surfaceID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlSurfaceCommit)

	return msg
}
