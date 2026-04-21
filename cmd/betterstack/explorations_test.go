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
	"sync"
	"testing"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

// telemetryTestServer records the last request method/path/body and returns
// configurable responses.
type telemetryTestServer struct {
	mu         sync.Mutex
	requests   []recordedRequest
	handler    http.HandlerFunc
	srv        *httptest.Server
}

type recordedRequest struct {
	method string
	path   string
	query  string
	body   []byte
}

func newTelemetryTestServer(t *testing.T, h http.HandlerFunc) *telemetryTestServer {
	t.Helper()
	s := &telemetryTestServer{handler: h}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.requests = append(s.requests, recordedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			body:   body,
		})
		s.mu.Unlock()
		// Reset body so the handler can still read
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		s.handler(w, r)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *telemetryTestServer) URL() string { return s.srv.URL }

func (s *telemetryTestServer) Requests() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedRequest, len(s.requests))
	copy(out, s.requests)
	return out
}

// setupTelemetryEnv configures env vars so telemetry-bound commands hit the
// given test server.
func setupTelemetryEnv(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("BETTERSTACK_TELEMETRY_TOKEN", "test-telemetry")
	t.Setenv("BETTERSTACK_API_TOKEN", "test-uptime")
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", baseURL)
	t.Setenv("BETTERSTACK_BASE_URL", baseURL)
	t.Setenv("BETTERSTACK_QUIET", "1")
}

func resetExplorationFlags(t *testing.T) {
	t.Helper()
	resetFlags(explorationsCreateCmd, "body-file", "source-id", "pattern", "name")
	resetBoolFlags(explorationsCreateCmd, "upsert")
	resetFlags(explorationsUpdateCmd, "body-file", "source-id", "pattern", "name")
	resetBoolFlags(explorationsDeleteCmd, "yes", "force")
	resetFlags(explorationsListCmd, "limit")
	resetBoolFlags(rootCmd, "json")
	rootCmd.PersistentFlags().Lookup("json").Changed = false
	if q := rootCmd.PersistentFlags().Lookup("quiet"); q != nil {
		_ = rootCmd.PersistentFlags().Set("quiet", "false")
		q.Changed = false
	}
}

func executeCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	stdoutBuf := new(strings.Builder)
	stderrBuf := new(strings.Builder)
	rootCmd.SetOut(stdoutBuf)
	rootCmd.SetErr(stderrBuf)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestExplorations_ListShowsTableWithColumns(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"5","attributes":{"name":"INF-3017","date_range_from":"now-1h","date_range_to":"now","updated_at":"2026-04-20T10:00:00Z"}}]}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	out, _, err := executeCmd(t, "explorations", "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"ID", "NAME", "DATE RANGE", "UPDATED"} {
		if !strings.Contains(out, col) {
			t.Errorf("missing column %q in output:\n%s", col, out)
		}
	}
	if !strings.Contains(out, "INF-3017") {
		t.Errorf("missing exploration name:\n%s", out)
	}
}

func TestExplorations_GetShowsDetail(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"id":"5","attributes":{"name":"INF-3017","chart":{"chart_type":"number_chart"},"date_range_from":"now-1h","date_range_to":"now","queries":[{"query_type":"tail_query","where_condition":"message CONTAINS \"err\"","source_variable":"42"}],"updated_at":"2026-04-20T10:00:00Z"}}}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	out, _, err := executeCmd(t, "explorations", "get", "5")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"INF-3017", "number_chart", "now-1h", "tail_query", "42"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestExplorations_CreateShorthandPostsCountMatchingPayload(t *testing.T) {
	var gotBody []byte
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			gotBody = body
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"new","attributes":{"name":"INF-3017"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	out, _, err := executeCmd(t, "explorations", "create",
		"--source-id", "123456",
		"--pattern", "AADSTS7000215",
		"--name", "INF-3017",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "INF-3017") {
		t.Errorf("expected created name in output, got:\n%s", out)
	}

	var parsed map[string]any
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatalf("body not JSON: %v (%s)", err, gotBody)
	}
	if parsed["name"] != "INF-3017" {
		t.Errorf("body name = %v", parsed["name"])
	}
	chart := parsed["chart"].(map[string]any)
	if chart["chart_type"] != "number_chart" {
		t.Errorf("chart_type = %v", chart["chart_type"])
	}
	if parsed["date_range_from"] != "now-1h" {
		t.Errorf("date_range_from = %v", parsed["date_range_from"])
	}
}

