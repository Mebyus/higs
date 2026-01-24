package proxy

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestEncodeClose(t *testing.T) {
	tests := []struct {
		cc CloseCode
	}{
		{cc: 0},
		{cc: 1},
		{cc: 2},
		{cc: 0x75BCC91A},
	}

	g := rand.NewChaCha8([32]byte{0, 1, 2, 3})
	for _, tt := range tests {
		t.Run(fmt.Sprintf("cc=%d", tt.cc), func(t *testing.T) {
			var s Close
			s.InitEncode(g, tt.cc)

			var buf [16]byte
			data := EncodeClose(&s, buf[:0])

			var got Close
			err := DecodeClose(&got, data)
			if err != nil {
				t.Errorf("DecodeClose() error = %v", err)
				return
			}

			if got.Code != tt.cc {
				t.Errorf("DecodeClose() got = %d, want %d", got.Code, tt.cc)
			}
		})
	}
}
