package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Mutate applies the supplied mutations atomically. Mutations are passed as
// already-encoded JSON values (each one a single mutation object).
func (c *client) Mutate(ctx context.Context, mutations []json.RawMessage, opts *MutateOptions) (*MutateResponse, error) {
	if len(mutations) == 0 {
		return nil, fmt.Errorf("at least one mutation is required")
	}

	hostBase := c.dataHost(false) // never CDN for writes
	path := c.dataPath("data/mutate")

	body := map[string]any{"mutations": mutations}
	if opts != nil && opts.TransactionID != "" {
		body["transactionId"] = opts.TransactionID
	}

	q := url.Values{}
	if opts != nil {
		if opts.ReturnIDs {
			q.Set("returnIds", "true")
		}
		if opts.ReturnDocuments {
			q.Set("returnDocuments", "true")
		}
		if opts.Visibility != "" {
			q.Set("visibility", opts.Visibility)
		}
		if opts.DryRun {
			q.Set("dryRun", "true")
		}
		if opts.AutoGenerateKeys {
			q.Set("autoGenerateArrayKeys", "true")
		}
	}

	urlStr := hostBase + path
	if encoded := q.Encode(); encoded != "" {
		urlStr += "?" + encoded
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling mutations: %w", err)
	}

	return doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(buf))
			if err != nil {
				return nil, err
			}
			c.setHeaders(req, "application/json")
			return c.httpClient.Do(req)
		},
		func(resp *http.Response) (*MutateResponse, error) {
			defer drainAndClose(resp.Body)
			var mr MutateResponse
			raw, err := readAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("reading mutate response: %w", err)
			}
			// Sanity returns either {transactionId, results} or
			// {transactionId, documents}. Decode both into RawMessage.
			var probe struct {
				TransactionID string          `json:"transactionId,omitempty"`
				Results       json.RawMessage `json:"results,omitempty"`
				Documents     json.RawMessage `json:"documents,omitempty"`
			}
			if err := json.Unmarshal(raw, &probe); err != nil {
				return nil, fmt.Errorf("decoding mutate response: %w", err)
			}
			mr.TransactionID = probe.TransactionID
			mr.Results = probe.Results
			mr.Documents = probe.Documents
			return &mr, nil
		},
	)
}
