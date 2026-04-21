package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// pathID returns id percent-encoded for safe interpolation into a URL path.
// BetterStack IDs are numeric, but defense-in-depth costs nothing.
func pathID(id string) string {
	return url.PathEscape(id)
}

const (
	telemetryBaseURL = "https://telemetry.betterstack.com"
	explorationsPath = "/api/v2/explorations"
	alertsPath       = "/api/v2/alerts"
	sourcesPath      = "/api/v2/sources"
)

// TelemetryClient talks to telemetry.betterstack.com (BetterStack Logs).
// It shares the transport pipeline with the uptime Client, guaranteeing
// identical timeout, retry, auth-header, and error semantics.
type TelemetryClient struct {
	t *transport
}

func telemetryBaseURLResolved() string {
	if override := os.Getenv("BETTERSTACK_TELEMETRY_BASE_URL"); override != "" {
		return override
	}
	return telemetryBaseURL
}

func NewTelemetry(token, version string) *TelemetryClient {
	return &TelemetryClient{t: newTransport(token, version, telemetryBaseURLResolved())}
}

func (c *TelemetryClient) withBaseURL(baseURL string) *TelemetryClient {
	c.t.baseURL = baseURL
	return c
}

func (c *TelemetryClient) httpClientField() *http.Client {
	return c.t.httpClient
}

// Explorations

func (c *TelemetryClient) ListExplorations(ctx context.Context, limit int) ([]json.RawMessage, error) {
	return c.t.fetchAll(ctx, c.t.baseURL+explorationsPath, nil, limit)
}

func (c *TelemetryClient) GetExploration(ctx context.Context, id string) (json.RawMessage, error) {
	return c.t.fetchOne(ctx, c.t.baseURL+explorationsPath+"/"+pathID(id))
}

func (c *TelemetryClient) CreateExploration(ctx context.Context, body []byte) (json.RawMessage, error) {
	return c.t.postOne(ctx, c.t.baseURL+explorationsPath, bytes.NewReader(body))
}

func (c *TelemetryClient) UpdateExploration(ctx context.Context, id string, body []byte) (json.RawMessage, error) {
	return c.t.patchOne(ctx, c.t.baseURL+explorationsPath+"/"+pathID(id), bytes.NewReader(body))
}

func (c *TelemetryClient) DeleteExploration(ctx context.Context, id string) error {
	return c.t.deleteOne(ctx, c.t.baseURL+explorationsPath+"/"+pathID(id))
}

// ExplorationAlertsFirstPage fetches one page of the scoped alerts endpoint
// for refcount prechecks. The caller uses per_page=1 to short-circuit.
func (c *TelemetryClient) ExplorationAlertsFirstPage(ctx context.Context, explorationID string, perPage int) ([]json.RawMessage, error) {
	q := url.Values{}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	endpoint := fmt.Sprintf("%s%s/%s/alerts", c.t.baseURL, explorationsPath, pathID(explorationID))
	return c.t.fetchPage(ctx, endpoint, q)
}

// Logs-alerts

func (c *TelemetryClient) ListAlerts(ctx context.Context, limit int) ([]json.RawMessage, error) {
	return c.t.fetchAll(ctx, c.t.baseURL+alertsPath, nil, limit)
}

func (c *TelemetryClient) GetAlert(ctx context.Context, id string) (json.RawMessage, error) {
	return c.t.fetchOne(ctx, c.t.baseURL+alertsPath+"/"+pathID(id))
}

func (c *TelemetryClient) CreateAlert(ctx context.Context, explorationID string, body []byte) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("%s%s/%s/alerts", c.t.baseURL, explorationsPath, pathID(explorationID))
	return c.t.postOne(ctx, endpoint, bytes.NewReader(body))
}

func (c *TelemetryClient) UpdateAlert(ctx context.Context, id string, body []byte) (json.RawMessage, error) {
	return c.t.patchOne(ctx, c.t.baseURL+alertsPath+"/"+pathID(id), bytes.NewReader(body))
}

func (c *TelemetryClient) DeleteAlert(ctx context.Context, id string) error {
	return c.t.deleteOne(ctx, c.t.baseURL+alertsPath+"/"+pathID(id))
}

// IterateAlerts walks every page of /api/v2/alerts, invoking visit on each
// alert. If visit returns true, pagination short-circuits and the alert is
// returned. If pagination completes without a match, (nil, nil) is returned.
func (c *TelemetryClient) IterateAlerts(ctx context.Context, visit func(json.RawMessage) bool) (json.RawMessage, error) {
	nextURL := c.t.baseURL + alertsPath
	for nextURL != "" {
		body, err := c.t.doRequestWithRetry(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, err
		}
		var page PaginatedResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		for _, item := range page.Data {
			if visit(item) {
				return item, nil
			}
		}
		if page.Pagination.Next != nil {
			nextURL = *page.Pagination.Next
		} else {
			nextURL = ""
		}
	}
	return nil, nil
}

// Logs-sources

func (c *TelemetryClient) ListSources(ctx context.Context, limit int) ([]json.RawMessage, error) {
	return c.t.fetchAll(ctx, c.t.baseURL+sourcesPath, nil, limit)
}

func (c *TelemetryClient) GetSource(ctx context.Context, id string) (json.RawMessage, error) {
	return c.t.fetchOne(ctx, c.t.baseURL+sourcesPath+"/"+pathID(id))
}
