# Sources Command Design

Add CRUD operations for BetterStack sources (webhook integrations) so users can create, list, inspect, and delete sources from the CLI without switching to the web UI.

Addresses [tommymorgan/betterstack-cli#1](https://github.com/tommymorgan/betterstack-cli/issues/1).

## Motivation

Setting up monitoring integrations (CloudWatch, Datadog, etc.) requires creating a source in BetterStack to get a webhook URL. Doing this through the UI breaks automation and IaC workflows. The CLI should support the full lifecycle so the webhook URL can be piped into Terraform, CloudFormation, or parameter stores.

## API

BetterStack Uptime API endpoint: `/api/v2/sources`

| Operation | Method | Path | Request Body | Response |
|-----------|--------|------|-------------|----------|
| List | GET | `/api/v2/sources` | - | Paginated list |
| Get | GET | `/api/v2/sources/{id}` | - | Single resource |
| Create | POST | `/api/v2/sources` | `{"name": "...", "type": "..."}` | Created resource |
| Delete | DELETE | `/api/v2/sources/{id}` | - | 204 No Content |

## Design

### Client Layer (`internal/client/client.go`)

**Generalize `doRequest`** to support all HTTP methods. The current signature:

```go
func (c *Client) doRequest(ctx context.Context, rawURL string) ([]byte, error)
```

Becomes:

```go
func (c *Client) doRequest(ctx context.Context, method, rawURL string, body io.Reader) ([]byte, error)
```

All existing callers (`fetchAll`, `fetchOne`) pass `http.MethodGet` and `nil` body. `doRequestWithRetry` passes through the new parameters. The retry logic is unchanged. When `body` is non-nil, `doRequest` sets `Content-Type: application/json`.

**New constant:**

```go
sourcesPath = "/api/v2/sources"
```

**New client methods:**

- `ListSources(ctx, limit) ([]json.RawMessage, error)` -- delegates to `fetchAll`
- `GetSource(ctx, id) (json.RawMessage, error)` -- delegates to `fetchOne`
- `CreateSource(ctx, name, sourceType) (json.RawMessage, error)` -- POST with JSON body, returns created resource via new `postOne` helper
- `DeleteSource(ctx, id) error` -- DELETE, expects 204, via new `deleteOne` helper

**New helpers:**

- `postOne(ctx, endpoint, body) (json.RawMessage, error)` -- POST request, unwraps `{"data": ...}` response
- `deleteOne(ctx, endpoint) error` -- DELETE request, succeeds on 204

### Output Layer (`internal/output/output.go`)

**New struct:**

```go
type Source struct {
    ID         string `json:"id"`
    Attributes struct {
        Name       string `json:"name"`
        Type       string `json:"type"`
        WebhookURL string `json:"webhook_url"`
        CreatedAt  string `json:"created_at"`
    } `json:"attributes"`
}
```

**New renderers:**

- `RenderSourcesTable(w, data)` -- columns: ID, NAME, TYPE, WEBHOOK URL
- `RenderSourceDetail(w, data)` -- key-value: ID, Name, Type, Webhook URL, Created At

### Command Layer (`cmd/betterstack/sources.go`)

```
betterstack sources list [--limit N] [--json]
betterstack sources get <id> [--json]
betterstack sources create --name <name> --type <type> [--json]
betterstack sources delete <id> [--json]
```

- `list`: paginated, optional `--limit` flag
- `get`: requires exactly one positional arg (source ID), renders detail view
- `create`: `--name` and `--type` are required flags (enforced via `MarkFlagRequired`). Renders detail view on success so the webhook URL is immediately visible.
- `delete`: requires exactly one positional arg (source ID). Prints `Deleted source <id>`. With `--json`, outputs `{"deleted": true, "id": "<id>"}`.

All subcommands follow the same pattern as monitors/incidents: resolve token, create client, call method, render output.

### Testing

**Extend `setupTestServer`** in `commands_test.go` with routes for:
- `GET /api/v2/sources` -- returns paginated source list
- `GET /api/v2/sources/{id}` -- returns single source
- `POST /api/v2/sources` -- validates JSON body, returns created source with webhook URL
- `DELETE /api/v2/sources/{id}` -- returns 204

**Command tests:**
- `sources list` shows table with name, type, webhook URL
- `sources list --json` returns valid JSON array
- `sources get <id>` shows detail view
- `sources create --name X --type Y` shows created source with webhook URL
- `sources create` without required flags returns error
- `sources delete <id>` prints confirmation
- `sources delete <id> --json` returns JSON confirmation

**Client tests:**
- `ListSources` sends GET to correct endpoint
- `GetSource` sends GET to correct endpoint with ID
- `CreateSource` sends POST with correct JSON body
- `DeleteSource` sends DELETE and succeeds on 204
- `CreateSource` and `DeleteSource` use the same retry/error handling as existing methods

## Out of Scope

- `sources update` (PATCH) -- not requested in the issue, can be added later
- Stdin/file-based JSON input for create -- flags-only is sufficient for the two required fields
- Source type validation/autocomplete -- the API validates types server-side