func TestExplorations_CreateBodyFileSendsContentsVerbatim(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.json")
	customBody := []byte(`{"custom":"payload","chart":{"chart_type":"line_chart"}}`)
	if err := os.WriteFile(bodyPath, customBody, 0o600); err != nil {
		t.Fatal(err)
	}

	var gotBody []byte
	var gotCT string
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			gotBody = body
			gotCT = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"x","attributes":{"name":"custom"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "create", "--body-file", bodyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != string(customBody) {
		t.Errorf("body altered:\n got: %s\nwant: %s", gotBody, customBody)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
}

func TestExplorations_CreateShorthandAndBodyFileAreMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.json")
	_ = os.WriteFile(bodyPath, []byte(`{}`), 0o600)

	setupTelemetryEnv(t, "http://unused")
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "create",
		"--body-file", bodyPath,
		"--pattern", "x",
	)
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error: %s", err.Error())
	}
}

func TestExplorations_CreateBodyFileMissingPathErrors(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "create", "--body-file", "/definitely/missing.json")
	if err == nil {
		t.Fatal("expected error")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
}

func TestExplorations_CreateBodyFileInvalidJSONErrors(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.txt")
	_ = os.WriteFile(bodyPath, []byte("not json"), 0o600)

	setupTelemetryEnv(t, "http://unused")
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "create", "--body-file", bodyPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "valid JSON") {
		t.Errorf("error: %s", err.Error())
	}
}

func TestExplorations_CreateBodyFileEmptyErrors(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "empty.json")
	_ = os.WriteFile(bodyPath, nil, 0o600)

	setupTelemetryEnv(t, "http://unused")
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "create", "--body-file", bodyPath)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestExplorations_UpdateShorthandPatches(t *testing.T) {
	var gotBody []byte
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			body, _ := io.ReadAll(r.Body)
			gotBody = body
			fmt.Fprint(w, `{"data":{"id":"5","attributes":{"name":"renamed"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "update", "5", "--name", "renamed")
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(gotBody, &parsed)
	if parsed["name"] != "renamed" {
		t.Errorf("patch body name = %v", parsed["name"])
	}
	if _, has := parsed["queries"]; has {
		t.Errorf("patch should not include queries when only name was provided")
	}
}

func TestExplorations_DeleteRequiresYes(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "delete", "5")
	if err == nil {
		t.Fatal("expected error when --yes missing")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
}

func TestExplorations_DeleteEmitsDependencyConflictWhenAlertsExist(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/alerts") {
			fmt.Fprint(w, `{"data":[{"id":"99"}]}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, stderr, err := executeCmd(t, "explorations", "delete", "5", "--yes")
	if err == nil {
		t.Fatal("expected dependency-conflict error")
	}
	if errs.CodeOf(err) != errs.ExitDependencyConflict {
		t.Errorf("exit code = %d, want 4", errs.CodeOf(err))
	}
	if !strings.Contains(stderr, "at least 1") {
		t.Errorf("stderr missing 'at least 1': %s", stderr)
	}
	if !strings.Contains(stderr, "logs-alerts list --exploration-id 5") {
		t.Errorf("stderr missing inspect command with literal ID: %s", stderr)
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("stderr should mention --force: %s", stderr)
	}
}

func TestExplorations_DeleteDependencyConflictJSONFormat(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/alerts") {
			fmt.Fprint(w, `{"data":[{"id":"99"}]}`)
			return
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, stderr, err := executeCmd(t, "explorations", "delete", "5", "--yes", "--json")
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
	if errObj["count"] != "at least 1" {
		t.Errorf("count = %v", errObj["count"])
	}
	if errObj["inspect_command"] != "logs-alerts list --exploration-id 5" {
		t.Errorf("inspect_command = %v", errObj["inspect_command"])
	}
	if errObj["resource_type"] != "exploration" {
		t.Errorf("resource_type = %v", errObj["resource_type"])
	}
	if errObj["resource_id"] != "5" {
		t.Errorf("resource_id = %v", errObj["resource_id"])
	}
}

func TestExplorations_DeleteForceSkipsPrecheck(t *testing.T) {
	var sawAlertsGET bool
	var sawDelete bool
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/alerts") {
			sawAlertsGET = true
		}
		if r.Method == "DELETE" {
			sawDelete = true
			w.WriteHeader(http.StatusNoContent)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "delete", "5", "--yes", "--force")
	if err != nil {
		t.Fatal(err)
	}
	if sawAlertsGET {
		t.Errorf("precheck should be skipped with --force")
	}
	if !sawDelete {
		t.Errorf("DELETE should have been issued")
	}
}

