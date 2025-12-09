package scf

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Error struct {
	Text string

	File string

	// Zero means that error does not correspond to any particular line.
	Line int
}

func (e *Error) Error() string {
	if e.Line <= 0 {
		return e.Text
	}
	if e.File == "" {
		return fmt.Sprintf("line %d: %s", e.Line, e.Text)
	}

	return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Text)
}

type Config interface {
	// Apply raw value to a field with specified name.
	Apply(name, rawValue string) error

	// Validate config values.
	Valid() error
}

// Load read and parse config from file specified by path.
func Load(c Config, path string) error {
	if c == nil {
		panic("nil config")
	}

	path = filepath.Clean(path)
	if path == "" || path == "." || path == ".." {
		return errors.New("empty or invalid config path")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = Parse(c, file)
	if err != nil {
		e, ok := err.(*Error)
		if ok {
			e.File = path
		}
		return err
	}
	return nil
}

// Parse populate config with data from supplied reader.
func Parse(c Config, r io.Reader) error {
	sc := bufio.NewScanner(r)
	ln := 0 // line number
	for sc.Scan() {
		ln += 1
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		split := strings.SplitN(line, ":", 2)
		if len(split) != 2 {
			return &Error{Line: ln, Text: "invalid field format"}
		}

		name := strings.TrimSpace(split[0])
		if name == "" {
			return &Error{Line: ln, Text: "empty field name"}
		}

		rawValue := strings.TrimSpace(split[1])
		if rawValue == "" {
			return &Error{Line: ln, Text: "empty field raw value"}
		}

		err := c.Apply(name, rawValue)
		if err != nil {
			return &Error{Line: ln, Text: fmt.Sprintf("apply field \"%s\" value (=%s): %v", name, rawValue, err)}
		}
	}
	err := sc.Err()
	if err != nil {
		return &Error{Text: err.Error()}
	}
	err = c.Valid()
	if err != nil {
		return fmt.Errorf("check config: %v", err)
	}

	return nil
}
