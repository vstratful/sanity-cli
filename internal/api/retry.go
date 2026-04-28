package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     DefaultMaxRetries,
		InitialBackoff: DefaultInitialBackoff,
		MaxBackoff:     DefaultMaxBackoff,
	}
}

func isSuccessStatus(code int) bool { return code >= 200 && code < 300 }

func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout ||
		code >= 500
}

func (c *client) shouldRetry(err error, statusCode int, attempt int) bool {
	if c.retry == nil || attempt >= c.retry.MaxRetries {
		return false
	}
	if err != nil {
		return true
	}
	return isRetryableStatus(statusCode)
}

func (c *client) calculateBackoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if c.retry != nil && retryAfter > c.retry.MaxBackoff {
			return c.retry.MaxBackoff
		}
		return retryAfter
	}
	if c.retry == nil {
		return 0
	}
	backoff := c.retry.InitialBackoff
	for i := 0; i < attempt; i++ {
		backoff *= 2
	}
	if backoff > c.retry.MaxBackoff {
		backoff = c.retry.MaxBackoff
	}
	jitterRange := int64(backoff / 10)
	if jitterRange > 0 {
		jitter, _ := rand.Int(rand.Reader, big.NewInt(jitterRange*2))
		backoff += time.Duration(jitter.Int64()) - time.Duration(jitterRange)
	}
	return backoff
}

func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

type requestFunc func(ctx context.Context) (*http.Response, error)
type responseHandler[T any] func(resp *http.Response) (T, error)

func doWithRetry[T any](ctx context.Context, c *client, reqFn requestFunc, handleFn responseHandler[T]) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, fmt.Errorf("context error: %w", err)
		}

		resp, err := reqFn(ctx)
		if err != nil {
			lastErr = fmt.Errorf("sending request: %w", err)
			if c.shouldRetry(err, 0, attempt) {
				if sleepErr := sleep(ctx, c.calculateBackoff(attempt, 0)); sleepErr != nil {
					return zero, sleepErr
				}
				continue
			}
			return zero, lastErr
		}

		statusCode := resp.StatusCode

		if !isSuccessStatus(statusCode) {
			body, readErr := io.ReadAll(resp.Body)
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			resp.Body.Close()

			if readErr != nil {
				lastErr = &APIError{StatusCode: statusCode, Message: fmt.Sprintf("failed to read error body: %v", readErr)}
			} else {
				lastErr = &APIError{StatusCode: statusCode, Body: string(body)}
			}

			if c.shouldRetry(nil, statusCode, attempt) {
				if sleepErr := sleep(ctx, c.calculateBackoff(attempt, retryAfter)); sleepErr != nil {
					return zero, sleepErr
				}
				continue
			}
			return zero, lastErr
		}

		return handleFn(resp)
	}
}
