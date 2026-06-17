package protocol

import wl "tissla-wallpaper/internal/wayland"

const (
	wlSurfaceAttach = 1
	wlSurfaceDamage = 2
	wlSurfaceCommit = 6
)

func Attach(wlc wl.Connection, surfaceID, bufferID uint32, x, y int32) error {
	return nil
}

func Damage(wlc wl.Connection, surfaceID uint32, x, y, width, height int32) error {
	return nil
}

func Commit(wlc wl.Connection, surfaceID uint32) error {
	return nil
}
