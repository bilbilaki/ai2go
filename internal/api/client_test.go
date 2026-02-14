package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bilbilaki/ai2go/internal/config"
)

func TestRetryDelayForAttemptIsCapped(t *testing.T) {
	if got := retryDelayForAttempt(1); got != 2*time.Second {
		t.Fatalf("attempt 1: expected 2s, got %s", got)
	}
	if got := retryDelayForAttempt(3); got != 8*time.Second {
		t.Fatalf("attempt 3: expected 8s, got %s", got)
	}
	if got := retryDelayForAttempt(7); got != maxRetryDelay {
		t.Fatalf("attempt 7: expected %s, got %s", maxRetryDelay, got)
	}
}

func TestRetryAfterDelayParsesSecondsAndDate(t *testing.T) {
	d, ok := retryAfterDelay("3")
	if !ok || d != 3*time.Second {
		t.Fatalf("seconds parse failed: ok=%v delay=%s", ok, d)
	}

	future := time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat)
	d, ok = retryAfterDelay(future)
	if !ok {
		t.Fatalf("expected valid date retry-after")
	}
	if d <= 0 || d > maxRetryDelay {
		t.Fatalf("expected bounded positive delay, got %s", d)
	}
}

func TestDoWithRetryHonorsCanceledContext(t *testing.T) {
	c := NewClient(&config.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.doWithRetry(ctx, func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected canceled error, got: %v", err)
	}
}

func TestRetryableStatuses(t *testing.T) {
	codes := []int{
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusInternalServerError,
	}
	for _, code := range codes {
		if !isRetryableStatus(code) {
			t.Fatalf("expected status %d to be retryable", code)
		}
	}
	if isRetryableStatus(http.StatusBadRequest) {
		t.Fatalf("did not expect status %d to be retryable", http.StatusBadRequest)
	}
}
