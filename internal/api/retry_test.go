package api

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestIsSuccessStatus(t *testing.T) {
	cases := map[int]bool{
		200: true, 204: true, 299: true,
		199: false, 300: false, 400: false, 500: false,
	}
	for code, want := range cases {
		if got := isSuccessStatus(code); got != want {
			t.Errorf("isSuccessStatus(%d)=%v, want %v", code, got, want)
		}
	}
}

func TestIsRetryableStatus(t *testing.T) {
	retryable := []int{
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		500, 502, 504, 599,
	}
	notRetryable := []int{200, 201, 400, 401, 403, 404, 422}
	for _, c := range retryable {
		if !isRetryableStatus(c) {
			t.Errorf("isRetryableStatus(%d)=false, want true", c)
		}
	}
	for _, c := range notRetryable {
		if isRetryableStatus(c) {
			t.Errorf("isRetryableStatus(%d)=true, want false", c)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter(""); got != 0 {
		t.Errorf("empty header => %v, want 0", got)
	}
	if got := parseRetryAfter("3"); got != 3*time.Second {
		t.Errorf("seconds=3 => %v, want 3s", got)
	}
	if got := parseRetryAfter("garbage"); got != 0 {
		t.Errorf("garbage => %v, want 0", got)
	}
	// HTTP-date in the future
	future := time.Now().Add(5 * time.Second).UTC().Format(http.TimeFormat)
	got := parseRetryAfter(future)
	if got <= 0 || got > 6*time.Second {
		t.Errorf("future http-date => %v, want ~5s", got)
	}
	// HTTP-date in the past
	past := time.Now().Add(-5 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(past); got != 0 {
		t.Errorf("past http-date => %v, want 0", got)
	}
}

func TestCalculateBackoffExponential(t *testing.T) {
	c := &client{retry: &RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
	}}
	// Attempt 0 should be roughly 100ms (with up to 10% jitter).
	got := c.calculateBackoff(0, 0)
	if got < 80*time.Millisecond || got > 120*time.Millisecond {
		t.Errorf("attempt 0 backoff=%v, want ~100ms", got)
	}
	// Attempt 4 doubles past MaxBackoff and should be clamped.
	got = c.calculateBackoff(4, 0)
	if got > 2*time.Second+200*time.Millisecond { // some jitter slack
		t.Errorf("attempt 4 backoff=%v should be clamped near MaxBackoff", got)
	}
}

func TestCalculateBackoffRetryAfter(t *testing.T) {
	c := &client{retry: &RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
	}}
	// Retry-After takes priority and is clamped to MaxBackoff.
	if got := c.calculateBackoff(0, 500*time.Millisecond); got != 500*time.Millisecond {
		t.Errorf("retry-after=500ms => %v, want 500ms", got)
	}
	if got := c.calculateBackoff(0, 30*time.Second); got != 2*time.Second {
		t.Errorf("retry-after=30s clamps to MaxBackoff=2s, got %v", got)
	}
}

func TestShouldRetryRespectsMax(t *testing.T) {
	c := &client{retry: &RetryConfig{MaxRetries: 2}}
	if !c.shouldRetry(nil, 503, 0) {
		t.Error("attempt 0 of 2 should retry")
	}
	if !c.shouldRetry(nil, 503, 1) {
		t.Error("attempt 1 of 2 should retry")
	}
	if c.shouldRetry(nil, 503, 2) {
		t.Error("attempt 2 of 2 should not retry")
	}
}

func TestAPIErrorUnwrap(t *testing.T) {
	cases := []struct {
		code int
		want error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusServiceUnavailable, ErrServiceUnavailable},
		{http.StatusBadRequest, nil},
	}
	for _, c := range cases {
		t.Run(strconv.Itoa(c.code), func(t *testing.T) {
			e := &APIError{StatusCode: c.code}
			if got := e.Unwrap(); got != c.want {
				t.Errorf("Unwrap()=%v, want %v", got, c.want)
			}
		})
	}
}
