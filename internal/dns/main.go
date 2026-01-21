package main

import (
	"fmt"
	"net"
	"net/netip"
	"os"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	data, err := os.ReadFile(".out/resp.dump")
	if err != nil {
		return err
	}

	var msg Message
	err = Decode(&msg, data)

	// return nil

	ip, err := netip.ParseAddr("8.8.8.8")
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IP(ip.AsSlice()),
		Port: 53,
	})
	if err != nil {
		panic(err)
	}

	n, err := conn.Write(data)
	if err != nil {
		panic(err)
	}
	fmt.Printf("sent %d bytes\n", n)

	var buf [1 << 16]byte
	n, addr, err := conn.ReadFrom(buf[:])
	if err != nil {
		panic(err)
	}
	fmt.Printf("received %d bytes from %s\n", n, addr)

	os.WriteFile(".out/resp.dump", buf[:n], 0o655)

	return err
}
