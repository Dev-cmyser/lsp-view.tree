package main

import (
	"log"
	"os"
)

func main() {
	// Set up logging to stderr (LSP uses stdin/stdout for communication)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	log.Println("[view.tree] Starting LSP server...")
	
	// Create and start the LSP server
	server := NewServer()
	if err := server.Run(); err != nil {
		log.Fatalf("[view.tree] Server failed: %v", err)
	}
}