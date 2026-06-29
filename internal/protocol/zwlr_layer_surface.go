package protocol

import (
	"encoding/binary"
	wl "tissla-wallpaper/internal/wayland"
)

// opcodes
const (
	ZwlrLayerSurfaceV1SetSize                  = 0
	ZwlrLayerSurfaceV1SetAnchor                = 1
	ZwlrLayerSurfaceV1SetExlusiveZone          = 2
	ZwlrLayerSurfaceV1SetMargin                = 3
	ZwlrLayerSurfaceV1SetKeyboardInteractivity = 4
	ZwlrLayerSurfaceV1GetPopup                 = 5

	ZwlrLayerSurfaceV1AckConfigure = 6
	ZwlrLayerSurfaceV1Destroy      = 7

	ZwlrLayerSurfaceV1SetLayer        = 8
	ZwlrLayerSurfaceV1SetExlusiveEdge = 9

	// events
	ZwlrLayerSurfaceV1Configure = 0
	ZwlrLayerSurfaceV1Closed    = 1

	AnchorTop    uint32 = 1
	AnchorBottom uint32 = 2
	AnchorLeft   uint32 = 3
	AnchorRight  uint32 = 4
)

func AckConfigure(wlc wl.Connection, layerSurfID, serial uint32) error {

	size := 12
	msg := wl.NewMessage(size)

	msg.WriteID(layerSurfID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(ZwlrLayerSurfaceV1AckConfigure)
	binary.LittleEndian.PutUint32(msg[8:12], serial)

	return wlc.Write(msg)
}

// should always be 0, 0 for fullscreen?
func SetSize(wlc wl.Connection, layerSurfID uint32, width, height uint32) error {
	size := 16
	msg := wl.NewMessage(size)
	msg.WriteID(layerSurfID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(ZwlrLayerSurfaceV1SetSize)
	binary.LittleEndian.PutUint32(msg[8:12], width)
	binary.LittleEndian.PutUint32(msg[12:16], height)
	return wlc.Write(msg)
}

// can be either?
func SetAnchor(wlc wl.Connection, layerSurfID uint32, anchor uint32) error {
	size := 12
	msg := wl.NewMessage(size)
	msg.WriteID(layerSurfID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(ZwlrLayerSurfaceV1SetAnchor)
	binary.LittleEndian.PutUint32(msg[8:12], anchor)
	return wlc.Write(msg)
}

// should be -1?
func SetExclusiveZone(wlc wl.Connection, layerSurfID uint32, zone int32) error {
	size := 12
	msg := wl.NewMessage(size)
	msg.WriteID(layerSurfID)
	msg.WriteSize(uint16(size))
	msg.WriteOpcode(ZwlrLayerSurfaceV1SetExlusiveZone)
	binary.LittleEndian.PutUint32(msg[8:12], uint32(zone))
	return wlc.Write(msg)
}
