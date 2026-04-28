package api

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrRateLimited        = errors.New("rate limited")
	ErrServiceUnavailable = errors.New("service unavailable")
)

// APIError represents an error returned by the Sanity API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

// Unwrap maps known status codes to sentinel errors.
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return nil
	}
}
