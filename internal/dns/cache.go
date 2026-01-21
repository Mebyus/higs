package main

import (
	"net/netip"
	"sync"
	"time"
)

// cache entry
type cent struct {
	list []netip.Addr

	// When entry was saved.
	//
	// Unix timestamp in seconds.
	ts int64

	// Lifetime in seconds.
	ttl uint32
}

type Cache struct {
	mu sync.Mutex

	m map[ /* domain name */ string]cent
}

func (c *Cache) init() {
	c.m = make(map[string]cent)
}

func (c *Cache) Set(name string, entry cent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[name] = entry
}

// Get returns stored address list and its remaining ttl.
func (c *Cache) Get(name string, now int64) ([]netip.Addr, uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.m[name]
	if !ok {
		return nil, 0
	}

	if now-int64(entry.ttl) >= entry.ts {
		// entry expired
		return nil, 0
	}

	return entry.list, entry.ttl - uint32(now-entry.ts)
}

func nowcent(list []netip.Addr, ttl uint32) cent {
	return cent{
		list: list,
		ttl:  ttl,
		ts:   time.Now().Unix(),
	}
}
