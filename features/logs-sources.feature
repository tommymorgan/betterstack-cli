Feature: Logs Sources
  As a user of the BetterStack CLI
  I want to list and inspect log sources
  So that I can find the source IDs I need for exploration shorthand

  @user
  Scenario: List log sources
    Given the user has a valid telemetry token configured
    When they run `logs-sources list`
    Then they see a table with columns ID, NAME, PLATFORM, and TEAM

  @user
  Scenario: Get a single log source by ID
    Given the user has a valid telemetry token configured
    When they run `logs-sources get <id>`
    Then they see the source's curated detail fields (name, platform, retention, team name)

  @technical
  Scenario: Logs-sources endpoints
    Given the telemetry client
    When it issues source read requests
    Then list sends GET to /api/v2/sources
    And get sends GET to /api/v2/sources/{id}

  @technical
  Scenario: Logs-sources table output
    Given a list of log sources
    When rendered as a table
    Then columns are: ID, NAME, PLATFORM, TEAM
