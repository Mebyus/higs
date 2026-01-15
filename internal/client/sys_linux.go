package client

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

func setupNAT(tcpPort uint16) error {
	return execProc(time.Second, "iptables", "-t", "nat",
		"-A", "OUTPUT", "-p", "tcp",
		"-m", "owner", "!", "--uid-owner", "root",
		"--dport", "443",
		"-j", "REDIRECT", "--to-port", strconv.FormatUint(uint64(tcpPort), 10),
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
