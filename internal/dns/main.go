package main

import (
	"fmt"
	"os"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	data, err := os.ReadFile(".out/192.168.0.129:41524-1768754489734292.dump")
	if err != nil {
		return err
	}

	var msg Message
	err = Decode(&msg, data)
	return err
}
