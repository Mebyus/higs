package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func loadRouter(r *Router, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	sc := bufio.NewScanner(file)
	var list []net.IP
	ln := 0 // line number
	for sc.Scan() {
		ln += 1
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		ip := net.ParseIP(line)
		if ip == nil {
			return fmt.Errorf("bad ip on line %d", ln)
		}
		list = append(list, ip)
	}
	err = sc.Err()
	if err != nil {
		return err
	}

	r.list = list
	return nil
}

type Router struct {
	list []net.IP
}

func (r *Router) ShouldProxy(ip net.IP) bool {
	for _, v := range r.list {
		if ip.Equal(v) {
			return true
		}
	}
	return false
}
