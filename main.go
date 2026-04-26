package main

import (
	"log"
	"os"
	"whole-pump/compute"
	"whole-pump/snapshot"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <command> [args]\nCommands: transfers, holders, stats, snapshot")
	}
	switch os.Args[1] {
	case "snapshot":
		snapshot.Run(os.Args[2:])
	case "compute":
		compute.Run(os.Args[2:])

	default:
		log.Fatalf("Unknown command: %s", os.Args[1])
	}
}
