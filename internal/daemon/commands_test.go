package daemon

import (
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"tissla-wallpaper/internal/ipc"
)

// fakeConn satisfies wl.Connection without a real compositor: events never fire
// and writes are no-ops, so the loop only ever reacts to commands.
type fakeConn struct{}

func (fakeConn) Write([]byte) error        { return nil }
func (fakeConn) WriteFd([]byte, int) error { return nil }
func (fakeConn) Read() ([]byte, error)     { select {} } // block forever
func (fakeConn) Listen() <-chan []byte     { return make(chan []byte) }

func TestHandleCommandMonitors(t *testing.T) {
	d := &Daemon{initialized: true, outputs: []*Output{
		{hName: "DP-1", width: 1920, height: 1080},
		{hName: "HDMI-A-1", width: 2560, height: 1440},
	}}
	resp, err := d.handleCommand(command{verb: "monitors"})
	if err != nil {
		t.Fatalf("monitors: %v", err)
	}
	for _, want := range []string{"DP-1 - 1920x1080", "HDMI-A-1 - 2560x1440"} {
		if !strings.Contains(resp, want) {
			t.Errorf("monitors response %q missing %q", resp, want)
		}
	}
}

func TestHandleCommandUnknownVerb(t *testing.T) {
	d := &Daemon{initialized: true}
	if _, err := d.handleCommand(command{verb: "bogus"}); err == nil {
		t.Error("expected error for unknown verb")
	}
}

func TestHandleCommandNotInitialized(t *testing.T) {
	d := &Daemon{initialized: false}
	if _, err := d.handleCommand(command{verb: "monitors"}); err == nil {
		t.Error("expected error when daemon not initialized")
	}
}

// --- integration test of the full request/reply path over a real unix socket ---

func dialWithRetry(t *testing.T, path string) net.Conn {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("unix", path); err == nil {
			return conn
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("listener never came up")
	return nil
}

// TestRequestReplyRoundTrip is the one that proves the reply channel wiring:
// client writes a request, the loop produces a response, the socket handler
// shuttles it back, the client reads it to EOF.
func TestRequestReplyRoundTrip(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	d := &Daemon{
		wlConn:      fakeConn{},
		commands:    make(chan command),
		initialized: true,
		outputs:     []*Output{{hName: "DP-1", width: 1920, height: 1080}},
	}
	go d.Run()
	go d.HandleCommands()

	conn := dialWithRetry(t, ipc.SocketPath())
	defer conn.Close()

	req, _ := json.Marshal(ipc.Request{Verb: "monitors"})
	if _, err := conn.Write(req); err != nil {
		t.Fatalf("write: %v", err)
	}
	resp, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(resp), "DP-1 - 1920x1080") {
		t.Errorf("round-trip response = %q", resp)
	}
}

// TestRequestReplyParseError: a malformed request is rejected by the socket
// handler before reaching the loop, and the client still gets an error string.
func TestRequestReplyParseError(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	d := &Daemon{
		wlConn:      fakeConn{},
		commands:    make(chan command),
		initialized: true,
	}
	go d.Run()
	go d.HandleCommands()

	conn := dialWithRetry(t, ipc.SocketPath())
	defer conn.Close()

	req, _ := json.Marshal(ipc.Request{Verb: "frobnicate"})
	conn.Write(req)
	resp, _ := io.ReadAll(conn)
	if !strings.HasPrefix(string(resp), "error:") {
		t.Errorf("expected error response, got %q", resp)
	}
}

func TestReleaseBufferDestroysAndRemoves(t *testing.T) {
	d := &Daemon{wlConn: fakeConn{}, buffers: map[uint32]uint32{10: 11}}
	d.releaseBuffer(10)
	if _, ok := d.buffers[10]; ok {
		t.Errorf("buffer 10 still tracked after release")
	}
}

func TestReleaseBufferUnknownIsNoop(t *testing.T) {
	d := &Daemon{wlConn: fakeConn{}, buffers: map[uint32]uint32{}}
	d.releaseBuffer(999) // must not panic or add anything
	if len(d.buffers) != 0 {
		t.Errorf("unexpected entries: %v", d.buffers)
	}
}

// Releasing an output's current buffer must forget it, since the compositor now
// owns the displayed copy and our buffer is gone.
func TestReleaseBufferZeroesCurrentOutput(t *testing.T) {
	out := &Output{bufferID: 10, poolID: 11}
	d := &Daemon{wlConn: fakeConn{}, buffers: map[uint32]uint32{10: 11}, outputs: []*Output{out}}
	d.releaseBuffer(10)
	if out.bufferID != 0 || out.poolID != 0 {
		t.Errorf("output not zeroed: buffer=%d pool=%d", out.bufferID, out.poolID)
	}
}

// clearOutput blanks and zeroes the output; destruction is release-driven, so
// it does not touch the buffers map itself.
func TestClearOutputZeroesOutput(t *testing.T) {
	out := &Output{surface: 3, bufferID: 10, poolID: 11}
	d := &Daemon{wlConn: fakeConn{}, buffers: map[uint32]uint32{10: 11}, outputs: []*Output{out}}
	if err := d.clearOutput(out); err != nil {
		t.Fatalf("clearOutput: %v", err)
	}
	if out.bufferID != 0 || out.poolID != 0 {
		t.Errorf("output not zeroed: buffer=%d pool=%d", out.bufferID, out.poolID)
	}
}
