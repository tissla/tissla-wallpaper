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
	return readMessage(w.conn)
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

// readMessage frames one Wayland message from r: read the 8-byte header, parse
// the length out of it, then read exactly that many more bytes. io.ReadFull
// means a message split across several reads on the stream is reassembled
// correctly instead of being truncated.
func readMessage(r io.Reader) ([]byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	size := Message(header).Size() // läser 6:8 — samma källa som skrivsidan
	if size < 8 {
		return nil, fmt.Errorf("invalid message size %d", size)
	}
	if size == 8 {
		return header, nil
	}

	msg := make([]byte, size)
	copy(msg, header)
	if _, err := io.ReadFull(r, msg[8:]); err != nil {
		return nil, err
	}
	return msg, nil
}
