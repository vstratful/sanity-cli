package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// UploadAsset uploads a binary asset. The body is sent as the request body
// (raw bytes; Sanity accepts this rather than multipart). Returns the raw
// `document` field which contains _id, url, etc.
func (c *client) UploadAsset(ctx context.Context, kind AssetKind, body io.Reader, opts *AssetUploadOptions) (json.RawMessage, error) {
	if opts == nil {
		opts = &AssetUploadOptions{}
	}
	pathSeg := "files"
	if kind == AssetKindImage {
		pathSeg = "images"
	}
	hostBase := c.dataHost(false)
	v := c.instance.EffectiveAPIVersion()
	urlStr := fmt.Sprintf("%s/v%s/assets/%s/%s", hostBase, v, pathSeg, c.instance.Dataset)

	q := url.Values{}
	if opts.Filename != "" {
		q.Set("filename", opts.Filename)
	}
	if opts.Label != "" {
		q.Set("label", opts.Label)
	}
	if opts.Title != "" {
		q.Set("title", opts.Title)
	}
	if encoded := q.Encode(); encoded != "" {
		urlStr += "?" + encoded
	}

	contentType := opts.ContentType
	if contentType == "" {
		if kind == AssetKindImage {
			contentType = "application/octet-stream"
		} else {
			contentType = "application/octet-stream"
		}
	}

	// Read the body once so retries can replay it.
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("reading asset body: %w", err)
	}

	resp, err := doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, newBytesReader(raw))
			if err != nil {
				return nil, err
			}
			req.ContentLength = int64(len(raw))
			c.setHeaders(req, contentType)
			return c.httpClient.Do(req)
		},
		func(resp *http.Response) (*AssetResponse, error) {
			defer drainAndClose(resp.Body)
			var ar AssetResponse
			if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
				return nil, fmt.Errorf("decoding asset response: %w", err)
			}
			return &ar, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return resp.Document, nil
}
