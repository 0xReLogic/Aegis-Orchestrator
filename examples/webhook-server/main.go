package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Alert represents an alert from Aegis-Orchestrator
type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ServiceName string            `json:"service_name"`
	Severity    string            `json:"severity"`
	State       string            `json:"state"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func main() {
	// Create a simple HTTP server
	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/health", handleHealth)

	// Start the server
	port := 8083
	fmt.Printf("Starting webhook server on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// handleWebhook handles webhook notifications from Aegis-Orchestrator
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse JSON
	var alert Alert
	if err := json.Unmarshal(body, &alert); err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Log the alert
	fmt.Printf("[%s] Alert received: %s - %s (Service: %s)\n",
		time.Now().Format("2006-01-02 15:04:05"),
		alert.Severity,
		alert.Name,
		alert.ServiceName)
	fmt.Printf("  Description: %s\n", alert.Description)
	fmt.Printf("  State: %s\n", alert.State)
	fmt.Printf("  Value: %.2f (Threshold: %.2f)\n", alert.Value, alert.Threshold)
	fmt.Printf("  Start Time: %s\n", alert.StartTime.Format("2006-01-02 15:04:05"))
	if !alert.EndTime.IsZero() {
		fmt.Printf("  End Time: %s\n", alert.EndTime.Format("2006-01-02 15:04:05"))
	}
	if len(alert.Labels) > 0 {
		fmt.Println("  Labels:")
		for k, v := range alert.Labels {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	fmt.Println()

	// Write to log file
	f, err := os.OpenFile("alerts.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		alertJSON, _ := json.MarshalIndent(alert, "", "  ")
		f.WriteString(fmt.Sprintf("[%s] %s\n\n",
			time.Now().Format("2006-01-02 15:04:05"),
			string(alertJSON)))
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"UP"}`))
}
