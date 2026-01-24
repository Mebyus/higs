package proxy

import (
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"slices"
)

type CloseCode uint32

const (
	CloseOK CloseCode = iota
)

type Close struct {
	Code CloseCode

	junk [4]byte

	ok bool
}

func (c *Close) InitEncode(g *rand.ChaCha8, cc CloseCode) {
	putJunk(g, c.junk[:])
	c.Code = cc
	c.ok = true
}

func EncodeClose(s *Close, buf []byte) []byte {
	if !s.ok {
		panic("no init")
	}

	c := encoder{buf: buf}
	return c.close(s)
}

func (c *encoder) close(s *Close) []byte {
	c.buf = slices.Grow(c.buf, 8)

	var cc [4]byte
	binary.LittleEndian.PutUint32(cc[:], uint32(s.Code))

	c.putb(cc[0])
	c.putb(s.junk[0])

	c.putb(cc[1])
	c.putb(s.junk[1])

	c.putb(cc[2])
	c.putb(s.junk[2])

	c.putb(cc[3])
	c.putb(s.junk[3])

	return c.buf
}

func DecodeClose(s *Close, data []byte) error {
	d := decoder{buf: data}
	return d.close(s)
}

var ErrBadCloseSize = errors.New("bad size")

func (d *decoder) close(s *Close) error {
	if d.len() != 8 {
		return ErrBadCloseSize
	}

	b := d.bytes(8)

	var cc [4]byte
	cc[0] = b[0]
	cc[1] = b[2]
	cc[2] = b[4]
	cc[3] = b[6]

	s.Code = CloseCode(binary.LittleEndian.Uint32(cc[:]))
	return nil
}
