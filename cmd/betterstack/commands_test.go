package betterstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v3/incidents" && r.Method == "GET":
			fmt.Fprint(w, `{"data":[{
				"id":"1",
				"attributes":{
					"name":"Test Incident",
					"cause":"Threshold crossed",
					"started_at":"2026-03-20T10:00:00Z",
					"resolved_at":"2026-03-20T10:01:00Z",
					"status":"Resolved"
				}
			}]}`)
		case strings.HasPrefix(r.URL.Path, "/api/v3/incidents/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/v3/incidents/")
			fmt.Fprintf(w, `{"data":{"id":"%s","attributes":{"name":"Incident %s","cause":"error","started_at":"2026-03-20T10:00:00Z","resolved_at":null,"status":"Started"}}}`, id, id)
		case r.URL.Path == "/api/v2/monitors" && r.Method == "GET":
			fmt.Fprint(w, `{"data":[{
				"id":"99",
				"attributes":{
					"pronounceable_name":"My Monitor",
					"url":"https://example.com",
					"status":"up",
					"check_frequency":60
				}
			}]}`)
		case strings.HasPrefix(r.URL.Path, "/api/v2/monitors/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/v2/monitors/")
			fmt.Fprintf(w, `{"data":{"id":"%s","attributes":{"pronounceable_name":"Monitor %s","url":"https://example.com","status":"up","check_frequency":30}}}`, id, id)
		case r.URL.Path == "/api/v2/sources" && r.Method == "GET":
			fmt.Fprint(w, `{"data":[{
				"id":"10",
				"attributes":{
					"name":"dev-cloudwatch",
					"type":"amazon_cloudwatch",
					"webhook_url":"https://uptime.betterstack.com/hook/abc123"
				}
			}]}`)
		case r.URL.Path == "/api/v2/sources" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{
				"id":"20",
				"attributes":{
					"name":"new-source",
					"type":"datadog",
					"webhook_url":"https://uptime.betterstack.com/hook/new123",
					"created_at":"2026-04-10T14:30:00Z"
				}
			}}`)
		case strings.HasPrefix(r.URL.Path, "/api/v2/sources/") && r.Method == "GET":
			id := strings.TrimPrefix(r.URL.Path, "/api/v2/sources/")
			fmt.Fprintf(w, `{"data":{"id":"%s","attributes":{"name":"Source %s","type":"amazon_cloudwatch","webhook_url":"https://uptime.betterstack.com/hook/xyz","created_at":"2026-04-10T14:30:00Z"}}}`, id, id)
		case strings.HasPrefix(r.URL.Path, "/api/v2/sources/") && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(404)
		}
	}))
}

func executeCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)
	// Reset persistent flags to avoid state leaking between tests
	rootCmd.PersistentFlags().Set("json", "false")
	// Reset subcommand flags that may leak between tests
	sourcesDeleteCmd.Flags().Set("yes", "false")
	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestVersion_ShowsVersionNumber(t *testing.T) {
	Version = "1.2.3"
	defer func() { Version = "dev" }()

	out, _, err := executeCommand(t, "version")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "1.2.3") {
		t.Errorf("expected version 1.2.3, got: %s", out)
	}
}

func TestHelp_ShowsAvailableCommands(t *testing.T) {
	out, _, err := executeCommand(t, "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, cmd := range []string{"incidents", "monitors", "version"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("help output missing command %q", cmd)
		}
	}
}

func TestIncidentsList_ShowsTableOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "incidents", "list")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "Test Incident") {
		t.Errorf("expected incident name in output:\n%s", out)
	}
	if !strings.Contains(out, "Threshold crossed") {
		t.Errorf("expected cause in output:\n%s", out)
	}
	if !strings.Contains(out, "Resolved") {
		t.Errorf("expected status in output:\n%s", out)
	}
}

func TestIncidentsList_JSONOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "incidents", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput:\n%s", err, out)
	}
}

func TestIncidentsGet_ShowsIncidentDetails(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "incidents", "get", "42")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "42") {
		t.Errorf("expected incident ID in output:\n%s", out)
	}
}

func TestIncidentsGet_JSONOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "incidents", "get", "42", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestMonitorsList_ShowsTableOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "monitors", "list")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "My Monitor") {
		t.Errorf("expected monitor name in output:\n%s", out)
	}
	if !strings.Contains(out, "https://example.com") {
		t.Errorf("expected URL in output:\n%s", out)
	}
}

func TestMonitorsList_JSONOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "monitors", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestMonitorsGet_ShowsMonitorDetails(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "monitors", "get", "99")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "99") {
		t.Errorf("expected monitor ID in output:\n%s", out)
	}
}

func TestNoToken_ShowsHelpfulError(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "")
	t.Setenv("HOME", t.TempDir())

	_, _, err := executeCommand(t, "incidents", "list")
	if err == nil {
		t.Fatal("expected error when no token configured")
	}

	if !strings.Contains(err.Error(), "BETTERSTACK_API_TOKEN") {
		t.Errorf("error should mention env var, got: %s", err.Error())
	}
}

func TestIncidentsGet_RequiresIDArgument(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")

	_, _, err := executeCommand(t, "incidents", "get")
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestMonitorsGet_RequiresIDArgument(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")

	_, _, err := executeCommand(t, "monitors", "get")
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestAPIError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"errors":["Invalid token"]}`)
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "bad-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	_, _, err := executeCommand(t, "incidents", "list")
	if err == nil {
		t.Fatal("expected error for 401")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain status code, got: %s", err.Error())
	}
}

