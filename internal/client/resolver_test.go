package client

import (
	"slices"
	"strings"
	"testing"
)

const testNames = `
# comment
# another comment
youtube.com, www.youtube.com:

    64.233.162.91
    64.233.162.93
    64.233.162.136
    64.233.162.190

youtube.ru, www.youtube.ru:

    209.85.233.91
    209.85.233.93
    209.85.233.136
    209.85.233.190
    209.85.233.198

instagram.com:

    188.186.154.88

# end comment
`

func Test_parseNames(t *testing.T) {
	var r Resolver
	err := parseNames(&r, strings.NewReader(testNames))
	if err != nil {
		t.Errorf("parseNames() error = %v", err)
		return
	}

	tests := []struct {
		name string
		list []string
	}{
		{
			name: "ya.ru",
			list: nil,
		},
		{
			name: "instagram.com",
			list: []string{"188.186.154.88"},
		},
		{
			name: "youtube.com",
			list: []string{
				"64.233.162.91",
				"64.233.162.93",
				"64.233.162.136",
				"64.233.162.190",
			},
		},
		{
			name: "www.youtube.com",
			list: []string{
				"64.233.162.91",
				"64.233.162.93",
				"64.233.162.136",
				"64.233.162.190",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := parseAddrList(tt.list)
			if err != nil {
				t.Errorf("parseAddrList() error = %v", err)
				return
			}

			got := r.Resolve(tt.name)
			if !slices.Equal(got, list) {
				t.Errorf("Resolver.Resolve() got = %s, want %s", got, list)
			}
		})
	}
}
