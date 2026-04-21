package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	uptimeBaseURL  = "https://uptime.betterstack.com"
	incidentsPath  = "/api/v3/incidents"
	monitorsPath   = "/api/v2/monitors"
	policiesPath   = "/api/v3/policies"
)

var integrationPaths = map[string]string{
	"aws-cloudwatch": "/api/v2/aws-cloudwatch-integrations",
}

func SupportedIntegrationTypes() []string {
	types := make([]string, 0, len(integrationPaths))
	for k := range integrationPaths {
		types = append(types, k)
	}
	return types
}

func resolveIntegrationPath(integrationType string) (string, error) {
	path, ok := integrationPaths[integrationType]
	if !ok {
		return "", fmt.Errorf("unknown integration type %q; supported types: %s",
			integrationType, strings.Join(SupportedIntegrationTypes(), ", "))
	}
	return path, nil
}

// Client is the uptime-host client. It talks to uptime.betterstack.com using
// a BETTERSTACK_API_TOKEN-compatible token.
type Client struct {
	t *transport
}

// uptimeBaseURLResolved returns the uptime base URL honoring the
// BETTERSTACK_BASE_URL override for tests.
func uptimeBaseURLResolved() string {
	if override := os.Getenv("BETTERSTACK_BASE_URL"); override != "" {
		return override
	}
	return uptimeBaseURL
}

func New(token, version string) *Client {
	return &Client{t: newTransport(token, version, uptimeBaseURLResolved())}
}

// withBaseURL is a test helper that swaps the base URL without going through
// the env-var override.
func (c *Client) withBaseURL(baseURL string) *Client {
	c.t.baseURL = baseURL
	return c
}

// httpClientForTest exposes the underlying http.Client so tests can tweak
// the timeout. Tests use this directly.
func (c *Client) httpClientField() *http.Client {
	return c.t.httpClient
}

type IncidentListParams struct {
	From         string
	To           string
	MonitorID    string
	Resolved     *bool
	Acknowledged *bool
}

type MonitorListParams struct {
	URL               string
	PronounceableName string
	Status            string
}

func (c *Client) ListIncidents(ctx context.Context, params IncidentListParams, limit int) ([]json.RawMessage, error) {
	q := url.Values{}
	if params.From != "" {
		q.Set("from", params.From)
	}
	if params.To != "" {
		q.Set("to", params.To)
	}
	if params.MonitorID != "" {
		q.Set("monitor_id", params.MonitorID)
	}
	if params.Resolved != nil {
		q.Set("resolved", strconv.FormatBool(*params.Resolved))
	}
	if params.Acknowledged != nil {
		q.Set("acknowledged", strconv.FormatBool(*params.Acknowledged))
	}

	return c.t.fetchAll(ctx, c.t.baseURL+incidentsPath, q, limit)
}

func (c *Client) GetIncident(ctx context.Context, id string) (json.RawMessage, error) {
	return c.t.fetchOne(ctx, fmt.Sprintf("%s%s/%s", c.t.baseURL, incidentsPath, id))
}

func (c *Client) ListMonitors(ctx context.Context, params MonitorListParams, limit int) ([]json.RawMessage, error) {
	q := url.Values{}
	if params.URL != "" {
		q.Set("url", params.URL)
	}
	if params.PronounceableName != "" {
		q.Set("pronounceable_name", params.PronounceableName)
	}

	results, err := c.t.fetchAll(ctx, c.t.baseURL+monitorsPath, q, limit)
	if err != nil {
		return nil, err
	}

	if params.Status != "" {
		results = filterByStatus(results, params.Status)
	}

	return results, nil
}

func filterByStatus(items []json.RawMessage, status string) []json.RawMessage {
	filtered := make([]json.RawMessage, 0)
	for _, item := range items {
		var wrapper struct {
			Attributes struct {
				Status string `json:"status"`
			} `json:"attributes"`
		}
		if err := json.Unmarshal(item, &wrapper); err != nil {
			continue
		}
		if wrapper.Attributes.Status == status {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (c *Client) GetMonitor(ctx context.Context, id string) (json.RawMessage, error) {
	return c.t.fetchOne(ctx, fmt.Sprintf("%s%s/%s", c.t.baseURL, monitorsPath, id))
}

func (c *Client) ListIntegrations(ctx context.Context, integrationType string, limit int) ([]json.RawMessage, error) {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return nil, err
	}
	return c.t.fetchAll(ctx, c.t.baseURL+path, nil, limit)
}

func (c *Client) GetIntegration(ctx context.Context, integrationType, id string) (json.RawMessage, error) {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return nil, err
	}
	return c.t.fetchOne(ctx, fmt.Sprintf("%s%s/%s", c.t.baseURL, path, id))
}

func (c *Client) CreateIntegration(ctx context.Context, integrationType, name string) (json.RawMessage, error) {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.t.postOne(ctx, c.t.baseURL+path, bytes.NewReader(payload))
}

func (c *Client) DeleteIntegration(ctx context.Context, integrationType, id string) error {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return err
	}
	return c.t.deleteOne(ctx, fmt.Sprintf("%s%s/%s", c.t.baseURL, path, id))
}

// APIError is the shared Go error type emitted by both the uptime and
// telemetry clients on non-2xx responses.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}
