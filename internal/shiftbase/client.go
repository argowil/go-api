// Package shiftbase is a thin wrapper around the Shiftbase REST API.
//
// Authentication: every request carries the API key in the
// "Authorization: API <key>" header, exactly as Shiftbase's developer
// portal documents.
//
// Base URL: https://api.shiftbase.com/api (configurable via SHIFTBASE_BASE_URL).
//
// If you have no Shiftbase subscription you can leave SHIFTBASE_API_KEY empty;
// the schedule module will fall back to the local database in that case.
package shiftbase

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client speaks to the Shiftbase REST API.
type Client struct {
	baseURL            string
	apiKey             string
	createEmployeePath string
	httpClient         *http.Client
}

// NewClient creates a Shiftbase API client.
// baseURL is typically "https://api.shiftbase.com/api".
func NewClient(baseURL, apiKey, createEmployeePath string) *Client {
	if createEmployeePath == "" {
		createEmployeePath = "/users"
	}
	return &Client{
		baseURL:            baseURL,
		apiKey:             apiKey,
		createEmployeePath: createEmployeePath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Enabled returns false when no API key is configured,
// allowing callers to skip Shiftbase and use local data instead.
func (c *Client) Enabled() bool {
	return c.apiKey != ""
}

// get performs an authenticated GET request and decodes the JSON response into dst.
func (c *Client) get(path string, dst any) error {
	url := c.baseURL + path
	log.Printf("shiftbase GET %s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "API "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shiftbase request %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("shiftbase GET %s -> %d %s", url, resp.StatusCode, string(body))
		return fmt.Errorf("shiftbase %s returned %d: %s", path, resp.StatusCode, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode shiftbase response: %w", err)
	}
	return nil
}
