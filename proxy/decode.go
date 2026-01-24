package proxy

import (
	"errors"
	"fmt"
)

// Decode packet from wire format. Correct packet salt must be
// set before decoding.
func Decode(p *Packet, data []byte) error {
	if !p.ok {
		panic("no init")
	}

	d := decoder{buf: data}
	return d.packet(p)
}

type decoder struct {
	buf []byte
	pos int
}

const minPacketLength = 2 + // style prefix
	8 + 1 + // prefix junk
	3 + 1 + // reserved junk + type
	16 + // cid
	0 + // packet data
	4 + // control sum
	8 + 1 + // suffix junk
	2 // style suffix

var (
	ErrPacketSize  = errors.New("bad size")
	ErrPacketStyle = errors.New("bad style")
	ErrPacketSum   = errors.New("bad sum")
)

func (d *decoder) packet(p *Packet) error {
	const debug = false

	if d.len() < minPacketLength {
		return ErrPacketSize
	}

	var s1 Style
	switch d.u8() {
	case '{':
		s1 = Style0
	case '[':
		s1 = Style1
	default:
		return ErrPacketStyle
	}
	if d.u8() != '"' {
		return ErrPacketStyle
	}

	prefix := d.bytes(8)

	var hasher Hasher
	hasher.reset(p.salt)
	hasher.put(prefix)
	h := hasher.val

	len1 := 1 + (h & 0b111)
	len2 := 1 + ((h >> 3) & 0b111)

	d.skip(int(len1) + 3) // skip prefix varlen tjunk + 3 bytes of reserved junk

	typ := PacketType(d.u8() & 0b1111)
	if typ.IsJunk() {
		typ = PacketJunk
	}
	cid := d.cid()

	// length of packet data
	dlen := d.len() -
		4 - // control sum
		int(len2) - // varlen tjunk suffix
		8 - // fixed tjunk suffix
		2 // style suffix
	if dlen < 0 {
		return ErrPacketSize
	}
	data := d.bytes(dlen)

	csum := d.u32()
	d.skip(int(len2))
	suffix := d.bytes(8)

	if debug {
		fmt.Printf("data(4): %v\n", data[:min(4, len(data))])
		fmt.Printf("csum:    %08X\n", csum)
		fmt.Printf("suffix:  %v\n", suffix)
	}

	i1 := prefix[0] & 0b111
	i2 := suffix[0] & 0b111
	hasher.reset(p.salt)
	hasher.putb(prefix[i1])
	hasher.put(cid[:])
	hasher.put(data[:min(4, len(data))])
	hasher.putb(suffix[i2])
	if hasher.get32() != csum {
		return ErrPacketSum
	}

	if d.u8() != '"' {
		return ErrPacketStyle
	}
	var s2 Style
	switch d.u8() {
	case '}':
		s2 = Style0
	case ']':
		s2 = Style1
	default:
		return ErrPacketStyle
	}
	if s1 != s2 {
		return ErrPacketStyle
	}

	p.Data = data
	p.CID = cid
	p.Type = typ
	return nil
}

// Returns next n bytes from buffer and advances decoder
// by this exact amount. Caller is responsible for checking
// buffer length boundaries.
func (d *decoder) bytes(n int) []byte {
	p := d.pos
	d.pos += n
	return d.buf[p:d.pos]
}

func (d *decoder) cid() ConnID {
	var cid ConnID
	b := d.bytes(16)
	copy(cid[:], b)
	return cid
}

func (d *decoder) skip(n int) {
	d.pos += n
}

// Returns number of remaining bytes available.
func (d *decoder) len() int {
	return len(d.buf) - d.pos
}

func (d *decoder) u32() uint32 {
	p := d.pos
	d.pos += 4

	b := d.buf[p:]
	_ = b[3] // compiler hint
	return uint32(b[0]) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
}

func (d *decoder) u16() uint16 {
	p := d.pos
	d.pos += 2

	b := d.buf[p:]
	_ = b[1] // compiler hint
	return uint16(b[0]) | (uint16(b[1]) << 8)
}

func (d *decoder) u8() uint8 {
	p := d.pos
	d.pos += 1
	return d.buf[p]
}
