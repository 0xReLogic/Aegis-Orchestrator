package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var running = true

func main() {
	// Start TCP server
	listener, err := net.Listen("tcp", ":5433")
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	log.Println("TCP server listening on :5433")

	// Handle connections in a goroutine
	go func() {
		for {
			if !running {
				time.Sleep(time.Second)
				continue
			}

			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v", err)
				continue
			}

			go handleConnection(conn)
		}
	}()

	// Toggle server status every 30 seconds
	go func() {
		for {
			time.Sleep(30 * time.Second)
			running = !running
			status := "UP"
			if !running {
				status = "DOWN"
			}
			log.Printf("TCP server status toggled to: %s", status)
		}
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	time.Sleep(time.Second) // Give time for any in-flight connections to complete
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Simple response
	fmt.Fprintf(conn, "Aegis-Orchestrator TCP Test Server\n")
}
