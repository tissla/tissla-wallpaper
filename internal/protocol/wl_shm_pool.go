package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

const (
	// requests to wl_shm_pool
	wlShmPoolCreateBuffer = 0
	wlShmPoolDestroy      = 1
	wlShmPoolResize       = 2
)

func CreateBuffer(poolID, newID uint32, offset, width, height, stride int32, format uint32) wl.Message {

	size := 32 //8 + (6 * 4)
	msg := wl.NewMessage(size)

	msg.WriteID(poolID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlShmPoolCreateBuffer)

	binary.LittleEndian.PutUint32(msg[8:12], newID)
	binary.LittleEndian.PutUint32(msg[12:16], uint32(offset))
	binary.LittleEndian.PutUint32(msg[16:20], uint32(width))
	binary.LittleEndian.PutUint32(msg[20:24], uint32(height))
	binary.LittleEndian.PutUint32(msg[24:28], uint32(stride))
	binary.LittleEndian.PutUint32(msg[28:32], format)

	return msg
}

func Destroy(poolID uint32) wl.Message {
	size := 8

	msg := wl.NewMessage(size)
	msg.WriteID(poolID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlShmPoolDestroy)

	return msg
}

func Resize(poolID uint32, newSize int) wl.Message {

	size := 12 // 8 + 4
	msg := wl.NewMessage(size)

	msg.WriteID(poolID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(wlShmPoolResize)

	binary.LittleEndian.PutUint32(msg[8:12], uint32(newSize))

	return msg

}
