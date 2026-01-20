package main

import (
	"errors"
	"net/netip"
)

type Resolver interface {
	Resolve(name string) []netip.Addr
}

type Proxy struct {
	resolver Resolver

	servers []netip.Addr
}

var ErrNoQuest = errors.New("no questions in request")

// Handle returns encoded response for a given request encoded request.
func (p *Proxy) Handle(request []byte) ([]byte, error) {
	var msg Message
	err := Decode(&msg, request)
	if err != nil {
		return nil, err
	}
	if len(msg.Quests) == 0 {
		return nil, ErrNoQuest
	}

	if p.resolver != nil {
		if len(msg.Quests) == 1 {
			q := msg.Quests[0]
			list := p.resolver.Resolve(q.Name)
			if len(list) != 0 {
				// TODO: fill and encode answer
				resp := Message{ID: msg.ID}
				resp.addAnswers(q.Name, list)
				return Encode(&resp, nil) // TODO: use local buffer?
			}
		}
	}

	return nil, nil
}