func TestExplorations_DeleteHandlesPostPrecheckRace(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/alerts") {
			// Precheck passes
			fmt.Fprint(w, `{"data":[]}`)
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"errors":["dependency exists now"]}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, stderr, err := executeCmd(t, "explorations", "delete", "5", "--yes")
	if err == nil {
		t.Fatal("expected race error")
	}
	if errs.CodeOf(err) != errs.ExitDependencyConflict {
		t.Errorf("exit code = %d, want 4", errs.CodeOf(err))
	}
	if !strings.Contains(stderr, "dependency_appeared_after_precheck:") {
		t.Errorf("stderr should contain machine-matchable prefix: %s", stderr)
	}
	if !strings.Contains(stderr, "logs-alerts list --exploration-id 5") {
		t.Errorf("stderr should contain inspect cmd with literal ID: %s", stderr)
	}
}

func TestExplorations_UpsertCreatesWhenNoMatch(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/explorations" {
			fmt.Fprint(w, `{"data":[{"id":"1","attributes":{"name":"different"}}]}`)
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"new","attributes":{"name":"INF-3017"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	out, _, err := executeCmd(t, "explorations", "create",
		"--source-id", "1", "--pattern", "abc", "--name", "INF-3017", "--upsert", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out)
	}
	if env["action"] != "created" {
		t.Errorf("action = %v, want created", env["action"])
	}
	if env["resource"] == nil {
		t.Errorf("resource missing")
	}
}

