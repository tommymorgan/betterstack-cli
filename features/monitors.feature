Feature: Monitor Management
  As a user of the BetterStack CLI
  I want to query monitor data
  So that I can understand which monitors are generating incidents

  @user
  Scenario: List monitors
    Given the user has a valid BetterStack API token configured
    When they run the CLI to list monitors
    Then they see a table of monitors showing name, URL, status, and check frequency

  @user
  Scenario: Filter monitors by status
    Given the user has a valid BetterStack API token configured
    When they run the CLI to list monitors with a specific status filter
    Then they see only monitors matching that status

  @user
  Scenario: Get a single monitor's details
    Given the user has a valid BetterStack API token configured
    When they run the CLI to get a specific monitor by ID
    Then they see the full details of that monitor

  @technical
  Scenario: Monitors list endpoint
    Given a valid API token
    When the client calls ListMonitors
    Then it sends GET to https://uptime.betterstack.com/api/v2/monitors
    And it supports query parameters: url, pronounceable_name

  @technical
  Scenario: Monitors get endpoint
    Given a valid API token and a monitor ID
    When the client calls GetMonitor
    Then it sends GET to https://uptime.betterstack.com/api/v2/monitors/{id}