func TestEmptyResults_ShowsEmptyTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "incidents", "list")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (header only), got %d lines:\n%s", len(lines), out)
	}
}

func TestEmptyResults_ShowsEmptyJSONArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "incidents", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.TrimSpace(out), "[]") {
		t.Errorf("expected empty JSON array, got: %s", out)
	}
}

func TestSourcesList_ShowsTableWithIDNameTypeWebhookURL(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "sources", "list")
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"10", "dev-cloudwatch", "amazon_cloudwatch", "https://uptime.betterstack.com/hook/abc123"} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output:\n%s", expected, out)
		}
	}
}

func TestSourcesList_JSONOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "sources", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput:\n%s", err, out)
	}
}

func TestSourcesGet_ShowsSourceDetailsWithWebhookURL(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "sources", "get", "42")
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"42", "Source 42", "amazon_cloudwatch", "https://uptime.betterstack.com/hook/xyz", "2026-04-10 14:30:00 UTC"} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output:\n%s", expected, out)
		}
	}
}

func TestSourcesGet_JSONOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "sources", "get", "42", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput:\n%s", err, out)
	}
}

func TestSourcesCreate_ShowsCreatedSourceWithWebhookURL(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "sources", "create", "--name", "new-source", "--type", "datadog")
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"20", "new-source", "datadog", "https://uptime.betterstack.com/hook/new123"} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output:\n%s", expected, out)
		}
	}
}

func TestSourcesCreate_FailsWithoutRequiredFlags(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")

	_, _, err := executeCommand(t, "sources", "create")
	if err == nil {
		t.Fatal("expected error when required flags not provided")
	}

	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention missing flag, got: %s", err.Error())
	}
}

func TestSourcesCreate_FailsWithoutTypeFlag(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")

	_, _, err := executeCommand(t, "sources", "create", "--name", "test")
	if err == nil {
		t.Fatal("expected error when --type flag not provided")
	}

	if !strings.Contains(err.Error(), "type") {
		t.Errorf("error should mention missing --type flag, got: %s", err.Error())
	}
}

func TestSourcesDelete_ShowsConfirmationMessage(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "sources", "delete", "42", "--yes")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "Deleted source 42") {
		t.Errorf("expected confirmation message, got:\n%s", out)
	}
}

func TestSourcesDelete_JSONOutput(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)

	out, _, err := executeCommand(t, "sources", "delete", "42", "--yes", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", result["deleted"])
	}
	if result["id"] != "42" {
		t.Errorf("expected id=42, got %v", result["id"])
	}
}

func TestSourcesDelete_FailsWithoutYesFlag(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")

	_, _, err := executeCommand(t, "sources", "delete", "42")
	if err == nil {
		t.Fatal("expected error when --yes flag not provided")
	}

	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("error should mention --yes flag, got: %s", err.Error())
	}
}

func TestSourcesGet_RequiresIDArgument(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")

	_, _, err := executeCommand(t, "sources", "get")
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestSourcesDelete_RequiresIDArgument(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test-token")

	_, _, err := executeCommand(t, "sources", "delete", "--yes")
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

// Test help flags separately to avoid cobra state contamination.
// This must be the LAST test alphabetically to avoid --help flag leaking.
func TestZZ_IncidentsListHelp_ShowsFlags(t *testing.T) {
	out, _, err := executeCommand(t, "incidents", "list", "--help")
	if err != nil {
		t.Fatal(err)
	}

	for _, flag := range []string{"--from", "--to", "--monitor-id", "--resolved", "--limit", "--json"} {
		if !strings.Contains(out, flag) {
			t.Errorf("help missing flag %q in:\n%s", flag, out)
		}
	}
}
