package server

import (
	"errors"
	"log/slog"

	"github.com/mebyus/higs/scf"
)

type Config struct {
	// Required.
	StaticDir string

	// Required.
	AuthToken string

	// Path to file for writing logs.
	// Standard output will be used if this field is empty.
	LogFile string

	// Zero value means info level.
	LogLevel slog.Level

	// Required.
	//
	// Listen port.
	Port uint16
}

func (c *Config) Apply(name, rawValue string) error {
	var err error
	switch name {
	case "static_dir":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.StaticDir = v
	case "auth_token":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.AuthToken = v
	case "log_file":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.LogFile = v
	case "log_level":
		var l slog.Level
		l, err = scf.ParseLogLevel(rawValue)
		c.LogLevel = l
	case "port":
		var v uint16
		v, err = scf.ParseUint16Value(rawValue)
		c.Port = v
	default:
		return errors.New("unknown field")
	}
	return err
}

func (c *Config) Valid() error {
	if c.StaticDir == "" {
		return errors.New("empty static directory")
	}
	if c.AuthToken == "" {
		return errors.New("empty auth token")
	}
	if c.Port == 0 {
		return errors.New("empty or zero listen port")
	}
	return nil
}
