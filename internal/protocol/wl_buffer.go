package protocol

import wl "tissla-wallpaper/internal/wayland"

const (
	wlBufferDestroy = 0 // request
	WlBufferRelease = 0 // event
)

// DestroyBuffer destroys a wl_buffer. Safe once the compositor has released it.
func DestroyBuffer(bufferID uint32) wl.Message {
	size := 8
	msg := wl.NewMessage(size)
	msg.WriteID(bufferID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlBufferDestroy)
	return msg
}
