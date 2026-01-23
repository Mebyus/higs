package dns

import (
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "1 empty message",
			msg:  Message{},
		},
		{
			name: "2 only id",
			msg:  Message{ID: 0xE30C},
		},
		{
			name: "3 one quest",
			msg: Message{
				ID:     0xE30C,
				Opcode: OpQuery,
				Quests: []Quest{{
					Name:  "ya.ru",
					Type:  TypeAddr,
					Class: Internet,
				}},
			},
		},
		{
			name: "4 one answer",
			msg: Message{
				ID:     0xE30C,
				Opcode: OpQuery,
				Answers: []Answer{{
					Name:  "ya.ru",
					Type:  TypeAddr,
					Class: Internet,
					TTL:   581,
					Data:  []byte{77, 88, 44, 242},
				}},
			},
		},
		{
			name: "5 two answers",
			msg: Message{
				ID:     0xE30C,
				Opcode: OpQuery,
				Answers: []Answer{{
					Name:  "ya.ru",
					Type:  TypeAddr,
					Class: Internet,
					TTL:   581,
					Data:  []byte{5, 255, 255, 242},
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// use small initial buffer to test that reallocations
			// are handled correctly inside Encode function
			var buf [32]byte
			data := Encode(&tt.msg, buf[:0])

			var msg Message
			err := Decode(&msg, data)
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
