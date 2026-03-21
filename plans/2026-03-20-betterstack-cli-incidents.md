# Feature: BetterStack CLI for Incident Analysis

**Created**: 2026-03-20
**Goal**: Give Claude Code (and other AI agents) access to BetterStack incident and monitor data via a Go CLI, enabling analysis of alert threshold noise.

## User Requirements

<!-- DONE -->
Scenario: List recent incidents
  Given the user has a valid BetterStack API token configured
  When they run the CLI to list incidents
  Then they see a table of incidents showing name, started at, length, and status

Scenario: Filter incidents by date range
  Given the user has a valid BetterStack API token configured
  When they run the CLI to list incidents with a start and end date
  Then they see only incidents within that date range

Scenario: Filter incidents by monitor
  Given the user has a valid BetterStack API token configured
  When they run the CLI to list incidents for a specific monitor
  Then they see only incidents belonging to that monitor

Scenario: Filter incidents by resolution status
  Given the user has a valid BetterStack API token configured
  When they run the CLI to list only resolved incidents
  Then they see only incidents that have been resolved

Scenario: Get a single incident's details
  Given the user has a valid BetterStack API token configured
  When they run the CLI to get a specific incident by ID
  Then they see the full details of that incident

Scenario: List monitors
  Given the user has a valid BetterStack API token configured
  When they run the CLI to list monitors
  Then they see a table of monitors showing name, URL, status, and check frequency

Scenario: Filter monitors by status
  Given the user has a valid BetterStack API token configured
  When they run the CLI to list monitors with a specific status filter
  Then they see only monitors matching that status

Scenario: Get a single monitor's details
  Given the user has a valid BetterStack API token configured
  When they run the CLI to get a specific monitor by ID
  Then they see the full details of that monitor

Scenario: Get machine-readable output
  Given the user has a valid BetterStack API token configured
  When they run any list or get command with the JSON output flag
  Then the output is valid JSON written to stdout

Scenario: Limit number of results
  Given the user has a valid BetterStack API token configured
  When they run a list command with a limit flag
  Then at most that many results are returned

Scenario: Authenticate via environment variable
  Given the user has set the BETTERSTACK_API_TOKEN environment variable
  When they run any CLI command
  Then the CLI authenticates using that token

Scenario: Authenticate via config file
  Given the user has a config file with their API token
  And no environment variable is set
  When they run any CLI command
  Then the CLI authenticates using the config file token

Scenario: See a helpful error when no token is configured
  Given the user has no API token configured anywhere
  When they run any CLI command
  Then they see an error message explaining how to configure authentication
  And the CLI exits with a non-zero status code

Scenario: See a clear error for API failures
  Given the user has a valid API token configured
  When the BetterStack API returns an error
  Then the error message and HTTP status are printed to stderr
  And the CLI exits with a non-zero status code

Scenario: Handle rate limiting gracefully
  Given the user has a valid API token configured
  When the BetterStack API returns a rate limit response
  Then the CLI waits and retries automatically
  And the user eventually receives their results

Scenario: Handle empty result sets
  Given the user has a valid API token configured
  When no incidents or monitors match the query
  Then the CLI prints an empty table or empty JSON array
  And the CLI exits with a zero status code

Scenario: Discover available commands via help
  Given the user has installed the CLI
  When they run the CLI with a help flag or no arguments
  Then they see a list of available commands with descriptions
  And each subcommand also supports a help flag showing its flags and usage

Scenario: Check the CLI version
  Given the user has installed the CLI
  When they run the version command
  Then they see the CLI version number

Scenario: Install the CLI
  Given the user has Go installed
  When they run go install for the CLI module
  Then the betterstack binary is available in their PATH

## Technical Specifications

<!-- DONE -->
Scenario: Go module structure
  Given a new Go module at github.com/tommymorgan/betterstack-cli
  When the project is initialized
  Then it has cmd/ and internal/ directories
  And internal/ contains client/, output/, and config/ packages
  And it uses cobra for CLI subcommand parsing
  And it uses gopkg.in/yaml.v3 for config file parsing
  And it targets Go 1.26
  And it is installable via go install

