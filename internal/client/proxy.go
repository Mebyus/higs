package client

import (
	"os"
	"path/filepath"
)

type ProxyConn struct {
}

func (c *ProxyConn) Read(b []byte) (int, error) {
	return 0, nil
}

func (c *ProxyConn) Write(b []byte) (int, error) {
	return 0, nil
}

func (c *ProxyConn) Close() error {
	return nil
}

// FileConn mainly used for testing as a simple implementation of Socket.
type FileConn struct {
	file *os.File
}

func (c *FileConn) Init(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	c.file = file
	return nil
}

func (c *FileConn) Read(b []byte) (int, error) {
	return 0, nil
}

func (c *FileConn) Write(b []byte) (int, error) {
	return c.file.Write(b)
}

func (c *FileConn) Close() error {
	return c.file.Close()
}
