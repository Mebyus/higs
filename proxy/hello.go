package proxy

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/netip"
)

const (
	NetworkTCP = 0
	NetworkUDP = 1
)

type Hello struct {
	AddrPort netip.AddrPort

	junk [8]byte

	Network uint8

	ok bool
}

func (h *Hello) InitEncode(g *rand.ChaCha8, network uint8, ap netip.AddrPort) {
	putJunk(g, h.junk[:])
	h.AddrPort = ap
	h.Network = network
	h.ok = true
}

func EncodeHello(h *Hello, buf []byte) []byte {
	if !h.ok {
		panic("no init")
	}

	c := encoder{buf: buf}
	return c.hello(h)
}

const (
	IPv4 = 0
	IPv6 = 1
)

func (c *encoder) hello(h *Hello) []byte {
	addr := h.AddrPort.Addr()
	port := h.AddrPort.Port()
	bl := addr.BitLen()
	var typ uint8
	switch bl {
	case 32:
		typ = IPv4
	case 128:
		typ = IPv6
	default:
		panic(fmt.Sprintf("unexpected address bit length (=%d)", bl))
	}

	c.putb((h.junk[0] & 0b11111100) | h.Network)
	c.putb((h.junk[1] & 0b11111100) | typ)
	c.u16(port)

	switch typ {
	case IPv4:
		a := addr.As4()

		c.putb(a[0])
		c.putb(h.junk[2])

		c.putb(a[1])
		c.putb(h.junk[3])

		c.putb(a[2])
		c.putb(h.junk[4])

		c.putb(a[3])
		c.putb(h.junk[5])
	case IPv6:
		panic("stub")
	default:
		panic(fmt.Sprintf("unexpected address type (=%d)", typ))
	}

	return c.buf
}

var (
	ErrAddrType  = errors.New("bad type")
	ErrHelloSize = errors.New("bad size")
	ErrNetwork   = errors.New("bad network")
)

const minHelloLength = 1 + // network
	1 + // address type
	2 + // port
	8 // address + junk (mixed)

func DecodeHello(h *Hello, data []byte) error {
	d := decoder{buf: data}
	return d.hello(h)
}

func (d *decoder) hello(h *Hello) error {
	const debug = false

	if d.len() < minHelloLength {
		return ErrHelloSize
	}

	network := d.u8() & 0b11
	switch network {
	case NetworkTCP, NetworkUDP:
		// continue execution
	default:
		return ErrNetwork
	}

	typ := d.u8() & 0b11
	switch typ {
	case IPv4, IPv6:
		// continue execution
	default:
		return ErrAddrType
	}

	port := d.u16()

	var ip netip.Addr
	switch typ {
	case IPv4:
		var a [4]byte
		b := d.bytes(8)
		a[0] = b[0]
		a[1] = b[2]
		a[2] = b[4]
		a[3] = b[6]
		ip = netip.AddrFrom4(a)
	case IPv6:
		panic("stub")
	default:
		panic(fmt.Sprintf("unexpected address type (=%d)", typ))
	}
	ap := netip.AddrPortFrom(ip, port)

	h.AddrPort = ap
	h.Network = network
	return nil
}
