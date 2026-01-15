package proxy

import (
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"net"
)

const (
	NetworkTCP = 0
	NetworkUDP = 1
)

type Hello struct {
	IP   net.IP
	Port uint16

	Network uint8
}

const helloDataLength = 4 + 4 + // ip + junk
	1 + 1 + 2 // junk + protocol (tcp/udp) + port

func EncodeHello(g *rand.ChaCha8, h *Hello) []byte {
	b := make([]byte, helloDataLength)
	junk := g.Uint64()

	ip := h.IP
	port := h.Port

	b[0] = ip[0]
	b[1] = byte(junk & 0xFF)
	b[2] = ip[1]
	b[3] = byte((junk >> 8) & 0xFF)
	b[4] = ip[2]
	b[5] = byte((junk >> 16) & 0xFF)
	b[6] = ip[3]
	b[7] = byte((junk >> 24) & 0xFF)

	b[8] = byte((junk >> 32) & 0xFF)
	b[9] = h.Network

	binary.LittleEndian.PutUint16(b[10:], uint16(port))
	return b
}

var (
	ErrBadHello   = errors.New("bad hello")
	ErrBadNetwork = errors.New("bad network")
)

func DecodeHello(b []byte, h *Hello) error {
	if len(b) != helloDataLength {
		return ErrBadHello
	}

	network := b[9]
	switch network {
	case NetworkTCP, NetworkUDP:
		// do nothing
	default:
		return ErrBadNetwork
	}

	ip := net.IPv4(b[0], b[2], b[4], b[6])
	port := binary.LittleEndian.Uint16(b[10:])

	h.IP = ip
	h.Port = port
	h.Network = network
	return nil
}
