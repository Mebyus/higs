package wsok

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name  string // description of this test case
		frame Frame
	}{
		{
			name: "1 empty no mask",
			frame: Frame{
				Data: nil,
				Op:   OpBin,
				Fin:  true,
			},
		},
		{
			name: "2 small data no mask",
			frame: Frame{
				Data: []byte("hello"),
				Op:   OpBin,
				Fin:  true,
			},
		},
		{
			name: "3 small data with mask",
			frame: Frame{
				Data:    []byte("hello"),
				Op:      OpBin,
				Fin:     true,
				Mask:    [4]byte{0xAC, 0x13, 0xE9, 0x06},
				UseMask: true,
			},
		},
		{
			name: "4 medium payload",
			frame: Frame{
				Data:    bytes.Repeat([]byte("hello world"), 1000),
				Op:      OpBin,
				Fin:     true,
				Mask:    [4]byte{0xAC, 0x13, 0xE9, 0x06},
				UseMask: true,
			},
		},
		{
			name: "5 big payload",
			frame: Frame{
				Data:    bytes.Repeat([]byte("hello world"), 10000),
				Op:      OpBin,
				Fin:     true,
				Mask:    [4]byte{0xAC, 0x13, 0xE9, 0x06},
				UseMask: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Encode(&buf, &tt.frame)
			if err != nil {
				t.Errorf("Encode() error = %v", err)
				return
			}

			var gotFrame Frame
			err = Decode(&buf, &gotFrame)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}
			if !reflect.DeepEqual(&gotFrame, &tt.frame) {
				t.Errorf("\nDecode() got = %#v\nwant %#v", &gotFrame, &tt.frame)
			}
		})
	}
}
