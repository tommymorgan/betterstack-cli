package payload

import (
	"encoding/json"
	"testing"
)

func intPtr(n int) *int { return &n }
func boolPtr(b bool) *bool { return &b }

func TestBuildAlertCreate_ThresholdShape(t *testing.T) {
	body, err := BuildAlertCreate(AlertShorthand{
		Name:       "INF-3017",
		Threshold:  intPtr(3),
		WindowSecs: 300,
		PolicyID:   "77",
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "INF-3017" {
		t.Errorf("name = %v", got["name"])
	}
	if got["alert_type"] != "threshold" {
		t.Errorf("alert_type = %v, want threshold", got["alert_type"])
	}
	if got["operator"] != "higher_than_or_equal" {
		t.Errorf("operator = %v", got["operator"])
	}
	if v, _ := got["value"].(float64); v != 3 {
		t.Errorf("value = %v, want 3", got["value"])
	}
	if cp, _ := got["check_period"].(float64); cp != 300 {
		t.Errorf("check_period = %v, want 300", got["check_period"])
	}
	if qp, _ := got["query_period"].(float64); qp != 300 {
		t.Errorf("query_period = %v, want 300", got["query_period"])
	}
	target := got["escalation_target"].(map[string]any)
	if pid, _ := target["policy_id"].(float64); pid != 77 {
		t.Errorf("policy_id = %v, want 77 (as integer)", target["policy_id"])
	}
}

func TestBuildAlertCreate_RequiresAllShorthandFields(t *testing.T) {
	cases := []AlertShorthand{
		{Threshold: intPtr(3), WindowSecs: 300, PolicyID: "77"},                     // missing name
		{Name: "n", WindowSecs: 300, PolicyID: "77"},                                // missing threshold
		{Name: "n", Threshold: intPtr(3), PolicyID: "77"},                           // missing window
		{Name: "n", Threshold: intPtr(3), WindowSecs: 300},                          // missing policy-id
	}
	for i, c := range cases {
		if _, err := BuildAlertCreate(c); err == nil {
			t.Errorf("case %d: expected error for incomplete shorthand", i)
		}
	}
}

func TestBuildAlertCreate_RejectsNonNumericPolicyID(t *testing.T) {
	_, err := BuildAlertCreate(AlertShorthand{
		Name: "n", Threshold: intPtr(1), WindowSecs: 60, PolicyID: "abc",
	})
	if err == nil {
		t.Fatal("expected error for non-numeric policy-id")
	}
}

func TestBuildAlertCreate_IncludesRecoveryAndPausedWhenSet(t *testing.T) {
	body, err := BuildAlertCreate(AlertShorthand{
		Name: "n", Threshold: intPtr(1), WindowSecs: 60, PolicyID: "1",
		RecoverySecs: 120, Paused: boolPtr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(body, &got)
	if rp, _ := got["recovery_period"].(float64); rp != 120 {
		t.Errorf("recovery_period = %v, want 120", got["recovery_period"])
	}
	if got["paused"] != true {
		t.Errorf("paused = %v, want true", got["paused"])
	}
}

func TestBuildAlertPatch_OnlyIncludesProvidedFields(t *testing.T) {
	body, err := BuildAlertPatch(AlertShorthand{Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(body, &got)
	if got["name"] != "renamed" {
		t.Errorf("name = %v", got["name"])
	}
	if _, has := got["value"]; has {
		t.Errorf("value should be omitted: %v", got)
	}
	if _, has := got["check_period"]; has {
		t.Errorf("check_period should be omitted: %v", got)
	}
}

func TestBuildAlertPatch_CheckPeriodOverridesWindow(t *testing.T) {
	body, err := BuildAlertPatch(AlertShorthand{
		WindowSecs: 300,
		CheckSecs:  60,
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(body, &got)
	if cp, _ := got["check_period"].(float64); cp != 60 {
		t.Errorf("check_period = %v, want 60 (explicit check overrides window)", got["check_period"])
	}
	if qp, _ := got["query_period"].(float64); qp != 300 {
		t.Errorf("query_period = %v, want 300 (from window)", got["query_period"])
	}
}

func TestBuildAlertPatch_PolicyIDAsInteger(t *testing.T) {
	body, err := BuildAlertPatch(AlertShorthand{PolicyID: "42"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(body, &got)
	target := got["escalation_target"].(map[string]any)
	if pid, _ := target["policy_id"].(float64); pid != 42 {
		t.Errorf("policy_id = %v, want 42 as integer", target["policy_id"])
	}
}

func TestParseDurationFlags_AllBlanksReturnZeros(t *testing.T) {
	w, c, q, r, err := ParseDurationFlags("", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if w != 0 || c != 0 || q != 0 || r != 0 {
		t.Errorf("expected all zero, got %d/%d/%d/%d", w, c, q, r)
	}
}

func TestParseDurationFlags_PropagatesErrors(t *testing.T) {
	if _, _, _, _, err := ParseDurationFlags("abc", "", "", ""); err == nil {
		t.Fatal("expected error for bad window")
	}
}
