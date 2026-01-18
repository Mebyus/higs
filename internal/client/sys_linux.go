package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type LocalNAT struct{}

func (n *LocalNAT) Setup(tcpPort, udpPort uint16) error {
	err := n.addRule("tcp", 443, tcpPort)
	if err != nil {
		return fmt.Errorf("add tls redirect rule: %v", err)
	}
	err = n.addRule("udp", 53, udpPort)
	if err != nil {
		return fmt.Errorf("add dns redirect rule: %v", err)
	}

	return nil
}

func (n *LocalNAT) Disable() error {
	return execProc(time.Second, "iptables", "-t", "nat", "-F")
}

func (n *LocalNAT) addRule(network string, destPort, redirectPort uint16) error {
	switch network {
	case "tcp", "udp":
		// continue execution
	default:
		panic(fmt.Sprintf("unexpected \"%s\" network", network))
	}

	return execProc(time.Second, "iptables", "-t", "nat",
		"-A", "OUTPUT", "-p", network,
		"-m", "owner", "!", "--uid-owner", "root",
		"--dport", strconv.FormatUint(uint64(destPort), 10),
		"-j", "REDIRECT", "--to-port", strconv.FormatUint(uint64(redirectPort), 10),
	)
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

func getOriginalDestination(conn net.Conn) (netip.AddrPort, string, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return netip.AddrPort{}, "", fmt.Errorf("not a tcp connection")
	}

	file, err := tcpConn.File()
	if err != nil {
		return netip.AddrPort{}, "", err
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
		return netip.AddrPort{}, "", fmt.Errorf("getsockopt failed: %v", errno)
	}

	if addr.Family != syscall.AF_INET {
		return netip.AddrPort{}, "", fmt.Errorf("not an IPv4 address")
	}

	ip := netip.AddrFrom4(addr.Addr)
	port := swapEndianUint16(addr.Port)

	return netip.AddrPortFrom(ip, port), name, nil
}

func swapEndianUint16(v uint16) uint16 {
	return (v >> 8) | (v << 8)
}
