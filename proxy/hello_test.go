package proxy

import (
	"errors"
	"math/rand/v2"
	"net/netip"
	"testing"
)

func TestDecodeHello(t *testing.T) {
	tests := []struct {
		addr string
		net  uint8
	}{
		{
			addr: "8.8.8.8:53",
			net:  NetworkUDP,
		},
		{
			addr: "127.0.0.1:8081",
		},
		{
			addr: "209.85.233.91:80",
		},
		{
			addr: "188.186.154.88:443",
		},
	}

	g := rand.NewChaCha8([32]byte{0, 1, 2, 3})
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			ap, err := netip.ParseAddrPort(tt.addr)
			if err != nil {
				t.Errorf("ParseAddrPort() error = %v", err)
				return
			}

			var h Hello
			h.InitEncode(g, tt.net, ap)

			var buf [16]byte
			data := EncodeHello(&h, buf[:0])

			var got Hello
			err = DecodeHello(&got, data)
			if err != nil {
				t.Errorf("DecodeHello() error = %v", err)
				return
			}

			err = compareHellos(&got, &h)
			if err != nil {
				t.Errorf("compare hellos: %v", err)
				logHello(t, "got", &got)
				logHello(t, "want", &h)
			}
		})
	}
}

func logHello(t *testing.T, title string, h *Hello) {
	t.Logf("%s hello:", title)

	var p string
	switch h.Network {
	case NetworkTCP:
		p = "tcp"
	case NetworkUDP:
		p = "udp"
	}
	t.Logf("  addr: %s://%s", p, h.AddrPort)
}

func compareHellos(a, b *Hello) error {
	if a.Network != b.Network {
		return errors.New("network not equal")
	}
	if a.AddrPort != b.AddrPort {
		return errors.New("address not equal")
	}
	return nil
}
