package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

const (
	// requests to wl_shm
	wlShmCreatePool = 0

	// format event
	wlShmFormat = 0

	// pixel format enums
	// maybe script this instead to keep up to date and synced to wayland_server?
	wlShmPixelFormatArgb8888 = 0
	wlShmPixelFormatXrgb8888 = 1
)

// CreatePool implements the wayland protocols create pool request on the wl_shm interface
func CreatePool(wlc wl.Connection, shmID, newID uint32, poolSize, fd int) error {

	size := 16 // 8 + 4 + 4
	msg := wl.NewMessage(size)

	msg.WriteID(shmID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlShmCreatePool)
	binary.LittleEndian.PutUint32(msg[8:12], newID)
	binary.LittleEndian.PutUint32(msg[12:16], uint32(poolSize))

	// fd sent oob
	return wlc.WriteFd(msg, fd)
}

// ParseFormat parses the wayland protocols format event on the wl_shm interface
func ParseFormat(msgData []byte) uint32 {
	return binary.LittleEndian.Uint32(msgData[:4])
}
