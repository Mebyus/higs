package main

import (
	"errors"
	"net/netip"
	"time"
)

type Resolver interface {
	Resolve(name string) []netip.Addr
}

type Proxy struct {
	resolver Resolver

	cache Cache

	servers []netip.Addr

	minTTL uint32
	maxTTL uint32
}

func NewProxy(r Resolver) *Proxy {
	p := &Proxy{
		resolver: r,

		minTTL: 10 * 60,
		maxTTL: 24 * 60 * 60,
	}
	p.cache.init()
	return p
}

var ErrNoQuest = errors.New("no questions in request")

// Handle returns encoded response for a given request encoded request.
func (p *Proxy) Handle(request []byte) []byte {
	var req Message
	err := Decode(&req, request)
	if err != nil {
		panic("not implemented")
		return nil
	}

	var resp Message
	err = p.handle(&resp, &req)
	if err != nil {
		panic("not implemented")
		return nil
	}

	return Encode(&resp, nil)
}

var ErrTooManyQuest = errors.New("request contains too many questions")

func (p *Proxy) handle(resp *Message, req *Message) error {
	if len(req.Quests) == 0 {
		return ErrNoQuest
	}

	const maxQuest = 8
	if len(req.Quests) > maxQuest {
		return ErrTooManyQuest
	}
	resp.ID = req.ID
	resp.Opcode = req.Opcode

	// flag for each question - was it resolved or not
	var resolved [maxQuest]bool

	// resolved quest count
	var rc int

	for i, q := range req.Quests {
		if q.Name == "" {
			rc += 1
			resolved[i] = true
			continue
		}

		list, ttl := p.getLocal(q.Name)
		if len(list) == 0 {
			continue
		}

		resp.addAnswers(q.Name, list, ttl)
		rc += 1
		resolved[i] = true
	}

	if rc >= len(req.Quests) {
		return nil
	}

	return nil
}

func (p *Proxy) getLocal(name string) ([]netip.Addr, uint32) {
	list := p.resolver.Resolve(name)
	if len(list) != 0 {
		return list, p.maxTTL
	}

	return p.cache.Get(name, time.Now().Unix())
}
