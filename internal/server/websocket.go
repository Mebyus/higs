package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/mebyus/higs/proxy"
	"github.com/mebyus/higs/wsok"
)

func (s *Server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	if !wsok.HasUpgradeHeaders(r.Header) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-Websocket-Key")
	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	authToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if authToken == "" || s.Config.AuthToken != authToken {
		// Check token for empty string in case we somehow allowed
		// empty auth token in config.
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hash := wsok.HashHandshakeKey(key)

	hj, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = conn.SetWriteDeadline(time.Time{})
	if err != nil {
		return
	}
	err = conn.SetReadDeadline(time.Time{})
	if err != nil {
		return
	}

	bufrw.WriteString("HTTP/1.1 101 Switching Protocols\n")
	bufrw.WriteString("Connection: Upgrade\n")
	bufrw.WriteString("Upgrade: websocket\n")
	bufrw.WriteString("Sec-Websocket-Accept: " + hash + "\n\n")
	err = bufrw.Flush()
	if err != nil {
		return
	}

	go serveTunnel(&Tunnel{
		conn:  conn,
		rb:    bufrw.Reader,
		wb:    bufrw.Writer,
		lg:    s.lg.WithGroup("tun"),
		done:  make(chan struct{}),
		conns: make(map[proxy.ConnID]*Conn),
	})
}
