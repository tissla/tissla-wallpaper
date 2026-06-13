// wayland comm layer
package wayland

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"
)

type WlConnection struct {
	conn *net.UnixConn
	est  time.Time
}

func New() (*WlConnection, error) {

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	display := os.Getenv("WAYLAND_DISPLAY")

	if display == "" {
		display = "wayland-0"
	}

	addr := &net.UnixAddr{
		Name: filepath.Join(runtimeDir, display),
		Net:  "unix",
	}

	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}

	wlConn := &WlConnection{
		conn: conn,
		est:  time.Now().UTC(),
	}

	return wlConn, nil

}

// write to connection
func (w *WlConnection) Write(data []byte) error {

}

// read connection
func (w *WlConnection) Read() ([]byte, error) {

	buf := make([]byte, 8)

	n, err := w.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != 8 {
		return nil, errors.New("invalid msg size")
	}

	return buf, nil
}
