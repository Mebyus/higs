package proxy

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestDecodePacket(t *testing.T) {
	tests := []struct {
		name string

		data string
		typ  PacketType
	}{
		{
			name: "1 hello",
			data: "",
			typ:  PacketHello,
		},
		{
			name: "2 close",
			data: "",
			typ:  PacketClose,
		},
		{
			name: "3 ping",
			data: "",
			typ:  PacketPing,
		},
		{
			name: "4 junk",
			data: "",
			typ:  PacketJunk,
		},
		{
			name: "5 empty",
			data: "",
			typ:  PacketData,
		},
		{
			name: "6 data",
			data: "hello",
			typ:  PacketData,
		},
		{
			name: "7 large data",
			data: strings.Repeat("++ hello !", 237),
			typ:  PacketData,
		},
	}

	g := rand.NewChaCha8([32]byte{0, 1, 2, 3})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid := NewConnID(g)
			packet := Packet{
				Data: []byte(tt.data),
				Type: tt.typ,
				CID:  cid,
			}

			const salt = 0x4A7BAAE0
			packet.InitEncode(g, salt)

			var buf [32]byte
			data := Encode(&packet, buf[:0])

			var got Packet
			got.InitDecode(salt)
			err := Decode(&got, data)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}

			err = comparePackets(&got, &packet)
			if err != nil {
				t.Errorf("compare packets: %v", err)
				logPacket(t, "got", &got)
				logPacket(t, "want", &packet)
			}
		})
	}
}

func logPacket(t *testing.T, title string, p *Packet) {
	t.Logf("%s packet:", title)
	t.Logf("  cid:  %s", p.CID)
	t.Logf("  data: %v", p.Data)
	t.Logf("  salt: %08X", p.salt)
	t.Logf("  type: %s", p.Type)
}

func comparePackets(a, b *Packet) error {
	if a.CID != b.CID {
		return errors.New("cid not equal")
	}
	if !bytes.Equal(a.Data, b.Data) {
		return errors.New("data not equal")
	}
	if a.salt != b.salt {
		return errors.New("salt not equal")
	}
	if a.Type.IsJunk() && b.Type.IsJunk() {
		return nil
	}
	if a.Type != b.Type {
		return errors.New("type not equal")
	}
	return nil
}
