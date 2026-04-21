Feature: Escalation Policies
  As a user of the BetterStack CLI
  I want to manage escalation policies
  So that logs-alerts (and uptime alerts) route through predictable notification chains

  @user
  Scenario: List escalation policies
    Given the user has a valid uptime token configured
    When they run `policies list`
    Then they see a table with columns ID, NAME, STEPS, and UPDATED

  @user
  Scenario: Get a single policy by ID
    Given the user has a valid uptime token configured
    When they run `policies get <id>`
    Then they see the policy's curated detail fields including step configuration

  @user
  Scenario: Create a policy from a JSON body file
    Given the user has a valid uptime token configured
    When they run `policies create --body-file path/to/policy.json`
    Then the CLI sends the file contents as the POST body

  @user
  Scenario: Policies create does not support --upsert
    Given the user runs `policies create --body-file path.json --upsert`
    Then the CLI errors with exit code 3 explaining --upsert is not supported for policies (policies require --body-file and the "shorthand OR body" rule rules out body-file upsert)

  @user
  Scenario: Update a policy from a JSON body file
    Given the user has a valid uptime token configured
    When they run `policies update <id> --body-file path/to/update.json`
    Then the CLI sends a PATCH with the file contents as the body

  @user
  Scenario: Delete a policy with no dependent alerts
    Given the user has a valid uptime token configured
    And no logs-alerts reference the policy
    When they run `policies delete <id> --yes`
    Then they see a confirmation message showing the deleted policy ID

  @user
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

  @user
  Scenario: Force-delete a policy with dependent alerts
    Given the user has a valid uptime token configured
    And a policy with dependent alerts
    When they run `policies delete <id> --yes --force`
    Then the CLI skips the dependency check and issues the DELETE

  @user
  Scenario: DELETE fails after policy refcount precheck passed
    Given the policy precheck found zero dependent alerts
    And a concurrent actor creates a dependent alert before the DELETE
    When the DELETE returns 409 or 422
    Then the CLI errors with stderr prefix `error: dependency_appeared_after_precheck:` followed by `logs-alerts list --policy-id <literal-id>`
    And the CLI exits with code 4

  @technical
  Scenario: Escalation policies endpoints
    Given the uptime client
    When it issues policies CRUD requests
    Then list sends GET to /api/v3/policies
    And get sends GET to /api/v3/policies/{id}
    And create sends POST to /api/v3/policies
    And update sends PATCH to /api/v3/policies/{id}
    And delete sends DELETE to /api/v3/policies/{id}

  @technical
  Scenario: Pre-delete refcount check on policies delete (paginate until first match)
    Given the user ran `policies delete <id> --yes` without --force
    When the CLI processes the command
    Then it paginates GET /api/v2/alerts, filtering client-side for alerts whose escalation_target.policy_id equals <id>
    And on the first matching alert, it short-circuits (does not continue paginating) and returns the DEPENDENCIES_EXIST error with count wording "at least 1"
    And if pagination completes with zero matches, it proceeds with the DELETE

  @technical
  Scenario: Refcount check is best-effort, not atomic
    Given the refcount check passes and the CLI issues the DELETE
    When a concurrent actor creates a dependent between the GET and the DELETE
    Then the DELETE may either orphan the dependent (if the server allows) or return 409/422 (if the server enforces)
    And this non-atomicity is documented in --help for both `explorations delete` and `policies delete`

  @technical
  Scenario: Policies table output
    Given a list of escalation policies
    When rendered as a table
    Then columns are: ID, NAME, STEPS, UPDATED
    And STEPS shows the count of steps in the policy
