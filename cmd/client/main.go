package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mebyus/higs/internal/client"
	"github.com/mebyus/higs/proc"
	"github.com/mebyus/higs/proxy"
	"github.com/mebyus/higs/scf"
	"go.uber.org/zap"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "client.scf", "path to client config file")
	flag.Parse()

	err := StartClient(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func StartClient(configPath string) error {
	var config client.Config
	err := scf.Load(&config, configPath)
	if err != nil {
		return fmt.Errorf("load config from \"%s\" file: %v", configPath, err)
	}

	lg, sink, err := proc.SetupLogger(config.LogFile, config.LogLevel)
	if err != nil {
		return fmt.Errorf("setup logger: %v", err)
	}
	if sink != nil {
		defer sink.Close()
	}
	defer lg.Sync()

	mainLog := lg.Named("main")
	defer mainLog.Info("exit")
	mainLog.Info("start")

	return start(proc.NewContext(), lg, &config)
}

func start(ctx context.Context, lg *zap.Logger, config *client.Config) error {
	startLog := lg.Named("init")

	var resolver client.Resolver
	if config.NamesFile != "" {
		path := config.NamesFile
		err := client.LoadNamesFromFile(&resolver, path)
		if err != nil {
			startLog.Error("load local dns config from file", zap.String("path", path), zap.Error(err))
			return fmt.Errorf("load local dns config from \"%s\" file: %v", path, err)
		}
	}

	var router client.Router
	if config.RoutesFile != "" {
		path := config.RoutesFile
		err := client.LoadRoutesFromFile(&router, path)
		if err != nil {
			startLog.Error("load local router config from file", zap.String("path", path), zap.Error(err))
			return fmt.Errorf("load local router config from \"%s\" file: %v", path, err)
		}
	}

	url := config.ProxyURL
	tunnel, err := proxy.Connect(ctx, url, config.AuthToken)
	if err != nil {
		startLog.Error("connect to proxy server", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("connect to proxy server: %v", err)
	}

	var nat client.LocalNAT
	err = nat.Setup(config.LocalTCPPort, config.LocalUDPPort)
	if err != nil {
		startLog.Error("setup local nat", zap.Uint16("tcp.port", config.LocalTCPPort), zap.Uint16("udp.port", config.LocalUDPPort), zap.Error(err))
		return fmt.Errorf("setup local nat: %v", err)
	}
	defer cleanupNAT(lg, &nat)

	go tunnel.Serve(ctx)

	err = client.RunLocalServer(ctx, lg, config, tunnel, &resolver, &router)
	if err != nil {
		lg.Error("run local server", zap.Error(err))
		return fmt.Errorf("run local server: %v", err)
	}

	return nil
}

func cleanupNAT(lg *zap.Logger, nat *client.LocalNAT) {
	err := nat.Disable()
	if err != nil {
		lg.Error("disable local nat", zap.Error(err))
	}
}
