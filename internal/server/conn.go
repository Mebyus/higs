package server

import (
	"io"
	"log/slog"
	"net"

	"github.com/mebyus/higs/proxy"
)

type Conn struct {
	// proxy target remote address and network
	// i.e. where to connect this client connection
	hello proxy.Hello

	// proxy target connection
	conn net.Conn

	cid proxy.ConnID

	// incoming packets from client connection
	in chan *proxy.Packet

	// packets with data from target remote
	// we need to relay them to client
	out chan *proxy.Packet

	// signals when connection serve should end
	done chan struct{}

	lg *slog.Logger
}

func serveConn(c *Conn) {
	if c.hello.IP == nil {
		panic("empty remote address")
	}
	lg := c.lg

	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   c.hello.IP,
		Port: int(c.hello.Port),
	})
	if err != nil {
		lg.Error("init conn", slog.String("error", err.Error()))
		return
	}

	c.conn = conn

	go c.serveIncomingPackets(lg)
	go c.serveRemoteReads(lg)

	<-c.done
}

func (c *Conn) serveIncomingPackets(lg *slog.Logger) {
	for {
		select {
		case <-c.done:
			return
		case packet := <-c.in:
			_, err := c.conn.Write(packet.Data)
			if err != nil {
				if err == net.ErrClosed {
					lg.Debug("exit serve incoming packets")
					return
				}
				lg.Error("relay incoming data from client", slog.String("error", err.Error()))
			}
		}
	}
}

func (c *Conn) serveRemoteReads(lg *slog.Logger) {
	var buf [1 << 16]byte
	for {
		n, err := c.conn.Read(buf[:])
		if err != nil {
			if err == io.EOF || err == net.ErrClosed {
				lg.Debug("exit serve remote reads")
				return
			}

			lg.Error("read data from remote", slog.String("error", err.Error()))
			continue
		}

		data := buf[:n]

		var packet proxy.Packet
		packet.PutData(nil, c.cid, data)
		proxy.Encode(&packet)
		// TODO: transform to packet and send to out channel
	}
}
