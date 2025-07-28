package main

import (
	"log"
	"os"

	"github.com/terraconstructs/tcons-signal/test/integration"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run test/helpers.go [up|down|test]")
	}

	command := os.Args[1]
	switch command {
	case "up":
		if err := integration.StartEnvironment(); err != nil {
			log.Fatal(err)
		}
	case "down":
		if err := integration.StopEnvironment(); err != nil {
			log.Fatal(err)
		}
	case "test":
		if err := integration.RunFullTest(); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unknown command: %s. Use 'up', 'down', or 'test'", command)
	}
}
