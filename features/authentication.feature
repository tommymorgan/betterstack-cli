Feature: Authentication
  As a user of the BetterStack CLI
  I want flexible authentication options
  So that I can securely configure API access

  @user
  Scenario: Authenticate via environment variable
    Given the user has set the BETTERSTACK_API_TOKEN environment variable
    When they run any CLI command
    Then the CLI authenticates using that token

  @user
  Scenario: Authenticate via config file
    Given the user has a config file with their API token
    And no environment variable is set
    When they run any CLI command
    Then the CLI authenticates using the config file token

  @user
  Scenario: See a helpful error when no token is configured
    Given the user has no API token configured anywhere
    When they run any CLI command
    Then they see an error message explaining how to configure authentication
    And the CLI exits with a non-zero status code

  @technical
  Scenario: Token resolution order
    Given both an environment variable and config file contain tokens
    When the config package resolves the token
    Then the environment variable token takes precedence

  @technical
  Scenario: Config file location
    Given the user has created a config file
    When the config package looks for it
    Then it checks ~/.config/betterstack/config.yaml
    And the file contains an api_token field

  @technical
  Scenario: Config file has restricted permissions
    Given the CLI creates or reads a config file
    When the config file is written
    Then its file permissions are set to 0600
    And the CLI warns to stderr if an existing config file has permissions more open than 0600

  @technical
  Scenario: HTTP client sends correct auth headers
    Given a configured API token
    When the client makes a request to the BetterStack API
    Then it sends an Authorization header with "Bearer <token>"
    And it sends an Accept header with "application/json"
    And it sends a User-Agent header identifying the CLI name and version
