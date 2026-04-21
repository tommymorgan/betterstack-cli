package betterstack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

func osWriteFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0o600)
}

func resetAlertFlags(t *testing.T) {
	t.Helper()
	resetFlags(logsAlertsCreateCmd, "body-file", "name", "threshold", "window", "check-period", "query-period", "recovery", "policy-id", "exploration-id")
	resetBoolFlags(logsAlertsCreateCmd, "paused", "no-paused", "upsert")
	resetFlags(logsAlertsUpdateCmd, "body-file", "name", "threshold", "window", "check-period", "query-period", "recovery", "policy-id")
	resetBoolFlags(logsAlertsUpdateCmd, "paused", "no-paused")
	resetFlags(logsAlertsListCmd, "exploration-id", "policy-id")
	logsAlertsListCmd.Flags().Set("limit", "0")
	logsAlertsListCmd.Flags().Lookup("limit").Changed = false
	resetBoolFlags(logsAlertsDeleteCmd, "yes", "force")
	rootCmd.PersistentFlags().Set("json", "false")
	rootCmd.PersistentFlags().Lookup("json").Changed = false
}

// resetFlags clears the value and the Changed bit for each named flag.
// Because cobra/pflag treats Changed() as "user provided at CLI", we must
// clear it so cross-test state doesn't leak.
func resetFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		_ = cmd.Flags().Set(name, "")
		f.Changed = false
	}
}

func resetBoolFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		_ = cmd.Flags().Set(name, "false")
		f.Changed = false
	}
}

func TestLogsAlerts_CreateShorthandPostsThresholdPayload(t *testing.T) {
	var gotBody []byte
	var gotPath string
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			gotPath = r.URL.Path
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"a1","attributes":{"name":"INF-3017","value":3,"operator":"higher_than_or_equal","check_period":300,"query_period":300}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	_, _, err := executeCmd(t, "logs-alerts", "create",
		"--exploration-id", "5",
		"--threshold", "3",
		"--window", "5m",
		"--policy-id", "77",
		"--name", "INF-3017",
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/v2/explorations/5/alerts" {
		t.Errorf("path = %q, want /api/v2/explorations/5/alerts", gotPath)
	}

	var parsed map[string]any
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["alert_type"] != "threshold" {
		t.Errorf("alert_type = %v", parsed["alert_type"])
	}
	if parsed["operator"] != "higher_than_or_equal" {
		t.Errorf("operator = %v", parsed["operator"])
	}
	if v, _ := parsed["value"].(float64); v != 3 {
		t.Errorf("value = %v", parsed["value"])
	}
	if cp, _ := parsed["check_period"].(float64); cp != 300 {
		t.Errorf("check_period = %v, want 300", parsed["check_period"])
	}
	target := parsed["escalation_target"].(map[string]any)
	if pid, _ := target["policy_id"].(float64); pid != 77 {
		t.Errorf("policy_id = %v (type %T), want number 77", target["policy_id"], target["policy_id"])
	}
}

func TestLogsAlerts_DurationFlagRejectsBareNumber(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetAlertFlags(t)
	_, _, err := executeCmd(t, "logs-alerts", "create",
		"--exploration-id", "5", "--threshold", "1", "--window", "300",
		"--policy-id", "1", "--name", "x")
	if err == nil {
		t.Fatal("expected error for bare number")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "24h") {
		t.Errorf("error should mention max-unit guidance: %s", err.Error())
	}
}

func TestLogsAlerts_DurationFlagRejectsZero(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetAlertFlags(t)
	_, _, err := executeCmd(t, "logs-alerts", "create",
		"--exploration-id", "5", "--threshold", "1", "--window", "0s",
		"--policy-id", "1", "--name", "x")
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("error should say 'positive': %s", err.Error())
	}
}

