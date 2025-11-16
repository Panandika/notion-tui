package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/Panandika/notion-tui/cmd"
)

func main() {
	// Load .env file if it exists (for development)
	// Ignoring error: it's OK if .env doesn't exist
	_ = godotenv.Load()

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
