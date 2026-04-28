package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Query runs a GROQ query against the dataset.
//
// For short queries we use GET (URL params), for long queries POST with a JSON
// body — Sanity supports both.
func (c *client) Query(ctx context.Context, groq string, params map[string]any) (json.RawMessage, error) {
	if groq == "" {
		return nil, fmt.Errorf("query is empty")
	}

	hostBase := c.dataHost(true)
	path := c.dataPath("data/query")
	perspective := c.instance.EffectivePerspective()

	// Estimate URL length to decide GET vs POST.
	q := url.Values{}
	q.Set("query", groq)
	for k, v := range params {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshaling param %q: %w", k, err)
		}
		q.Set("$"+k, string(b))
	}
	if perspective != "" {
		q.Set("perspective", perspective)
	}

	encoded := q.Encode()
	usePost := len(hostBase)+len(path)+len(encoded)+1 > QueryGetThreshold

	resp, err := doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			if usePost {
				body := map[string]any{"query": groq}
				if len(params) > 0 {
					body["params"] = params
				}
				if perspective != "" {
					body["perspective"] = perspective
				}
				buf, err := json.Marshal(body)
				if err != nil {
					return nil, fmt.Errorf("marshaling query body: %w", err)
				}
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, hostBase+path, bytes.NewReader(buf))
				if err != nil {
					return nil, err
				}
				c.setHeaders(req, "application/json")
				return c.httpClient.Do(req)
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, hostBase+path+"?"+encoded, nil)
			if err != nil {
				return nil, err
			}
			c.setHeaders(req, "")
			return c.httpClient.Do(req)
		},
		func(resp *http.Response) (*QueryResponse, error) {
			defer drainAndClose(resp.Body)
			var qr QueryResponse
			if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
				return nil, fmt.Errorf("decoding query response: %w", err)
			}
			return &qr, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}
