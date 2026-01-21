package main

import (
	"net/netip"
)

type Message struct {
	Quests  []Quest
	Answers []Answer
	Records []Record

	ID uint16

	Opcode Opcode
	Rcode  Rcode
}

// Message header data.
//
// Internal helper struct for parsing and encoding.
type header struct {
	// Packet id.
	//
	// 16 bit identifier assigned by the program that
	// generates any kind of query. This identifier is copied
	// the corresponding reply and can be used by the requester
	// to match up replies to outstanding queries.
	id uint16

	// Number of entries in the question section.
	quests uint16

	// Number of resource records in the answer section.
	answers uint16

	// Number of name server resource records in the authority
	// records section.
	servers uint16

	// Number of resource records in the additional records section.
	records uint16

	// A four bit field that specifies kind of query in this
	// message. This value is set by the originator of a query
	// and copied into the response.
	opcode Opcode

	// Response code. This 4 bit field is set as part of
	// responses.
	rcode Rcode

	// A one bit field that specifies whether this message is a
	// query, or a response.
	//
	// false - means query
	// true  - means response
	resp bool

	// Authoritative Answer - this bit is valid in responses,
	// and specifies that the responding name server is an
	// authority for the domain name in question section.
	//
	// Note that the contents of the answer section may have
	// multiple owner names because of aliases. The AA bit
	// corresponds to the name which matches the query name, or
	// the first owner name in the answer section.
	auth bool

	// Truncate caution. If true specifies that this message
	// was truncated due to length greater than that permitted
	// on the transmission channel.
	trunc bool

	// Recursion desired. This bit may be set in a query and
	// is copied into the response. If true, it directs
	// the name server to pursue the query recursively.
	recd bool

	// Recursion Available. This bit is set or cleared in a
	// response, and denotes whether recursive query support is
	// available in the name server.
	reca bool
}

// Opcode query opcode.
type Opcode uint8

const (
	// Standard query.
	OpQuery Opcode = 0

	// Inverse query.
	OpInvQuery Opcode = 1

	// Server status request.
	OpStatus Opcode = 2
)

type Rcode uint8

const (
	// No error.
	RcOk Rcode = 0

	// Format error.
	RcFormat Rcode = 1

	// Server failure.
	RcFail Rcode = 2

	// Name Error. Meaningful only for responses from an authoritative
	// name server, this code signifies that the
	// domain name referenced in the query does not exist.
	RcNotFound Rcode = 3

	RcRefused Rcode = 5
)

// Type of resource record.
type Type uint16

const (
	// Host address.
	TypeAddr Type = 1

	// Authoritative name server.
	TypeAuth Type = 2

	// Canonical name for an alias.
	TypeCanon Type = 5
)

type Class uint16

const (
	Internet Class = 1
	Chaos    Class = 3
	Hesiod   Class = 4
)

type Quest struct {
	Name  string
	Type  Type
	Class Class
}

type Answer = Record

type Record struct {
	Data []byte

	Name string

	TTL uint32

	Type  Type
	Class Class
}

func (m *Message) addAnswer(name string, ip netip.Addr, ttl uint32) {
	m.Answers = append(m.Answers, Answer{
		Name:  name,
		Type:  TypeAddr,
		Class: Internet,
		TTL:   ttl,
		Data:  ip.AsSlice(),
	})
}

func (m *Message) addAnswers(name string, list []netip.Addr, ttl uint32) {
	if len(list) == 0 {
		panic("empty list")
	}

	for _, ip := range list {
		m.addAnswer(name, ip, ttl)
	}
}