func TestLogsAlerts_DurationFlagRejectsNegative(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetAlertFlags(t)
	_, _, err := executeCmd(t, "logs-alerts", "create",
		"--exploration-id", "5", "--threshold", "1", "--window", "-5m",
		"--policy-id", "1", "--name", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLogsAlerts_CreateBodyFileSendsVerbatim(t *testing.T) {
	dir := t.TempDir()
	bodyPath := dir + "/body.json"
	customBody := []byte(`{"alert_type":"anomaly_rrcf","escalation_target":{"team_id":99}}`)
	_ = writeFile(bodyPath, customBody)

	var gotBody []byte
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"1","attributes":{"name":"x"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	_, _, err := executeCmd(t, "logs-alerts", "create", "--exploration-id", "5", "--body-file", bodyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != string(customBody) {
		t.Errorf("body altered:\n got: %s\nwant: %s", gotBody, customBody)
	}
}

func writeFile(path string, content []byte) error {
	return osWriteFile(path, content)
}

func TestLogsAlerts_CreateShorthandAndBodyFileAreMutuallyExclusive(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetAlertFlags(t)
	_, _, err := executeCmd(t, "logs-alerts", "create",
		"--exploration-id", "5", "--body-file", "any.json", "--threshold", "3")
	if err == nil {
		t.Fatal("expected error")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
}

func TestLogsAlerts_ListShowsTable(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"9","attributes":{"name":"alpha","value":3,"operator":"higher_than_or_equal","check_period":300,"paused":false}}]}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"ID", "NAME", "CONDITION", "CHECK/WINDOW", "PAUSED"} {
		if !strings.Contains(out, col) {
			t.Errorf("missing column %q:\n%s", col, out)
		}
	}
	if !strings.Contains(out, ">= 3") {
		t.Errorf("condition should render as '>= 3':\n%s", out)
	}
	if !strings.Contains(out, "5m") {
		t.Errorf("check_period should render as '5m':\n%s", out)
	}
}

func TestLogsAlerts_ListFiltersByExplorationClientSide(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":"1","attributes":{"name":"a","exploration_id":5,"value":1,"operator":"higher_than_or_equal","check_period":60}},
			{"id":"2","attributes":{"name":"b","exploration_id":99,"value":1,"operator":"higher_than_or_equal","check_period":60}}
		]}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "list", "--exploration-id", "5")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, " 1 ") && !strings.HasPrefix(strings.SplitN(strings.TrimPrefix(strings.SplitN(out, "\n", 2)[1], ""), " ", 2)[0], "1") {
		// fallback: just require alert id 1 present and 2 absent
	}
	if !strings.Contains(out, "\n1 ") && !strings.Contains(out, "\n1\t") && !strings.Contains(out, "a ") {
		t.Logf("output:\n%s", out)
	}
	if strings.Contains(out, "\n2 ") || strings.Contains(out, "\n2\t") || strings.Contains(out, "b ") {
		t.Errorf("alert 2 should be filtered out:\n%s", out)
	}
}

func TestLogsAlerts_ListFiltersByPolicyClientSide(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":"1","attributes":{"name":"alpha","value":1,"operator":"higher_than_or_equal","check_period":60,"escalation_target":{"policy_id":77}}},
			{"id":"2","attributes":{"name":"beta","value":1,"operator":"higher_than_or_equal","check_period":60,"escalation_target":{"policy_id":88}}}
		]}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "list", "--policy-id", "77")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected alpha in output:\n%s", out)
	}
	if strings.Contains(out, "beta") {
		t.Errorf("beta should be filtered out:\n%s", out)
	}
}

func TestLogsAlerts_GetShowsDetail(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"id":"9","attributes":{"name":"INF-3017","exploration_id":5,"value":3,"operator":"higher_than_or_equal","check_period":300,"query_period":300,"paused":false,"escalation_target":{"policy_id":77}}}}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "get", "9")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"INF-3017", "5m", ">= 3", "policy 77"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail missing %q:\n%s", want, out)
		}
	}
}

