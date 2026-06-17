package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

func GetRegistry(wlc wl.Connection) error {

	buf := make([]byte, 12)

	// header
	binary.LittleEndian.PutUint32(buf[0:4], DisplayID)
	// upper 16 bites is size, lower is opcode
	binary.LittleEndian.PutUint32(buf[4:8], (12<<16)|1)
	// args
	binary.LittleEndian.PutUint32(buf[8:12], RegistryID)

	return wlc.Write(buf)
}