Scenario: HTTP client sends correct auth headers
  Given a configured API token
  When the client makes a request to the BetterStack API
  Then it sends an Authorization header with "Bearer <token>"
  And it sends an Accept header with "application/json"
  And it sends a User-Agent header identifying the CLI name and version

Scenario: HTTP client auto-paginates
  Given the BetterStack API returns paginated results with next page links
  When the client fetches a list endpoint
  Then it follows pagination links until all pages are fetched
  And it combines results from all pages into a single slice

Scenario: HTTP client respects limit flag
  Given the user specified a limit of N results
  When the client fetches a list endpoint
  Then it stops fetching pages once N results are collected
  And it returns at most N results

Scenario: HTTP client retries on 429
  Given the BetterStack API returns HTTP 429 with a Retry-After header
  When the client receives the response
  Then it waits for the duration specified in Retry-After
  And it retries the request up to 3 times
  And it returns an error if all retries are exhausted

Scenario: Incidents list endpoint
  Given a valid API token
  When the client calls ListIncidents
  Then it sends GET to https://uptime.betterstack.com/api/v3/incidents
  And it supports query parameters: from, to, monitor_id, resolved, acknowledged

Scenario: Incidents get endpoint
  Given a valid API token and an incident ID
  When the client calls GetIncident
  Then it sends GET to https://uptime.betterstack.com/api/v3/incidents/{id}

Scenario: Monitors list endpoint
  Given a valid API token
  When the client calls ListMonitors
  Then it sends GET to https://uptime.betterstack.com/api/v2/monitors
  And it supports query parameters: url, pronounceable_name

Scenario: Monitors get endpoint
  Given a valid API token and a monitor ID
  When the client calls GetMonitor
  Then it sends GET to https://uptime.betterstack.com/api/v2/monitors/{id}

Scenario: Table output formatting
  Given a list of incidents
  When rendered as a table
  Then columns are: ID, Name, Cause, Started At, Resolved At, Length, Status
  And timestamps are displayed in UTC
  And empty fields show a dash character
  And columns are aligned with tabwriter

Scenario: JSON output formatting
  Given a list of incidents
  When rendered as JSON
  Then the output is the full API response data as indented JSON
  And it is written to stdout

Scenario: Token resolution order
  Given both an environment variable and config file contain tokens
  When the config package resolves the token
  Then the environment variable token takes precedence

Scenario: Config file location
  Given the user has created a config file
  When the config package looks for it
  Then it checks ~/.config/betterstack/config.yaml
  And the file contains an api_token field

Scenario: Config file has restricted permissions
  Given the CLI creates or reads a config file
  When the config file is written
  Then its file permissions are set to 0600
  And the CLI warns to stderr if an existing config file has permissions more open than 0600

Scenario: Errors go to stderr
  Given any error occurs during execution
  When the CLI reports the error
  Then the error message is written to stderr
  And the exit code is 1

Scenario: HTTP client enforces request timeouts
  Given the client makes a request to the BetterStack API
  When the request takes longer than 30 seconds
  Then the client cancels the request
  And returns a timeout error

Scenario: Unknown JSON fields are ignored
  Given the BetterStack API returns fields not present in the Go structs
  When the client deserializes the response
  Then unknown fields are silently ignored
  And known fields are correctly populated

## Affected Documentation

No existing documentation is affected by these changes (greenfield project).

## Notes

- **API versions differ**: Incidents use `/api/v3/`, monitors use `/api/v2/` — this is how BetterStack's API is structured today.
- **Read-only first**: This initial version is read-only. Write operations (acknowledge, resolve, create incidents) can be added later.
- **Design for expansion**: The subcommand structure (`betterstack <resource> <verb>`) scales to heartbeats, status pages, escalation policies, and telemetry endpoints in future iterations.
- **The primary consumer is Claude Code**: Output must be clean, parseable, and errors must go to stderr so Claude can distinguish success from failure via exit codes.
- **Auth token is for the Uptime API**: The user's existing team-scoped Uptime API token covers incidents and monitors. Telemetry (logs) uses a separate token and will be handled in a future iteration.