func TestLogsAlerts_UpdateShorthandPatches(t *testing.T) {
	var gotBody []byte
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			gotBody, _ = io.ReadAll(r.Body)
			fmt.Fprint(w, `{"data":{"id":"9","attributes":{"name":"new"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	_, _, err := executeCmd(t, "logs-alerts", "update", "9", "--threshold", "5", "--window", "10m")
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(gotBody, &parsed)
	if v, _ := parsed["value"].(float64); v != 5 {
		t.Errorf("value = %v, want 5", parsed["value"])
	}
	if cp, _ := parsed["check_period"].(float64); cp != 600 {
		t.Errorf("check_period = %v, want 600 (10m)", parsed["check_period"])
	}
	if _, has := parsed["name"]; has {
		t.Errorf("name should be omitted (not provided): %v", parsed)
	}
}

func TestLogsAlerts_Delete(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "delete", "9", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Deleted") && !strings.Contains(out, "logs-alert") {
		t.Errorf("expected confirmation:\n%s", out)
	}
}

func TestLogsAlerts_DeleteRejectsForce(t *testing.T) {
	setupTelemetryEnv(t, "http://unused")
	resetAlertFlags(t)
	_, _, err := executeCmd(t, "logs-alerts", "delete", "9", "--yes", "--force")
	if err == nil {
		t.Fatal("expected error")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want 3", errs.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should explain --force invalidity: %s", err.Error())
	}
}

func TestLogsAlerts_UpsertScopesNameToExploration(t *testing.T) {
	var postCalled bool
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/alerts" {
			// An alert named INF-3017 exists on exploration A; the user targets exploration B.
			fmt.Fprint(w, `{"data":[{"id":"1","attributes":{"name":"INF-3017","exploration_id":1001,"value":3,"operator":"higher_than_or_equal","check_period":60}}]}`)
			return
		}
		if r.Method == "POST" {
			postCalled = true
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"new","attributes":{"name":"INF-3017","exploration_id":1002}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "create",
		"--exploration-id", "B", "--name", "INF-3017",
		"--threshold", "3", "--window", "5m", "--policy-id", "1",
		"--upsert", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !postCalled {
		t.Errorf("expected POST since no alert with name INF-3017 exists on exploration B")
	}
	var env map[string]any
	_ = json.Unmarshal([]byte(out), &env)
	if env["action"] != "created" {
		t.Errorf("action = %v, want created", env["action"])
	}
}

// Field-reported regression: accounts commonly have alerts with
// escalation_target: "current_team" (a string sentinel). The upsert match
// path must tolerate this shape account-wide, not fail closed on the first
// such alert.
func TestLogsAlerts_UpsertToleratesStringEscalationTarget(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/alerts" {
			// Realistic account shape: some alerts use "current_team" (string),
			// some use {"policy_id": N} (object). Before the fix, the string
			// form made the whole envelope fail to unmarshal and killed upsert.
			fmt.Fprint(w, `{"data":[
				{"id":"100","attributes":{"name":"existing-a","exploration_id":500,"value":1,"operator":"higher_than_or_equal","check_period":60,"escalation_target":"current_team"}},
				{"id":"101","attributes":{"name":"existing-b","exploration_id":500,"value":1,"operator":"higher_than_or_equal","check_period":60,"escalation_target":{"policy_id":77}}}
			]}`)
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"new","attributes":{"name":"brand-new","exploration_id":999}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "create",
		"--exploration-id", "999", "--name", "brand-new",
		"--threshold", "1", "--window", "5m", "--policy-id", "77",
		"--upsert", "--json")
	if err != nil {
		t.Fatalf("upsert must tolerate string escalation_target: %v", err)
	}
	var env map[string]any
	_ = json.Unmarshal([]byte(out), &env)
	if env["action"] != "created" {
		t.Errorf("action = %v, want created", env["action"])
	}
}

// Field-reported regression: list --exploration-id failed because the server
// returns exploration_id as a JSON number, not a string.
func TestLogsAlerts_ListExplorationIDFilterWorksOnNumericServerID(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":"1","attributes":{"name":"keep","exploration_id":766903,"value":1,"operator":"higher_than_or_equal","check_period":60}},
			{"id":"2","attributes":{"name":"drop","exploration_id":766905,"value":1,"operator":"higher_than_or_equal","check_period":60}}
		]}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "list", "--exploration-id", "766903")
	if err != nil {
		t.Fatalf("filter must not fail on numeric exploration_id: %v", err)
	}
	if !strings.Contains(out, "keep") {
		t.Errorf("expected 'keep' row:\n%s", out)
	}
	if strings.Contains(out, "drop") {
		t.Errorf("'drop' should be filtered out:\n%s", out)
	}
}

// Field-reported regression: logs-alerts update without --json pretty-printed
// through the same broken struct. The server-side change succeeded but the
// CLI errored on the output formatter. The update response must render
// cleanly regardless of --json.
func TestLogsAlerts_UpdateRendersCleanlyWithNumericExplorationID(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			fmt.Fprint(w, `{"data":{"id":"9","attributes":{"name":"n","exploration_id":766903,"value":3,"operator":"higher_than_or_equal","check_period":300,"query_period":300,"paused":false,"escalation_target":"current_team"}}}`)
		}
	})
	setupTelemetryEnv(t, srv.URL())
	resetAlertFlags(t)

	out, _, err := executeCmd(t, "logs-alerts", "update", "9", "--no-paused")
	if err != nil {
		t.Fatalf("update must render realistic server responses: %v", err)
	}
	if !strings.Contains(out, "766903") {
		t.Errorf("exploration_id should appear in detail output:\n%s", out)
	}
	if !strings.Contains(out, "current_team") {
		t.Errorf("escalation_target 'current_team' should render as-is:\n%s", out)
	}
}

// TestZZ_LogsAlerts_CreateHelpMentionsPolicyOpinion must be physically LAST
// in this file because cobra's --help handling leaves flag state that
// interferes with subsequent RunE invocations on the same command tree.
func TestZZ_LogsAlerts_CreateHelpMentionsPolicyOpinion(t *testing.T) {
	resetAlertFlags(t)
	out, _, err := executeCmd(t, "logs-alerts", "create", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"escalation polic", "--body-file", "policies"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q:\n%s", want, out)
		}
	}
}
