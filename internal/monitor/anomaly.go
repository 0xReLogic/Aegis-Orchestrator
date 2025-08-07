package monitor

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
)

// AnomalyType represents the type of anomaly
type AnomalyType string

const (
	// AnomalyTypeLatency represents a latency anomaly
	AnomalyTypeLatency AnomalyType = "LATENCY"
	// AnomalyTypeThroughput represents a throughput anomaly
	AnomalyTypeThroughput AnomalyType = "THROUGHPUT"
	// AnomalyTypeErrorRate represents an error rate anomaly
	AnomalyTypeErrorRate AnomalyType = "ERROR_RATE"
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	ServiceName string
	Type        AnomalyType
	Value       float64
	Threshold   float64
	Timestamp   time.Time
	Description string
}

// MetricSample represents a sample of a metric
type MetricSample struct {
	Value     float64
	Timestamp time.Time
}

// AnomalyDetector detects anomalies in metrics
type AnomalyDetector struct {
	// Metrics history
	latencyHistory    map[string][]MetricSample
	throughputHistory map[string][]MetricSample
	errorRateHistory  map[string][]MetricSample

	// Thresholds
	latencyThresholds    map[string]float64
	throughputThresholds map[string]float64
	errorRateThresholds  map[string]float64

	// Configuration
	historySize       int
	detectionInterval time.Duration

	// State
	anomalies []Anomaly
	mu        sync.RWMutex
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(historySize int, detectionInterval time.Duration) *AnomalyDetector {
	return &AnomalyDetector{
		latencyHistory:       make(map[string][]MetricSample),
		throughputHistory:    make(map[string][]MetricSample),
		errorRateHistory:     make(map[string][]MetricSample),
		latencyThresholds:    make(map[string]float64),
		throughputThresholds: make(map[string]float64),
		errorRateThresholds:  make(map[string]float64),
		historySize:          historySize,
		detectionInterval:    detectionInterval,
		anomalies:            make([]Anomaly, 0),
		stopChan:             make(chan struct{}),
	}
}

// Start starts the anomaly detector
func (a *AnomalyDetector) Start() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(a.detectionInterval)
		defer ticker.Stop()

		for {
			select {
			case <-a.stopChan:
				logger.Info("Stopping anomaly detector")
				return
			case <-ticker.C:
				a.detectAnomalies()
			}
		}
	}()

	logger.Info("Started anomaly detector with interval %s", a.detectionInterval)
}

// Stop stops the anomaly detector
func (a *AnomalyDetector) Stop() {
	close(a.stopChan)
	a.wg.Wait()
	logger.Info("Anomaly detector stopped")
}

// AddLatencySample adds a latency sample for a service
func (a *AnomalyDetector) AddLatencySample(serviceName string, latency float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sample := MetricSample{
		Value:     latency,
		Timestamp: time.Now(),
	}

	// Add sample to history
	a.latencyHistory[serviceName] = append(a.latencyHistory[serviceName], sample)

	// Trim history if needed
	if len(a.latencyHistory[serviceName]) > a.historySize {
		a.latencyHistory[serviceName] = a.latencyHistory[serviceName][1:]
	}
}

// AddThroughputSample adds a throughput sample for a service
func (a *AnomalyDetector) AddThroughputSample(serviceName string, throughput float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sample := MetricSample{
		Value:     throughput,
		Timestamp: time.Now(),
	}

	// Add sample to history
	a.throughputHistory[serviceName] = append(a.throughputHistory[serviceName], sample)

	// Trim history if needed
	if len(a.throughputHistory[serviceName]) > a.historySize {
		a.throughputHistory[serviceName] = a.throughputHistory[serviceName][1:]
	}
}

// AddErrorRateSample adds an error rate sample for a service
func (a *AnomalyDetector) AddErrorRateSample(serviceName string, errorRate float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sample := MetricSample{
		Value:     errorRate,
		Timestamp: time.Now(),
	}

	// Add sample to history
	a.errorRateHistory[serviceName] = append(a.errorRateHistory[serviceName], sample)

	// Trim history if needed
	if len(a.errorRateHistory[serviceName]) > a.historySize {
		a.errorRateHistory[serviceName] = a.errorRateHistory[serviceName][1:]
	}
}

// SetLatencyThreshold sets the latency threshold for a service
func (a *AnomalyDetector) SetLatencyThreshold(serviceName string, threshold float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.latencyThresholds[serviceName] = threshold
}

// SetThroughputThreshold sets the throughput threshold for a service
func (a *AnomalyDetector) SetThroughputThreshold(serviceName string, threshold float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.throughputThresholds[serviceName] = threshold
}

