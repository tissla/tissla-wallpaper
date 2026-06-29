package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

const (
	wlDisplaySync        = 0 // request
	wlDisplayGetRegistry = 1
	wlDisplayError       = 0 // event
)

func GetRegistry() wl.Message {

	size := 12
	msg := wl.NewMessage(size)

	// header

	msg.WriteID(DisplayID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlDisplayGetRegistry)

	//args
	binary.LittleEndian.PutUint32(msg[8:12], RegistryID)

	return msg
}

// Sync sends a sync request to the wayland server. When the server emits an event with the callbackID the sync is complete.
func Sync(callbackID uint32) wl.Message {
	size := 12
	msg := wl.NewMessage(size)
	msg.WriteID(DisplayID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlDisplaySync)

	//TODO: abstract the arg part
	binary.LittleEndian.PutUint32(msg[8:12], callbackID)

	return msg
}

func ParseSyncDone(msg wl.Message, callbackID uint32) bool {
	return msg.ObjectID() == callbackID && msg.Opcode() == 0
}
