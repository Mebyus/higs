package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
)

var ErrConnClosed = errors.New("connection closed")

// Conn represents a single proxied connection.
type Conn struct {
	// Contains leftover bytes from previous incoming packets.
	// This must be read first before processing new incoming packets.
	inbuf bytes.Buffer

	cid ConnID

	// Total number of bytes received from incoming stream.
	// Counts bytes only from useful payload, excluding proxy protocol overhead.
	received uint64

	// Total number of bytes sent to outgoing stream.
	// Counts bytes only from useful payload, excluding proxy protocol overhead.
	sent uint64

	// Incoming stream of encoded packets.
	in chan []byte

	// Outgoing stream of encoded packets.
	out chan<- []byte

	g *rand.ChaCha8
}

func (c *Conn) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

	if c.inbuf.Len() != 0 {
		// use incoming data buffer, before processing new packets
		n, _ := c.inbuf.Read(b)
		return n, nil
	}

	encoded, ok := <-c.in
	if !ok {
		return 0, ErrConnClosed
	}

	var packet Packet
	err := Decode(encoded, &packet)
	if err != nil {
		return 0, err
	}

	switch packet.Type {
	case PacketHello:
		return 0, fmt.Errorf("incoming hello packet in client connection")
	case PacketClose:
		panic("not implemented")
	case PacketData:
		// proceed further
	default:
		return 0, fmt.Errorf("unexpected packet type (=%d)", packet.Type)
	}

	c.received += uint64(len(packet.Data))

	n := copy(b, packet.Data)
	if n == len(packet.Data) {
		return n, nil
	}

	// save leftover bytes to incoming data buffer
	c.inbuf.Write(packet.Data[n:])

	return 0, nil
}

func (c *Conn) Write(b []byte) (int, error) {
	// TODO: split big writes into smaller packets

	var packet Packet
	packet.PutData(c.g, c.cid, b)

	c.out <- Encode(&packet)

	c.sent += uint64(len(b))
	return len(b), nil
}

func (c *Conn) Close() error {
	return nil
}
