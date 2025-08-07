//go:build ignore

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var healthy = true

func main() {
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if healthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"UP"}`)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"DOWN"}`)
		}
	})

	// Toggle health status endpoint
	http.HandleFunc("/toggle", func(w http.ResponseWriter, r *http.Request) {
		healthy = !healthy
		status := "UP"
		if !healthy {
			status = "DOWN"
		}
		fmt.Fprintf(w, `{"status":"%s"}`, status)
		log.Printf("Health status toggled to: %s", status)
	})

	// Root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Test HTTP Server for Aegis-Orchestrator")
	})

	// Start server in a goroutine
	go func() {
		log.Println("Starting HTTP server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	time.Sleep(time.Second) // Give time for any in-flight requests to complete
}
