package wayland

import (
	"bytes"
	"net"
	"testing"
	"time"
)

// buildMessage assembles a wire message using the production builders, so the
// test exercises the same encoding path the daemon uses.
func buildMessage(id uint32, opcode uint16, payload []byte) Message {
	size := uint16(8 + len(payload))
	msg := NewMessage(int(size))
	msg.WriteID(id)
	msg.WriteSize(size)
	msg.WriteOpcode(opcode)
	copy(msg[8:], payload)
	return msg
}

// TestReadMessageFraming sends a message across a net.Pipe and asserts the
// framing reassembles it byte-for-byte. The write side and the read side are
// independent code paths, so this catches a size parsed from the wrong header
// offset (the bug we fixed): a misread length yields the wrong byte count.
func TestReadMessageFraming(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	server.SetReadDeadline(time.Now().Add(2 * time.Second))

	want := buildMessage(7, 1, []byte{0xDE, 0xAD, 0xBE, 0xEF})

	go client.Write(want)

	got, err := readMessage(server)
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if Message(got).Size() != Message(want).Size() {
		t.Errorf("framed size: got %d, want %d", Message(got).Size(), Message(want).Size())
	}
	if !bytes.Equal(got, want) {
		t.Errorf("framed bytes mismatch:\n got %x\nwant %x", got, want)
	}
}

// TestReadMessageHeaderOnly covers the size==8 fast path (no payload).
func TestReadMessageHeaderOnly(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	server.SetReadDeadline(time.Now().Add(2 * time.Second))

	want := buildMessage(1, 0, nil)

	go client.Write(want)

	got, err := readMessage(server)
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("len: got %d, want 8", len(got))
	}
	if !bytes.Equal(got, want) {
		t.Errorf("header mismatch:\n got %x\nwant %x", got, want)
	}
}

// TestReadMessagePartialWrites dribbles one message out in two writes, splitting
// the header itself. net.Pipe hands the reader exactly what each Write provides,
// so a reader doing a single Read per section would come up short. Passing here
// is what proves the framing uses io.ReadFull rather than a lone Read.
func TestReadMessagePartialWrites(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	server.SetReadDeadline(time.Now().Add(2 * time.Second))

	want := buildMessage(3, 2, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})

	go func() {
		client.Write(want[:5]) // partial header
		client.Write(want[5:]) // rest of header + payload
	}()

	got, err := readMessage(server)
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("partial reassembly mismatch:\n got %x\nwant %x", got, want)
	}
}

// TestReadMessageTruncated feeds a header promising more bytes than ever arrive;
// closing the writer must surface as an error, not a hang or a short message.
func TestReadMessageTruncated(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	server.SetReadDeadline(time.Now().Add(2 * time.Second))

	hdr := buildMessage(1, 0, []byte{0, 0, 0, 0}) // claims size 12
	go func() {
		client.Write(hdr[:8]) // header only, then hang up
		client.Close()
	}()

	if _, err := readMessage(server); err == nil {
		t.Error("expected error on truncated message, got nil")
	}
}
