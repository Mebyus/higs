package main

import "slices"

// Encode appends encoded message to a given slice buffer and returns
// resulting slice.
func Encode(m *Message, buf []byte) ([]byte, error) {
	g := encoder{buf: buf}

	h := header{
		id: m.ID,

		resp: len(m.Answers) != 0,

		quests:  uint16(len(m.Quests)),
		answers: uint16(len(m.Answers)),
	}
	g.header(&h)

	return g.buf, nil
}

type encoder struct {
	buf []byte
}

func (g *encoder) header(h *header) {
	g.buf = slices.Grow(g.buf, 12)

	g.u16(h.id)

	// STUB: only for testing tests
	for range 10 {
		g.u16(0)
	}
}

func (g *encoder) u16(v uint16) {
	g.buf = append(g.buf, byte(v>>8))
	g.buf = append(g.buf, byte(v))
}
