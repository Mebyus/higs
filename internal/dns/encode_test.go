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
		{
			name: "3 one quest",
			msg: main.Message{
				ID:     0xE30C,
				Opcode: main.OpQuery,
				Quests: []main.Quest{{
					Name:  "ya.ru",
					Type:  main.TypeAddr,
					Class: main.Internet,
				}},
			},
		},
		{
			name: "4 one answer",
			msg: main.Message{
				ID:     0xE30C,
				Opcode: main.OpQuery,
				Answers: []main.Answer{{
					Name:  "ya.ru",
					Type:  main.TypeAddr,
					Class: main.Internet,
					TTL:   581,
					Data:  []byte{77, 88, 44, 242},
				}},
			},
		},
		{
			name: "5 two answers",
			msg: main.Message{
				ID:     0xE30C,
				Opcode: main.OpQuery,
				Answers: []main.Answer{{
					Name:  "ya.ru",
					Type:  main.TypeAddr,
					Class: main.Internet,
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
			data := main.Encode(&tt.msg, buf[:0])

			var msg main.Message
			err := main.Decode(&msg, data)
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