// SetErrorRateThreshold sets the error rate threshold for a service
func (a *AnomalyDetector) SetErrorRateThreshold(serviceName string, threshold float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.errorRateThresholds[serviceName] = threshold
}

// GetAnomalies returns the detected anomalies
func (a *AnomalyDetector) GetAnomalies() []Anomaly {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make([]Anomaly, len(a.anomalies))
	copy(result, a.anomalies)

	return result
}

// ClearAnomalies clears the detected anomalies
func (a *AnomalyDetector) ClearAnomalies() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.anomalies = make([]Anomaly, 0)
}

// detectAnomalies detects anomalies in the metrics
func (a *AnomalyDetector) detectAnomalies() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Detect latency anomalies
	for serviceName, samples := range a.latencyHistory {
		threshold, ok := a.latencyThresholds[serviceName]
		if !ok {
			// No threshold set, use dynamic threshold based on standard deviation
			threshold = a.calculateDynamicThreshold(samples)
		}

		// Need at least a few samples to detect anomalies
		if len(samples) < 3 {
			continue
		}

		// Get the latest sample
		latestSample := samples[len(samples)-1]

		// Check if the latest sample exceeds the threshold
		if latestSample.Value > threshold {
			anomaly := Anomaly{
				ServiceName: serviceName,
				Type:        AnomalyTypeLatency,
				Value:       latestSample.Value,
				Threshold:   threshold,
				Timestamp:   latestSample.Timestamp,
				Description: fmt.Sprintf("Latency of %.2fms exceeds threshold of %.2fms", latestSample.Value, threshold),
			}

			a.anomalies = append(a.anomalies, anomaly)
			logger.Warn("Detected latency anomaly for service %s: %.2fms (threshold: %.2fms)",
				serviceName, latestSample.Value, threshold)
		}
	}

	// Detect throughput anomalies (low throughput)
	for serviceName, samples := range a.throughputHistory {
		threshold, ok := a.throughputThresholds[serviceName]
		if !ok {
			// No threshold set, skip
			continue
		}

		// Need at least a few samples to detect anomalies
		if len(samples) < 3 {
			continue
		}

		// Get the latest sample
		latestSample := samples[len(samples)-1]

		// Check if the latest sample is below the threshold
		if latestSample.Value < threshold {
			anomaly := Anomaly{
				ServiceName: serviceName,
				Type:        AnomalyTypeThroughput,
				Value:       latestSample.Value,
				Threshold:   threshold,
				Timestamp:   latestSample.Timestamp,
				Description: fmt.Sprintf("Throughput of %.2f requests/sec below threshold of %.2f",
					latestSample.Value, threshold),
			}

			a.anomalies = append(a.anomalies, anomaly)
			logger.Warn("Detected throughput anomaly for service %s: %.2f req/s (threshold: %.2f)",
				serviceName, latestSample.Value, threshold)
		}
	}

	// Detect error rate anomalies
	for serviceName, samples := range a.errorRateHistory {
		threshold, ok := a.errorRateThresholds[serviceName]
		if !ok {
			// No threshold set, skip
			continue
		}

		// Need at least a few samples to detect anomalies
		if len(samples) < 3 {
			continue
		}

		// Get the latest sample
		latestSample := samples[len(samples)-1]

		// Check if the latest sample exceeds the threshold
		if latestSample.Value > threshold {
			anomaly := Anomaly{
				ServiceName: serviceName,
				Type:        AnomalyTypeErrorRate,
				Value:       latestSample.Value,
				Threshold:   threshold,
				Timestamp:   latestSample.Timestamp,
				Description: fmt.Sprintf("Error rate of %.2f%% exceeds threshold of %.2f%%",
					latestSample.Value*100, threshold*100),
			}

			a.anomalies = append(a.anomalies, anomaly)
			logger.Warn("Detected error rate anomaly for service %s: %.2f%% (threshold: %.2f%%)",
				serviceName, latestSample.Value*100, threshold*100)
		}
	}
}

// calculateDynamicThreshold calculates a dynamic threshold based on the mean and standard deviation
func (a *AnomalyDetector) calculateDynamicThreshold(samples []MetricSample) float64 {
	// Need at least a few samples
	if len(samples) < 3 {
		return 1000.0 // Default high threshold
	}

	// Calculate mean
	var sum float64
	for _, sample := range samples {
		sum += sample.Value
	}
	mean := sum / float64(len(samples))

	// Calculate standard deviation
	var variance float64
	for _, sample := range samples {
		variance += math.Pow(sample.Value-mean, 2)
	}
	variance /= float64(len(samples))
	stdDev := math.Sqrt(variance)

	// Threshold is mean + 3 standard deviations (covers 99.7% of normal distribution)
	return mean + 3*stdDev
}
