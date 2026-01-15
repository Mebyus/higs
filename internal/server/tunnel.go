package server

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/mebyus/higs/proxy"
	"github.com/mebyus/higs/wsok"
)

type Tunnel struct {
	conn net.Conn

	// signals when tunnel serve should end
	done chan struct{}

	rb *bufio.Reader
	wb *bufio.Writer

	lg *slog.Logger

	// Protects map with active connections.
	mu sync.RWMutex

	conns map[proxy.ConnID]*Conn

	g *rand.ChaCha8
}

func serveTunnel(t *Tunnel) {
	addr := t.conn.RemoteAddr().String()
	if t.g == nil {
		var seed [32]byte
		time.Now().AppendBinary(seed[:16])
		copy(seed[16:], addr)
		t.g = rand.NewChaCha8(seed)
	}

	lg := t.lg

	lg.Info("new client", slog.String("addr", addr))
	defer lg.Info("drop client")

	defer func() {
		p := recover()
		if p == nil {
			return
		}

		lg.Error("panic", slog.Any("cause", p))
	}()

	go t.serveIncomingFrames(lg)

	<-t.done
}

// serve frames that come from the client
func (t *Tunnel) serveIncomingFrames(lg *slog.Logger) {
	for {
		err := t.readNextFrame()
		if err != nil {
			if err == io.EOF {
				return
			}
			lg.Error("read frame", slog.String("error", err.Error()))
		}
	}
}

func (t *Tunnel) readNextFrame() error {
	var frame wsok.Frame
	err := wsok.Decode(t.rb, &frame)
	if err != nil {
		return err
	}

	var packet proxy.Packet
	err = proxy.Decode(frame.Data, &packet)
	if err != nil {
		return err
	}

	cid := packet.CID
	typ := packet.Type
	c := t.getConn(cid)

	switch typ {
	case proxy.PacketHello:
		if c != nil {
			return fmt.Errorf("hello packet from already existing connection (cid=%s)", cid)
		}

		var hello proxy.Hello
		err = proxy.DecodeHello(packet.Data, &hello)
		if err != nil {
			return err
		}

		c = &Conn{
			cid:   cid,
			in:    make(chan *proxy.Packet, 64),
			lg:    t.lg,
			done:  make(chan struct{}),
			hello: hello,
		}
		t.addConn(c)
		go serveConn(c)
		return nil
	case proxy.PacketData:
		if c == nil {
			return fmt.Errorf("packet from unknown connection (cid=%s)", cid)
		}
		c.in <- &packet
		return nil
	case proxy.PacketClose:
		if c == nil {
			return fmt.Errorf("packet from unknown connection (cid=%s)", cid)
		}
		return nil
	default:
		return fmt.Errorf("unexpected packet type (=%d)", typ)
	}
}

func (t *Tunnel) addConn(c *Conn) {
	t.mu.Lock()
	t.conns[c.cid] = c
	t.mu.Unlock()
}

func (t *Tunnel) getConn(cid proxy.ConnID) *Conn {
	t.mu.RLock()
	c := t.conns[cid]
	t.mu.RUnlock()

	return c
}

func (t *Tunnel) dropConn(cid proxy.ConnID) {
	t.mu.Lock()
	delete(t.conns, cid)
	t.mu.Unlock()
}
