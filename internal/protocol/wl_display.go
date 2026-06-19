package protocol

import (
	"encoding/binary"
	"tissla-wallpaper/internal/wayland"
	wl "tissla-wallpaper/internal/wayland"
)

const (
	wlDisplaySync        = 0 // request
	wlDisplayGetRegistry = 1
	wlDisplayError       = 0 // event
)

func GetRegistry(wlc wl.Connection) error {

	buf := make([]byte, 12)

	// header
	binary.LittleEndian.PutUint32(buf[0:4], DisplayID)
	// upper 16 bites is size, lower is opcode
	binary.LittleEndian.PutUint32(buf[4:8], (12<<16)|wlDisplayGetRegistry)
	// args
	binary.LittleEndian.PutUint32(buf[8:12], RegistryID)

	return wlc.Write(buf)
}

// Sync sends a sync request to the wayland server. When the server emits an event with the callbackID the sync is complete.
func Sync(wlc wl.Connection, callbackID uint32) error {
	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf[0:4], DisplayID)
	binary.LittleEndian.PutUint32(buf[4:8], (12<<16)|wlDisplaySync)
	binary.LittleEndian.PutUint32(buf[8:12], callbackID)
	return wlc.Write(buf)
}

// GetID
func ParseSyncDone(msg wayland.Message, callbackID uint32) bool {
	return msg.ObjectID() == callbackID && msg.Opcode() == wlDisplaySync
}
