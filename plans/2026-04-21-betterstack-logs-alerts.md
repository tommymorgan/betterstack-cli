# Feature: BetterStack Logs Alerts, Explorations, Escalation Policies

**Created**: 2026-04-21
**Goal**: Let users codify log-pattern → incident rules (and the escalation policies they route through) from the CLI, so signals like INF-3017 can be caught automatically instead of via CloudWatch metric filter workarounds.

## User Requirements

### Authentication (dual-host)

<!-- Living: features/authentication.feature::Telemetry commands prefer BETTERSTACK_TELEMETRY_TOKEN -->
<!-- Action: extends -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry commands prefer BETTERSTACK_TELEMETRY_TOKEN
  Given the user has set both BETTERSTACK_API_TOKEN and BETTERSTACK_TELEMETRY_TOKEN
  When they run a telemetry command (logs-alerts, explorations, or logs-sources)
  Then the CLI authenticates using BETTERSTACK_TELEMETRY_TOKEN

<!-- Living: features/authentication.feature::Telemetry commands fall back to BETTERSTACK_API_TOKEN -->
<!-- Action: extends -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry commands fall back to BETTERSTACK_API_TOKEN when telemetry token is unset
  Given the user has set BETTERSTACK_API_TOKEN but not BETTERSTACK_TELEMETRY_TOKEN
  When they run a telemetry command on an interactive (TTY) session
  Then the CLI authenticates using BETTERSTACK_API_TOKEN
  And stderr emits a one-line notice once per process: "note: BETTERSTACK_TELEMETRY_TOKEN unset; sent BETTERSTACK_API_TOKEN to telemetry.betterstack.com (works only for global tokens)"

<!-- Living: features/authentication.feature::Telemetry fallback notice suppression -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry fallback notice is suppressed on non-TTY and when --quiet / BETTERSTACK_QUIET is set
  Given the user has set BETTERSTACK_API_TOKEN but not BETTERSTACK_TELEMETRY_TOKEN
  When the CLI runs in a non-TTY context (CI, piped) OR --quiet is passed OR BETTERSTACK_QUIET is set to a truthy value
  Then the fallback notice is not emitted
  And the CLI still authenticates using BETTERSTACK_API_TOKEN

<!-- Living: features/authentication.feature::Uptime commands never use the telemetry token -->
<!-- Action: extends -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Uptime commands never use the telemetry token
  Given the user has set both BETTERSTACK_API_TOKEN and BETTERSTACK_TELEMETRY_TOKEN
  When they run an uptime command (incidents, monitors, integrations, policies)
  Then the CLI authenticates using BETTERSTACK_API_TOKEN only

<!-- Living: features/authentication.feature::Uptime command with only telemetry token set -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Uptime command fails explicitly when only the telemetry token is set
  Given the user has BETTERSTACK_TELEMETRY_TOKEN set and BETTERSTACK_API_TOKEN unset (no config file)
  When they run an uptime command
  Then the CLI errors before issuing any HTTP request
  And the error names BETTERSTACK_API_TOKEN as the env var required for uptime commands
  And the CLI exits with status code 2 (auth failure)

<!-- Living: features/authentication.feature::Auth error names which env vars are set -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Auth error on telemetry command names which env vars are currently set
  Given the user has BETTERSTACK_API_TOKEN set but it is team-scoped to uptime only
  And BETTERSTACK_TELEMETRY_TOKEN is unset
  When they run a telemetry command and telemetry.betterstack.com returns 401 or 403
  Then the CLI prints a single actionable error to stderr naming the HTTP status code
  And the error lists which token env vars are currently set and which are unset
  And the error suggests setting BETTERSTACK_TELEMETRY_TOKEN or using a global BETTERSTACK_API_TOKEN
  And the CLI exits with status code 2 (auth failure)

<!-- Living: features/authentication.feature::Telemetry token config file precedence -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry token env var takes precedence over config file
  Given ~/.config/betterstack/config.yaml contains telemetry_api_token: "from-file"
  And BETTERSTACK_TELEMETRY_TOKEN is set to "from-env"
  When a telemetry command runs
  Then the CLI uses "from-env"

<!-- Living: features/authentication.feature::Telemetry token from config file only -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry token resolved from config file when env is unset
  Given ~/.config/betterstack/config.yaml contains telemetry_api_token
  And BETTERSTACK_TELEMETRY_TOKEN is unset
  When a telemetry command runs
  Then the CLI uses the config file's telemetry_api_token
  And the existing 0600 permissions warning applies (matching current api_token handling)

<!-- Living: features/authentication.feature::Telemetry token env var only -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry token resolved from env when no config file has telemetry_api_token
  Given the config file does not contain telemetry_api_token
  And BETTERSTACK_TELEMETRY_TOKEN is set
  When a telemetry command runs
  Then the CLI uses the env var value

