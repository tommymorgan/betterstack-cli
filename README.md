# betterstack-cli

A command-line tool for querying BetterStack incidents and monitors, designed for use by AI agents (Claude Code, etc.) and humans alike.

## Install

```bash
go install github.com/tommymorgan/betterstack-cli@latest
```

## Authentication

Set your BetterStack Uptime API token via environment variable:

```bash
export BETTERSTACK_API_TOKEN=<your-token>
```

Or create a config file at `~/.config/betterstack/config.yaml`:

```yaml
api_token: <your-token>
```

The environment variable takes precedence if both are set.

## Usage

### Incidents

```bash
# List all incidents
betterstack-cli incidents list

# Filter by date range
betterstack-cli incidents list --from 2026-03-01 --to 2026-03-31

# Filter by monitor
betterstack-cli incidents list --monitor-id 123

# Filter by resolution status
betterstack-cli incidents list --resolved

# Limit results
betterstack-cli incidents list --limit 10

# Get a single incident
betterstack-cli incidents get <id>
```

### Monitors

```bash
# List all monitors
betterstack-cli monitors list

# Filter by status
betterstack-cli monitors list --status down

# Get a single monitor
betterstack-cli monitors get <id>
```

### JSON output

Add `--json` to any command for machine-readable output:

```bash
betterstack-cli incidents list --from 2026-03-25 --json
```

## Design

- **Errors go to stderr**, data goes to stdout, so AI agents can distinguish success from failure via exit codes
- **Auto-paginates** all list endpoints by default
- **Retries automatically** on rate limiting (HTTP 429)
- **30-second request timeout** per API call
