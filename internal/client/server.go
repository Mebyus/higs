package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/mebyus/higs/proxy"
)

type Server struct {
	resolver *Resolver
	router   *Router

	listener net.Listener

	// listen address
	address string

	// next accepted connection id
	next uint64

	tunnel *proxy.Tunnel

	lg *zap.Logger
}

func relayData(client, backend Socket) {
	go func() {
		io.Copy(backend, client)
		backend.Close()
	}()
	io.Copy(client, backend)
}

func RunLocalServer(ctx context.Context, lg *zap.Logger, config *Config, tunnel *proxy.Tunnel, resolver *Resolver, router *Router) error {
	lg = lg.Named("local")

	port := config.LocalTCPPort
	address := fmt.Sprintf(":%d", port)

	var server Server
	server.address = address
	server.tunnel = tunnel
	server.lg = lg
	server.resolver = resolver
	server.router = router

	go func() {
		err := server.ListenUDP(ctx, config.LocalUDPPort)
		if err != nil {
			lg.Error("listen udp", zap.Error(err))
		}
	}()

	err := server.ListenAndHandleConnections(ctx)
	if err != nil {
		return fmt.Errorf("listen on %d port: %v\n", port, err)
	}
	return nil
}

func getRemotePort(conn net.Conn) int {
	_, s, _ := net.SplitHostPort(conn.RemoteAddr().String())
	port, _ := strconv.ParseInt(s, 10, 64)
	return int(port)
}

func (s *Server) ListenUDP(ctx context.Context, port uint16) error {
	lg := s.lg.Named("udp")

	listener, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	done := ctx.Done()
	for {
		_, ok := <-done
		if ok {
			return nil
		}

		var buf [1 << 12]byte
		n, addr, err := listener.ReadFrom(buf[:])
		if err != nil {
			lg.Error("read data", zap.Error(err))
			continue
		}

		data := buf[:n]
		os.WriteFile(fmt.Sprintf(".out/%s-%d.dump", addr, time.Now().UnixMicro()), data, 0o655)
	}
}

func (s *Server) ListenAndHandleConnections(ctx context.Context) error {
	if s.address == "" {
		panic("empty listen address")
	}

	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen %s: %v", s.address, err)
	}
	s.listener = listener
	defer listener.Close()

	done := ctx.Done()
	go s.acceptAndHandleConnections(done)

	<-done
	return nil
}

func (s *Server) acceptAndHandleConnections(done <-chan struct{}) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			_, ok := <-done
			if !ok {
				return
			}

			fmt.Printf("Error accepting connection: %v (%T)\n", err, err)
			continue
		}

		id := s.next
		s.next += 1

		go s.handleConnection(&Connection{
			in: conn,
			id: id,
		}, done)
	}
}

func (s *Server) handleConnection(c *Connection, done <-chan struct{}) {
	defer c.in.Close()

	clientAddress := c.in.RemoteAddr()
	ap, procName, err := getOriginalDestination(c.in)
	if err != nil {
		fmt.Printf("get original destination %s: %v\n", clientAddress, err)
		return
	}

	fmt.Printf("accepted connection (id=%d) from %s (%s) to %s\n", c.id, procName, clientAddress, ap)

	var out Socket

	act := s.router.Lookup(ap.Addr())
	switch act {
	case ActionDirect, ActionAuto:
		destConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
			IP:   net.IP(ap.Addr().AsSlice()),
			Port: int(ap.Port()),
		})
		if err != nil {
			fmt.Printf("direct destination %s dial: %v\n", ap, err)
			return
		}
		fmt.Printf("new direct connection (id=%d) from %v to %s established\n", c.id, clientAddress, ap)
		out = destConn
	case ActionProxy:
		var f FileConn
		err := f.Init(filepath.Join("tcp", strconv.FormatUint(c.id, 10)))
		if err != nil {
			fmt.Printf("create dump file for connection (id=%d): %v\n", c.id, err)
			return
		}
		fmt.Printf("new file dump for connection (id=%d) created\n", c.id)
		out = &f
	case ActionBlock:
		return
	default:
		panic(fmt.Sprintf("unexpected action (=%d)", act))
	}

	c.out = out
	defer out.Close()

	relayData(c.in, out)

	fmt.Printf("relay of connection (id=%d) ended\n", c.id)
}

// Socket represents a two-way full duplex connection between client and server.
type Socket interface {
	Read([]byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
}

// Connection represents active connection between client (that is being proxied or directed)
// and proxy server or direct destination.
type Connection struct {
	// outgoing server connection
	out Socket

	// incoming client connection
	in net.Conn

	id uint64
}
