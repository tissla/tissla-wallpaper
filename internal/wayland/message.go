package wayland

import "encoding/binary"

type Message []byte

func NewMessage(size int) Message {
	return make(Message, size)
}

// read
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

// write
func (m Message) WriteID(id uint32) {
	binary.LittleEndian.PutUint32(m[0:4], id)
}

func (m Message) WriteSize(size uint16) {
	binary.LittleEndian.PutUint16(m[4:6], size)
}
func (m Message) WriteOpcode(opcode uint16) {
	binary.LittleEndian.PutUint16(m[6:8], opcode)
}
