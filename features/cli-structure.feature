Feature: CLI Structure
  As a user or AI agent
  I want a well-structured CLI with discoverable commands
  So that I can find and use the right commands

  @user
  Scenario: Discover available commands via help
    Given the user has installed the CLI
    When they run the CLI with a help flag or no arguments
    Then they see a list of available commands with descriptions
    And each subcommand also supports a help flag showing its flags and usage

  @user
  Scenario: Check the CLI version
    Given the user has installed the CLI
    When they run the version command
    Then they see the CLI version number

  @user
  Scenario: Install the CLI
    Given the user has Go installed
    When they run go install for the CLI module
    Then the betterstack binary is available in their PATH

  @technical
  Scenario: Go module structure
    Given a new Go module at github.com/tommymorgan/betterstack-cli
    When the project is initialized
    Then it has cmd/ and internal/ directories
    And internal/ contains client/, output/, and config/ packages
    And it uses cobra for CLI subcommand parsing
    And it uses gopkg.in/yaml.v3 for config file parsing
    And it targets Go 1.26
    And it is installable via go install

  @technical
  Scenario: HTTP client auto-paginates
    Given the BetterStack API returns paginated results with next page links
    When the client fetches a list endpoint
    Then it follows pagination links until all pages are fetched
    And it combines results from all pages into a single slice

  @technical
  Scenario: HTTP client respects limit flag
    Given the user specified a limit of N results
    When the client fetches a list endpoint
    Then it stops fetching pages once N results are collected
    And it returns at most N results

  @technical
  Scenario: Unknown JSON fields are ignored
    Given the BetterStack API returns fields not present in the Go structs
    When the client deserializes the response
    Then unknown fields are silently ignored
    And known fields are correctly populated
