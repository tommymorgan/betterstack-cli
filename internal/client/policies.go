package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// Policies (escalation policies) live on the uptime host at /api/v3/policies.

func (c *Client) ListPolicies(ctx context.Context, limit int) ([]json.RawMessage, error) {
	return c.t.fetchAll(ctx, c.t.baseURL+policiesPath, nil, limit)
}

func (c *Client) GetPolicy(ctx context.Context, id string) (json.RawMessage, error) {
	return c.t.fetchOne(ctx, fmt.Sprintf("%s%s/%s", c.t.baseURL, policiesPath, pathID(id)))
}

func (c *Client) CreatePolicy(ctx context.Context, body []byte) (json.RawMessage, error) {
	return c.t.postOne(ctx, c.t.baseURL+policiesPath, bytes.NewReader(body))
}

func (c *Client) UpdatePolicy(ctx context.Context, id string, body []byte) (json.RawMessage, error) {
	return c.t.patchOne(ctx, fmt.Sprintf("%s%s/%s", c.t.baseURL, policiesPath, pathID(id)), bytes.NewReader(body))
}

func (c *Client) DeletePolicy(ctx context.Context, id string) error {
	return c.t.deleteOne(ctx, fmt.Sprintf("%s%s/%s", c.t.baseURL, policiesPath, pathID(id)))
}
