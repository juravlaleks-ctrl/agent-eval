package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "serve":
		// phase 3
		fmt.Println("serve: not implemented yet")
	case "history":
		// phase 4
		fmt.Println("history: not implemented yet")
	case "export":
		// phase 4
		fmt.Println("export: not implemented yet")
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(2)
	}
	_ = args
	_ = flag.CommandLine
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: agent-eval <serve|history|export>")
}
