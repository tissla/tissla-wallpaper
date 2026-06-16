package protocol

import "encoding/binary"

type Global struct {
	Name      uint32
	Interface string
	Version   uint32
}

func ParseGlobal(msg []byte) (Global, bool) {
	objectID := binary.LittleEndian.Uint32(msg[0:4])
	sizeAndOp := binary.LittleEndian.Uint32(msg[4:8])
	opcode := uint16(sizeAndOp & 0xFFFF)

	// registry-objektet har id=2
	// global-event har opcode=0
	if objectID != 2 || opcode != 0 {
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
