package client

import (
	"bufio"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"
)

func loadRoutesFromFile(r *Router, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return parseRoutes(r, file)
}

func parseRoutes(g *Router, r io.Reader) error {
	sc := bufio.NewScanner(r)
	var entries []RouteEntry
	ln := 0 // line number
	for sc.Scan() {
		ln += 1
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, err := parseRoutesLine(line)
		if err != nil {
			return fmt.Errorf("parse line %d: %v", ln, err)
		}
		entries = append(entries, entry)
	}
	err := sc.Err()
	if err != nil {
		return err
	}

	g.init(entries)
	return nil
}

type Action uint8

const (
	// Default value for entries not present in routes file.
	// Actual action depends on client config.
	ActionAuto Action = iota

	// Use direct connection.
	ActionDirect

	// Use proxy connection.
	ActionProxy

	// Block connections.
	ActionBlock
)

var actionText = [...]string{
	ActionAuto:   "auto",
	ActionDirect: "direct",
	ActionProxy:  "proxy",
	ActionBlock:  "block",
}

func (a Action) String() string {
	return actionText[a]
}

type RouteEntry struct {
	IP  netip.Addr
	Act Action
}

func parseRoutesLine(line string) (RouteEntry, error) {
	act := ActionProxy

	fields := strings.Fields(line)
	switch len(fields) {
	case 0:
		panic("impossible condition")
	case 1:
		// do nothing
	case 2:
		s := fields[1]
		switch s {
		case "direct":
			act = ActionDirect
		case "proxy":
			act = ActionProxy
		case "block":
			act = ActionBlock
		default:
			return RouteEntry{}, fmt.Errorf("unknown \"%s\" action", s)
		}
	default:
		return RouteEntry{}, fmt.Errorf("bad format (%d fields)", len(fields))
	}

	ip, err := netip.ParseAddr(fields[0])
	if err != nil {
		return RouteEntry{}, err
	}

	return RouteEntry{
		IP:  ip,
		Act: act,
	}, nil
}

type Router struct {
	m map[netip.Addr]Action
}

func (r *Router) init(entries []RouteEntry) {
	m := make(map[netip.Addr]Action, len(entries))

	for _, entry := range entries {
		m[entry.IP] = entry.Act
	}

	r.m = m
}

func (r *Router) Lookup(ip netip.Addr) Action {
	act, ok := r.m[ip]
	if ok {
		return act
	}
	return ActionAuto
}
