package wsok

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
)

const debug = false

type OpCode uint8

const (
	OpFrag  OpCode = 0x0
	OpText  OpCode = 0x1
	OpBin   OpCode = 0x2
	OpClose OpCode = 0x8
	OpPing  OpCode = 0x9
)

const handshakeMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func GenHandshakeKey(g *rand.ChaCha8) string {
	var buf [16]byte
	g.Read(buf[:])
	return base64.StdEncoding.EncodeToString(buf[:])
}

func HashHandshakeKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(handshakeMagic))
	return string(base64.StdEncoding.AppendEncode(nil, h.Sum(nil)))
}

type Frame struct {
	// Raw unmasked payload bytes.
	Data []byte

	// Opcode from header.
	Op OpCode

	Mask [4]byte

	// Extension bits from header.
	//
	//	7 6 5 4 3 2 1 0 # bits order: most -> least significant
	//	- - - - - 1 2 3 # order of extension bits
	Ext uint8

	// Final (last) frame in message.
	Fin bool

	UseMask bool
}

func Decode(r io.Reader, f *Frame) error {
	// fixed buffer for reading frame header
	var buf [12]byte

	n, err := r.Read(buf[:2])
	if err != nil {
		return err
	}
	if n != 2 {
		return fmt.Errorf("unable to read 2 frame header bytes (got %d)", n)
	}

	fin := (buf[0] >> 7) == 1
	ext := (buf[0] >> 4) & 0x7
	op := OpCode(buf[0] & 0xf)
	useMask := (buf[1] >> 7) == 1
	sizeBits := buf[1] & 0x7f

	if debug {
		fmt.Printf("fin: %v\n", fin)
		fmt.Printf("ext bits: %03b\n", ext)
		fmt.Printf("op: 0x%x\n", op)
		fmt.Printf("use mask: %v\n", useMask)
		fmt.Printf("size bits: %b\n", sizeBits)
	}

	// TODO: check extension bits
	// TODO: check opcode value

	size := uint64(sizeBits)
	pos := 0
	headerExtraSize := getHeaderExtraSize(useMask, sizeBits)
	if headerExtraSize > 0 {
		n, err := r.Read(buf[:headerExtraSize])
		if err != nil {
			return err
		}
		if n != headerExtraSize {
			return fmt.Errorf("unable to read extra header bytes: got %d, want %d", n, headerExtraSize)
		}

		if sizeBits == 126 {
			size = uint64(binary.BigEndian.Uint16(buf[:2]))
			pos = 2
		} else if sizeBits == 127 {
			size = binary.BigEndian.Uint64(buf[:8])
			pos = 8
		}
	}

	if debug {
		fmt.Printf("size: %d\n", size)
	}

	var mask [4]byte
	if useMask {
		copy(mask[:], buf[pos:headerExtraSize])
		if debug {
			fmt.Printf("mask: 0x%04x\n", mask)
		}
	}

	var data []byte
	if size != 0 {
		data = make([]byte, size)
		_, err := io.ReadFull(r, data)
		if err != nil {
			return err
		}
	}

	if useMask {
		// unmask payload
		for i := 0; i < len(data); i++ {
			data[i] ^= mask[i&0b11]
		}
	}

	f.Data = data
	f.Op = op
	f.Ext = ext
	f.Fin = fin
	f.Mask = mask
	f.UseMask = useMask
	return nil
}

func getHeaderExtraSize(useMask bool, sizeBits uint8) int {
	size := 0
	if useMask {
		size += 4
	}

	if sizeBits == 126 {
		size += 2
	} else if sizeBits == 127 {
		size += 8
	}

	return size
}

func Encode(w io.Writer, f *Frame) error {
	// fixed buffer for encoding frame header
	var hbuf [14]byte

	sizeBits, extraSize := getSizeBits(uint64(len(f.Data)))

	hbuf[0] = (boolBit(f.Fin) << 7) | ((f.Ext & 0x7) << 4) | (uint8(f.Op) & 0xF)
	hbuf[1] = (boolBit(f.UseMask) << 7) | sizeBits
	// encoded header length
	n := 2

	switch extraSize {
	case 0:
		// do nothing
	case 2:
		binary.BigEndian.PutUint16(hbuf[2:], uint16(len(f.Data)))
	case 8:
		binary.BigEndian.PutUint64(hbuf[2:], uint64(len(f.Data)))
	default:
		panic(fmt.Sprintf("unexpected extra size %d", extraSize))
	}
	n += extraSize

	if f.UseMask {
		copy(hbuf[n:n+4], f.Mask[:])
		n += 4
	}

	_, err := w.Write(hbuf[:n])
	if err != nil {
		return err
	}

	if len(f.Data) == 0 {
		return nil
	}

	if !f.UseMask {
		_, err := w.Write(f.Data)
		return err
	}

	const shiftSize = 14
	var buf [1 << shiftSize]byte

	// position inside payload slice
	var pos int

	for j := 0; j < (len(f.Data) >> shiftSize); j += 1 {
		// outer loop cycles through full buffer chunks

		for i := 0; i < len(buf); i += 1 {
			// place masked byte into buffer
			buf[i] = f.Data[pos] ^ f.Mask[i&0b11]
			pos += 1
		}

		_, err := w.Write(buf[:])
		if err != nil {
			return err
		}
	}

	if pos >= len(f.Data) {
		return nil
	}

	// write remaining portion of payload
	r := len(f.Data) & (len(buf) - 1)
	for i := 0; i < r; i += 1 {
		buf[i] = f.Data[pos] ^ f.Mask[i&0b11]
		pos += 1
	}

	_, err = w.Write(buf[:r])
	return err
}

// returns header size bits and header extra size
func getSizeBits(size uint64) (uint8, int) {
	if size <= 125 {
		return uint8(size), 0
	}
	if size <= 0xFFFF {
		return 126, 2
	}
	return 127, 8
}

func boolBit(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}
