package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mebyus/higs/internal/server"
	"github.com/mebyus/higs/proc"
	"github.com/mebyus/higs/scf"
)

func main() {
	err := RunServer("server.scf")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func RunServer(configPath string) error {
	var server server.Server
	err := scf.Load(&server.Config, configPath)
	if err != nil {
		return err
	}

	var logSink io.Writer
	if server.Config.LogFile != "" {
		dir := filepath.Dir(server.Config.LogFile)
		if dir != "" && dir != "." {
			err := os.MkdirAll(dir, 0o750)
			if err != nil {
				return err
			}
		}

		file, err := os.Create(server.Config.LogFile)
		if err != nil {
			return err
		}
		defer file.Close()
		logSink = file
	} else {
		logSink = os.Stdout
	}
	lg := slog.New(slog.NewJSONHandler(logSink, &slog.HandlerOptions{
		Level: server.Config.LogLevel,
	}))

	lg.Info("start")
	err = server.Run(proc.NewContext(), lg)
	if err != nil {
		lg.Error("exit", slog.String("error", err.Error()))
		return err
	}
	lg.Info("exit")

	return nil
}
