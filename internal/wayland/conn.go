// Wayland comm layer
package wayland

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
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
	mu   sync.Mutex
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
	w.mu.Lock()
	defer w.mu.Unlock()

	_, _, err := w.conn.WriteMsgUnix(data, nil, nil)
	return err
}

// WriteFd writes with fd-argument to oob
func (w *WlConnection) WriteFd(data []byte, fd int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	oob := syscall.UnixRights(fd)
	_, _, err := w.conn.WriteMsgUnix(data, oob, nil)
	return err
}

// Read WlConnection
func (w *WlConnection) Read() ([]byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(w.conn, header); err != nil {
		return nil, err
	}

	msgSize := Message(header).Size() // läser 6:8, samma källa som skrivsidan
	if msgSize < 8 {
		return nil, fmt.Errorf("invalid message size %d", msgSize)
	}
	if msgSize == 8 {
		return header, nil
	}

	rest := make([]byte, msgSize-8)
	if _, err := io.ReadFull(w.conn, rest); err != nil {
		return nil, err
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
