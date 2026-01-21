package main

import (
	"slices"
	"strings"
)

// Encode appends encoded message to a given slice buffer and returns
// resulting slice.
func Encode(m *Message, buf []byte) []byte {
	g := encoder{
		buf: buf,
		m:   make(map[string]uint16),
	}

	h := header{
		id: m.ID,

		resp:   len(m.Answers) != 0 || m.Rcode != RcOk,
		opcode: m.Opcode,
		rcode:  m.Rcode,
		auth:   false,
		trunc:  false,
		recd:   false,
		reca:   false,

		quests:  uint16(len(m.Quests)),
		answers: uint16(len(m.Answers)),
		records: uint16(len(m.Records)),
	}
	g.header(&h)

	for i := range len(m.Quests) {
		q := &m.Quests[i]
		g.quest(q)
	}

	for i := range len(m.Answers) {
		a := &m.Answers[i]
		g.answer(a)
	}

	for i := range len(m.Records) {
		r := &m.Records[i]
		g.record(r)
	}

	return g.buf
}

type encoder struct {
	buf []byte

	// Lookup table with suffix offsets.
	//
	// Used for names compression (backward pointer scheme).
	m map[string]uint16
}

func (g *encoder) header(h *header) {
	g.buf = slices.Grow(g.buf, 12)

	g.u16(h.id)
	g.u8(bitbool(h.resp, 7) | uint8(h.opcode) | bitbool(h.auth, 2) | bitbool(h.trunc, 1) | bitbool(h.recd, 0))
	g.u8(bitbool(h.reca, 7) | uint8(h.rcode))

	g.u16(h.quests)
	g.u16(h.answers)
	g.u16(h.servers)
	g.u16(h.records)
}

func (g *encoder) quest(q *Quest) {
	g.name(q.Name)
	g.u16(uint16(q.Type))
	g.u16(uint16(q.Class))
}

func (g *encoder) answer(a *Answer) {
	g.record(a)
}

func (g *encoder) record(r *Record) {
	g.name(r.Name)
	g.u16(uint16(r.Type))
	g.u16(uint16(r.Class))
	g.u32(r.TTL)
	g.u16(uint16(len(r.Data)))
	g.buf = append(g.buf, r.Data...)
}

func (g *encoder) name(s string) {
	if s == "" {
		// zero terminator for root name
		g.u8(0)
		return
	}

	offset, ok := g.lookup(s)
	if ok {
		g.ptr(offset)
		return
	}
	g.store(s)

	prefix, suffix, _ := strings.Cut(s, ".")
	g.label(prefix)
	g.name(suffix)
}

// Lookup for previously encoded name suffix and obtain its
// offset if found.
func (g *encoder) lookup(s string) (uint16, bool) {
	offset, ok := g.m[s]
	return offset, ok
}

// Encodes name suffix pointer from given offset
func (g *encoder) ptr(offset uint16) {
	g.u16(offset | 0xC000)
}

func (g *encoder) label(s string) {
	g.u8(uint8(len(s)))
	g.buf = append(g.buf, s...)
}

// Store suffix with current encoder offset. After this call
// lookup for the stored suffix will return stored offset.
func (g *encoder) store(s string) {
	g.m[s] = uint16(len(g.buf))
}

func (g *encoder) u8(v uint8) {
	g.buf = append(g.buf, v)
}

func (g *encoder) u16(v uint16) {
	g.buf = append(g.buf, byte(v>>8))
	g.buf = append(g.buf, byte(v))
}

func (g *encoder) u32(v uint32) {
	g.buf = append(g.buf, byte(v>>24))
	g.buf = append(g.buf, byte(v>>16))
	g.buf = append(g.buf, byte(v>>8))
	g.buf = append(g.buf, byte(v))
}

func bitbool(v bool, pos int) uint8 {
	var b uint8
	if v {
		b = 1
	} else {
		b = 0
	}
	return b << pos
}
