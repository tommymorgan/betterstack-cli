Feature: Logs Alerts
  As a user of the BetterStack CLI
  I want to attach threshold alerts to explorations
  So that I can codify log-pattern → incident rules routed through escalation policies

  @user
  Scenario: Create a logs-alert on an existing exploration
    Given the user has a valid telemetry token configured
    And an existing exploration ID and an existing escalation policy ID
    When they run `logs-alerts create --exploration-id <eid> --threshold 3 --window 5m --policy-id <pid> --name "INF-3017"`
    Then the CLI creates an alert that fires when the exploration's count is >= 3 over 5 minutes
    And the alert routes through the named escalation policy
    And they see the created alert's ID, name, condition, check period, and paused status

  @user
  Scenario: `logs-alerts create --help` documents the policy-routing opinion
    Given the user runs `logs-alerts create --help`
    Then the help text mentions that alerts route via escalation policies
    And names `policies create` or `policies list` as the way to pick one
    And names `--body-file` as the escape hatch for ad-hoc direct channels

  @user
  Scenario: Duration flags accept Go-style duration strings with units
    Given the user has a valid telemetry token configured
    When they run a command with `--window 5m`, `--check-period 30s`, or `--recovery 2h`
    Then the CLI parses the value via time.ParseDuration
    And converts it to integer seconds for the API request

  @user
  Scenario: Duration flags reject values without units
    Given the user runs a command with a bare number (e.g., `--window 300`)
    Then the CLI errors with exit code 3 showing the expected format with units
    And the error mentions the maximum supported unit is hours (e.g., use 24h, not 1d)

  @user
  Scenario: Duration flags reject zero and negative values
    Given the user runs a command with `--window 0s` or `--window -5m`
    Then the CLI errors with exit code 3 explaining durations must be strictly positive

  @user
  Scenario: Create a logs-alert from a raw JSON body file
    Given the user has a valid telemetry token configured
    When they run `logs-alerts create --exploration-id <eid> --body-file path/to/alert.json`
    Then the CLI sends the file contents verbatim as the alert body

  @user
  Scenario: Shorthand flags and --body-file are mutually exclusive on logs-alerts create
    Given the user has a valid telemetry token configured
    When they run `logs-alerts create` with both shorthand flags (e.g. --threshold) and --body-file
    Then the CLI errors with exit code 3 explaining the two modes are mutually exclusive

  @user
  Scenario: List logs-alerts
    Given the user has a valid telemetry token configured
    When they run `logs-alerts list`
    Then they see a table with columns ID, NAME, CONDITION, CHECK/WINDOW, and PAUSED

  @user
  Scenario: Filter logs-alerts by exploration
    Given the user has a valid telemetry token configured
    When they run `logs-alerts list --exploration-id <eid>`
    Then they see only alerts attached to that exploration (filter applied client-side after full pagination)

  @user
  Scenario: Filter logs-alerts by escalation policy
    Given the user has a valid telemetry token configured
    When they run `logs-alerts list --policy-id <pid>`
    Then they see only alerts whose escalation_target references that policy (filter applied client-side after full pagination)

  @user
  Scenario: Get a single logs-alert by ID
    Given the user has a valid telemetry token configured
    When they run `logs-alerts get <id>`
    Then they see the alert's curated detail fields including exploration_id, condition, periods, and escalation target

  @user
  Scenario: Update a logs-alert via shorthand flags
    Given the user has a valid telemetry token configured
    And an existing logs-alert
    When they run `logs-alerts update <id>` with any of --name, --threshold, --window, --check-period, --query-period, --recovery, --policy-id, --paused, --no-paused
    Then the CLI sends a PATCH containing only the user-provided fields
    And they see the updated alert's details

  @user
  Scenario: Update a logs-alert via body file for fields outside the shorthand
    Given the user has a valid telemetry token configured
    When they run `logs-alerts update <id> --body-file path/to/update.json`
    Then the CLI sends a PATCH with the file contents as the body
    And the CLI uses --body-file for nested shapes (anomaly_rrcf alerts, custom escalation_target shapes, metadata objects)

  @user
  Scenario: Delete a logs-alert
    Given the user has a valid telemetry token configured
    When they run `logs-alerts delete <id> --yes`
    Then they see a confirmation message showing the deleted alert ID

  @user
  Scenario: logs-alerts delete rejects --force explicitly
    Given the user runs `logs-alerts delete <id> --yes --force`
    Then the CLI errors with exit code 3 explaining --force is only valid on commands with dependency checks (explorations delete, policies delete)
    And no DELETE is sent

  @user
  Scenario: Upsert on logs-alerts scopes the name match to the target exploration
    Given the user has a valid telemetry token configured
    And an existing alert named "INF-3017" attached to exploration A
    When they run `logs-alerts create --exploration-id B --name "INF-3017" ... --upsert`
    Then the CLI creates a new alert on exploration B (no match found within exploration B's alerts)
    And the alert on exploration A is unchanged

  @technical
  Scenario: Logs-alerts endpoints
    Given the telemetry client
    When it issues logs-alerts CRUD requests
    Then list sends GET to /api/v2/alerts (filters --exploration-id and --policy-id applied client-side after full pagination)
    And get sends GET to /api/v2/alerts/{id}
    And create sends POST to /api/v2/explorations/{exploration_id}/alerts
    And update sends PATCH to /api/v2/alerts/{id}
    And delete sends DELETE to /api/v2/alerts/{id}

  @technical
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

  @technical
  Scenario: Duration parsing
    Given a duration flag value (e.g., "5m", "30s", "2h")
    When the CLI parses it
    Then it uses time.ParseDuration
    And converts to integer seconds for API payloads
    And rejects values without units with an error that mentions the "max unit: h; use 24h, not 1d" guidance
    And rejects zero or negative values with a "strictly positive" error

  @technical
  Scenario: Logs-alerts table output
    Given a list of logs-alerts
    When rendered as a table
    Then columns are: ID, NAME, CONDITION, CHECK/WINDOW, PAUSED
    And CONDITION combines operator and value (e.g., ">= 3")
    And CHECK/WINDOW shows check_period in human-readable form (e.g., "5m")
