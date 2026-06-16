// Package ipc
package ipc

import (
	"os"
	"path/filepath"
)

func SocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	return filepath.Join(runtimeDir, "tissla-wallpaperd.sock")
}
