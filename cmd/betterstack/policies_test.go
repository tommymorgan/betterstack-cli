package betterstack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

func resetPolicyFlags(t *testing.T) {
	t.Helper()
	resetFlags(policiesCreateCmd, "body-file")
	resetBoolFlags(policiesCreateCmd, "upsert")
	resetFlags(policiesUpdateCmd, "body-file")
	resetFlags(policiesListCmd, "limit")
	resetBoolFlags(policiesDeleteCmd, "yes", "force")
	resetBoolFlags(rootCmd, "json")
	rootCmd.PersistentFlags().Lookup("json").Changed = false
}

func TestPolicies_ListShowsTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/policies" {
			fmt.Fprint(w, `{"data":[{"id":"42","attributes":{"name":"oncall","steps":[{"s":1},{"s":2}],"updated_at":"2026-04-20T10:00:00Z"}}]}`)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	resetPolicyFlags(t)

	out, _, err := executeCmd(t, "policies", "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"ID", "NAME", "STEPS", "UPDATED"} {
		if !strings.Contains(out, col) {
			t.Errorf("missing column %q:\n%s", col, out)
		}
	}
	if !strings.Contains(out, "oncall") {
		t.Errorf("missing name:\n%s", out)
	}
	if !strings.Contains(out, "2") {
		t.Errorf("missing step count:\n%s", out)
	}
}

func TestPolicies_GetShowsDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v3/policies/") {
			fmt.Fprint(w, `{"data":{"id":"42","attributes":{"name":"oncall","steps":[{"step_type":"call"}]}}}`)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	resetPolicyFlags(t)

	out, _, err := executeCmd(t, "policies", "get", "42")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "oncall") {
		t.Errorf("missing name:\n%s", out)
	}
}

