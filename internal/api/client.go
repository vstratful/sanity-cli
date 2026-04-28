// Package api provides a thin HTTP client for the Sanity.io APIs.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vstratful/sanity-cli/internal/config"
)

// Client is the interface for interacting with the Sanity API.
type Client interface {
	// Query runs a GROQ query and returns the raw `result` field.
	Query(ctx context.Context, groq string, params map[string]any) (json.RawMessage, error)
	// Mutate applies a list of mutations atomically.
	Mutate(ctx context.Context, mutations []json.RawMessage, opts *MutateOptions) (*MutateResponse, error)
	// UploadAsset uploads a binary asset.
	UploadAsset(ctx context.Context, kind AssetKind, body io.Reader, opts *AssetUploadOptions) (json.RawMessage, error)
	// ListProjects lists projects accessible to the token via the Manage API.
	ListProjects(ctx context.Context) ([]Project, error)
	// ListDatasets lists datasets in a project via the Manage API.
	ListDatasets(ctx context.Context, projectID string) ([]Dataset, error)
}

// ClientConfig is the construction-time configuration for a Client.
type ClientConfig struct {
	Instance   *config.Instance
	Timeout    time.Duration
	HTTPClient *http.Client
	Retry      *RetryConfig
	UserAgent  string
}

// DefaultClient creates a new Client from an instance with default settings.
func DefaultClient(inst *config.Instance, timeout time.Duration) Client {
	rc := DefaultRetryConfig()
	return NewClient(ClientConfig{
		Instance:  inst,
		Timeout:   timeout,
		Retry:     &rc,
		UserAgent: UserAgent,
	})
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfg ClientConfig) Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = UserAgent
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &client{
		instance:   cfg.Instance,
		httpClient: httpClient,
		retry:      cfg.Retry,
		userAgent:  cfg.UserAgent,
	}
}

type client struct {
	instance   *config.Instance
	httpClient *http.Client
	retry      *RetryConfig
	userAgent  string
}

// dataHost returns the host used for data-plane operations.
// CDN host is only valid for read operations.
func (c *client) dataHost(allowCDN bool) string {
	if allowCDN && c.instance.UseCDN {
		return fmt.Sprintf("https://%s.apicdn.sanity.io", c.instance.ProjectID)
	}
	return fmt.Sprintf("https://%s.api.sanity.io", c.instance.ProjectID)
}

func (c *client) dataPath(operation string) string {
	v := c.instance.EffectiveAPIVersion()
	return fmt.Sprintf("/v%s/%s/%s", v, operation, c.instance.Dataset)
}

func (c *client) setHeaders(req *http.Request, contentType string) {
	if c.instance.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.instance.Token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
}

// drainAndClose ensures the body is fully read & closed before returning.
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
