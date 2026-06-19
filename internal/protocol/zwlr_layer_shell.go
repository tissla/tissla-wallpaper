package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

const (
	zwlrLayerShellV1GetLayerSurface = 0
	zwlrLayerShellV1Destroy         = 1

	ZwlrLayerBackground = 0
	ZwlrLayerBottom     = 1
	ZwlrLayerTop        = 2
	ZwlrLayerOverlay    = 3
)

func GetLayerSurface(wlc wl.Connection, layerShellID, newID, surface, output, layer uint32, namespace string) error {

	nsBytes := encodeString(namespace)

	size := 24 + len(nsBytes)

	msg := wl.NewMessage(size)

	msg.WriteID(layerShellID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(zwlrLayerShellV1GetLayerSurface)

	binary.LittleEndian.PutUint32(msg[8:12], newID)
	binary.LittleEndian.PutUint32(msg[12:16], surface)
	binary.LittleEndian.PutUint32(msg[16:20], output)
	binary.LittleEndian.PutUint32(msg[20:24], layer)

	copy(msg[24:], nsBytes)

	return wlc.Write(msg)
}
