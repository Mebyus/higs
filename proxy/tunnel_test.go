package proxy

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"sync"
	"testing"

	"github.com/mebyus/higs/wsok"
)

// EchoSocket test socket implementation. Echoes back all frames with data packets.
type EchoSocket struct {
	mu  sync.Mutex
	buf bytes.Buffer

	r *io.PipeReader
	w *io.PipeWriter
}

func (s *EchoSocket) init() {
	r, w := io.Pipe()
	s.r = r
	s.w = w
	go s.serveEcho()
}

func (s *EchoSocket) Read(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, err := s.buf.Read(b)
	if err == io.EOF {
		return 0, nil
	}
	return n, nil
}

func (s *EchoSocket) Write(b []byte) (int, error) {
	return s.w.Write(b)
}

func (s *EchoSocket) serveEcho() {
	for {
		s.echo()
	}
}

func (s *EchoSocket) echo() {
	var frame wsok.Frame
	err := wsok.Decode(s.r, &frame)
	if err != nil {
		fmt.Printf("[error] decode frame: %v\n", err)
		return
	}

	typ, err := PeekPacketType(frame.Data)
	if err != nil {
		fmt.Printf("[error] peek frame type: %v\n", err)
		return
	}

	if typ != PacketData {
		return
	}

	s.mu.Lock()
	wsok.Encode(&s.buf, &frame)
	s.mu.Unlock()
}

func (s *EchoSocket) Close() error {
	return nil
}

func Test_Tunnel(t *testing.T) {
	var sock EchoSocket
	sock.init()

	tun := newTunnel(&sock, nil)
	go tun.Serve(t.Context())

	g := rand.NewChaCha8([32]byte{0, 1, 2, 3})
	c, err := tun.Hello(g, NewConnID(g), net.ParseIP("127.0.0.1"), 443)
	if err != nil {
		t.Errorf("create new connection: %v", err)
		return
	}

	testData := []byte("123")
	_, err = c.Write(testData)
	if err != nil {
		t.Errorf("write test data to connection: %v", err)
		return
	}

	var buf [1 << 14]byte
	n, err := io.ReadAtLeast(c, buf[:], len(testData))
	if err != nil {
		t.Errorf("read data from connection: %v", err)
		return
	}

	got := buf[:n]
	if !bytes.Equal(got, testData) {
		t.Errorf("read data echo got = %s, want %s", got, testData)
		return
	}
}