### Explorations

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Create a count-matching exploration with shorthand flags
  Given the user has a valid telemetry token configured
  When they run `explorations create --source-id <id> --pattern "AADSTS7000215" --name "INF-3017"`
  Then the CLI creates an exploration scoped to that source that counts occurrences of the pattern
  And they see the created exploration's ID, name, chart type, date range, and query summary

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Create an exploration from a raw JSON body file
  Given the user has a valid telemetry token configured
  And they have a JSON file describing a full exploration payload
  When they run `explorations create --body-file path/to/exploration.json`
  Then the CLI sends the file's contents verbatim as the request body
  And they see the created exploration's details

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --body-file - reads from stdin
  Given the user has a valid telemetry token configured
  When they pipe a JSON exploration payload into `explorations create --body-file -`
  Then the CLI reads the body from stdin and sends it verbatim as the request body

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Shorthand flags and --body-file are mutually exclusive on explorations create
  Given the user has a valid telemetry token configured
  When they run `explorations create` with both a shorthand flag (e.g. --pattern) and --body-file
  Then the CLI errors with exit code 3 (user input error) explaining the two modes are mutually exclusive
  And no request is sent to the API

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --body-file path is unreadable or does not exist
  Given the user passes --body-file path/that/does/not/exist.json
  When the CLI validates the flag
  Then it errors with exit code 3 naming the unresolvable path
  And no HTTP request is sent

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --body-file contents are not valid JSON
  Given the user passes --body-file path/to/not-json.txt
  When the CLI reads and parses the file
  Then it errors with exit code 3 explaining the file is not valid JSON
  And no HTTP request is sent

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --body-file contents are empty
  Given the user passes --body-file pointing at an empty file (or pipes empty stdin into --body-file -)
  When the CLI reads the content
  Then it errors with exit code 3 explaining an empty body is not a valid request
  And no HTTP request is sent

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --body-file contents exceed 10MB safety cap
  Given the user passes --body-file pointing at a file larger than 10MB (or pipes >10MB through stdin)
  When the CLI reads the content
  Then it errors with exit code 3 naming the cap
  And no HTTP request is sent

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: List explorations
  Given the user has a valid telemetry token configured
  When they run `explorations list`
  Then they see a table with columns ID, NAME, DATE RANGE, and UPDATED

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Get a single exploration by ID
  Given the user has a valid telemetry token configured
  When they run `explorations get <id>`
  Then they see the exploration's curated detail fields (name, date range, chart type, query summary, timestamps)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Update an exploration via shorthand flags
  Given the user has a valid telemetry token configured
  And an existing exploration
  When they run `explorations update <id>` with any of --name, --pattern, --source-id
  Then the CLI sends a PATCH containing only the user-provided fields
  And they see the updated exploration's details

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Update an exploration via body file for fields outside the shorthand
  Given the user has a valid telemetry token configured
  When they run `explorations update <id> --body-file path/to/update.json`
  Then the CLI sends a PATCH with the file contents as the body
  And they see the updated exploration's details

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Delete an exploration with no dependent alerts
  Given the user has a valid telemetry token configured
  And an exploration with zero alerts attached
  When they run `explorations delete <id> --yes`
  Then they see a confirmation message showing the deleted exploration ID

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Refuse to delete an exploration that has live alerts attached
  Given the user has a valid telemetry token configured
  And an exploration with one or more dependent alerts
  When they run `explorations delete <id> --yes`
  Then the CLI issues GET /api/v2/explorations/<id>/alerts?per_page=1 and observes a non-empty result
  And prints an error using the shared DEPENDENCIES_EXIST template (see "Consistent error template" below)
  And the error uses the phrase "at least 1 dependent alert" (short-circuit wording)
  And includes the command `logs-alerts list --exploration-id <id>` with the literal ID (not a placeholder)
  And mentions --force as the override
  And exits with code 4 (dependency conflict)
  And the exploration is not deleted

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Force-delete an exploration that has live alerts attached
  Given the user has a valid telemetry token configured
  And an exploration with dependent alerts
  When they run `explorations delete <id> --yes --force`
  Then the CLI skips the dependency check and issues the DELETE
  And they see a confirmation message showing the deleted exploration ID

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Refuse to delete without --yes
  Given the user has a valid telemetry token configured
  When they run `explorations delete <id>` without --yes
  Then the CLI errors with exit code 3 explaining --yes is required for destructive operations
  And no DELETE is sent

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: DELETE fails after refcount precheck passed (concurrent dependency creation)
  Given the precheck found zero dependent alerts
  And a concurrent actor attaches a dependent alert before the DELETE is issued
  When the DELETE returns 409 or 422
  Then the CLI errors with stderr prefix `error: dependency_appeared_after_precheck:` followed by the specific inspect command with the literal ID
  And the CLI exits with code 4 (dependency conflict)
  And the CLI does not retry the DELETE

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Upsert creates when no exploration matches the name
  Given the user has a valid telemetry token configured
  And no existing exploration has the name "INF-3017"
  When they run `explorations create --source-id <id> --pattern "AADSTS7000215" --name "INF-3017" --upsert`
  Then the CLI creates a new exploration
  And the JSON output is {"action": "created", "resource": { <full exploration object> }}

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Upsert updates when exactly one exploration matches the name
  Given the user has a valid telemetry token configured
  And exactly one exploration exists with the name "INF-3017" whose fields differ from the provided shorthand
  When they run `explorations create --source-id <id> --pattern "AADSTS7000215" --name "INF-3017" --upsert`
  Then the CLI issues a PATCH containing ONLY the fields corresponding to provided shorthand flags (not defaulted fields)
  And the JSON output is {"action": "updated", "resource": { <full exploration object> }}

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Upsert is a no-op when existing state already matches provided shorthand flags
  Given the user has a valid telemetry token configured
  And exactly one exploration named "INF-3017" exists whose fields corresponding to provided shorthand already equal the provided values
  When they run `explorations create ... --upsert`
  Then the CLI does not send a PATCH (equality is over provided-shorthand fields only, ignoring unprovided fields)
  And the JSON output is {"action": "unchanged", "resource": { <full exploration object> }}
  And the CLI exits with code 0

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Upsert refuses when two or more explorations match the name
  Given the user has a valid telemetry token configured
  And two or more explorations exist with the same name
  When they run `explorations create ... --upsert`
  Then the CLI errors with exit code 3 naming the duplicate count
  And the error says to use --name with a unique value, or to rename the duplicates via `explorations update`
  And no POST or PATCH is sent

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --upsert is incompatible with --body-file
  Given the user has a valid telemetry token configured
  When they run `explorations create --body-file path.json --upsert`
  Then the CLI errors with exit code 3 explaining --upsert requires shorthand flags

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Upsert is best-effort, not atomic
  Given --upsert found zero matches and the CLI issues the POST
  When a concurrent actor creates a resource with the same name between the list and the POST
  Then both creates may succeed, producing duplicates on the next run's upsert
  And this non-atomicity is documented in the --help output for every `<resource> create --upsert`

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Upsert handles PATCH 404 as a race condition
  Given --upsert found exactly one match and the CLI issues a PATCH
  When the PATCH returns 404 (the matched resource was deleted between list and PATCH)
  Then the CLI errors with stderr prefix `error: upsert_match_deleted_after_precheck:`
  And the CLI exits with code 5 (upstream API error)
  And the CLI does not retry

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Pattern with special characters is encoded safely into the where_condition
  Given the user passes --pattern containing quotes, backslashes, or regex metacharacters (e.g., `"; DROP *\.`)
  When the CLI builds the exploration's where_condition
  Then the pattern is escaped so the resulting query behaves as a literal message match
  And no query-syntax error is raised by BetterStack

