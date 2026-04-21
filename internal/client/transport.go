package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// transport encapsulates the request pipeline shared by every BetterStack
// host client: Bearer auth, 30s timeout, Retry-After-aware 429 retries, and
// uniform APIError emission on non-2xx responses.
type transport struct {
	token      string
	version    string
	baseURL    string
	httpClient *http.Client
}

func newTransport(token, version, baseURL string) *transport {
	return &transport{
		token:   token,
		version: version,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// PaginatedResponse is the envelope returned by list endpoints.
type PaginatedResponse struct {
	Data       []json.RawMessage `json:"data"`
	Pagination struct {
		Next *string `json:"next"`
	} `json:"pagination"`
}

type singleResponse struct {
	Data json.RawMessage `json:"data"`
}

func (t *transport) fetchAll(ctx context.Context, endpoint string, params url.Values, limit int) ([]json.RawMessage, error) {
	all := make([]json.RawMessage, 0)
	nextURL := endpoint
	if len(params) > 0 {
		nextURL += "?" + params.Encode()
	}

	for nextURL != "" {
		body, err := t.doRequestWithRetry(ctx, http.MethodGet, nextURL, nil)
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

// fetchPage reads a single page worth of results. Used when a caller wants to
// peek at one page (e.g., the `per_page=1` refcount precheck) rather than
// auto-paginating to completion.
func (t *transport) fetchPage(ctx context.Context, endpoint string, params url.Values) ([]json.RawMessage, error) {
	fullURL := endpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}
	body, err := t.doRequestWithRetry(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	var page PaginatedResponse
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return page.Data, nil
}

func (t *transport) fetchOne(ctx context.Context, endpoint string) (json.RawMessage, error) {
	body, err := t.doRequestWithRetry(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	return unwrapData(body)
}

func (t *transport) postOne(ctx context.Context, endpoint string, reqBody io.Reader) (json.RawMessage, error) {
	body, err := t.doRequestWithRetry(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	return unwrapData(body)
}

func (t *transport) patchOne(ctx context.Context, endpoint string, reqBody io.Reader) (json.RawMessage, error) {
	body, err := t.doRequestWithRetry(ctx, http.MethodPatch, endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	return unwrapData(body)
}

func (t *transport) deleteOne(ctx context.Context, endpoint string) error {
	_, err := t.doRequestWithRetry(ctx, http.MethodDelete, endpoint, nil)
	return err
}

func unwrapData(body []byte) (json.RawMessage, error) {
	if len(body) == 0 {
		return nil, nil
	}
	var wrapper singleResponse
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return wrapper.Data, nil
}

type retryableError struct {
	apiError       *APIError
	retryAfter     time.Duration
	hasRetryHeader bool
}

func (e *retryableError) Error() string {
	return e.apiError.Error()
}

func (t *transport) doRequestWithRetry(ctx context.Context, method, rawURL string, body io.Reader) ([]byte, error) {
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

		respBody, err := t.doRequest(ctx, method, rawURL, reqBody)
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

func (t *transport) doRequest(ctx context.Context, method, rawURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "betterstack-cli/"+t.version)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.httpClient.Do(req)
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
