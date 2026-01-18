package client

import (
	"errors"
	"fmt"

	"github.com/mebyus/higs/scf"
)

type Config struct {
	ProxyURL  string
	AuthToken string

	RoutesFile string

	NamesFile string

	// Path to file for writing logs.
	// Standard output will be used if this field is empty.
	LogFile string

	// Zero value means info level.
	LogLevel string

	LocalTCPPort uint16
	LocalUDPPort uint16
}

func (c *Config) Apply(name, rawValue string) error {
	var err error
	switch name {
	case "proxy_url":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.ProxyURL = v
	case "auth_token":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.AuthToken = v
	case "routes_file":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.RoutesFile = v
	case "names_file":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.NamesFile = v
	case "log_file":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.LogFile = v
	case "log_level":
		var v string
		v, err = scf.ParseStringValue(rawValue)
		c.LogLevel = v
	case "local_tcp_port":
		var v uint16
		v, err = scf.ParseUint16Value(rawValue)
		c.LocalTCPPort = v
	case "local_udp_port":
		var v uint16
		v, err = scf.ParseUint16Value(rawValue)
		c.LocalUDPPort = v
	default:
		return fmt.Errorf("unknown field")
	}
	return err
}

func (c *Config) Valid() error {
	if c.ProxyURL == "" {
		return errors.New("empty proxy url")
	}
	if c.AuthToken == "" {
		return errors.New("empty auth token")
	}
	if c.LocalTCPPort == 0 {
		return errors.New("empty or zero local tcp port")
	}
	if c.LocalUDPPort == 0 {
		return errors.New("empty or zero local udp port")
	}
	return nil
}
