Feature: Incident Management
  As a user of the BetterStack CLI
  I want to query incident data
  So that I can analyze alert patterns and tune thresholds

  @user
  Scenario: List recent incidents
    Given the user has a valid BetterStack API token configured
    When they run the CLI to list incidents
    Then they see a table of incidents showing name, started at, length, and status

  @user
  Scenario: Filter incidents by date range
    Given the user has a valid BetterStack API token configured
    When they run the CLI to list incidents with a start and end date
    Then they see only incidents within that date range

  @user
  Scenario: Filter incidents by monitor
    Given the user has a valid BetterStack API token configured
    When they run the CLI to list incidents for a specific monitor
    Then they see only incidents belonging to that monitor

  @user
  Scenario: Filter incidents by resolution status
    Given the user has a valid BetterStack API token configured
    When they run the CLI to list only resolved incidents
    Then they see only incidents that have been resolved

  @user
  Scenario: Get a single incident's details
    Given the user has a valid BetterStack API token configured
    When they run the CLI to get a specific incident by ID
    Then they see the full details of that incident

  @technical
  Scenario: Incidents list endpoint
    Given a valid API token
    When the client calls ListIncidents
    Then it sends GET to https://uptime.betterstack.com/api/v3/incidents
    And it supports query parameters: from, to, monitor_id, resolved, acknowledged

  @technical
  Scenario: Incidents get endpoint
    Given a valid API token and an incident ID
    When the client calls GetIncident
    Then it sends GET to https://uptime.betterstack.com/api/v3/incidents/{id}
