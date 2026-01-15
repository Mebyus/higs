package client

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mebyus/higs/proxy"
)

type Server struct {
	router Router

	listener net.Listener

	// listen address
	address string

	// next accepted connection id
	next uint64

	tunnel *proxy.Tunnel
}

func (s *Server) Init(address string, tunnel *proxy.Tunnel) error {
	address = strings.TrimSpace(address)
	if address == "" {
		panic("empty listen address")
	}

	err := loadRoutesFromFile(&s.router, "routes.txt")
	if err != nil {
		return fmt.Errorf("load routes from file: %v", err)
	}

	s.address = address
	s.tunnel = tunnel
	return nil
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
