# Feature: Integrations Command

**Created**: 2026-04-10
**Goal**: Let users manage BetterStack integrations (starting with AWS CloudWatch) from the CLI, using the correct per-type API endpoints so webhook URLs can be consumed by automation.

## User Requirements

<!-- DONE -->
Scenario: List integrations by type
  Given the user has a valid BetterStack API token configured
  When they run `integrations list --type aws-cloudwatch`
  Then they see a table with columns ID, Name, Webhook URL, and Paused

<!-- DONE -->
Scenario: Get an integration's details
  Given the user has a valid BetterStack API token configured
  When they run `integrations get --type aws-cloudwatch <id>`
  Then they see the integration's ID, name, webhook URL, paused status, and team name

<!-- DONE -->
Scenario: Create an integration and receive its webhook URL
  Given the user has a valid BetterStack API token configured
  When they run `integrations create --type aws-cloudwatch --name "My Integration"`
  Then they see the created integration's ID, name, webhook URL, and paused status

<!-- DONE -->
Scenario: Delete an integration
  Given the user has a valid BetterStack API token configured
  When they run `integrations delete --type aws-cloudwatch <id> --yes`
  Then they see a confirmation message showing the deleted integration ID

<!-- DONE -->
Scenario: See a clear error when creating without a type
  Given the user has a valid BetterStack API token configured
  When they run the create command without providing --type
  Then they see an error message naming --type as required
  And the CLI exits with a non-zero status code

<!-- DONE -->
Scenario: See a clear error when creating without a name
  Given the user has a valid BetterStack API token configured
  When they run `integrations create --type aws-cloudwatch` without --name
  Then they see an error message naming --name as required
  And the CLI exits with a non-zero status code

<!-- DONE -->
Scenario: See a clear error when deleting without confirmation flag
  Given the user has a valid BetterStack API token configured
  When they run the CLI to delete an integration by type and ID without the --yes flag
  Then they see an error message explaining that --yes is required for destructive operations
  And the CLI exits with a non-zero status code
  And the integration is not deleted

<!-- DONE -->
Scenario: See a clear error for unknown integration type
  Given the user has a valid BetterStack API token configured
  When they run any integrations subcommand with an unsupported type
  Then they see an error message naming the unsupported type they provided and listing supported types
  And the CLI exits with a non-zero status code

## Technical Specifications

<!-- DONE -->
Scenario: Integration type maps to correct API path
  Given a registry mapping integration types to API paths
  When the client receives type "aws-cloudwatch"
  Then it resolves to path /api/v2/aws-cloudwatch-integrations
  And the client prepends the configured base URL to form the full request URL
  And unknown types return a Go error value listing supported types (not an HTTP call)

<!-- DONE -->
Scenario: Integrations list endpoint
  Given a valid API token and type "aws-cloudwatch"
  When the client calls ListIntegrations
  Then it sends GET to /api/v2/aws-cloudwatch-integrations on the configured base URL
  And it returns all integrations across all pages (not just the first page)

<!-- DONE -->
Scenario: Integrations get endpoint
  Given a valid API token, type "aws-cloudwatch", and an integration ID
  When the client calls GetIntegration
  Then it sends GET to /api/v2/aws-cloudwatch-integrations/{id} on the configured base URL
  And it returns the integration data including ID, name, and webhook URL

<!-- DONE -->
Scenario: Integrations create endpoint
  Given a valid API token, type "aws-cloudwatch", and a name
  When the client calls CreateIntegration
  Then it sends POST to /api/v2/aws-cloudwatch-integrations on the configured base URL
  And the request body is JSON with a name field
  And the request sets Content-Type to application/json
  And the API returns 201 Created on success
  And on non-201 responses the client returns an error with the HTTP status code
  And it returns the created integration data including the webhook URL

<!-- DONE -->
Scenario: Integrations delete endpoint
  Given a valid API token, type "aws-cloudwatch", and an integration ID
  When the client calls DeleteIntegration
  Then it sends DELETE to /api/v2/aws-cloudwatch-integrations/{id} on the configured base URL
  And it succeeds when the API returns 204 No Content
  And it returns an error when the API responds with 404 (integration not found)

<!-- DONE -->
Scenario: Integrations table output
  Given a list of integrations
  When rendered as a table
  Then columns are: ID, NAME, WEBHOOK URL, PAUSED
  And column values are aligned consistently across all rows

<!-- DONE -->
Scenario: Integrations detail output
  Given a single integration
  When rendered as a detail view
  Then it shows key-value pairs in this order: ID, Name, Webhook URL, Paused, Team Name

<!-- DONE -->
Scenario: Delete confirmation output
  Given an integration has been deleted
  When rendered as text
  Then it prints "Deleted integration <id>"
  And with JSON flag it outputs {"deleted": true, "id": "<id>"}
  And both formats are written to stdout

## Affected Documentation

- [x] Update README.md — replace "Sources" section with "Integrations" section showing `--type aws-cloudwatch`
- [x] Rename features/sources.feature → features/integrations.feature with updated scenarios
- [ ] Remove or archive plans/2026-04-10-sources-command.md (superseded by this plan)
- [ ] Remove docs/superpowers/specs/2026-04-10-sources-command-design.md (superseded)
- [x] Update PKGBUILD if build or binary name changes — no changes needed, binary name is unchanged

## Notes

- **Per-type API paths**: BetterStack has no generic `/api/v2/sources` endpoint. Each integration type has its own path (e.g., `/api/v2/aws-cloudwatch-integrations`, `/api/v2/new-relic-integrations`). The `--type` flag maps to the correct path via a registry.
- **AWS CloudWatch only for now**: Only `aws-cloudwatch` is supported initially. Adding new types is a one-line map entry plus any type-specific create fields.
- **Rename from sources**: The `sources` command never worked against the real API. "Integrations" matches BetterStack's own terminology.
- **POST/DELETE support already exists**: The generalized `doRequest` from the sources work carries over. The `postOne` and `deleteOne` helpers are type-agnostic.
- **Source struct → Integration struct**: The output struct is renamed and fields updated to match the CloudWatch API response (adds `paused`, `team_name`; removes `type` and `created_at` since the API doesn't return those for CloudWatch).
