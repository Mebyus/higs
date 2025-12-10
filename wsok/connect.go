package wsok

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ConnectConfig struct {
	// Optional.
	//
	// Extra headers for request.
	ExtraHeaders []Header

	// Optional.
	//
	// Acceptable websocket extensions.
	Extensions []string

	// Optional.
	AcceptEncodings []string

	// Required.
	Path string

	// Required.
	Key string

	// Required.
	//
	// Includes optional port.
	Host string

	// Optional.
	UserAgent string

	// Optional.
	Origin string

	// Optional.
	AuthToken string
}

type Header struct {
	Name  string
	Value string
}

// WriteConnectRequest writes websocket connect request in
// plain text into supplied Writer.
//
// Example of websocket connect request from browser:
//
//	GET /ext/user/me/events HTTP/1.1
//	Host: localhost:8733
//	User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:145.0) Gecko/20100101 Firefox/145.0
//	Accept: */*
//	Accept-Language: en-US,en;q=0.5
//	Accept-Encoding: gzip, deflate, br, zstd
//	Sec-WebSocket-Version: 13
//	Origin: http://localhost:8733
//	Sec-WebSocket-Extensions: permessage-deflate
//	Sec-WebSocket-Key: rdwCAuY2qmzrQbTkg2fZhA==
//	DNT: 1
//	Sec-GPC: 1
//	Connection: keep-alive, Upgrade
//	Sec-Fetch-Dest: empty
//	Sec-Fetch-Mode: websocket
//	Sec-Fetch-Site: same-origin
//	Pragma: no-cache
//	Cache-Control: no-cache
//	Upgrade: websocket
func WriteConnectRequest(w io.Writer, c *ConnectConfig) error {
	err := writeGetPath(w, c.Path)
	if err != nil {
		return err
	}

	headers := []Header{
		{"Host", c.Host},
		{"User-Agent", c.UserAgent},
		{"Accept-Encoding", joinHeaderValues(c.AcceptEncodings)},
		{"Sec-WebSocket-Version", "13"},
		{"Origin", c.Origin},
		{"Sec-WebSocket-Extensions", joinHeaderValues(c.Extensions)},
		{"Sec-Websocket-Key", c.Key},
		{"Connection", "keep-alive, Upgrade"},
		{"Sec-Fetch-Dest", "empty"},
		{"Sec-Fetch-Mode", "websocket"},
		{"Sec-Fetch-Site", "same-origin"},
		{"Pragma", "no-cache"},
		{"Cache-Control", "no-cache"},
		{"Upgrade", "websocket"},
		makeAuthHeader(c.AuthToken),
	}

	for _, h := range headers {
		err = writeHeader(w, h.Name, h.Value)
		if err != nil {
			return err
		}
	}

	for _, h := range c.ExtraHeaders {
		err = writeHeader(w, h.Name, h.Value)
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(w, "\n")
	return err
}

func makeAuthHeader(token string) Header {
	if token == "" {
		return Header{}
	}

	return Header{
		Name:  "Authorization",
		Value: "Bearer " + token,
	}
}

func joinHeaderValues(s []string) string {
	return strings.Join(s, ", ")
}

func writeGetPath(w io.Writer, path string) error {
	_, err := io.WriteString(w, "GET ")
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, path)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, " HTTP/1.1\n")
	return err
}

func writeHeader(w io.Writer, name, value string) error {
	if value == "" || name == "" {
		return nil
	}

	_, err := io.WriteString(w, name)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, ": ")
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, value)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

// Correct websocket connect response should be like:
//
//	HTTP/1.1 101 Switching Protocols
//	Connection: Upgrade
//	Upgrade: websocket
//	Sec-Websocket-Accept: <hash>
//
// Note that response must end with double new line ("\n\n").
func CheckConnectResponse(b []byte, key string) error {
	split := bytes.Split(b, []byte{'\n'})
	if len(split) != 6 {
		return fmt.Errorf("unexpected number of headers (=%d)", len(split))
	}

	if !bytes.Equal(split[0], []byte("HTTP/1.1 101 Switching Protocols")) {
		return fmt.Errorf("bad status: %s", split[0])
	}

	if !bytes.Equal(split[1], []byte("Connection: Upgrade")) {
		return fmt.Errorf("bad connection header: %s", split[1])
	}

	if !bytes.Equal(split[2], []byte("Upgrade: websocket")) {
		return fmt.Errorf("bad upgrade header: %s", split[2])
	}

	acceptSplit := bytes.SplitN(split[3], []byte{':'}, 2)
	if len(acceptSplit) != 2 {
		return fmt.Errorf("bad accept header: %s", split[3])
	}

	// TODO: check accept header name

	gotHash := bytes.TrimSpace(acceptSplit[1])
	wantHash := HashHandshakeKey(key)
	if !bytes.Equal(gotHash, []byte(wantHash)) {
		return fmt.Errorf("handshake key hash mismatch: %s", gotHash)
	}

	if len(split[4]) != 0 {
		return fmt.Errorf("bad end: %s", split[4])
	}

	return nil
}

func HasUpgradeHeaders(headers http.Header) bool {
	if headers.Get("Upgrade") != "websocket" {
		return false
	}

	return strings.Contains(headers.Get("Connection"), "Upgrade")
}
