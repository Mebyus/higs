package proxy

import (
	"fmt"
	"slices"
)

func Encode(p *Packet, buf []byte) []byte {
	if !p.ok {
		panic("no init")
	}

	c := encoder{buf: buf}
	return c.packet(p)
}

type encoder struct {
	buf []byte
}

func (c *encoder) packet(p *Packet) []byte {
	const debug = false

	c.buf = slices.Grow(c.buf,
		2+ // style prefix
			8+ // fixed start tjunk
			8+ // at most 8 bytes of varlen tjunk
			3+ // reserved tjunk
			1+ // packet type
			16+ // connection id
			len(p.Data)+
			4+ // control sum
			8+ // at most 8 bytes of varlen tjunk
			8+ // fixed end tjunk
			2+ // style suffix
			0) // 0 just for formatting and newline rules of Go

	var hasher Hasher
	hasher.reset(p.salt)
	hasher.put(p.junk1[:8])
	h := hasher.val

	len1 := 1 + (h & 0b111)
	len2 := 1 + ((h >> 3) & 0b111)

	typ := uint8(p.Type) | (p.junk1[20] & 0xF0)

	i1 := p.junk1[0] & 0b111
	i2 := uint8(len2) + p.junk2[len2]&0b111
	hasher.reset(p.salt)
	hasher.putb(p.junk1[i1])
	hasher.put(p.CID[:])
	hasher.put(p.Data[:min(4, len(p.Data))])
	hasher.putb(p.junk2[i2])

	csum := hasher.get32()

	if debug {
		suffix := p.junk2[len2 : 8+len2]
		fmt.Printf("data(4): %v\n", p.Data[:min(4, len(p.Data))])
		fmt.Printf("csum:    %08X\n", csum)
		fmt.Printf("suffix:  %v\n", suffix)
	}

	c.start(p.style)
	c.put(p.junk1[:8+len1+3])
	c.putb(typ)
	c.put(p.CID[:])
	c.put(p.Data)
	c.u32(csum)
	c.put(p.junk2[:8+len2])
	c.end(p.style)

	return c.buf
}

func (c *encoder) start(s Style) {
	switch s {
	case Style0:
		c.puts(`{"`)
	case Style1:
		c.puts(`["`)
	default:
		panic(fmt.Sprintf("unexpected style (=%d)", s))
	}
}

func (c *encoder) end(s Style) {
	switch s {
	case Style0:
		c.puts(`"}`)
	case Style1:
		c.puts(`"]`)
	default:
		panic(fmt.Sprintf("unexpected style (=%d)", s))
	}
}

func (c *encoder) put(data []byte) {
	c.buf = append(c.buf, data...)
}

func (c *encoder) putb(b byte) {
	c.buf = append(c.buf, b)
}

func (c *encoder) puts(s string) {
	c.buf = append(c.buf, s...)
}

func (c *encoder) u16(v uint16) {
	c.buf = append(c.buf, byte(v), byte(v>>8))
}

func (c *encoder) u32(v uint32) {
	c.buf = append(c.buf, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}
