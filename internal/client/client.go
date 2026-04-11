package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout   = 30 * time.Second
	maxRetries       = 3
	uptimeBaseURL    = "https://uptime.betterstack.com"
	incidentsPath = "/api/v3/incidents"
	monitorsPath  = "/api/v2/monitors"
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

type Client struct {
	token      string
	version    string
	baseURL    string
	httpClient *http.Client
}

func New(token, version string) *Client {
	base := uptimeBaseURL
	if override := os.Getenv("BETTERSTACK_BASE_URL"); override != "" {
		base = override
	}
	return &Client{
		token:   token,
		version: version,
		baseURL: base,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) withBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

type PaginatedResponse struct {
	Data       []json.RawMessage `json:"data"`
	Pagination struct {
		Next *string `json:"next"`
	} `json:"pagination"`
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

	return c.fetchAll(ctx, c.baseURL+incidentsPath, q, limit)
}

func (c *Client) GetIncident(ctx context.Context, id string) (json.RawMessage, error) {
	return c.fetchOne(ctx, fmt.Sprintf("%s%s/%s", c.baseURL, incidentsPath, id))
}

func (c *Client) ListMonitors(ctx context.Context, params MonitorListParams, limit int) ([]json.RawMessage, error) {
	q := url.Values{}
	if params.URL != "" {
		q.Set("url", params.URL)
	}
	if params.PronounceableName != "" {
		q.Set("pronounceable_name", params.PronounceableName)
	}

	results, err := c.fetchAll(ctx, c.baseURL+monitorsPath, q, limit)
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
	return c.fetchOne(ctx, fmt.Sprintf("%s%s/%s", c.baseURL, monitorsPath, id))
}

func (c *Client) ListIntegrations(ctx context.Context, integrationType string, limit int) ([]json.RawMessage, error) {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return nil, err
	}
	return c.fetchAll(ctx, c.baseURL+path, nil, limit)
}

func (c *Client) GetIntegration(ctx context.Context, integrationType, id string) (json.RawMessage, error) {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return nil, err
	}
	return c.fetchOne(ctx, fmt.Sprintf("%s%s/%s", c.baseURL, path, id))
}

func (c *Client) CreateIntegration(ctx context.Context, integrationType, name string) (json.RawMessage, error) {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]string{
		"name": name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.postOne(ctx, c.baseURL+path, bytes.NewReader(payload))
}

func (c *Client) DeleteIntegration(ctx context.Context, integrationType, id string) error {
	path, err := resolveIntegrationPath(integrationType)
	if err != nil {
		return err
	}
	return c.deleteOne(ctx, fmt.Sprintf("%s%s/%s", c.baseURL, path, id))
}

func (c *Client) fetchAll(ctx context.Context, endpoint string, params url.Values, limit int) ([]json.RawMessage, error) {
	all := make([]json.RawMessage, 0)
	nextURL := endpoint
	if len(params) > 0 {
		nextURL += "?" + params.Encode()
	}

	for nextURL != "" {
		body, err := c.doRequestWithRetry(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, err
		}

		var page PaginatedResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		all = append(all, page.Data...)

		if limit > 0 && len(all) >= limit {
			all = all[:limit]
			break
		}

		if page.Pagination.Next != nil {
			nextURL = *page.Pagination.Next
		} else {
			nextURL = ""
		}
	}

	return all, nil
}

func (c *Client) fetchOne(ctx context.Context, endpoint string) (json.RawMessage, error) {
	body, err := c.doRequestWithRetry(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return wrapper.Data, nil
}

func (c *Client) postOne(ctx context.Context, endpoint string, reqBody io.Reader) (json.RawMessage, error) {
	body, err := c.doRequestWithRetry(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return wrapper.Data, nil
}

func (c *Client) deleteOne(ctx context.Context, endpoint string) error {
	_, err := c.doRequestWithRetry(ctx, http.MethodDelete, endpoint, nil)
	return err
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

type retryableError struct {
	apiError       *APIError
	retryAfter     time.Duration
	hasRetryHeader bool
}

func (e *retryableError) Error() string {
	return e.apiError.Error()
}

func (c *Client) doRequestWithRetry(ctx context.Context, method, rawURL string, body io.Reader) ([]byte, error) {
	var lastErr error
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	for attempt := range maxRetries {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		respBody, err := c.doRequest(ctx, method, rawURL, reqBody)
		if err == nil {
			return respBody, nil
		}

		re, ok := err.(*retryableError)
		if !ok {
			return nil, err
		}
		lastErr = re.apiError

		if attempt == maxRetries-1 {
			break
		}

		delay := re.retryAfter
		if !re.hasRetryHeader {
			delay = time.Duration(attempt+1) * time.Second
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, lastErr
}

func (c *Client) doRequest(ctx context.Context, method, rawURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "betterstack-cli/"+c.version)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryHeader := resp.Header.Get("Retry-After")
		retryAfter := parseRetryAfter(retryHeader)
		return nil, &retryableError{
			apiError:       &APIError{StatusCode: resp.StatusCode, Message: string(respBody)},
			retryAfter:     retryAfter,
			hasRetryHeader: retryHeader != "",
		}
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	return respBody, nil
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
