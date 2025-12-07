package client

import "net"

type DirectConn struct {
	conn net.Conn
}

func (c *DirectConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *DirectConn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c *DirectConn) Close() error {
	return c.conn.Close()
}