func TestPolicies_CreateFromBodyFile(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "p.json")
	custom := []byte(`{"name":"paging","steps":[{"step_type":"call","recipient":"oncall"}]}`)
	_ = os.WriteFile(bodyPath, custom, 0o600)

	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v3/policies" {
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"1","attributes":{"name":"paging"}}}`)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	resetPolicyFlags(t)

	_, _, err := executeCmd(t, "policies", "create", "--body-file", bodyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != string(custom) {
		t.Errorf("body altered:\n got: %s\nwant: %s", gotBody, custom)
	}
}

func TestPolicies_CreateWithoutBodyFileErrors(t *testing.T) {
	t.Setenv("BETTERSTACK_API_TOKEN", "test")
	t.Setenv("BETTERSTACK_BASE_URL", "http://unused")
	resetPolicyFlags(t)

	_, _, err := executeCmd(t, "policies", "create")
	if err == nil {
		t.Fatal("expected error without --body-file")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
}

func TestPolicies_CreateRejectsUpsert(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "p.json")
	_ = os.WriteFile(bodyPath, []byte(`{}`), 0o600)

	t.Setenv("BETTERSTACK_API_TOKEN", "test")
	t.Setenv("BETTERSTACK_BASE_URL", "http://unused")
	resetPolicyFlags(t)

	_, _, err := executeCmd(t, "policies", "create", "--body-file", bodyPath, "--upsert")
	if err == nil {
		t.Fatal("expected error rejecting --upsert")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
}

func TestPolicies_UpdateFromBodyFile(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "p.json")
	custom := []byte(`{"name":"renamed"}`)
	_ = os.WriteFile(bodyPath, custom, 0o600)

	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			gotBody, _ = io.ReadAll(r.Body)
			fmt.Fprint(w, `{"data":{"id":"42","attributes":{"name":"renamed"}}}`)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "test")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	resetPolicyFlags(t)

	_, _, err := executeCmd(t, "policies", "update", "42", "--body-file", bodyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != string(custom) {
		t.Errorf("body altered:\n got: %s\nwant: %s", gotBody, custom)
	}
}

// Policy delete requires pagination of alerts on the telemetry host.
// The test server must respond on BOTH the telemetry path (for precheck)
// and the uptime path (for DELETE).
func TestPolicies_DeleteRefusesWhenReferencedByAlerts(t *testing.T) {
	// Combined server: serves both uptime and telemetry hosts.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" {
			fmt.Fprint(w, `{"data":[{"id":"99","attributes":{"name":"a","escalation_target":{"policy_id":42}}}]}`)
			return
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "uptime")
	t.Setenv("BETTERSTACK_TELEMETRY_TOKEN", "telemetry")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_QUIET", "1")
	resetPolicyFlags(t)

	_, stderr, err := executeCmd(t, "policies", "delete", "42", "--yes")
	if err == nil {
		t.Fatal("expected dependency-conflict error")
	}
	if errs.CodeOf(err) != errs.ExitDependencyConflict {
		t.Errorf("exit code = %d, want 4", errs.CodeOf(err))
	}
	if !strings.Contains(stderr, "at least 1") {
		t.Errorf("stderr missing 'at least 1': %s", stderr)
	}
	if !strings.Contains(stderr, "logs-alerts list --policy-id 42") {
		t.Errorf("stderr missing inspect command with literal ID: %s", stderr)
	}
}

func TestPolicies_DeleteJSONFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" {
			fmt.Fprint(w, `{"data":[{"id":"99","attributes":{"name":"a","escalation_target":{"policy_id":42}}}]}`)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "uptime")
	t.Setenv("BETTERSTACK_TELEMETRY_TOKEN", "telemetry")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_QUIET", "1")
	resetPolicyFlags(t)

	_, stderr, err := executeCmd(t, "policies", "delete", "42", "--yes", "--json")
	if err == nil {
		t.Fatal("expected error")
	}
	var parsed map[string]any
	if jerr := json.Unmarshal([]byte(stderr), &parsed); jerr != nil {
		t.Fatalf("stderr not JSON: %v\n%s", jerr, stderr)
	}
	errObj := parsed["error"].(map[string]any)
	if errObj["code"] != "DEPENDENCIES_EXIST" {
		t.Errorf("code = %v", errObj["code"])
	}
	if errObj["resource_type"] != "policy" {
		t.Errorf("resource_type = %v", errObj["resource_type"])
	}
}

func TestPolicies_DeleteProceedsWhenNoReferences(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" {
			fmt.Fprint(w, `{"data":[]}`)
			return
		}
		if r.Method == "DELETE" && r.URL.Path == "/api/v3/policies/42" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "uptime")
	t.Setenv("BETTERSTACK_TELEMETRY_TOKEN", "telemetry")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_QUIET", "1")
	resetPolicyFlags(t)

	_, _, err := executeCmd(t, "policies", "delete", "42", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !deleteCalled {
		t.Errorf("DELETE should have been called after empty precheck")
	}
}

func TestPolicies_DeleteForceSkipsPrecheck(t *testing.T) {
	var sawAlerts bool
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" {
			sawAlerts = true
		}
		if r.Method == "DELETE" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "uptime")
	t.Setenv("BETTERSTACK_TELEMETRY_TOKEN", "telemetry")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_QUIET", "1")
	resetPolicyFlags(t)

	_, _, err := executeCmd(t, "policies", "delete", "42", "--yes", "--force")
	if err != nil {
		t.Fatal(err)
	}
	if sawAlerts {
		t.Errorf("--force should skip precheck")
	}
	if !deleteCalled {
		t.Errorf("DELETE should have been called")
	}
}

func TestPolicies_DeletePostPrecheckRace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" {
			fmt.Fprint(w, `{"data":[]}`)
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"errors":["deps appeared"]}`)
		}
	}))
	defer srv.Close()

	t.Setenv("BETTERSTACK_API_TOKEN", "uptime")
	t.Setenv("BETTERSTACK_TELEMETRY_TOKEN", "telemetry")
	t.Setenv("BETTERSTACK_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", srv.URL)
	t.Setenv("BETTERSTACK_QUIET", "1")
	resetPolicyFlags(t)

	_, stderr, err := executeCmd(t, "policies", "delete", "42", "--yes")
	if err == nil {
		t.Fatal("expected race error")
	}
	if errs.CodeOf(err) != errs.ExitDependencyConflict {
		t.Errorf("exit code = %d, want 4", errs.CodeOf(err))
	}
	if !strings.Contains(stderr, "dependency_appeared_after_precheck:") {
		t.Errorf("stderr should contain prefix: %s", stderr)
	}
	if !strings.Contains(stderr, "logs-alerts list --policy-id 42") {
		t.Errorf("stderr should contain literal ID: %s", stderr)
	}
}
