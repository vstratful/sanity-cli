package api

import "time"

const (
	// QueryGetThreshold is the URL-length threshold (in bytes) above which
	// GROQ queries are sent via POST instead of GET. Sanity accepts both.
	QueryGetThreshold = 1024

	// ManageBaseURL is the host for the Sanity Manage API.
	ManageBaseURL = "https://api.sanity.io/v2021-06-07"

	// DefaultTimeout is the default HTTP timeout.
	DefaultTimeout = 2 * time.Minute

	// DefaultMaxRetries is the default maximum number of retries.
	DefaultMaxRetries = 3

	// DefaultInitialBackoff is the default initial backoff duration.
	DefaultInitialBackoff = 500 * time.Millisecond

	// DefaultMaxBackoff is the default maximum backoff duration.
	DefaultMaxBackoff = 10 * time.Second

	// UserAgent is sent on every request.
	UserAgent = "sanity-cli (+https://github.com/vstratful/sanity-cli)"
)
