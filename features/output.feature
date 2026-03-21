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
