package main

import "fmt"

func printHeader(h *header) {
	if h.resp {
		fmt.Printf("kind: response\n")
	} else {
		fmt.Printf("kind: request\n")
	}
	fmt.Printf("id:   0x%04X\n", h.id)
	fmt.Println()

	fmt.Printf("resp:   %d\n", intbool(h.resp))
	fmt.Printf("opcode: %d\n", h.opcode)
	fmt.Printf("auth:   %d\n", intbool(h.auth))
	fmt.Printf("trunc:  %d\n", intbool(h.trunc))
	fmt.Printf("recd:   %d\n", intbool(h.recd))
	fmt.Printf("reca:   %d\n", intbool(h.reca))
	fmt.Printf("rcode:  %d\n", h.rcode)

	fmt.Println()
	fmt.Printf("quests:  %d\n", h.quests)
	fmt.Printf("answers: %d\n", h.answers)
	fmt.Printf("servers: %d\n", h.servers)
	fmt.Printf("records: %d\n", h.records)
}

func printQuest(q *Quest) {
	fmt.Println()
	fmt.Printf("name:  %s\n", q.Name)
	fmt.Printf("type:  %d\n", q.Type)
	fmt.Printf("class: %d\n", q.Class)
}

func printRecord(r *Record) {
	fmt.Println()
	fmt.Printf("name:  %s\n", r.Name)
	fmt.Printf("type:  %d\n", r.Type)
	fmt.Printf("class: %d\n", r.Class)
	fmt.Printf("ttl:   %d\n", r.TTL)
	fmt.Printf("data:  %v\n", r.Data)
}

func intbool(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}
