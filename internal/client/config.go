package client

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ProxyURL  string
	AuthToken string

	LocalPort uint16
}

func LoadConfig(c *Config, path string) error {
	if path == "" || path == "." || path == ".." {
		return errors.New("empty or invalid config path")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	sc := bufio.NewScanner(file)
	ln := 0 // line number
	for sc.Scan() {
		ln += 1
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		err := c.apply(line)
		if err != nil {
			return fmt.Errorf("load config %s:%d: %v", path, ln, err)
		}
	}
	err = sc.Err()
	if err != nil {
		return err
	}
	err = c.Valid()
	if err != nil {
		return fmt.Errorf("check config: %v", err)
	}

	return nil
}

func (c *Config) apply(line string) error {
	split := strings.SplitN(line, ":", 2)
	if len(split) != 2 {
		return errors.New("invalid field format")
	}

	name := strings.TrimSpace(split[0])
	value := strings.TrimSpace(split[1])
	if value == "" {
		return errors.New("empty field value")
	}

	var err error
	switch name {
	case "":
		return errors.New("empty field name")
	case "proxy_url":
		var v string
		v, err = parseStringValue(value)
		c.ProxyURL = v
	case "auth_token":
		var v string
		v, err = parseStringValue(value)
		c.AuthToken = v
	case "local_port":
		var v uint16
		v, err = parseUint16Value(value)
		c.LocalPort = v
	default:
		return fmt.Errorf("unknown field \"%s\"", name)
	}
	return err
}

func parseStringValue(v string) (string, error) {
	if len(v) < 2 {
		return "", errors.New("invalid string")
	}

	n := len(v)
	if v[0] != '"' && v[n-1] != '"' {
		return "", errors.New("invalid string")
	}

	return v[1 : n-1], nil
}

func parseUint16Value(v string) (uint16, error) {
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value \"%s\" is not a number", v)
	}
	if n > 0xFFFF {
		return 0, errors.New("number cannot be greater than 65535")
	}

	return uint16(n), nil
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
