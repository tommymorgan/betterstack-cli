Feature: Authentication
  As a user of the BetterStack CLI
  I want flexible authentication options for both uptime and telemetry hosts
  So that I can securely configure API access with least-privilege team-scoped tokens

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

  @user
  Scenario: Telemetry commands prefer BETTERSTACK_TELEMETRY_TOKEN
    Given the user has set both BETTERSTACK_API_TOKEN and BETTERSTACK_TELEMETRY_TOKEN
    When they run a telemetry command (logs-alerts, explorations, or logs-sources)
    Then the CLI authenticates using BETTERSTACK_TELEMETRY_TOKEN

  @user
  Scenario: Telemetry commands fall back to BETTERSTACK_API_TOKEN when telemetry token is unset
    Given the user has set BETTERSTACK_API_TOKEN but not BETTERSTACK_TELEMETRY_TOKEN
    When they run a telemetry command on an interactive (TTY) session
    Then the CLI authenticates using BETTERSTACK_API_TOKEN
    And stderr emits a one-line notice once per process: "note: BETTERSTACK_TELEMETRY_TOKEN unset; sent BETTERSTACK_API_TOKEN to telemetry.betterstack.com (works only for global tokens)"

  @user
  Scenario: Telemetry fallback notice is suppressed on non-TTY and when --quiet / BETTERSTACK_QUIET is set
    Given the user has set BETTERSTACK_API_TOKEN but not BETTERSTACK_TELEMETRY_TOKEN
    When the CLI runs in a non-TTY context (CI, piped) OR --quiet is passed OR BETTERSTACK_QUIET is set to a truthy value
    Then the fallback notice is not emitted
    And the CLI still authenticates using BETTERSTACK_API_TOKEN

  @user
  Scenario: Uptime commands never use the telemetry token
    Given the user has set both BETTERSTACK_API_TOKEN and BETTERSTACK_TELEMETRY_TOKEN
    When they run an uptime command (incidents, monitors, integrations, policies)
    Then the CLI authenticates using BETTERSTACK_API_TOKEN only

  @user
  Scenario: Uptime command fails explicitly when only the telemetry token is set
    Given the user has BETTERSTACK_TELEMETRY_TOKEN set and BETTERSTACK_API_TOKEN unset (no config file)
    When they run an uptime command
    Then the CLI errors before issuing any HTTP request
    And the error names BETTERSTACK_API_TOKEN as the env var required for uptime commands
    And the CLI exits with status code 2 (auth failure)

  @user
  Scenario: Auth error on telemetry command names which env vars are currently set
    Given the user has BETTERSTACK_API_TOKEN set but it is team-scoped to uptime only
    And BETTERSTACK_TELEMETRY_TOKEN is unset
    When they run a telemetry command and telemetry.betterstack.com returns 401 or 403
    Then the CLI prints a single actionable error to stderr naming the HTTP status code
    And the error lists which token env vars are currently set and which are unset
    And the error suggests setting BETTERSTACK_TELEMETRY_TOKEN or using a global BETTERSTACK_API_TOKEN
    And the CLI exits with status code 2 (auth failure)

  @user
  Scenario: Telemetry token env var takes precedence over config file
    Given ~/.config/betterstack/config.yaml contains telemetry_api_token: "from-file"
    And BETTERSTACK_TELEMETRY_TOKEN is set to "from-env"
    When a telemetry command runs
    Then the CLI uses "from-env"

  @user
  Scenario: Telemetry token resolved from config file when env is unset
    Given ~/.config/betterstack/config.yaml contains telemetry_api_token
    And BETTERSTACK_TELEMETRY_TOKEN is unset
    When a telemetry command runs
    Then the CLI uses the config file's telemetry_api_token
    And the existing 0600 permissions warning applies (matching current api_token handling)

  @user
  Scenario: Telemetry token resolved from env when no config file has telemetry_api_token
    Given the config file does not contain telemetry_api_token
    And BETTERSTACK_TELEMETRY_TOKEN is set
    When a telemetry command runs
    Then the CLI uses the env var value

  @technical
  Scenario: Telemetry token resolution
    Given config.ResolveTelemetryToken is called
    When BETTERSTACK_TELEMETRY_TOKEN is set (env)
    Then it returns that value
    When BETTERSTACK_TELEMETRY_TOKEN is unset and ~/.config/betterstack/config.yaml contains telemetry_api_token
    Then it returns the config file value
    When neither env nor config has a telemetry_api_token
    Then it falls back to the uptime token resolution (BETTERSTACK_API_TOKEN then config's api_token)
    When no token can be resolved
    Then it returns an error whose message names all env vars and config fields, and explains the fallback rule

  @technical
  Scenario: Telemetry base URL
    Given the telemetry client is constructed
    When it issues a request
    Then the base URL is https://telemetry.betterstack.com
    And a BETTERSTACK_TELEMETRY_BASE_URL env var overrides the base URL (mirroring BETTERSTACK_BASE_URL for uptime)

  @technical
  Scenario: Both clients honor Retry-After on 429 identically
    Given a client (uptime or telemetry) receives HTTP 429 with a Retry-After header
    When the client processes the response
    Then it waits for the duration specified in Retry-After and retries up to 3 times
    And the observable retry behavior is identical across uptime and telemetry clients

  @technical
  Scenario: Both clients apply a 30s request timeout
    Given a client (uptime or telemetry) issues a request
    When the request takes longer than 30 seconds
    Then the client cancels and returns a timeout error (identical across clients)

  @technical
  Scenario: Both clients set Bearer auth and emit identical error types on 4xx/5xx
    Given a client (uptime or telemetry) issues a request
    Then it sets an Authorization header with "Bearer <token>"
    And on 4xx/5xx responses it returns an error whose Go type and field shape is identical across clients
