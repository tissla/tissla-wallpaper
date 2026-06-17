package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

type Global struct {
	Name      uint32
	Interface string
	Version   uint32
}

// opcodes for wl_registry
const (
	wlRegistryBind         = 0 // request
	wlRegistryGlobal       = 0 // event
	wlRegistryGlobalRemove = 1
)

func ParseGlobal(msg []byte) (Global, bool) {
	objectID := binary.LittleEndian.Uint32(msg[0:4])
	sizeAndOp := binary.LittleEndian.Uint32(msg[4:8])
	opcode := uint16(sizeAndOp & 0xFFFF)

	// global-event har opcode=0
	if objectID != RegistryID || opcode != wlRegistryGlobal {
		return Global{}, false
	}

	name := binary.LittleEndian.Uint32(msg[8:12])

	strLen := binary.LittleEndian.Uint32(msg[12:16])
	iface := string(msg[16 : 16+strLen-1])

	padded := (strLen + 3) &^ 3
	versionOffset := 16 + padded

	version := binary.LittleEndian.Uint32(msg[versionOffset : versionOffset+4])

	return Global{Name: name, Interface: iface, Version: version}, true

}

// ParseGlobalRemove parses the wl_registry.global_remove global-event
// returns ok=false if msg is not global_remove-event
func ParseGlobalRemove(msg []byte) (uint32, bool) {

	objectID := binary.LittleEndian.Uint32(msg[0:4])
	sizeAndOp := binary.LittleEndian.Uint32(msg[4:8])

	opCode := uint16(sizeAndOp & 0xFFFF)

	if objectID != RegistryID || opCode != wlRegistryGlobalRemove {
		return 0, false
	}

	name := binary.LittleEndian.Uint32(msg[8:12])
	return name, true
}

// Bind requests a local handle (newID) for a global the server
// advertised under (name). After this, the client refers to the
// object using newID; the server's "name" is no longer used.
func Bind(wlc wl.Connection, registryID, name, newID uint32, iface string, version uint32) error {
	ifaceBytes := encodeString(iface)

	size := 8 + 4 + len(ifaceBytes) + 4 + 4 // header + name + iface + version + new_id
	buf := make([]byte, size)

	binary.LittleEndian.PutUint32(buf[0:4], registryID)
	binary.LittleEndian.PutUint32(buf[4:8], (uint32(size)<<16)|wlRegistryBind)

	offset := 8
	binary.LittleEndian.PutUint32(buf[offset:offset+4], name)
	offset += 4

	copy(buf[offset:], ifaceBytes)
	offset += len(ifaceBytes)

	binary.LittleEndian.PutUint32(buf[offset:offset+4], version)
	offset += 4

	binary.LittleEndian.PutUint32(buf[offset:offset+4], newID)

	return wlc.Write(buf)
}

func encodeString(s string) []byte {
	n := len(s) + 1
	padded := (n + 3) &^ 3
	b := make([]byte, 4+padded)
	binary.LittleEndian.PutUint32(b[0:4], uint32(n))
	copy(b[4:], s)
	return b
}
