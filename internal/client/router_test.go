package client

import (
	"net/netip"
	"strings"
	"testing"
)

const testRoutes = `
# youtube.com

64.233.162.91        proxy
64.233.162.93        proxy
64.233.162.136
64.233.162.190

142.250.102.198

# youtube.ru

209.85.233.91       direct
209.85.233.93
209.85.233.136
209.85.233.190
209.85.233.198      direct

# instagram.com

188.186.154.88      block
`

func Test_parseRoutes(t *testing.T) {
	var r Router
	err := parseRoutes(&r, strings.NewReader(testRoutes))
	if err != nil {
		t.Errorf("parseRoutes() error = %v", err)
		return
	}

	tests := []struct {
		ips  string
		want Action
	}{
		{
			ips:  "10.10.10.10",
			want: ActionAuto,
		},
		{
			ips:  "209.85.233.198",
			want: ActionDirect,
		},
		{
			ips:  "64.233.162.190",
			want: ActionProxy,
		},
		{
			ips:  "64.233.162.91",
			want: ActionProxy,
		},
		{
			ips:  "188.186.154.88",
			want: ActionBlock,
		},
	}
	for _, tt := range tests {
		t.Run(tt.ips, func(t *testing.T) {
			ip, err := netip.ParseAddr(tt.ips)
			if err != nil {
				t.Errorf("ParseAddr() error = %v", err)
				return
			}

			got := r.Lookup(ip)
			if got != tt.want {
				t.Errorf("Router.Lookup() got = %s, want %s", got, tt.want)
			}
		})
	}
}
