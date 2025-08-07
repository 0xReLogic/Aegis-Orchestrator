package health

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
)

// RetryPolicy defines how retries should be performed
type RetryPolicy struct {
	// MaxRetries is the maximum number of retries
	MaxRetries int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffFactor is the factor by which the backoff increases
	BackoffFactor float64
	// Jitter is the maximum random jitter to add to backoff
	Jitter float64
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0.2,
	}
}

// CalculateBackoff calculates the backoff duration for a retry
func (p *RetryPolicy) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate backoff with exponential increase
	backoff := float64(p.InitialBackoff) * math.Pow(p.BackoffFactor, float64(attempt-1))

	// Apply maximum backoff
	if backoff > float64(p.MaxBackoff) {
		backoff = float64(p.MaxBackoff)
	}

	// Apply jitter
	if p.Jitter > 0 {
		jitter := (1 - p.Jitter) + (2 * p.Jitter * rand.Float64())
		backoff = backoff * jitter
	}

	return time.Duration(backoff)
}

// HealthCheckWithRetry performs a health check with retries according to the policy
func HealthCheckWithRetry(checker plugin.HealthChecker, policy *RetryPolicy) (*plugin.HealthCheckResult, error) {
	var lastResult *plugin.HealthCheckResult
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// If this is a retry, log it
		if attempt > 0 {
			logger.Debug("Retry attempt %d/%d for health check %s", attempt, policy.MaxRetries, checker.Name())

			// Calculate and wait for backoff
			backoff := policy.CalculateBackoff(attempt)
			logger.Debug("Waiting %s before retry", backoff)
			time.Sleep(backoff)
		}

		// Perform the health check
		result, err := checker.Check()

		// If successful, return immediately
		if err == nil && result.Status == plugin.StatusUp {
			if attempt > 0 {
				logger.Info("Health check %s succeeded after %d retries", checker.Name(), attempt)
			}
			return result, nil
		}

		// Store the last result and error
		lastResult = result
		lastErr = err

		// Log the failure
		if err != nil {
			logger.Warn("Health check %s failed with error: %v", checker.Name(), err)
		} else {
			logger.Warn("Health check %s failed with status: %s - %s", checker.Name(), result.Status, result.Message)
		}
	}

	// If we get here, all retries failed
	logger.Error("Health check %s failed after %d retries", checker.Name(), policy.MaxRetries)

	// If we have a result, update it to indicate retries were exhausted
	if lastResult != nil {
		lastResult.Message = fmt.Sprintf("%s (after %d retries)", lastResult.Message, policy.MaxRetries)
		if lastResult.Metadata == nil {
			lastResult.Metadata = make(map[string]string)
		}
		lastResult.Metadata["retries"] = fmt.Sprintf("%d", policy.MaxRetries)
	}

	return lastResult, lastErr
}

// TimeoutPolicy defines how timeouts should be handled
type TimeoutPolicy struct {
	// Timeout is the maximum duration for a health check
	Timeout time.Duration
	// FailOnTimeout indicates whether to fail the check on timeout
	FailOnTimeout bool
}

// DefaultTimeoutPolicy returns a default timeout policy
func DefaultTimeoutPolicy() *TimeoutPolicy {
	return &TimeoutPolicy{
		Timeout:       5 * time.Second,
		FailOnTimeout: true,
	}
}
