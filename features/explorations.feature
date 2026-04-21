Feature: Explorations
  As a user of the BetterStack CLI
  I want to create and manage saved log queries (explorations)
  So that I can reuse count-matching queries as the query half of a logs-alert rule

  @user
  Scenario: Create a count-matching exploration with shorthand flags
    Given the user has a valid telemetry token configured
    When they run `explorations create --source-id <id> --pattern "AADSTS7000215" --name "INF-3017"`
    Then the CLI creates an exploration scoped to that source that counts occurrences of the pattern
    And they see the created exploration's ID, name, chart type, date range, and query summary

  @user
  Scenario: Create an exploration from a raw JSON body file
    Given the user has a valid telemetry token configured
    And they have a JSON file describing a full exploration payload
    When they run `explorations create --body-file path/to/exploration.json`
    Then the CLI sends the file's contents verbatim as the request body
    And they see the created exploration's details

  @user
  Scenario: --body-file - reads from stdin
    Given the user has a valid telemetry token configured
    When they pipe a JSON exploration payload into `explorations create --body-file -`
    Then the CLI reads the body from stdin and sends it verbatim as the request body

  @user
  Scenario: Shorthand flags and --body-file are mutually exclusive on explorations create
    Given the user has a valid telemetry token configured
    When they run `explorations create` with both a shorthand flag (e.g. --pattern) and --body-file
    Then the CLI errors with exit code 3 (user input error) explaining the two modes are mutually exclusive
    And no request is sent to the API

  @user
  Scenario: --body-file path is unreadable or does not exist
    Given the user passes --body-file path/that/does/not/exist.json
    When the CLI validates the flag
    Then it errors with exit code 3 naming the unresolvable path
    And no HTTP request is sent

  @user
  Scenario: --body-file contents are not valid JSON
    Given the user passes --body-file path/to/not-json.txt
    When the CLI reads and parses the file
    Then it errors with exit code 3 explaining the file is not valid JSON
    And no HTTP request is sent

  @user
  Scenario: --body-file contents are empty
    Given the user passes --body-file pointing at an empty file (or pipes empty stdin into --body-file -)
    When the CLI reads the content
    Then it errors with exit code 3 explaining an empty body is not a valid request
    And no HTTP request is sent

  @user
  Scenario: --body-file contents exceed 10MB safety cap
    Given the user passes --body-file pointing at a file larger than 10MB (or pipes >10MB through stdin)
    When the CLI reads the content
    Then it errors with exit code 3 naming the cap
    And no HTTP request is sent

  @user
  Scenario: List explorations
    Given the user has a valid telemetry token configured
    When they run `explorations list`
    Then they see a table with columns ID, NAME, DATE RANGE, and UPDATED

  @user
  Scenario: Get a single exploration by ID
    Given the user has a valid telemetry token configured
    When they run `explorations get <id>`
    Then they see the exploration's curated detail fields (name, date range, chart type, query summary, timestamps)

  @user
  Scenario: Update an exploration via shorthand flags
    Given the user has a valid telemetry token configured
    And an existing exploration
    When they run `explorations update <id>` with any of --name, --pattern, --source-id
    Then the CLI sends a PATCH containing only the user-provided fields
    And they see the updated exploration's details

  @user
  Scenario: Update an exploration via body file for fields outside the shorthand
    Given the user has a valid telemetry token configured
    When they run `explorations update <id> --body-file path/to/update.json`
    Then the CLI sends a PATCH with the file contents as the body
    And they see the updated exploration's details

  @user
  Scenario: Delete an exploration with no dependent alerts
    Given the user has a valid telemetry token configured
    And an exploration with zero alerts attached
    When they run `explorations delete <id> --yes`
    Then they see a confirmation message showing the deleted exploration ID

  @user
  Scenario: Refuse to delete an exploration that has live alerts attached
    Given the user has a valid telemetry token configured
    And an exploration with one or more dependent alerts
    When they run `explorations delete <id> --yes`
    Then the CLI issues GET /api/v2/explorations/<id>/alerts?per_page=1 and observes a non-empty result
    And prints an error using the shared DEPENDENCIES_EXIST template
    And the error uses the phrase "at least 1 dependent alert" (short-circuit wording)
    And includes the command `logs-alerts list --exploration-id <id>` with the literal ID (not a placeholder)
    And mentions --force as the override
    And exits with code 4 (dependency conflict)
    And the exploration is not deleted

  @user
  Scenario: Force-delete an exploration that has live alerts attached
    Given the user has a valid telemetry token configured
    And an exploration with dependent alerts
    When they run `explorations delete <id> --yes --force`
    Then the CLI skips the dependency check and issues the DELETE
    And they see a confirmation message showing the deleted exploration ID

  @user
  Scenario: Refuse to delete without --yes
    Given the user has a valid telemetry token configured
    When they run `explorations delete <id>` without --yes
    Then the CLI errors with exit code 3 explaining --yes is required for destructive operations
    And no DELETE is sent

  @user
  Scenario: DELETE fails after refcount precheck passed (concurrent dependency creation)
    Given the precheck found zero dependent alerts
    And a concurrent actor attaches a dependent alert before the DELETE is issued
    When the DELETE returns 409 or 422
    Then the CLI errors with stderr prefix `error: dependency_appeared_after_precheck:` followed by the specific inspect command with the literal ID
    And the CLI exits with code 4 (dependency conflict)
    And the CLI does not retry the DELETE

  @user
  Scenario: Upsert creates when no exploration matches the name
    Given the user has a valid telemetry token configured
    And no existing exploration has the name "INF-3017"
    When they run `explorations create --source-id <id> --pattern "AADSTS7000215" --name "INF-3017" --upsert`
    Then the CLI creates a new exploration
    And the JSON output is {"action": "created", "resource": { <full exploration object> }}

  @user
  Scenario: Upsert updates when exactly one exploration matches the name
    Given the user has a valid telemetry token configured
    And exactly one exploration exists with the name "INF-3017" whose fields differ from the provided shorthand
    When they run `explorations create --source-id <id> --pattern "AADSTS7000215" --name "INF-3017" --upsert`
    Then the CLI issues a PATCH containing ONLY the fields corresponding to provided shorthand flags (not defaulted fields)
    And the JSON output is {"action": "updated", "resource": { <full exploration object> }}

  @user
  Scenario: Upsert is a no-op when existing state already matches provided shorthand flags
    Given the user has a valid telemetry token configured
    And exactly one exploration named "INF-3017" exists whose fields corresponding to provided shorthand already equal the provided values
    When they run `explorations create ... --upsert`
    Then the CLI does not send a PATCH
    And the JSON output is {"action": "unchanged", "resource": { <full exploration object> }}
    And the CLI exits with code 0

  @user
  Scenario: Upsert refuses when two or more explorations match the name
    Given the user has a valid telemetry token configured
    And two or more explorations exist with the same name
    When they run `explorations create ... --upsert`
    Then the CLI errors with exit code 3 naming the duplicate count
    And the error says to use --name with a unique value, or to rename the duplicates via `explorations update`
    And no POST or PATCH is sent

  @user
  Scenario: --upsert is incompatible with --body-file
    Given the user has a valid telemetry token configured
    When they run `explorations create --body-file path.json --upsert`
    Then the CLI errors with exit code 3 explaining --upsert requires shorthand flags

  @user
  Scenario: Upsert is best-effort, not atomic
    Given --upsert found zero matches and the CLI issues the POST
    When a concurrent actor creates a resource with the same name between the list and the POST
    Then both creates may succeed, producing duplicates on the next run's upsert
    And this non-atomicity is documented in the --help output for every `<resource> create --upsert`

  @user
  Scenario: Upsert handles PATCH 404 as a race condition
    Given --upsert found exactly one match and the CLI issues a PATCH
    When the PATCH returns 404 (the matched resource was deleted between list and PATCH)
    Then the CLI errors with stderr prefix `error: upsert_match_deleted_after_precheck:`
    And the CLI exits with code 5 (upstream API error)
    And the CLI does not retry

  @user
  Scenario: Pattern with special characters is encoded safely into the where_condition
    Given the user passes --pattern containing quotes, backslashes, or regex metacharacters (e.g., `"; DROP *\.`)
    When the CLI builds the exploration's where_condition
    Then the pattern is escaped so the resulting query behaves as a literal message match
    And no query-syntax error is raised by BetterStack

  @technical
  Scenario: Explorations endpoints
    Given the telemetry client
    When it issues exploration CRUD requests
    Then list sends GET to /api/v2/explorations
    And get sends GET to /api/v2/explorations/{id}
    And create sends POST to /api/v2/explorations
    And update sends PATCH to /api/v2/explorations/{id}
    And delete sends DELETE to /api/v2/explorations/{id}

  @technical
  Scenario: Count-matching exploration shorthand payload
    Given the user passed --source-id X --pattern Y --name Z
    When the client builds the create payload
    Then chart.chart_type is "number_chart"
    And queries[0].query_type is "tail_query"
    And queries[0].where_condition encodes the pattern as a literal message match with safe escaping for quotes, backslashes, and regex metacharacters
    And queries[0].source_variable references source ID X
    And date_range_from defaults to "now-1h" and date_range_to defaults to "now"
    And name is Z

  @technical
  Scenario: Source ID format
    Given the BetterStack API treats source IDs as numeric strings
    When the CLI accepts --source-id
    Then it accepts both a numeric string (e.g., "123456") and raw integer inputs
    And passes the value through to the API unchanged in request payloads

  @technical
  Scenario: --body-file sends contents verbatim
    Given the user passed --body-file path.json (or --body-file -)
    When the client issues the POST or PATCH
    Then the request body is the file/stdin contents byte-for-byte after JSON-parse validation
    And no merging with shorthand flags occurs
    And Content-Type is application/json

  @technical
  Scenario: Pre-delete refcount check on explorations delete (short-circuit via per_page=1)
    Given the user ran `explorations delete <id> --yes` without --force
    When the CLI processes the command
    Then it sends a single GET /api/v2/explorations/{id}/alerts?per_page=1
    And if the response contains one or more alerts, it returns the DEPENDENCIES_EXIST error with count wording "at least 1"
    And if the response is empty, it proceeds with the DELETE

  @technical
  Scenario: Explorations table output
    Given a list of explorations
    When rendered as a table
    Then columns are: ID, NAME, DATE RANGE, UPDATED
    And columns are aligned with tabwriter
