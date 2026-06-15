// Wayland comm layer
package wayland

import (
	"encoding/binary"
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type Connection interface {
	Write([]byte) error
	WriteFd([]byte, int) error
	Read() ([]byte, error)
	Listen() <-chan []byte
}
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

// Write to WlConnection
func (w *WlConnection) Write(data []byte) error {

	_, _, err := w.conn.WriteMsgUnix(data, nil, nil)
	return err
}

// WriteFd writes with fd-argument to oob
func (w *WlConnection) WriteFd(data []byte, fd int) error {
	oob := syscall.UnixRights(fd)
	_, _, err := w.conn.WriteMsgUnix(data, oob, nil)
	return err
}

// Read WlConnection
func (w *WlConnection) Read() ([]byte, error) {

	header := make([]byte, 8)

	n, err := w.conn.Read(header)
	if err != nil {
		return nil, err
	}
	if n != 8 {
		return nil, errors.New("incomplete header")
	}

	// get size
	msgSize := binary.LittleEndian.Uint16(header[4:6])
	if msgSize == 8 {
		return header, nil
	}

	rest := make([]byte, msgSize-8)
	n, err = w.conn.Read(rest)
	if err != nil {
		return nil, err
	}

	if n != int(msgSize-8) {
		return nil, errors.New("incomplete msg")
	}

	return append(header, rest...), nil
}

// Listen starts a go-routine that listens to the WlConnection and writes the messages to a channel. Returns the channel to which messages are written
func (w *WlConnection) Listen() <-chan []byte {
	ch := make(chan []byte)
	go func() {
		for {
			msg, err := w.Read()
			if err != nil {
				close(ch)
				return
			}
			ch <- msg
		}
	}()
	return ch
}
