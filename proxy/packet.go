package proxy

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/netip"
)

// ConnID connection id.
type ConnID [16]byte

func (c ConnID) String() string {
	return fmt.Sprintf("%x-%x-%x", c[:4], c[4:8], c[8:])
}

func NewConnID(g *rand.ChaCha8) ConnID {
	var cid ConnID
	g.Read(cid[:])
	return cid
}

type PacketType uint8

var typeText = [...]string{
	PacketPing:  "ping",
	PacketHello: "hello",
	PacketClose: "close",
	PacketData:  "data",
	PacketJunk:  "junk",
}

func (t PacketType) String() string {
	if t > PacketJunk {
		return "junk"
	}
	return typeText[t]
}

func (t PacketType) IsJunk() bool {
	return t >= PacketJunk
}

const (
	// Such packets are send between client and server to check tunnel or specific
	// connection state and keep tunnel alive.
	PacketPing PacketType = iota

	// Create new connection over the tunnel.
	//
	// When server receives such packet from client it should start new
	// proxied connection to target specified inside packet data.
	//
	// When client receives such packet from server it indicates that
	// server successfully opened proxied connection to target.
	PacketHello

	// Close connection inside the tunnel or indicate failure to create a new one.
	//
	// When server receives such packet from client it should close connection
	// to target and stop relaying data for this connection.
	//
	// When client receives such packet from server it indicates that connection
	// was closed on behalf of the target and client should stop sending and receiving
	// data for this connection.
	PacketClose

	// Packet with regular connection data transmission between client and server.
	PacketData

	// All other values of PacketType must be considered junk packets.
	// Junk packets are ignored for data transmission and connection
	// managment. However server and client should still check packet
	// checksum in order to detect third-party meddling and intervention.
	PacketJunk
)

// Style determines packet encoding masquerade style.
type Style uint8

const (
	// {"..."}
	Style0 Style = iota

	// ["..."]
	Style1
)

var ErrBadPacketType = errors.New("bad packet type")

func (t PacketType) Valid() error {
	if t > PacketData {
		return ErrBadPacketType
	}
	return nil
}

// Packet carries a single packet of connection data between proxy client and server.
//
// Wire encoding of packets is designed to be harder to detect with DPI methods and
// mimic (at least for packet head and tail) valid json message. Encoding inserts
// random junk data in various places and avoids having fixed header. Each type of
// packet has its own format for carried data. Connection managment packets by
// design have variable length to make detection harder.
//
// Each encoded packet gives a sequence of bytes which starts and ends with characters
// which are valid for json message. Depending on packet style byte sequence will look as:
//
//	{"..."}
//	["..."]
//
// Where instead of ... comes encoded data with variable content. This content is called
// "inner part" of encoded packet. Inner part starts and ends with text junk (tjunk) data.
//
//	<start>
//	tjunk        - 8 bytes
//	tjunk        - varlen    (1 - 8 bytes)
//	junk         - 3 bytes   (reserved for future use)
//	type         - 1 byte    (low 4 bits is type, high 4 bits are junk)
//	cid          - 16 bytes
//	packet data  - varlen    (arbitrary)
//	csum         - 4 bytes   (control sum)
//	tjunk        - varlen    (1 - 8 bytes)
//	tjunk        - 8 bytes
//	<end>
//
// In order to determine length of two varlen tjunk parts during encoding
// and decoding one needs to take djb2 hash of first 8 bytes in inner part
// (with salt). First tjunk varlen comes from lower 3 bits of hash value
// and second comes from next 3 bits.
//
// Control sum is calculated from:
//   - packet salt
//   - 1 chosen byte of first fixed tjunk
//   - 16 bytes of cid
//   - at most 4 bytes of packet data (if present)
//   - 1 chosen byte of last fixed tjunk
//
// What byte to chose in tjunks is determined by lower 3 bits of first byte
// for each corresponding tjunk.
type Packet struct {
	Data []byte

	// Connection id.
	CID ConnID

	// junk padding at packet start
	//
	// we use at most 8 + 8 + 3 bytes of this array
	junk1 [24]byte

	// junk padding at packet end
	//
	// we use at most 8 + 8 bytes of this array
	junk2 [16]byte

	// Used for calculating control sum.
	//
	// Should be randomized for each packet and be calculatable
	// from packet wrapper in wire.
	//
	// Usually we use auth token issued by server when tunnel
	// is established + websocket mask.
	salt uint32

	// Control sum stored in packet.
	csum uint32

	Type PacketType

	// Should be chosen at random.
	style Style

	// Equals true after generating junk for various fields.
	//
	// Used for detecting incorrect usage in encoding.
	ok bool
}

// InitEncode initializes packet fields before it can be used for encoding.
//
// Generates and stores junk for future encoding call.
func (p *Packet) InitEncode(g *rand.ChaCha8, salt uint32) {
	putTextJunk(g, p.junk1[:])
	putTextJunk(g, p.junk2[:])

	v := g.Uint64() // random integer for style and type
	p.style = Style(v & 1)

	if p.Type.IsJunk() {
		// We want to randomize encoded junk type in range [4 - 15],
		// thus it will fit in 4 bits.
		//
		// To do so we add a random integer in range [0 - 11] to 4.
		// Since 11 = 7 + 3 + 1 we can use 3 + 2 + 1 random bits to
		// generate number in range [0 - 11].
		x1 := uint8(v>>8) & 0b1
		x2 := uint8(v>>16) & 0b11
		x3 := uint8(v>>24) & 0b111

		p.Type = PacketJunk + PacketType(x1+x2+x3)

		if len(p.Data) == 0 {
			// generate junk data
			n := 1 + (v>>32)&0x3F // number of junk bytes
			p.Data = make([]byte, n)
			putJunk(g, p.Data)
		}
	}

	p.salt = salt
	p.ok = true
}

// InitDecode initializes packet fields before it can be used for decoding.
func (p *Packet) InitDecode(salt uint32) {
	p.salt = salt
	p.ok = true
}

func (p *Packet) PutHelloTCP(g *rand.ChaCha8, salt uint32, cid ConnID, ap netip.AddrPort) {
	var h Hello
	h.InitEncode(g, NetworkTCP, ap)

	p.CID = cid
	p.Data = EncodeHello(&h, nil)
	p.Type = PacketHello

	p.InitEncode(g, salt)
}

func (p *Packet) PutClose(g *rand.ChaCha8, salt uint32, cid ConnID, cc CloseCode) {
	var s Close
	s.InitEncode(g, cc)

	p.CID = cid
	p.Data = EncodeClose(&s, nil)
	p.Type = PacketClose

	p.InitEncode(g, salt)
}

func (p *Packet) PutData(g *rand.ChaCha8, salt uint32, cid ConnID, data []byte) {
	p.CID = cid
	p.Data = data
	p.Type = PacketData

	p.InitEncode(g, salt)
}

func (p *Packet) PutJunk(g *rand.ChaCha8, salt uint32) {
	p.CID = NewConnID(g)
	p.Type = PacketJunk

	p.InitEncode(g, salt)
}

func putJunk(g *rand.ChaCha8, b []byte) {
	g.Read(b)
}

func putTextJunk(g *rand.ChaCha8, b []byte) {
	putJunk(g, b)
	for i := range len(b) {
		b[i] = mapByteAlphanum(b[i])
	}
}

const mapText = `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_:`

func mapByteAlphanum(b byte) byte {
	return mapText[b&0x3F]
}
