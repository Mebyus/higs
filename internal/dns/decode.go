package main

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNoHeader = errors.New("no header")
)

func Decode(m *Message, data []byte) error {
	const debug = true

	dec := decoder{buf: data}

	var h header
	err := dec.header(&h)
	if err != nil {
		return err
	}
	if debug {
		printHeader(&h)
	}

	quests, err := dec.quests(h.quests)
	if err != nil {
		return err
	}
	answers, err := dec.answers(h.answers)
	if err != nil {
		return err
	}
	records, err := dec.records(h.records)
	if err != nil {
		return err
	}

	if debug {
		fmt.Println()
		fmt.Printf("decoded %d/%d bytes\n", dec.pos, len(data))
	}

	_ = records
	m.Quests = quests
	m.Answers = answers
	m.ID = h.id
	m.Opcode = h.opcode
	return nil
}

type decoder struct {
	buf []byte

	// current decoding position (offset into data)
	pos int

	// saved position before jumping to offset (for decompression)
	mark int
}

func (d *decoder) jump(offset uint16) {
	d.mark = d.pos
	d.pos = int(offset)
}

// restore decoder position if jump occured previously
func (d *decoder) restore() {
	if d.mark != 0 {
		d.pos = d.mark
		d.mark = 0
	}
}

var (
	ErrEmptyName     = errors.New("empty name")
	ErrNotEnoughData = errors.New("not enough data")
)

func (d *decoder) header(h *header) error {
	if d.len() < 12 {
		return ErrNoHeader
	}

	h.id = d.u16()

	b := d.u8()
	h.resp = b>>7 != 0
	h.opcode = Opcode((b >> 3) & 0b1111)
	h.auth = b&0b100 != 0
	h.trunc = b&0b10 != 0
	h.recd = b&0b1 != 0

	b = d.u8()
	h.reca = b>>7 != 0
	h.rcode = Rcode(b & 0b1111)

	h.quests = d.u16()
	h.answers = d.u16()
	h.servers = d.u16()
	h.records = d.u16()

	return nil
}

func (d *decoder) quests(num uint16) ([]Quest, error) {
	if num == 0 {
		return nil, nil
	}

	quests := make([]Quest, num)
	for i := range num {
		q := &quests[i]
		err := d.quest(q)
		if err != nil {
			return nil, err
		}
	}
	return quests, nil
}

func (d *decoder) quest(q *Quest) error {
	const debug = true

	name, err := d.name()
	if err != nil {
		return err
	}
	if name == "" {
		return ErrEmptyName
	}

	if d.len() < 4 {
		return ErrNotEnoughData
	}

	typ := d.u16()
	class := d.u16()

	q.Name = name
	q.Type = Type(typ)
	q.Class = Class(class)

	if debug {
		printQuest(q)
	}

	return nil
}

func (d *decoder) answers(num uint16) ([]Answer, error) {
	if num == 0 {
		return nil, nil
	}

	answers := make([]Answer, num)
	for i := range num {
		a := &answers[i]
		err := d.answer(a)
		if err != nil {
			return nil, err
		}
	}
	return answers, nil
}

func (d *decoder) answer(a *Answer) error {
	return d.record(a)
}

func (d *decoder) records(num uint16) ([]Record, error) {
	if num == 0 {
		return nil, nil
	}

	records := make([]Record, num)
	for i := range num {
		r := &records[i]
		err := d.record(r)
		if err != nil {
			return nil, err
		}
	}
	return records, nil
}

func (d *decoder) record(r *Record) error {
	const debug = true

	name, err := d.name()
	if err != nil {
		return err
	}

	if d.len() < 2+2+4+2 {
		return ErrNotEnoughData
	}

	typ := d.u16()
	class := d.u16()
	ttl := d.u32()
	length := d.u16()

	if d.len() < int(length) {
		return ErrNotEnoughData
	}
	data := d.data(int(length))

	r.Name = name
	r.Type = Type(typ)
	r.Class = Class(class)
	r.TTL = ttl
	r.Data = data

	if debug {
		printRecord(r)
	}

	return nil
}

// Decodes a name from a sequence of labels in data.
//
// Sequence is terminated by zero byte.
func (d *decoder) name() (string, error) {
	var b strings.Builder

	start, err := d.label()
	if err != nil {
		return "", err
	}
	if len(start) == 0 {
		return "", nil
	}
	b.Write(start)

	for {
		label, err := d.label()
		if err != nil {
			return "", err
		}
		if len(label) == 0 {
			return b.String(), nil
		}

		b.WriteByte('.')
		b.Write(label)
	}
}

var ErrRecursivePointer = errors.New("recursive pointer")

func (d *decoder) label() ([]byte, error) {
	if d.len() < 1 {
		return nil, ErrNotEnoughData
	}

	length := d.u8()
	if length == 0 {
		d.restore()
		return nil, nil
	}
	if length>>6 == 0b11 {
		// handle pointer logic
		if d.mark != 0 {
			return nil, ErrRecursivePointer
		}

		if d.len() < 1 {
			return nil, ErrNotEnoughData
		}

		b := d.u8()
		offset := (uint16(length&0b111111) << 8) | uint16(b)
		d.jump(offset)
		return d.label()
	}

	if int(length) > d.len() {
		return nil, ErrNotEnoughData
	}

	p := d.pos
	d.pos += int(length)
	return d.buf[p:d.pos], nil
}

func (d *decoder) u8() uint8 {
	p := d.pos
	d.pos += 1
	return d.buf[p]
}

func (d *decoder) u16() uint16 {
	p := d.pos
	d.pos += 2

	b := d.buf[p:]
	_ = b[1] // compiler hint
	return (uint16(b[0]) << 8) | uint16(b[1])
}

func (d *decoder) u32() uint32 {
	p := d.pos
	d.pos += 4

	b := d.buf[p:]
	_ = b[3] // compiler hint
	return (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | uint32(b[3])
}

func (d *decoder) data(size int) []byte {
	p := d.pos
	d.pos += size
	return d.buf[p:d.pos]
}

// Returns number of remaining bytes available.
func (d *decoder) len() int {
	return len(d.buf) - d.pos
}
