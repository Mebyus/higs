package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mebyus/higs/proxy"
)

func relayData(client, backend Socket) {
	go func() {
		io.Copy(backend, client)
		backend.Close()
	}()
	io.Copy(client, backend)
}

func RunLocalServer(ctx context.Context, config *Config, tunnel *proxy.Tunnel) error {
	var server Server
	err := server.Init(fmt.Sprintf(":%d", config.LocalPort), tunnel)
	if err != nil {
		return fmt.Errorf("configure local server: %v\n", err)
	}

	err = setupNAT(config.LocalPort)
	if err != nil {
		return fmt.Errorf("setup up local nat: %v\n", err)
	}
	defer cleanupNAT()

	err = server.ListenAndHandleConnections(ctx)
	if err != nil {
		return fmt.Errorf("exit: %v\n", err)
	}
	return nil
}

func execProc(timeout time.Duration, path string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c := exec.CommandContext(ctx, path, args...)
	var out bytes.Buffer
	c.Stdout = &out

	var errout strings.Builder
	c.Stderr = &errout

	err := c.Run()
	io.Copy(os.Stdout, &out)
	if err != nil {
		return errors.New(errout.String())
	}
	return nil
}

func getRemotePort(conn net.Conn) int {
	_, s, _ := net.SplitHostPort(conn.RemoteAddr().String())
	port, _ := strconv.ParseInt(s, 10, 64)
	return int(port)
}
