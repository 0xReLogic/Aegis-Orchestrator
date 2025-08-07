package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	serviceStatus = "UP"
	mu            sync.Mutex
)

func main() {
	// Register handlers
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/toggle", toggleHandler)
	http.HandleFunc("/metrics", metricsHandler)

	// Start server
	port := 8082
	fmt.Printf("Starting example HTTP service on port %d...\n", port)
	fmt.Printf("Health endpoint: http://localhost:%d/health\n", port)
	fmt.Printf("Toggle endpoint: http://localhost:%d/toggle\n", port)
	fmt.Printf("Metrics endpoint: http://localhost:%d/metrics\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// healthHandler returns the health status of the service
func healthHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	status := serviceStatus
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	// Simulate high latency (600ms) to trigger anomaly detection
	latency := 600
	time.Sleep(time.Duration(latency) * time.Millisecond)

	if status == "UP" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"status":"UP","message":"Service is healthy","metadata":{"duration_ms":"%d"}}`, latency)))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf(`{"status":"DOWN","message":"Service is unhealthy","metadata":{"duration_ms":"%d"}}`, latency)))
	}

	fmt.Printf("[%s] Health check: %s (latency: %dms)\n",
		time.Now().Format("2006-01-02 15:04:05"),
		status,
		latency)
}

// toggleHandler toggles the service status between UP and DOWN
func toggleHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	if serviceStatus == "UP" {
		serviceStatus = "DOWN"
	} else {
		serviceStatus = "UP"
	}
	status := serviceStatus
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status":"%s","message":"Service status toggled"}`, status)))

	fmt.Printf("[%s] Service status toggled to: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		status)
}

// metricsHandler returns some example metrics
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	status := serviceStatus
	mu.Unlock()

	statusValue := 1
	if status == "DOWN" {
		statusValue = 0
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	metrics := fmt.Sprintf(`# HELP example_service_status Current status of the service (1=up, 0=down)
# TYPE example_service_status gauge
example_service_status %d

# HELP example_service_requests_total Total number of requests
# TYPE example_service_requests_total counter
example_service_requests_total %d

# HELP example_service_latency_ms Request latency in milliseconds
# TYPE example_service_latency_ms gauge
example_service_latency_ms %.2f
`, statusValue, time.Now().Unix()%1000, 50+float64(time.Now().UnixNano()%150))

	w.Write([]byte(metrics))
}
