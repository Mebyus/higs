package proxy

import (
	"bufio"
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/mebyus/higs/wsok"
)

const fakeUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:145.0) Gecko/20100101 Firefox/145.0"

func Connect(ctx context.Context, proxyURL, token string) (*Tunnel, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	host, portString, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip address \"%s\"", host)
	}
	port, err := strconv.ParseUint(portString, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid port \"%s\"", portString)
	}
	if port == 0 || port > 0xFFFF {
		return nil, fmt.Errorf("invalid port \"%s\"", portString)
	}

	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   ip,
		Port: int(port),
	})
	if err != nil {
		return nil, err
	}

	/* Send connect request to proxy server */

	var seed [32]byte
	time.Now().AppendBinary(seed[:16])
	copy(seed[16:], u.Host)
	g := rand.NewChaCha8(seed)
	key := wsok.GenHandshakeKey(g)

	w := bufio.NewWriter(conn)
	err = wsok.WriteConnectRequest(w, &wsok.ConnectConfig{
		Extensions:      []string{"permessage-deflate"},
		AcceptEncodings: []string{"gzip", "deflate", "br", "zstd"},
		Origin:          u.Scheme + "://" + u.Host,
		Host:            u.Host,
		Path:            u.Path,
		UserAgent:       fakeUserAgent,
		AuthToken:       token,
		Key:             key,
		ExtraHeaders: []wsok.Header{
			{"Accept", "*/*"},
			{"Accept-Language", "en-US,en;q=0.5"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("initiate connection: %v", err)
	}
	err = w.Flush()
	if err != nil {
		return nil, fmt.Errorf("initiate connection: %v", err)
	}

	/* Read response from proxy server */
	var buf [1 << 14]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		return nil, fmt.Errorf("read connect response: %v", err)
	}

	err = wsok.CheckConnectResponse(buf[:n], key)
	if err != nil {
		return nil, fmt.Errorf("check connect response: %v", err)
	}

	return newTunnel(conn, g), nil
}