func TestExplorations_UpsertUpdatesWhenOneMatches(t *testing.T) {
	var patchCalled bool
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/explorations" {
			fmt.Fprint(w, `{"data":[{"id":"77","attributes":{"name":"INF-3017","queries":[{"query_type":"tail_query","where_condition":"old","source_variable":"old"}]}}]}`)
			return
		}
		if r.Method == "PATCH" {
			patchCalled = true
			fmt.Fprint(w, `{"data":{"id":"77","attributes":{"name":"INF-3017"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	out, _, err := executeCmd(t, "explorations", "create",
		"--source-id", "1", "--pattern", "abc", "--name", "INF-3017", "--upsert", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !patchCalled {
		t.Errorf("expected PATCH for the match")
	}
	var env map[string]any
	_ = json.Unmarshal([]byte(out), &env)
	if env["action"] != "updated" {
		t.Errorf("action = %v, want updated", env["action"])
	}
}

func TestExplorations_UpsertUnchangedWhenMatchIdentical(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/explorations" {
			fmt.Fprint(w, `{"data":[{"id":"77","attributes":{"name":"INF-3017","queries":[{"query_type":"tail_query","where_condition":"message CONTAINS \"abc\"","source_variable":"1"}]}}]}`)
			return
		}
		if r.Method == "PATCH" || r.Method == "POST" {
			t.Errorf("no write expected; got %s", r.Method)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	out, _, err := executeCmd(t, "explorations", "create",
		"--source-id", "1", "--pattern", "abc", "--name", "INF-3017", "--upsert", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	_ = json.Unmarshal([]byte(out), &env)
	if env["action"] != "unchanged" {
		t.Errorf("action = %v, want unchanged", env["action"])
	}
}

func TestExplorations_UpsertRefusesOnMultipleMatches(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"1","attributes":{"name":"dup"}},{"id":"2","attributes":{"name":"dup"}}]}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "create",
		"--source-id", "1", "--pattern", "x", "--name", "dup", "--upsert")
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("error should name duplicate count: %s", err.Error())
	}
}

func TestExplorations_UpsertIncompatibleWithBodyFile(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetExplorationFlags(t)

	_, _, err := executeCmd(t, "explorations", "create", "--body-file", "any.json", "--upsert")
	if err == nil {
		t.Fatal("expected error")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
}

// Field-reported concern: upsert envelope claim is that BOTH POST and PATCH
// paths wrap the resource as {"action": ..., "resource": ...}. This locks in
// that contract so non-upsert (bare resource) vs upsert (wrapped) is the only
// shape split callers need to handle.
func TestExplorations_UpsertWrapsBothPOSTAndPATCHIdentically(t *testing.T) {
	// POST path: no match → action=created, wrapped.
	srv1 := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/explorations" {
			fmt.Fprint(w, `{"data":[]}`)
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"111","attributes":{"name":"X"}}}`)
		}
	})
	setupTelemetryEnv(t, srv1.URL())
	resetExplorationFlags(t)

	out1, _, err := executeCmd(t, "explorations", "create",
		"--source-id", "1", "--pattern", "p", "--name", "X", "--upsert", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env1 map[string]any
	if err := json.Unmarshal([]byte(out1), &env1); err != nil {
		t.Fatalf("POST-path output not JSON: %v\n%s", err, out1)
	}
	if env1["action"] != "created" {
		t.Errorf("POST path action = %v, want created", env1["action"])
	}
	if env1["resource"] == nil {
		t.Errorf("POST path must include 'resource' key (wrapped envelope): %v", env1)
	}

	// PATCH path: one match, differs → action=updated, wrapped.
	srv2 := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/explorations" {
			fmt.Fprint(w, `{"data":[{"id":"111","attributes":{"name":"X","queries":[{"query_type":"tail_query","where_condition":"old","source_variable":"old"}]}}]}`)
			return
		}
		if r.Method == "PATCH" {
			fmt.Fprint(w, `{"data":{"id":"111","attributes":{"name":"X"}}}`)
		}
	})
	setupTelemetryEnv(t, srv2.URL())
	resetExplorationFlags(t)

	out2, _, err := executeCmd(t, "explorations", "create",
		"--source-id", "2", "--pattern", "p2", "--name", "X", "--upsert", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env2 map[string]any
	if err := json.Unmarshal([]byte(out2), &env2); err != nil {
		t.Fatalf("PATCH-path output not JSON: %v\n%s", err, out2)
	}
	if env2["action"] != "updated" {
		t.Errorf("PATCH path action = %v, want updated", env2["action"])
	}
	if env2["resource"] == nil {
		t.Errorf("PATCH path must include 'resource' key (wrapped envelope): %v", env2)
	}

	// Both envelopes must carry the same set of top-level keys.
	for _, key := range []string{"action", "resource"} {
		if _, ok := env1[key]; !ok {
			t.Errorf("POST envelope missing key %q", key)
		}
		if _, ok := env2[key]; !ok {
			t.Errorf("PATCH envelope missing key %q", key)
		}
	}
}

func TestExplorations_UpsertPatch404ReportsRace(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/explorations" {
			fmt.Fprint(w, `{"data":[{"id":"77","attributes":{"name":"INF-3017"}}]}`)
			return
		}
		if r.Method == "PATCH" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"errors":["deleted"]}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetExplorationFlags(t)

	_, stderr, err := executeCmd(t, "explorations", "create",
		"--source-id", "99", "--pattern", "new", "--name", "INF-3017", "--upsert")
	if err == nil {
		t.Fatal("expected race error")
	}
	if errs.CodeOf(err) != errs.ExitUpstream {
		t.Errorf("exit code = %d, want 5", errs.CodeOf(err))
	}
	if !strings.Contains(stderr, "upsert_match_deleted_after_precheck:") {
		t.Errorf("stderr should contain race prefix: %s", stderr)
	}
}
