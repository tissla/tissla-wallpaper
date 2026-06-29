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

func ParseGlobal(msg wl.Message) (Global, bool) {

	// global-event har opcode=0
	if msg.ObjectID() != RegistryID || msg.Opcode() != wlRegistryGlobal {
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
func ParseGlobalRemove(msg wl.Message) (uint32, bool) {

	if msg.ObjectID() != RegistryID || msg.Opcode() != wlRegistryGlobalRemove {
		return 0, false
	}

	name := binary.LittleEndian.Uint32(msg[8:12])
	return name, true
}

// Bind requests a local handle (newID) for a global the server
// advertised under (name). After this, the client refers to the
// object using newID; the server's "name" is no longer used.
func Bind(registryID, name, newID uint32, iface string, version uint32) wl.Message {
	ifaceBytes := encodeString(iface)

	size := 8 + 4 + len(ifaceBytes) + 4 + 4 // header + name + iface + version + new_id

	msg := wl.NewMessage(size)

	msg.WriteID(registryID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlRegistryBind)

	offset := 8
	binary.LittleEndian.PutUint32(msg[offset:offset+4], name)
	offset += 4

	copy(msg[offset:], ifaceBytes)
	offset += len(ifaceBytes)

	binary.LittleEndian.PutUint32(msg[offset:offset+4], version)
	offset += 4

	binary.LittleEndian.PutUint32(msg[offset:offset+4], newID)

	return msg
}

func encodeString(s string) []byte {
	n := len(s) + 1
	padded := (n + 3) &^ 3
	b := make([]byte, 4+padded)
	binary.LittleEndian.PutUint32(b[0:4], uint32(n))
	copy(b[4:], s)
	return b
}
