# betterstack-cli

A command-line tool for BetterStack designed for use by AI agents (Claude Code, etc.) and humans alike. It covers the uptime host (incidents, monitors, integrations, escalation policies) and the telemetry host (logs sources, explorations, logs alerts).

## Install

```bash
go install github.com/tommymorgan/betterstack-cli@latest
```

## Authentication

BetterStack has two hosts:

- **Uptime** at `uptime.betterstack.com` — incidents, monitors, integrations, escalation policies.
- **Telemetry** at `telemetry.betterstack.com` — log sources, explorations, logs alerts.

A *global* token works for both hosts. A *team-scoped* token is host-specific.

| Env var                        | Used for                                       |
|--------------------------------|------------------------------------------------|
| `BETTERSTACK_API_TOKEN`        | Uptime host. Also the fallback for telemetry.  |
| `BETTERSTACK_TELEMETRY_TOKEN`  | Telemetry host.                                |

Or set them in `~/.config/betterstack/config.yaml` (0600 permissions required):

```yaml
api_token: <uptime-token>
telemetry_api_token: <telemetry-token>   # optional; falls back to api_token
```

Telemetry commands prefer `BETTERSTACK_TELEMETRY_TOKEN` → `telemetry_api_token` → `BETTERSTACK_API_TOKEN` → `api_token`. When the fallback kicks in on an interactive terminal, the CLI emits one notice per process reminding the user that the fallback only works for global tokens. Suppress the notice with `--quiet` or `BETTERSTACK_QUIET=1`.

Uptime commands *only* read the uptime token — they never fall back to the telemetry token.

## Usage

### Incidents and monitors (uptime)

```bash
betterstack-cli incidents list --from 2026-03-01 --to 2026-03-31
betterstack-cli incidents get <id>

betterstack-cli monitors list --status down
betterstack-cli monitors get <id>
```

### Integrations (uptime)

```bash
betterstack-cli integrations list --type aws-cloudwatch
betterstack-cli integrations get --type aws-cloudwatch <id>
betterstack-cli integrations create --type aws-cloudwatch --name "dev-cloudwatch"
betterstack-cli integrations delete --type aws-cloudwatch <id> --yes
```

### Escalation policies (uptime)

```bash
betterstack-cli policies list
betterstack-cli policies get <id>

# Create/update require --body-file (policies have nested step shapes that
# shorthand flags can't express).
betterstack-cli policies create --body-file policy.json
betterstack-cli policies update <id> --body-file update.json

# Delete refuses if any logs-alerts reference the policy; --force bypasses.
betterstack-cli policies delete <id> --yes
betterstack-cli policies delete <id> --yes --force
```

### Log sources (telemetry)

```bash
betterstack-cli logs-sources list
betterstack-cli logs-sources get <id>
```

### Explorations (telemetry)

An exploration is a saved query. The shorthand flags create a count-matching query over a source.

```bash
# Shorthand: count occurrences of a literal pattern
betterstack-cli explorations create \
  --source-id 123456 \
  --pattern "AADSTS7000215" \
  --name "INF-3017"

# Full JSON payload (escape hatch for custom shapes)
betterstack-cli explorations create --body-file exploration.json
cat exploration.json | betterstack-cli explorations create --body-file -

# Upsert (create or patch the sole name match)
betterstack-cli explorations create \
  --source-id 1 --pattern "foo" --name "INF-3017" --upsert

betterstack-cli explorations list
betterstack-cli explorations get <id>
betterstack-cli explorations update <id> --name "renamed"
betterstack-cli explorations update <id> --body-file update.json

# Delete refuses if dependent alerts exist; --force bypasses.
betterstack-cli explorations delete <id> --yes
betterstack-cli explorations delete <id> --yes --force
```

### Logs alerts (telemetry)

A logs alert attaches to an exploration and fires when the exploration's count crosses a threshold over a window.

```bash
# Shorthand: threshold >= 3 over 5 minutes, routed through policy 77
betterstack-cli logs-alerts create \
  --exploration-id 5 \
  --threshold 3 \
  --window 5m \
  --policy-id 77 \
  --name "INF-3017"

# --body-file as the escape hatch (anomaly alerts, team/schedule/user
# escalation targets, custom metadata):
betterstack-cli logs-alerts create --exploration-id 5 --body-file alert.json

# Upsert matches by name scoped to the target exploration
betterstack-cli logs-alerts create \
  --exploration-id 5 --name "INF-3017" --threshold 3 --window 5m \
  --policy-id 77 --upsert

betterstack-cli logs-alerts list
betterstack-cli logs-alerts list --exploration-id 5   # client-side filter
betterstack-cli logs-alerts list --policy-id 77       # client-side filter
betterstack-cli logs-alerts get <id>
betterstack-cli logs-alerts update <id> --threshold 5 --window 10m
betterstack-cli logs-alerts delete <id> --yes
```

## Durations

All duration flags (`--window`, `--check-period`, `--query-period`, `--recovery`) use Go-style duration strings with required units. The max unit is hour — use `24h`, not `1d`. Values must be strictly positive.

Good: `5m`, `30s`, `2h`, `1h30m`, `24h`.

Rejected: `300` (no unit), `0s` (not positive), `-5m` (negative), `1d` (unsupported unit).

## JSON output envelope

Pass `--json` for machine-readable output. The envelope is consistent across verbs:

```
list:   {"items": [ <resource>, <resource>, ... ]}
get:    <resource>
create/update: <resource>
upsert: {"action": "created" | "updated" | "unchanged", "resource": <resource>}
delete: {"id": "<id>", "deleted": true}
error:  {"error": {...}}   # to stderr
```

`--json` never auto-enables based on TTY detection — piping through `less` stays human-readable.

## Exit codes

| Code | Meaning                                                                 |
|------|-------------------------------------------------------------------------|
| 0    | Success                                                                 |
| 1    | Generic / unclassified error                                            |
| 2    | Authentication failure (missing, invalid, or wrong-scope token)         |
| 3    | User input error (bad flag combination, malformed body, missing `--yes`)|
| 4    | Dependency conflict (refcount refusal, `dependency_appeared_after_precheck`) |
| 5    | Upstream API error (5xx, unexpected 4xx, `upsert_match_deleted_after_precheck`) |

## `--yes` and `--force`

- `--yes` is required for every destructive operation. It confirms intent.
- `--force` is only accepted by `explorations delete` and `policies delete`. It bypasses the refcount precheck that refuses deletes when dependents exist. `logs-alerts delete` explicitly rejects `--force`.

## Race-condition error prefixes

Refcount prechecks and upsert matching are both best-effort. Concurrent activity surfaces deterministic, machine-matchable error prefixes:

- `error: dependency_appeared_after_precheck:` — a dependent was created between the precheck and the DELETE. Exit code 4.
- `error: upsert_match_deleted_after_precheck:` — the matched resource was deleted between the list and the PATCH. Exit code 5.

## Design

- **Errors go to stderr**, data goes to stdout.
- **Auto-paginates** all list endpoints by default.
- **Retries automatically** on rate limiting (HTTP 429) honoring `Retry-After`.
- **30-second request timeout** per API call.
- **List filters are client-side** (invariant: future internal optimizations must preserve this contract).
- **Shorthand OR body-file, never both** — no merging, no precedence rules.
- **Upsert is non-atomic** — concurrent creation between list and POST can produce duplicates.
