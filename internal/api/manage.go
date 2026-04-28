package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListProjects calls the Manage API to list projects accessible to the token.
func (c *client) ListProjects(ctx context.Context) ([]Project, error) {
	urlStr := ManageBaseURL + "/projects"
	return doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
			if err != nil {
				return nil, err
			}
			c.setHeaders(req, "")
			return c.httpClient.Do(req)
		},
		func(resp *http.Response) ([]Project, error) {
			defer drainAndClose(resp.Body)
			var projects []Project
			if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
				return nil, fmt.Errorf("decoding projects: %w", err)
			}
			return projects, nil
		},
	)
}

// ListDatasets calls the Manage API to list datasets in a project.
func (c *client) ListDatasets(ctx context.Context, projectID string) ([]Dataset, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}
	urlStr := fmt.Sprintf("%s/projects/%s/datasets", ManageBaseURL, projectID)
	return doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
			if err != nil {
				return nil, err
			}
			c.setHeaders(req, "")
			return c.httpClient.Do(req)
		},
		func(resp *http.Response) ([]Dataset, error) {
			defer drainAndClose(resp.Body)
			var datasets []Dataset
			if err := json.NewDecoder(resp.Body).Decode(&datasets); err != nil {
				return nil, fmt.Errorf("decoding datasets: %w", err)
			}
			return datasets, nil
		},
	)
}
