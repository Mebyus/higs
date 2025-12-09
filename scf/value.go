package scf

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

func ParseStringValue(v string) (string, error) {
	if len(v) < 2 {
		return "", errors.New("invalid string")
	}

	n := len(v)
	if v[0] != '"' && v[n-1] != '"' {
		return "", errors.New("invalid string")
	}

	return v[1 : n-1], nil
}

func ParseUint16Value(v string) (uint16, error) {
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value \"%s\" is not a number", v)
	}
	if n > 0xFFFF {
		return 0, errors.New("number cannot be greater than 65535")
	}

	return uint16(n), nil
}

func ParseLogLevel(v string) (slog.Level, error) {
	level, err := ParseStringValue(v)
	if err != nil {
		return 0, err
	}

	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, errors.New("unknown log level")
	}
}
