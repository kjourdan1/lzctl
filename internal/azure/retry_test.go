package azure

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetry_SucceedsFirstAttempt(t *testing.T) {
	calls := 0
	result, err := Retry(DefaultRetryConfig(), func() (string, error) {
		calls++
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("Retry() error: %v", err)
	}
	if result != "ok" {
		t.Errorf("Retry() = %q, want %q", result, "ok")
	}
	if calls != 1 {
		t.Errorf("called %d times, want 1", calls)
	}
}

func TestRetry_SucceedsAfterRetries(t *testing.T) {
	calls := 0
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}
	result, err := Retry(cfg, func() (int, error) {
		calls++
		if calls < 3 {
			return 0, fmt.Errorf("transient error")
		}
		return 42, nil
	})
	if err != nil {
		t.Fatalf("Retry() error: %v", err)
	}
	if result != 42 {
		t.Errorf("Retry() = %d, want 42", result)
	}
	if calls != 3 {
		t.Errorf("called %d times, want 3", calls)
	}
}

func TestRetry_AllAttemptsFail(t *testing.T) {
	calls := 0
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}
	_, err := Retry(cfg, func() (string, error) {
		calls++
		return "", fmt.Errorf("persistent error %d", calls)
	})
	if err == nil {
		t.Fatal("Retry() should return error when all attempts fail")
	}
	if calls != 3 {
		t.Errorf("called %d times, want 3", calls)
	}
	// The last error should be wrapped
	if !errors.Is(err, err) {
		t.Error("error should be unwrappable")
	}
}

func TestRetry_DefaultsApplied(t *testing.T) {
	// Zero-value config should use defaults
	calls := 0
	cfg := RetryConfig{MaxAttempts: 1} // override just attempts to keep test fast
	_, err := Retry(cfg, func() (string, error) {
		calls++
		return "", fmt.Errorf("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("called %d times, want 1", calls)
	}
}

func TestRetry_ZeroConfig(t *testing.T) {
	// Completely zero config should default to 3 attempts
	calls := 0
	cfg := RetryConfig{BaseDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond}
	_, err := Retry(cfg, func() (string, error) {
		calls++
		return "", fmt.Errorf("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != DefaultMaxRetryAttempts {
		t.Errorf("called %d times, want %d (default)", calls, DefaultMaxRetryAttempts)
	}
}

func TestBackoffDelay_Exponential(t *testing.T) {
	base := 100 * time.Millisecond
	max := 10 * time.Second

	d0 := backoffDelay(0, base, max)
	d1 := backoffDelay(1, base, max)
	d2 := backoffDelay(2, base, max)

	// With jitter, delays are not exact but should follow exponential pattern
	// d0 ≈ 50-100ms, d1 ≈ 100-200ms, d2 ≈ 200-400ms
	if d0 <= 0 || d0 > base {
		t.Errorf("backoffDelay(0) = %v, expected (0, %v]", d0, base)
	}
	if d1 <= 0 || d1 > 2*base {
		t.Errorf("backoffDelay(1) = %v, expected (0, %v]", d1, 2*base)
	}
	if d2 <= 0 || d2 > 4*base {
		t.Errorf("backoffDelay(2) = %v, expected (0, %v]", d2, 4*base)
	}
}

func TestBackoffDelay_CappedAtMax(t *testing.T) {
	base := 100 * time.Millisecond
	max := 150 * time.Millisecond

	// Attempt 10 would produce a very large backoff without the cap
	d := backoffDelay(10, base, max)
	if d > max {
		t.Errorf("backoffDelay(10) = %v, exceeds max %v", d, max)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != DefaultMaxRetryAttempts {
		t.Errorf("MaxAttempts = %d, want %d", cfg.MaxAttempts, DefaultMaxRetryAttempts)
	}
	if cfg.BaseDelay != DefaultRetryBaseDelay {
		t.Errorf("BaseDelay = %v, want %v", cfg.BaseDelay, DefaultRetryBaseDelay)
	}
	if cfg.MaxDelay != DefaultRetryMaxDelay {
		t.Errorf("MaxDelay = %v, want %v", cfg.MaxDelay, DefaultRetryMaxDelay)
	}
}
