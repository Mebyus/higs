package proc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func SetupLogger(path string, level string) (*zap.Logger, io.Closer, error) {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, nil, fmt.Errorf("parse log level from \"%s\" string: %v", level, err)
	}

	var sink zapcore.WriteSyncer
	var closer io.Closer
	if path != "" {
		file, err := createLogFile(path)
		if err != nil {
			return nil, nil, err
		}
		sink = file
		closer = file
	} else {
		sink = os.Stdout
	}

	var encoder zapcore.Encoder
	if path != "" {
		encoder = zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())
	} else {
		encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}
	lg := zap.New(zapcore.NewCore(encoder, sink, lvl))

	return lg, closer, nil
}

func createLogFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			return nil, err
		}
	}
	return os.Create(path)
}
