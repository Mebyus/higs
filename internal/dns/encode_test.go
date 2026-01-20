package main_test

import (
	"reflect"
	"testing"

	main "github.com/mebyus/higs/internal/dns"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		msg  main.Message
	}{
		{
			name: "1 empty message",
			msg:  main.Message{},
		},
		{
			name: "2 only id",
			msg:  main.Message{ID: 0xE30C},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// use small initial buffer to test that reallocations
			// are handled correctly inside Encode function
			var buf [32]byte
			data, err := main.Encode(&tt.msg, buf[:0])
			if err != nil {
				t.Errorf("Encode() error = %v", err)
				return
			}

			var msg main.Message
			err = main.Decode(&msg, data)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}

			if !reflect.DeepEqual(&msg, &tt.msg) {
				t.Errorf("Decode() = %#v, want %#v", &msg, &tt.msg)
			}
		})
	}
}
