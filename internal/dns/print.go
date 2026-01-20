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

	fmt.Printf("opcode: %d\n", h.opcode)

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
