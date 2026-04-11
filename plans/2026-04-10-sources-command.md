# Feature: Sources Command

**Created**: 2026-04-10
**Goal**: Let users create, list, inspect, and delete BetterStack sources (webhook integrations) from the CLI, so webhook URLs can be consumed by automation without switching to the web UI.

## User Requirements

<!-- DONE -->
Scenario: List all sources
  Given the user has a valid BetterStack API token configured
  When they run the CLI to list sources
  Then they see a list of sources showing ID, name, type, and webhook URL

<!-- DONE -->
Scenario: Get a source's details including webhook URL
  Given the user has a valid BetterStack API token configured
  When they run the CLI to get a specific source by ID
  Then they see the source's ID, name, type, webhook URL, and creation date

<!-- DONE -->
Scenario: Create a source and receive its webhook URL
  Given the user has a valid BetterStack API token configured
  When they run the CLI to create a source with a name and type
  Then they see the created source's ID, name, type, and webhook URL

<!-- DONE -->
Scenario: Delete a source
  Given the user has a valid BetterStack API token configured
  When they run the CLI to delete a specific source by ID
  Then the CLI requires a --yes flag to confirm the destructive operation
  And they see a confirmation message showing the deleted source ID

<!-- DONE -->
Scenario: See a clear error when creating a source without required fields
  Given the user has a valid BetterStack API token configured
  When they run the create command without providing a name or type
  Then they see an error message indicating the missing required fields
  And the CLI exits with a non-zero status code

<!-- DONE -->
Scenario: See a clear error when deleting without confirmation flag
  Given the user has a valid BetterStack API token configured
  When they run the CLI to delete a source by ID without the --yes flag
  Then they see an error message explaining that --yes is required for destructive operations
  And the CLI exits with a non-zero status code
  And the source is not deleted

## Technical Specifications

<!-- DONE -->
Scenario: HTTP client supports POST and DELETE methods
  Given the existing doRequest method only supports GET
  When the method signature is generalized to accept an HTTP method and optional body
  Then existing GET callers continue to work unchanged
  And POST requests send Content-Type application/json
  And DELETE requests send no body

<!-- DONE -->
Scenario: Sources list endpoint
  Given a valid API token
  When the client calls ListSources
  Then it sends GET to /api/v2/sources on the configured base URL
  And it auto-paginates using the existing fetchAll helper

<!-- DONE -->
Scenario: Sources get endpoint
  Given a valid API token and a source ID
  When the client calls GetSource
  Then it sends GET to /api/v2/sources/{id} on the configured base URL
  And it returns the source data including ID, name, type, and webhook URL

<!-- DONE -->
Scenario: Sources create endpoint
  Given a valid API token, a name, and a source type
  When the client calls CreateSource
  Then it sends POST to /api/v2/sources on the configured base URL
  And the request body is JSON with name and type fields
  And the API returns 201 Created on success
  And it returns the created source data including the webhook URL

<!-- DONE -->
Scenario: Sources delete endpoint
  Given a valid API token and a source ID
  When the client calls DeleteSource
  Then it sends DELETE to /api/v2/sources/{id} on the configured base URL
  And it succeeds when the API returns 204 No Content

<!-- DONE -->
Scenario: Sources table output
  Given a list of sources
  When rendered as a table
  Then columns are: ID, NAME, TYPE, WEBHOOK URL
  And columns are aligned with tabwriter

<!-- DONE -->
Scenario: Sources detail output
  Given a single source
  When rendered as a detail view
  Then it shows key-value pairs: ID, Name, Type, Webhook URL, Created At
  And timestamps are displayed in UTC

<!-- DONE -->
Scenario: Delete confirmation output
  Given a source has been deleted
  When rendered as text
  Then it prints "Deleted source <id>"
  And with JSON flag it outputs {"deleted": true, "id": "<id>"}

## Affected Documentation

- [ ] Update README.md — add Sources section with usage examples for list, get, create, and delete

## Notes

- **First write operations**: This is the first feature that adds POST and DELETE to the CLI. The `doRequest` generalization is designed to be reusable for future write operations on other resources.
- **Sources use /api/v2/**: Consistent with the monitors endpoint, not /api/v3/ like incidents.
- **Webhook URL is the key output**: The primary automation use case is creating a source and piping the webhook URL into infrastructure tools. The detail view (used for both get and create) makes the URL prominent.
- **--yes flag for delete**: Destructive operations require explicit confirmation since deleting a source immediately breaks any automation pointing at its webhook URL.
