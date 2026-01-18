package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"
)

type Resolver struct {
	m map[string]*ResolveEntry
}

func LoadNamesFromFile(r *Resolver, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return parseNames(r, file)
}

type ResolveEntry struct {
	Names []string
	List  []netip.Addr
}

func parseNames(g *Resolver, r io.Reader) error {
	sc := bufio.NewScanner(r)

	var entry ResolveEntry
	var entries []ResolveEntry
	ln := 0 // line number
	for sc.Scan() {
		ln += 1
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasSuffix(line, ":") {
			if len(entry.Names) != 0 {
				if len(entry.List) == 0 {
					return fmt.Errorf("empty address list for names %v", entry.Names)
				}
				entries = append(entries, entry)
			}

			names, err := parseNamesLine(line)
			if err != nil {
				return fmt.Errorf("parse line %d: %v", ln, err)
			}
			entry = ResolveEntry{Names: names}
		} else {
			if len(entry.Names) == 0 {
				return fmt.Errorf("line %d contains address before any names", ln)
			}

			addr, err := netip.ParseAddr(line)
			if err != nil {
				return fmt.Errorf("parse address line %d: %v", ln, err)
			}
			entry.List = append(entry.List, addr)
		}
	}
	err := sc.Err()
	if err != nil {
		return err
	}
	if len(entry.Names) != 0 {
		if len(entry.List) == 0 {
			return fmt.Errorf("empty address list for names %v", entry.Names)
		}
		entries = append(entries, entry)
	}

	g.init(entries)
	return nil
}

func parseAddrList(ss []string) ([]netip.Addr, error) {
	if len(ss) == 0 {
		return nil, nil
	}

	list := make([]netip.Addr, 0, len(ss))
	for _, s := range ss {
		ip, err := netip.ParseAddr(s)
		if err != nil {
			return nil, err
		}
		list = append(list, ip)
	}
	return list, nil
}

func parseNamesLine(line string) ([]string, error) {
	line = strings.TrimSuffix(line, ":")
	split := strings.Split(line, ",")
	if len(split) == 0 {
		return nil, errors.New("bad names list format")
	}

	names := make([]string, 0, len(split))
	for _, s := range split {
		name := strings.TrimSpace(s)
		if name == "" {
			return nil, errors.New("empty name")
		}
		names = append(names, name)
	}
	return names, nil
}

func (r *Resolver) init(entries []ResolveEntry) {
	m := make(map[string]*ResolveEntry)
	for i := range len(entries) {
		entry := &entries[i]

		for _, name := range entry.Names {
			m[name] = entry
		}
	}

	r.m = m
}

// Resolve returns nil if there is no entry with specified name.
func (r *Resolver) Resolve(name string) []netip.Addr {
	entry := r.m[name]
	if entry == nil {
		return nil
	}
	return entry.List
}
