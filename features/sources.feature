Feature: Source Management
  As a user of the BetterStack CLI
  I want to manage sources (webhook integrations)
  So that I can automate the creation of monitoring integrations without switching to the web UI

  @user
  Scenario: List all sources
    Given the user has a valid BetterStack API token configured
    When they run the CLI to list sources
    Then they see a list of sources showing ID, name, type, and webhook URL

  @user
  Scenario: Get a source's details including webhook URL
    Given the user has a valid BetterStack API token configured
    When they run the CLI to get a specific source by ID
    Then they see the source's ID, name, type, webhook URL, and creation date

  @user
  Scenario: Create a source and receive its webhook URL
    Given the user has a valid BetterStack API token configured
    When they run the CLI to create a source with a name and type
    Then they see the created source's ID, name, type, and webhook URL

  @user
  Scenario: Delete a source
    Given the user has a valid BetterStack API token configured
    When they run the CLI to delete a specific source by ID
    Then the CLI requires a --yes flag to confirm the destructive operation
    And they see a confirmation message showing the deleted source ID

  @user
  Scenario: See a clear error when creating a source without required fields
    Given the user has a valid BetterStack API token configured
    When they run the create command without providing a name or type
    Then they see an error message indicating the missing required fields
    And the CLI exits with a non-zero status code

  @user
  Scenario: See a clear error when deleting without confirmation flag
    Given the user has a valid BetterStack API token configured
    When they run the CLI to delete a source by ID without the --yes flag
    Then they see an error message explaining that --yes is required for destructive operations
    And the CLI exits with a non-zero status code
    And the source is not deleted

  @technical
  Scenario: HTTP client supports GET, POST, and DELETE methods
    Given a client configured with a valid token
    When GET, POST, and DELETE requests are made
    Then each uses the correct HTTP method
    And POST requests send Content-Type application/json
    And DELETE requests send no body

  @technical
  Scenario: Sources list endpoint
    Given a valid API token
    When the client calls ListSources
    Then it sends GET to /api/v2/sources on the configured base URL
    And it auto-paginates using the existing fetchAll helper

  @technical
  Scenario: Sources get endpoint
    Given a valid API token and a source ID
    When the client calls GetSource
    Then it sends GET to /api/v2/sources/{id} on the configured base URL
    And it returns the source data including ID, name, type, and webhook URL

  @technical
  Scenario: Sources create endpoint
    Given a valid API token, a name, and a source type
    When the client calls CreateSource
    Then it sends POST to /api/v2/sources on the configured base URL
    And the request body is JSON with name and type fields
    And the API returns 201 Created on success
    And it returns the created source data including the webhook URL

  @technical
  Scenario: Sources delete endpoint
    Given a valid API token and a source ID
    When the client calls DeleteSource
    Then it sends DELETE to /api/v2/sources/{id} on the configured base URL
    And it succeeds when the API returns 204 No Content

  @technical
  Scenario: Sources table output
    Given a list of sources
    When rendered as a table
    Then columns are: ID, NAME, TYPE, WEBHOOK URL
    And columns are aligned with tabwriter

  @technical
  Scenario: Sources detail output
    Given a single source
    When rendered as a detail view
    Then it shows key-value pairs: ID, Name, Type, Webhook URL, Created At
    And timestamps are displayed in UTC

  @technical
  Scenario: Delete confirmation output
    Given a source has been deleted
    When rendered as text
    Then it prints "Deleted source <id>"
    And with JSON flag it outputs {"deleted": true, "id": "<id>"}
