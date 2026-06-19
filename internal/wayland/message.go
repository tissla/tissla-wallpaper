package wayland

import "encoding/binary"

type Message []byte

func (m Message) ObjectID() uint32 {
	return binary.LittleEndian.Uint32(m[0:4])
}

func (m Message) Size() uint16 {
	return binary.LittleEndian.Uint16(m[4:6])
}

func (m Message) Opcode() uint16 {
	return binary.LittleEndian.Uint16(m[6:8])
}

func (m Message) Data() []byte {
	return m[8:]
}
