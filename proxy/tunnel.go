package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/mebyus/higs/wsok"
)

// Socket represents a two-way full duplex connection between client and server.
type Socket interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

// Tunnel proxies multiple connections via a single network socket.
type Tunnel struct {
	sock Socket

	rb *bufio.Reader
	wb *bufio.Writer

	// Incoming encoded packets (from active client connections).
	// These will be transformed into frames and sent to network socket.
	in chan []byte

	// Protects map with active connections.
	mu sync.RWMutex

	conns map[ConnID]*Conn

	g *rand.ChaCha8
}

func newTunnel(sock Socket, g *rand.ChaCha8) *Tunnel {
	if g == nil {
		var seed [32]byte
		time.Now().AppendBinary(seed[:16])
		g = rand.NewChaCha8(seed)
	}

	return &Tunnel{
		sock:  sock,
		rb:    bufio.NewReader(sock),
		wb:    bufio.NewWriter(sock),
		in:    make(chan []byte, 256),
		conns: make(map[ConnID]*Conn, 32),
		g:     g,
	}
}

// Serve start handling incoming (from proxy server) and outgoing (from client connections)
// packets. Blocks until supplied context is cancelled.
//
// Should be called after Tunnel is created and before any other methods are used.
func (t *Tunnel) Serve(ctx context.Context) error {
	done := ctx.Done()

	go t.closeWhenDone(done)
	go t.serveIn(done)
	go t.serveOut()

	<-done
	return nil
}

func (t *Tunnel) serveIn(done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case data := <-t.in:
			err := t.sendFrame(data)
			if err != nil {
				fmt.Printf("[error] send frame: %v\n", err)
			}
		}
	}
}

func (t *Tunnel) sendFrame(data []byte) error {
	frame := wsok.Frame{
		Data:    data,
		Op:      wsok.OpBin,
		Fin:     true,
		UseMask: true,
	}
	putFrameMask(t.g, &frame)

	err := wsok.Encode(t.wb, &frame)
	if err != nil {
		return err
	}
	return t.wb.Flush()
}

func (t *Tunnel) serveOut() {
	for {
		err := t.readNextFrame()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Printf("[error] read frame: %v\n", err)
		}
	}
}

func (t *Tunnel) closeWhenDone(done <-chan struct{}) {
	<-done
	err := t.sock.Close()
	if err != nil {
		fmt.Printf("[error] close proxy server socket: %v\n", err)
	}
}

func (t *Tunnel) readNextFrame() error {
	var frame wsok.Frame
	err := wsok.Decode(t.rb, &frame)
	if err != nil {
		return err
	}

	cid, err := PeekConnID(frame.Data)
	if err != nil {
		return err
	}

	c := t.getConn(cid)
	if c == nil {
		return fmt.Errorf("packet from unknown connection (cid=%s)", cid)
	}

	c.in <- frame.Data
	return nil
}

func (t *Tunnel) Hello(g *rand.ChaCha8, cid ConnID, ip net.IP, port int) (*Conn, error) {
	var packet Packet
	packet.PutHelloTCP(g, cid, ip, port)

	frame := wsok.Frame{
		Data:    Encode(&packet),
		Op:      wsok.OpBin,
		Fin:     true,
		UseMask: true,
	}
	putFrameMask(g, &frame)

	in := make(chan []byte, 64)
	c := Conn{
		cid: cid,
		g:   g,
		in:  in,
		out: t.in,
	}

	t.addConn(&c)
	err := wsok.Encode(t.wb, &frame) // TODO: fix Socket usage race
	if err != nil {
		t.dropConn(cid)
		return nil, err
	}
	err = t.wb.Flush()
	if err != nil {
		t.dropConn(cid)
		return nil, err
	}

	return &c, nil
}

func (t *Tunnel) addConn(c *Conn) {
	t.mu.Lock()
	t.conns[c.cid] = c
	t.mu.Unlock()
}

func (t *Tunnel) getConn(cid ConnID) *Conn {
	t.mu.Lock()
	c := t.conns[cid]
	t.mu.Unlock()

	return c
}

func (t *Tunnel) dropConn(cid ConnID) {
	t.mu.Lock()
	delete(t.conns, cid)
	t.mu.Unlock()
}

func (t *Tunnel) Drop(cid ConnID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	c, ok := t.conns[cid]
	if !ok {
		panic(fmt.Sprintf("connection (cid=%s) not found", cid))
	}

	close(c.in)
	delete(t.conns, cid)

	return nil
}

func putFrameMask(g *rand.ChaCha8, f *wsok.Frame) {
	g.Read(f.Mask[:])
}
