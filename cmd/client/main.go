package main

import (
	"fmt"
	"os"

	"github.com/mebyus/higs/internal/client"
	"github.com/mebyus/higs/proc"
	"github.com/mebyus/higs/proxy"
	"github.com/mebyus/higs/scf"
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
	err := scf.Load(&config, configPath)
	if err != nil {
		return err
	}

	ctx := proc.NewContext()

	tunnel, err := proxy.Connect(ctx, config.ProxyURL, config.AuthToken)
	if err != nil {
		return err
	}

	go tunnel.Serve(ctx)

	return client.RunLocalServer(ctx, &config, tunnel)
}
