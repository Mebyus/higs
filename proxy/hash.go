package proxy

import "encoding/binary"

type Hasher struct {
	val uint64
}

func (h *Hasher) reset(salt uint32) {
	h.val = 5381 // magic

	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], salt)
	h.put(buf[:])
}

func (h *Hasher) put(data []byte) {
	v := h.val
	for i := range len(data) {
		b := data[i]
		v += (v << 5) + uint64(b)
	}
	h.val = v
}

func (h *Hasher) putb(b byte) {
	h.val += (h.val << 5) + uint64(b)
}

func (h *Hasher) get32() uint32 {
	v := h.val
	return uint32(v) + uint32(v>>32)
}
