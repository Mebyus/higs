package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
)

// ConnID connection id.
type ConnID [16]byte

func (c ConnID) String() string {
	return fmt.Sprintf("%x-%x", c[:8], c[8:])
}

func NewConnID(g *rand.ChaCha8) ConnID {
	var cid ConnID
	g.Read(cid[:])
	return cid
}

type PacketType uint8

const (
	PacketHello PacketType = iota
	PacketClose
	PacketData
)

var ErrBadPacketType = errors.New("bad packet type")

func (t PacketType) Valid() error {
	if t > PacketData {
		return ErrBadPacketType
	}
	return nil
}

// Packet carries a single packet of connection data between proxy client and server.
type Packet struct {
	Data []byte

	// connection id
	CID ConnID

	// junk padding at packet start
	junk1 [8]byte

	// junk padding at packet end
	junk2 [8]byte

	// reserved
	rsv [3]byte

	Type PacketType
}

func (p *Packet) PutHello(g *rand.ChaCha8, cid ConnID, h *Hello) {
	ip := h.IP
	a := ip.To4()
	if a == nil {
		panic(fmt.Sprintf("unexpected ip address: %v", ip))
	}

	p.CID = cid
	p.Data = EncodeHello(g, h)
	p.Type = PacketHello
	p.PutJunk(g)
}

func (p *Packet) PutData(g *rand.ChaCha8, cid ConnID, data []byte) {
	p.CID = cid
	p.Data = data
	p.Type = PacketData
	p.PutJunk(g)
}

const minPacketLength = 2 + 2 + // prefix + suffix
	8 + 16 + 8 + 4 // junk1 + cid + junk2 + typ + reserved

// Get connection id from encoded packet without fully decoding it.
func PeekConnID(b []byte) (ConnID, error) {
	if len(b) < minPacketLength {
		return ConnID{}, ErrBadPacket
	}

	var cid ConnID
	copy(cid[:], b[10:26])
	return cid, nil
}

func PeekPacketType(b []byte) (PacketType, error) {
	if len(b) < minPacketLength {
		return 0, ErrBadPacket
	}

	return PacketType(b[29] - '0'), nil
}

func Encode(p *Packet) []byte {
	// resulting length of encoded packet
	var n = minPacketLength + len(p.Data)

	b := make([]byte, n)

	if p.Type == PacketData {
		b[0] = '{'
		b[1] = '"'

		b[n-2] = '"'
		b[n-1] = '}'
	} else {
		b[0] = '['
		b[1] = '"'

		b[n-2] = '"'
		b[n-1] = ']'
	}

	copy(b[2:10], p.junk1[:])
	copy(b[10:26], p.CID[:])
	copy(b[26:29], p.rsv[:])
	b[29] = uint8(p.Type) + '0'
	copy(b[30:n-10], p.Data)
	copy(b[n-10:], p.junk2[:])

	return b
}

var ErrBadPacket = errors.New("bad packet")

func Decode(b []byte, p *Packet) error {
	if len(b) < minPacketLength {
		return ErrBadPacket
	}

	typ := PacketType(b[29] - '0')
	err := typ.Valid()
	if err != nil {
		return err
	}

	n := len(b)
	copy(p.junk1[:], b[2:10])
	copy(p.CID[:], b[10:26])
	copy(p.rsv[:], b[26:29])
	p.Type = typ
	copy(p.junk2[:], b[n-10:n-2])
	p.Data = bytes.Clone(b[30 : n-10])
	return nil
}

func putJunk(g *rand.ChaCha8, b []byte) {
	g.Read(b)
}

func putTextJunk(g *rand.ChaCha8, b []byte) {
	putJunk(g, b)
	for i := range len(b) {
		b[i] = mapByteAlphanum(b[i])
	}
}

const mapText = `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_:`

func mapByteAlphanum(b byte) byte {
	return mapText[b&0x3F]
}

func (p *Packet) PutJunk(g *rand.ChaCha8) {
	putTextJunk(g, p.junk1[:])
	putTextJunk(g, p.rsv[:])
	putTextJunk(g, p.junk2[:])
}
