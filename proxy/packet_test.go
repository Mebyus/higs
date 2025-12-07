package proxy

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
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
			name: "3 empty",
			data: "",
			typ:  PacketData,
		},
		{
			name: "4 data",
			data: "hello",
			typ:  PacketData,
		},
	}

	g := rand.NewChaCha8([32]byte{0, 1, 2, 3})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := Packet{
				data: []byte(tt.data),
				typ:  tt.typ,
				cid:  NewConnID(g),
			}
			packet.PutJunk(g)
			b := Encode(&packet)
			cid, err := PeekConnID(b)
			if err != nil {
				t.Errorf("PeekConnID() error = %v", err)
				return
			}
			if cid != packet.cid {
				t.Errorf("PeekConnID() got = %s, want %s", cid, packet.cid)
				return
			}

			var gotPacket Packet
			err = Decode(b, &gotPacket)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}
			if !reflect.DeepEqual(&gotPacket, &packet) {
				t.Errorf("\nDecode() got = %#v\nwant %#v", &gotPacket, &packet)
			}
		})
	}
}
