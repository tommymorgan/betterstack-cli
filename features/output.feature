Feature: Output Formatting
  As a user or AI agent consuming the BetterStack CLI
  I want both human-readable and machine-readable output
  So that I can view or programmatically process results

  @user
  Scenario: Get machine-readable output
    Given the user has a valid BetterStack API token configured
    When they run any list or get command with the JSON output flag
    Then the output is valid JSON written to stdout

  @user
  Scenario: Limit number of results
    Given the user has a valid BetterStack API token configured
    When they run a list command with a limit flag
    Then at most that many results are returned

  @user
  Scenario: Handle empty result sets
    Given the user has a valid API token configured
    When no incidents or monitors match the query
    Then the CLI prints an empty table or empty JSON array
    And the CLI exits with a zero status code

  @technical
  Scenario: Table output formatting
    Given a list of incidents
    When rendered as a table
    Then columns are: ID, Name, Cause, Started At, Resolved At, Length, Status
    And timestamps are displayed in UTC
    And empty fields show a dash character
    And columns are aligned with tabwriter

  @technical
  Scenario: JSON output formatting
    Given a list of incidents
    When rendered as JSON
    Then the output is the full API response data as indented JSON
    And it is written to stdout

  @user
  Scenario: Durations render human-readable in table output
    Given any list or get command with a period field (e.g., check_period, query_period)
    When rendered as a table or curated detail view
    Then durations appear as human-readable strings (e.g., "5m", "30s", "2h")

  @user
  Scenario: Durations render as integer seconds in JSON output
    Given any list or get command with a period field
    When rendered as JSON with `--json`
    Then period fields are integer seconds (matching the API)

  @user
  Scenario: JSON output envelope is consistent across verbs
    Given any command with --json
    When the CLI renders the response
    Then list commands output {"items": [ <resource objects> ]}
    And get commands output <resource object> directly
    And create/update (non-upsert) commands output <resource object> directly
    And upsert commands output {"action": "created"|"updated"|"unchanged", "resource": <resource object>}
    And delete commands output {"id": "<id>", "deleted": true}
    And error responses output {"error": {...}} to stderr (not stdout)

  @user
  Scenario: --json is a global flag
    Given any command
    When --json is passed at any position
    Then the CLI emits the documented JSON envelope for that verb
    And without --json, the CLI emits the curated human-readable output (TTY behavior does not auto-flip to JSON)

  @user
  Scenario: Consistent error template for dependency conflicts
    Given any delete command refuses due to dependent resources
    When the CLI emits the error
    Then in --json mode stderr contains {"error": {"code": "DEPENDENCIES_EXIST", "count": "<at-least-N>", "inspect_command": "<literal command with ID>", "resource_type": "<type>", "resource_id": "<id>"}}
    And in default mode stderr contains a parameterized human template referencing the same fields
    And the exit code is 4 (dependency conflict)

  @user
  Scenario: Exit code taxonomy
    Given any CLI invocation
    When the CLI terminates
    Then exit code 0 means success
    And exit code 1 means a generic/unclassified error
    And exit code 2 means authentication failure (missing, invalid, or wrong-scope token)
    And exit code 3 means user input error (bad flag combination, malformed body, missing --yes, etc.)
    And exit code 4 means dependency conflict (refcount refusal, dependency_appeared_after_precheck)
    And exit code 5 means upstream API error (5xx, unexpected 4xx, upsert_match_deleted_after_precheck)

  @user
  Scenario: List commands fetch all pages by default
    Given any list command
    When the BetterStack API returns paginated results
    Then the CLI fetches all pages and returns a combined result
    And a --limit flag caps the returned count across pages (matching the existing incidents/monitors list behavior)

  @technical
  Scenario: Curated detail views for get commands
    Given any `<resource> get <id>` command (human mode)
    When rendered as a detail view
    Then it shows a stable curated set of fields documented in --help (not every field the API returned)
    And --json mode passes through the raw API response for scripting use

  @technical
  Scenario: --yes and --force are orthogonal
    Given a delete command
    When flags are validated
    Then --yes alone skips only the destructive-action confirmation
    And --force alone (without --yes) still errors because destructive-action confirmation is required
    And --force is only accepted by `explorations delete` and `policies delete` (which have dependency checks to skip)
    And `logs-alerts delete` explicitly rejects --force (exit code 3)
