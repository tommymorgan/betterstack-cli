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

  @user
  Scenario: dependency_appeared_after_precheck prefix on delete races
    Given a delete precheck found zero dependents
    And a concurrent actor creates a dependent between the precheck and the DELETE
    When the DELETE returns 409 or 422
    Then stderr contains the prefix `error: dependency_appeared_after_precheck:` followed by the inspect command with the literal resource ID
    And the CLI exits with status code 4 (dependency conflict)
    And the CLI does not retry the DELETE

  @user
  Scenario: upsert_match_deleted_after_precheck prefix on upsert races
    Given --upsert found exactly one match and the CLI issues a PATCH
    When the PATCH returns 404 (match was deleted between list and PATCH)
    Then stderr contains the prefix `error: upsert_match_deleted_after_precheck:`
    And the CLI exits with status code 5 (upstream API error)
    And the CLI does not retry