### Logs-alerts

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Create a logs-alert on an existing exploration
  Given the user has a valid telemetry token configured
  And an existing exploration ID and an existing escalation policy ID
  When they run `logs-alerts create --exploration-id <eid> --threshold 3 --window 5m --policy-id <pid> --name "INF-3017"`
  Then the CLI creates an alert that fires when the exploration's count is >= 3 over 5 minutes
  And the alert routes through the named escalation policy
  And they see the created alert's ID, name, condition, check period, and paused status

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: `logs-alerts create --help` documents the policy-routing opinion
  Given the user runs `logs-alerts create --help`
  Then the help text mentions that alerts route via escalation policies
  And names `policies create` or `policies list` as the way to pick one
  And names `--body-file` as the escape hatch for ad-hoc direct channels

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Duration flags accept Go-style duration strings with units
  Given the user has a valid telemetry token configured
  When they run a command with `--window 5m`, `--check-period 30s`, or `--recovery 2h`
  Then the CLI parses the value via time.ParseDuration
  And converts it to integer seconds for the API request

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Duration flags reject values without units
  Given the user runs a command with a bare number (e.g., `--window 300`)
  Then the CLI errors with exit code 3 showing the expected format with units
  And the error mentions the maximum supported unit is hours (e.g., use 24h, not 1d)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Duration flags reject zero and negative values
  Given the user runs a command with `--window 0s` or `--window -5m`
  Then the CLI errors with exit code 3 explaining durations must be strictly positive

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Create a logs-alert from a raw JSON body file
  Given the user has a valid telemetry token configured
  When they run `logs-alerts create --exploration-id <eid> --body-file path/to/alert.json`
  Then the CLI sends the file contents verbatim as the alert body

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Shorthand flags and --body-file are mutually exclusive on logs-alerts create
  Given the user has a valid telemetry token configured
  When they run `logs-alerts create` with both shorthand flags (e.g. --threshold) and --body-file
  Then the CLI errors with exit code 3 explaining the two modes are mutually exclusive

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: List logs-alerts
  Given the user has a valid telemetry token configured
  When they run `logs-alerts list`
  Then they see a table with columns ID, NAME, CONDITION, CHECK/WINDOW, and PAUSED

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Filter logs-alerts by exploration
  Given the user has a valid telemetry token configured
  When they run `logs-alerts list --exploration-id <eid>`
  Then they see only alerts attached to that exploration (filter applied client-side after full pagination)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Filter logs-alerts by escalation policy
  Given the user has a valid telemetry token configured
  When they run `logs-alerts list --policy-id <pid>`
  Then they see only alerts whose escalation_target references that policy (filter applied client-side after full pagination)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Get a single logs-alert by ID
  Given the user has a valid telemetry token configured
  When they run `logs-alerts get <id>`
  Then they see the alert's curated detail fields including exploration_id, condition, periods, and escalation target

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Update a logs-alert via shorthand flags
  Given the user has a valid telemetry token configured
  And an existing logs-alert
  When they run `logs-alerts update <id>` with any of --name, --threshold, --window, --check-period, --query-period, --recovery, --policy-id, --paused, --no-paused
  Then the CLI sends a PATCH containing only the user-provided fields
  And they see the updated alert's details

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Update a logs-alert via body file for fields outside the shorthand
  Given the user has a valid telemetry token configured
  When they run `logs-alerts update <id> --body-file path/to/update.json`
  Then the CLI sends a PATCH with the file contents as the body
  And the CLI uses --body-file for nested shapes (anomaly_rrcf alerts, custom escalation_target shapes, metadata objects)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Delete a logs-alert
  Given the user has a valid telemetry token configured
  When they run `logs-alerts delete <id> --yes`
  Then they see a confirmation message showing the deleted alert ID

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: logs-alerts delete rejects --force explicitly
  Given the user runs `logs-alerts delete <id> --yes --force`
  Then the CLI errors with exit code 3 explaining --force is only valid on commands with dependency checks (explorations delete, policies delete)
  And no DELETE is sent

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Upsert on logs-alerts scopes the name match to the target exploration
  Given the user has a valid telemetry token configured
  And an existing alert named "INF-3017" attached to exploration A
  When they run `logs-alerts create --exploration-id B --name "INF-3017" ... --upsert`
  Then the CLI creates a new alert on exploration B (no match found within exploration B's alerts)
  And the alert on exploration A is unchanged

### Policies (escalation policies)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: List escalation policies
  Given the user has a valid uptime token configured
  When they run `policies list`
  Then they see a table with columns ID, NAME, STEPS, and UPDATED

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Get a single policy by ID
  Given the user has a valid uptime token configured
  When they run `policies get <id>`
  Then they see the policy's curated detail fields including step configuration

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Create a policy from a JSON body file
  Given the user has a valid uptime token configured
  When they run `policies create --body-file path/to/policy.json`
  Then the CLI sends the file contents as the POST body

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Policies create does not support --upsert
  Given the user runs `policies create --body-file path.json --upsert`
  Then the CLI errors with exit code 3 explaining --upsert is not supported for policies (policies require --body-file and the "shorthand OR body" rule rules out body-file upsert)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Update a policy from a JSON body file
  Given the user has a valid uptime token configured
  When they run `policies update <id> --body-file path/to/update.json`
  Then the CLI sends a PATCH with the file contents as the body

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Delete a policy with no dependent alerts
  Given the user has a valid uptime token configured
  And no logs-alerts reference the policy
  When they run `policies delete <id> --yes`
  Then they see a confirmation message showing the deleted policy ID

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Refuse to delete a policy with dependent alerts
  Given the user has a valid uptime token configured
  And one or more logs-alerts reference the policy via escalation_target
  When they run `policies delete <id> --yes`
  Then the CLI paginates GET /api/v2/alerts and short-circuits on the first alert whose escalation_target.policy_id equals <id>
  And prints an error using the shared DEPENDENCIES_EXIST template
  And the error uses the phrase "at least 1 dependent alert"
  And includes the command `logs-alerts list --policy-id <id>` with the literal ID
  And mentions --force as the override
  And exits with code 4 (dependency conflict)
  And the policy is not deleted

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Force-delete a policy with dependent alerts
  Given the user has a valid uptime token configured
  And a policy with dependent alerts
  When they run `policies delete <id> --yes --force`
  Then the CLI skips the dependency check and issues the DELETE

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: DELETE fails after policy refcount precheck passed
  Given the policy precheck found zero dependent alerts
  And a concurrent actor creates a dependent alert before the DELETE
  When the DELETE returns 409 or 422
  Then the CLI errors with stderr prefix `error: dependency_appeared_after_precheck:` followed by `logs-alerts list --policy-id <literal-id>`
  And the CLI exits with code 4

### Logs-sources

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: List log sources
  Given the user has a valid telemetry token configured
  When they run `logs-sources list`
  Then they see a table with columns ID, NAME, PLATFORM, and TEAM

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Get a single log source by ID
  Given the user has a valid telemetry token configured
  When they run `logs-sources get <id>`
  Then they see the source's curated detail fields (name, platform, retention, team name)

### Cross-cutting

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Durations render human-readable in table output
  Given any list or get command with a period field (e.g., check_period, query_period)
  When rendered as a table or curated detail view
  Then durations appear as human-readable strings (e.g., "5m", "30s", "2h")

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Durations render as integer seconds in JSON output
  Given any list or get command with a period field
  When rendered as JSON with `--json`
  Then period fields are integer seconds (matching the API)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: JSON output envelope is consistent across verbs
  Given any command with --json
  When the CLI renders the response
  Then list commands output {"items": [ <resource objects> ]}
  And get commands output <resource object> directly
  And create/update (non-upsert) commands output <resource object> directly
  And upsert commands output {"action": "created"|"updated"|"unchanged", "resource": <resource object>}
  And delete commands output {"id": "<id>", "deleted": true}
  And error responses output {"error": {...}} to stderr (not stdout)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --json is a global flag
  Given any command
  When --json is passed at any position
  Then the CLI emits the documented JSON envelope for that verb
  And without --json, the CLI emits the curated human-readable output (TTY behavior does not auto-flip to JSON)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Consistent error template for dependency conflicts
  Given any delete command refuses due to dependent resources
  When the CLI emits the error
  Then in --json mode stderr contains {"error": {"code": "DEPENDENCIES_EXIST", "count": "<at-least-N>", "inspect_command": "<literal command with ID>", "resource_type": "<type>", "resource_id": "<id>"}}
  And in default mode stderr contains a parameterized human template referencing the same fields
  And the exit code is 4 (dependency conflict)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Exit code taxonomy
  Given any CLI invocation
  When the CLI terminates
  Then exit code 0 means success
  And exit code 1 means a generic/unclassified error
  And exit code 2 means authentication failure (missing, invalid, or wrong-scope token)
  And exit code 3 means user input error (bad flag combination, malformed body, missing --yes, etc.)
  And exit code 4 means dependency conflict (refcount refusal, dependency_appeared_after_precheck)
  And exit code 5 means upstream API error (5xx, unexpected 4xx, upsert_match_deleted_after_precheck)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: List commands fetch all pages by default
  Given any list command
  When the BetterStack API returns paginated results
  Then the CLI fetches all pages and returns a combined result
  And a --limit flag caps the returned count across pages (matching the existing incidents/monitors list behavior)

## Technical Specifications

### Client architecture (dual-host)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry base URL
  Given the telemetry client is constructed
  When it issues a request
  Then the base URL is https://telemetry.betterstack.com
  And a BETTERSTACK_TELEMETRY_BASE_URL env var overrides the base URL (mirroring BETTERSTACK_BASE_URL for uptime)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Both clients honor Retry-After on 429 identically
  Given a client (uptime or telemetry) receives HTTP 429 with a Retry-After header
  When the client processes the response
  Then it waits for the duration specified in Retry-After and retries up to 3 times
  And the observable retry behavior is identical across uptime and telemetry clients

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Both clients apply a 30s request timeout
  Given a client (uptime or telemetry) issues a request
  When the request takes longer than 30 seconds
  Then the client cancels and returns a timeout error (identical across clients)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Both clients set Bearer auth and emit identical error types on 4xx/5xx
  Given a client (uptime or telemetry) issues a request
  Then it sets an Authorization header with "Bearer <token>"
  And on 4xx/5xx responses it returns an error whose Go type and field shape is identical across clients

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Telemetry token resolution
  Given config.ResolveTelemetryToken is called
  When BETTERSTACK_TELEMETRY_TOKEN is set (env)
  Then it returns that value
  When BETTERSTACK_TELEMETRY_TOKEN is unset and ~/.config/betterstack/config.yaml contains telemetry_api_token
  Then it returns the config file value
  When neither env nor config has a telemetry_api_token
  Then it falls back to the uptime token resolution (BETTERSTACK_API_TOKEN then config's api_token)
  When no token can be resolved
  Then it returns an error whose message names all env vars and config fields, and explains the fallback rule

### Explorations API

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Explorations endpoints
  Given the telemetry client
  When it issues exploration CRUD requests
  Then list sends GET to /api/v2/explorations
  And get sends GET to /api/v2/explorations/{id}
  And create sends POST to /api/v2/explorations
  And update sends PATCH to /api/v2/explorations/{id}
  And delete sends DELETE to /api/v2/explorations/{id}

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Count-matching exploration shorthand payload
  Given the user passed --source-id X --pattern Y --name Z
  When the client builds the create payload
  Then chart.chart_type is "number_chart"
  And queries[0].query_type is "tail_query"
  And queries[0].where_condition encodes the pattern as a literal message match with safe escaping for quotes, backslashes, and regex metacharacters
  And queries[0].source_variable references source ID X
  And date_range_from defaults to "now-1h" and date_range_to defaults to "now"
  And name is Z

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Source ID format
  Given the BetterStack API treats source IDs as numeric strings
  When the CLI accepts --source-id
  Then it accepts both a numeric string (e.g., "123456") and raw integer inputs
  And passes the value through to the API unchanged in request payloads
  (Name-to-ID resolution is out of scope for this plan; users run `logs-sources list` to discover IDs)

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --body-file sends contents verbatim
  Given the user passed --body-file path.json (or --body-file -)
  When the client issues the POST or PATCH
  Then the request body is the file/stdin contents byte-for-byte after JSON-parse validation
  And no merging with shorthand flags occurs
  And Content-Type is application/json

### Logs-alerts API

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Logs-alerts endpoints
  Given the telemetry client
  When it issues logs-alerts CRUD requests
  Then list sends GET to /api/v2/alerts (filters --exploration-id and --policy-id applied client-side after full pagination)
  And get sends GET to /api/v2/alerts/{id}
  And create sends POST to /api/v2/explorations/{exploration_id}/alerts
  And update sends PATCH to /api/v2/alerts/{id}
  And delete sends DELETE to /api/v2/alerts/{id}

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Logs-alerts create payload from shorthand flags
  Given the user passed --exploration-id E --threshold N --window W --policy-id P --name Z
  When the client builds the create payload
  Then alert_type is "threshold"
  And operator is "higher_than_or_equal"
  And value is N
  And check_period is W in integer seconds
  And query_period is W in integer seconds
  And escalation_target is {"policy_id": P} where P is an integer
  And name is Z
  And non-policy escalation targets (team_id, user_id, schedule_id) require --body-file

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Duration parsing
  Given a duration flag value (e.g., "5m", "30s", "2h")
  When the CLI parses it
  Then it uses time.ParseDuration
  And converts to integer seconds for API payloads
  And rejects values without units with an error that mentions the "max unit: h; use 24h, not 1d" guidance
  And rejects zero or negative values with a "strictly positive" error

### Escalation policies API

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Escalation policies endpoints
  Given the uptime client
  When it issues policies CRUD requests
  Then list sends GET to /api/v3/policies
  And get sends GET to /api/v3/policies/{id}
  And create sends POST to /api/v3/policies
  And update sends PATCH to /api/v3/policies/{id}
  And delete sends DELETE to /api/v3/policies/{id}

### Logs-sources API

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Logs-sources endpoints
  Given the telemetry client
  When it issues source read requests
  Then list sends GET to /api/v2/sources
  And get sends GET to /api/v2/sources/{id}

### Cascade refcount semantics

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Pre-delete refcount check on explorations delete (short-circuit via per_page=1)
  Given the user ran `explorations delete <id> --yes` without --force
  When the CLI processes the command
  Then it sends a single GET /api/v2/explorations/{id}/alerts?per_page=1
  And if the response contains one or more alerts, it returns the DEPENDENCIES_EXIST error with count wording "at least 1"
  And if the response is empty, it proceeds with the DELETE

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Pre-delete refcount check on policies delete (paginate until first match)
  Given the user ran `policies delete <id> --yes` without --force
  When the CLI processes the command
  Then it paginates GET /api/v2/alerts, filtering client-side for alerts whose escalation_target.policy_id equals <id>
  And on the first matching alert, it short-circuits (does not continue paginating) and returns the DEPENDENCIES_EXIST error with count wording "at least 1"
  And if pagination completes with zero matches, it proceeds with the DELETE

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Refcount check is best-effort, not atomic
  Given the refcount check passes and the CLI issues the DELETE
  When a concurrent actor creates a dependent between the GET and the DELETE
  Then the DELETE may either orphan the dependent (if the server allows) or return 409/422 (if the server enforces)
  And this non-atomicity is documented in --help for both `explorations delete` and `policies delete`

### Upsert semantics

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --upsert flag implementation
  Given --upsert was passed with shorthand flags
  When the CLI processes the create command
  Then it first lists resources of that type scoped by name and the relevant parent (team for explorations; exploration_id for logs-alerts)
  And equality is evaluated over ONLY the fields corresponding to provided shorthand flags (unprovided fields are ignored)
  And on 0 matches it issues the POST and outputs action=created
  And on 1 match whose provided-field values differ, it issues a PATCH containing only the provided shorthand fields and outputs action=updated
  And on 1 match whose provided-field values already equal the provided values, it issues no write and outputs action=unchanged
  And on 2+ matches it errors without issuing a POST or PATCH

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --upsert is incompatible with --body-file
  Given --upsert and --body-file are both set
  When the CLI validates flags
  Then it returns an error before issuing any HTTP request
  And the CLI exits with code 3

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --upsert is not supported for policies
  Given the user passes --upsert with policies create
  When the CLI validates flags
  Then it errors with exit code 3 explaining policies require --body-file (no shorthand exists) and --upsert requires shorthand (the "shorthand OR body" rule precludes body-file upsert)

### Output

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Explorations table output
  Given a list of explorations
  When rendered as a table
  Then columns are: ID, NAME, DATE RANGE, UPDATED
  And columns are aligned with tabwriter

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Logs-alerts table output
  Given a list of logs-alerts
  When rendered as a table
  Then columns are: ID, NAME, CONDITION, CHECK/WINDOW, PAUSED
  And CONDITION combines operator and value (e.g., ">= 3")
  And CHECK/WINDOW shows check_period in human-readable form (e.g., "5m")

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Policies table output
  Given a list of escalation policies
  When rendered as a table
  Then columns are: ID, NAME, STEPS, UPDATED
  And STEPS shows the count of steps in the policy

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Logs-sources table output
  Given a list of log sources
  When rendered as a table
  Then columns are: ID, NAME, PLATFORM, TEAM

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: Curated detail views for get commands
  Given any `<resource> get <id>` command (human mode)
  When rendered as a detail view
  Then it shows a stable curated set of fields documented in --help (not every field the API returned)
  And --json mode passes through the raw API response for scripting use

### Flag semantics

<!-- Living: none (initial implementation) -->
<!-- Action: creates -->
<!-- Status: DONE -->
<!-- Living updated: YES -->
Scenario: --yes and --force are orthogonal
  Given a delete command
  When flags are validated
  Then --yes alone skips only the destructive-action confirmation
  And --force alone (without --yes) still errors because destructive-action confirmation is required
  And --force is only accepted by `explorations delete` and `policies delete` (which have dependency checks to skip)
  And `logs-alerts delete` explicitly rejects --force (exit code 3)

## Affected Documentation

- [ ] Update README.md — add sections for `explorations`, `logs-alerts`, `policies`, and `logs-sources` commands; document the new BETTERSTACK_TELEMETRY_TOKEN env var and the telemetry fallback rule; document the `--body-file`, `--upsert`, `--force`, and `--quiet` flags; note the duration-string format (max unit: h) and the exit code taxonomy
- [ ] Extend features/authentication.feature — add the dual-host token scenarios (preferred token, fallback with TTY notice, non-TTY suppression, uptime-with-only-telemetry error, auth error body, config precedence, config-only path, env-only path)
- [ ] Create features/explorations.feature — new file with all exploration scenarios from this plan
- [ ] Create features/logs-alerts.feature — new file with all logs-alerts scenarios from this plan
- [ ] Create features/policies.feature — new file with all policies scenarios from this plan
- [ ] Create features/logs-sources.feature — new file with logs-sources scenarios from this plan
- [ ] Extend features/output.feature — add the JSON envelope, consistent error template, exit code taxonomy, and pagination scenarios (cross-cutting)
- [ ] Extend features/error-handling.feature — add the `dependency_appeared_after_precheck` and `upsert_match_deleted_after_precheck` error-prefix scenarios and their exit code mappings
- [ ] PKGBUILD — no changes required (binary name unchanged)

## Notes

### Why this shape

- **Dual-host split**: BetterStack Logs (Telemetry) lives at `telemetry.betterstack.com` with its own team-scoped API token distinct from the uptime token at `uptime.betterstack.com`. A global token works for both hosts; a team-scoped token is host-specific. The telemetry-prefers-telemetry, falls-back-to-uptime rule lets users keep least-privilege team-scoped tokens without losing convenience for global-token users. The fallback is a silent-UX-leak risk (not a security breach — the fallback token will 401 against the wrong host), but surfacing a one-line TTY notice teaches users that they are relying on a global-token property before a team-scoped rotation breaks them.
- **Alert model mismatch**: BetterStack doesn't have a standalone "logs alert rule" resource. An alert is attached to an *exploration* (saved query). The exploration carries the query; the alert carries the firing condition. This plan matches BetterStack's model rather than abstracting it — two commands for one user goal, but no hidden state and explorations can be reused.
- **Opinion: alerts route via policies**: The API itself confirms this opinion — when `escalation_target` is set, individual notification channels (`call`, `sms`, etc.) are ignored. Shipping the CLI without direct-channel flags encodes this opinion. `--body-file` remains as the escape hatch for users who genuinely need direct channels.
- **Count-matching shorthand**: 95% of real log-pattern alerts (INF-3017, rate-limit breaches, error-class counts, third-party 5xx spikes) are count-over-window shapes. The shorthand flags (`--source-id`, `--pattern`, `--name` on explorations; `--threshold`, `--window`, `--policy-id` on alerts) make this case terse. `--body-file` covers the rest.

### Design invariants (non-goals and constraints that must not drift)

- **Always client-side filtering for list commands (invariant)**. Filters like `logs-alerts list --policy-id` and `--exploration-id` apply after full pagination, client-side. If a future plan adds server-side filtering as a performance optimization, it MUST be a pure internal change with no user-observable behavior difference — no flag, no opt-in, no notice. The mental contract from v1 is "client-side filtering," and the implementation must preserve that contract forever, even if the code path diverges.
- **Shorthand OR body, never both**. `--body-file` is mutually exclusive with shorthand flags on every create/update. No merging, no precedence rules, no silent override. This is a uniform rule across all commands and must not admit exceptions (which is why `--upsert` is not supported for policies — see below).
- **Upsert equality is over provided-shorthand fields only**. The PATCH on an upsert match contains only the fields corresponding to user-provided shorthand flags; unprovided fields are not compared and not overwritten. This keeps upsert idempotent across CLI-version changes that might alter defaulted fields.
- **Policies are shorthand-less and therefore upsert-less**. Shorthand flags can't reasonably express nested escalation steps, per-step channels, and on-call rotations. Since upsert requires shorthand (per the uniform rule above), policies get full CRUD via `--body-file` without upsert. Users who need idempotent policy provisioning can `policies list --json | jq -e '…'` and gate `create` accordingly.
- **--yes and --force are orthogonal**. `--yes` skips destructive-action confirmation; `--force` skips the refcount precheck. Conflating them is a bug.

### Design refinements captured during brainstorming + expert review

- `--body-file` accepts a path or `-` for stdin. Validated before any HTTP request: file must exist, be readable, parse as JSON, be non-empty, and be ≤10MB.
- Upsert non-atomicity and delete non-atomicity are both documented in `--help`. The CLI treats post-precheck races with deterministic, machine-matchable error prefixes: `error: dependency_appeared_after_precheck:` (delete race, exit 4) and `error: upsert_match_deleted_after_precheck:` (upsert race, exit 5).
- The fallback token notice is TTY-only and emitted once per process. `--quiet` and `BETTERSTACK_QUIET` suppress it.
- Duration flags use `time.ParseDuration` (max unit: h, units required, strictly positive).
- JSON output envelope is uniform across verbs. Upsert wraps: `{"action": "created"|"updated"|"unchanged", "resource": {...}}`. Delete emits `{"id": "...", "deleted": true}`. Errors emit `{"error": {...}}` to stderr. `--json` is a global flag; non-TTY stdout does NOT auto-flip to JSON (keeps piping to `less` predictable).
- Exit codes: 0=success, 1=generic, 2=auth, 3=user input, 4=dependency conflict, 5=upstream API error.
- Consistent DEPENDENCIES_EXIST error template across both delete commands — structured in `--json` mode, parameterized human template otherwise. Error strings substitute the literal resource ID so agent consumers don't have to construct the inspect command themselves.
- Help-text "assertions" in Gherkin are keyword-level only (e.g., "mentions policies", "mentions --body-file"). Exact copy stays out of specs and lives in docs review.

### Implementation notes

- The existing `internal/client` package currently models a single host. This plan extends it to two clients (uptime, telemetry) that share the request pipeline (timeout, retry, headers). Factor the shared pipeline into a common request-executor type before adding telemetry endpoints to avoid duplicated retry/timeout logic — the behavioral parity scenarios (Retry-After, 30s timeout, Bearer auth, error types) will test that the factoring worked.
- `internal/config` adds `ResolveTelemetryToken()` alongside the existing `ResolveToken()`. The telemetry resolver checks `BETTERSTACK_TELEMETRY_TOKEN`, then `telemetry_api_token` in the config file, then falls back to the uptime resolver. The fallback TTY notice lives at the call site (CLI command handler), not in the resolver, so unit tests of config stay pure.
- `--body-file` reads file contents into bytes, JSON-parses it for validation (discarding the parsed result), checks size, then passes the original bytes through verbatim. `--body-file -` reads from stdin with the same validation pipeline.
- Escalation target shape for policy reference: `{"policy_id": <id>}`. The policy ID is an integer in the API. The CLI accepts either a numeric string or a raw integer on `--policy-id` and passes it through as a JSON number.
- Pattern-to-`where_condition` encoding must escape quotes, backslashes, and any regex metacharacters that BetterStack's tail query DSL interprets. Implement a dedicated encoder function with round-trip tests for adversarial patterns.
- Refcount-check commands are distinct: `explorations delete` uses the scoped endpoint (single request with `per_page=1`); `policies delete` paginates the flat alerts endpoint and short-circuits on first match. Name this asymmetry in the code and in doc comments so a future refactor doesn't accidentally collapse them.
