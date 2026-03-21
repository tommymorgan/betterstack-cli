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
