Feature: Error Handling
  As a user of the BetterStack CLI
  I want clear error messages
  So that I can diagnose and fix issues

  @user
  Scenario: See a clear error for API failures
    Given the user has a valid API token configured
    When the BetterStack API returns an error
    Then the error message and HTTP status are printed to stderr
    And the CLI exits with a non-zero status code

  @user
  Scenario: Handle rate limiting gracefully
    Given the user has a valid API token configured
    When the BetterStack API returns a rate limit response
    Then the CLI waits and retries automatically
    And the user eventually receives their results

  @technical
  Scenario: Errors go to stderr
    Given any error occurs during execution
    When the CLI reports the error
    Then the error message is written to stderr
    And the exit code is 1

  @technical
  Scenario: HTTP client retries on 429
    Given the BetterStack API returns HTTP 429 with a Retry-After header
    When the client receives the response
    Then it waits for the duration specified in Retry-After
    And it retries the request up to 3 times
    And it returns an error if all retries are exhausted

  @technical
  Scenario: HTTP client enforces request timeouts
    Given the client makes a request to the BetterStack API
    When the request takes longer than 30 seconds
    Then the client cancels the request
    And returns a timeout error
