package client

import (
	"errors"
	"fmt"

	"github.com/mebyus/higs/scf"
)

type Config struct {
	ProxyURL  string
	AuthToken string

	LocalPort uint16
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
	case "local_port":
		var v uint16
		v, err = scf.ParseUint16Value(rawValue)
		c.LocalPort = v
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
	if c.LocalPort == 0 {
		return errors.New("empty or zero local port")
	}
	return nil
}
