package azure

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig configures exponential backoff retry behavior.
type RetryConfig struct {
	MaxAttempts int           // Maximum number of attempts (default: DefaultMaxRetryAttempts)
	BaseDelay   time.Duration // Initial delay between retries (default: DefaultRetryBaseDelay)
	MaxDelay    time.Duration // Maximum delay cap (default: DefaultRetryMaxDelay)
}

const (
	DefaultMaxRetryAttempts = 3
	DefaultRetryBaseDelay   = 1 * time.Second
	DefaultRetryMaxDelay    = 30 * time.Second
)

// DefaultRetryConfig returns sensible defaults for Azure CLI retries.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: DefaultMaxRetryAttempts,
		BaseDelay:   DefaultRetryBaseDelay,
		MaxDelay:    DefaultRetryMaxDelay,
	}
}

// Retry executes fn with exponential backoff and jitter.
// It retries on any error up to cfg.MaxAttempts times.
// Returns the result of the last attempt if all retries fail.
func Retry[T any](cfg RetryConfig, fn func() (T, error)) (T, error) {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = DefaultMaxRetryAttempts
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = DefaultRetryBaseDelay
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = DefaultRetryMaxDelay
	}

	var lastErr error
	var zero T
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < cfg.MaxAttempts-1 {
			delay := backoffDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)
			time.Sleep(delay)
		}
	}
	return zero, fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// backoffDelay computes delay with exponential backoff and jitter.
func backoffDelay(attempt int, base, max time.Duration) time.Duration {
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if delay > max {
		delay = max
	}
	// Add ±25% jitter
	jitter := time.Duration(rand.Int63n(int64(delay) / 2)) //nolint:gosec // jitter doesn't need crypto/rand
	return delay/2 + jitter
}
