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
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func relayData(client, backend Socket) {
	go func() {
		io.Copy(backend, client)
		backend.Close()
	}()
	io.Copy(client, backend)
}

func main() {
	var config Config
	err := LoadConfig(&config, "client.scf")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	ctx := newProcessContext()

	err = setupNAT()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup up local nat: %v\n", err)
		return
	}
	defer cleanupNAT()

	var server Server
	err = server.Init(":8080")
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure local server: %v\n", err)
		return
	}

	err = server.ListenAndHandleConnections(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "exit: %v\n", err)
		return
	}
}

func newProcessContext() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	return ctx
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

func setupNAT() error {
	return execProc(time.Second, "iptables", "-t", "nat",
		"-A", "OUTPUT", "-p", "tcp",
		"-m", "owner", "!", "--uid-owner", "root",
		"--dport", "443",
		"-j", "REDIRECT", "--to-port", "8080",
	)
}

func cleanupNAT() {
	err := disableNAT()
	if err != nil {
		fmt.Printf("Disable NAT: %v\n", err)
	}
}

func disableNAT() error {
	return execProc(time.Second, "iptables", "-t", "nat", "-F")
}

func getRemotePort(conn net.Conn) int {
	_, s, _ := net.SplitHostPort(conn.RemoteAddr().String())
	port, _ := strconv.ParseInt(s, 10, 64)
	return int(port)
}

func getOriginalDestination(conn net.Conn) (net.IP, int, string, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, 0, "", fmt.Errorf("not a tcp connection")
	}

	file, err := tcpConn.File()
	if err != nil {
		return nil, 0, "", err
	}
	defer file.Close()

	fd := file.Fd()
	name, err := getLocalNameByPort(getRemotePort(conn))
	if err != nil {
		fmt.Printf("unable to get proc name for %d socket: %v\n", fd, err)
	}

	// Method 1: Using SO_ORIGINAL_DST
	var addr syscall.RawSockaddrInet4
	addrLen := uint32(unsafe.Sizeof(addr))

	// SO_ORIGINAL_DST constant
	const SO_ORIGINAL_DST = 80

	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(fd),
		syscall.SOL_IP,
		SO_ORIGINAL_DST,
		uintptr(unsafe.Pointer(&addr)),
		uintptr(unsafe.Pointer(&addrLen)),
		0,
	)

	if errno != 0 {
		return nil, 0, "", fmt.Errorf("getsockopt failed: %v", errno)
	}

	if addr.Family != syscall.AF_INET {
		return nil, 0, "", fmt.Errorf("not an IPv4 address")
	}

	ip := net.IPv4(addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3])
	port := int(swapEndianUint16(addr.Port))

	return ip, port, name, nil
}

func swapEndianUint16(v uint16) uint16 {
	return (v >> 8) | (v << 8)
}
