package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"os"

	"github.com/mebyus/higs/internal/client"
	"github.com/mebyus/higs/proxy"
)

func main() {
	err := StartClient("client.scf")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func StartClient(configPath string) error {
	var config client.Config
	err := client.LoadConfig(&config, configPath)
	if err != nil {
		return err
	}

	tunnel, err := proxy.Connect(context.TODO(), config.ProxyURL, config.AuthToken)
	if err != nil {
		return err
	}

	go tunnel.Serve(context.TODO())

	g := rand.NewChaCha8([32]byte{0, 1, 2, 3})

	conn, err := tunnel.Hello(g, proxy.NewConnID(g), net.IPv4(8, 8, 8, 8), 443)
	if err != nil {
		return err
	}

	_ = conn

	return nil
}
