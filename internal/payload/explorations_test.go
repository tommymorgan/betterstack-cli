package payload

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildExplorationCreate_CountMatchingShape(t *testing.T) {
	body, err := BuildExplorationCreate(ExplorationShorthand{
		Name:     "INF-3017",
		SourceID: "123456",
		Pattern:  "AADSTS7000215",
	})
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}

	if got["name"] != "INF-3017" {
		t.Errorf("name = %v, want INF-3017", got["name"])
	}
	chart := got["chart"].(map[string]any)
	if chart["chart_type"] != "number_chart" {
		t.Errorf("chart_type = %v, want number_chart", chart["chart_type"])
	}
	queries := got["queries"].([]any)
	if len(queries) != 1 {
		t.Fatalf("queries len = %d, want 1", len(queries))
	}
	q := queries[0].(map[string]any)
	if q["query_type"] != "tail_query" {
		t.Errorf("query_type = %v, want tail_query", q["query_type"])
	}
	if q["source_variable"] != "123456" {
		t.Errorf("source_variable = %v, want 123456", q["source_variable"])
	}
	where := q["where_condition"].(string)
	if !strings.Contains(where, "AADSTS7000215") {
		t.Errorf("where_condition missing pattern: %s", where)
	}
	if got["date_range_from"] != "now-1h" {
		t.Errorf("date_range_from = %v, want now-1h", got["date_range_from"])
	}
	if got["date_range_to"] != "now" {
		t.Errorf("date_range_to = %v, want now", got["date_range_to"])
	}
}

func TestBuildExplorationCreate_EscapesAdversarialPattern(t *testing.T) {
	body, err := BuildExplorationCreate(ExplorationShorthand{
		Name:     "bad",
		SourceID: "1",
		Pattern:  `"; DROP *\.`,
	})
	if err != nil {
		t.Fatal(err)
	}
	// The raw attacker quote should not terminate the outer JSON string;
	// json.Unmarshal validates overall structure.
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("JSON invalid with adversarial pattern: %v\nbody: %s", err, body)
	}
}

func TestBuildExplorationPatch_OnlyIncludesProvidedFields(t *testing.T) {
	body, err := BuildExplorationPatch(ExplorationShorthand{Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "renamed" {
		t.Errorf("name = %v, want renamed", got["name"])
	}
	if _, hasQ := got["queries"]; hasQ {
		t.Errorf("queries should be omitted when only name was provided: %v", got)
	}
	if _, hasChart := got["chart"]; hasChart {
		t.Errorf("chart should be omitted: %v", got)
	}
}

func TestBuildExplorationPatch_PatternAloneSetsWhereCondition(t *testing.T) {
	body, err := BuildExplorationPatch(ExplorationShorthand{Pattern: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	queries := got["queries"].([]any)
	q := queries[0].(map[string]any)
	if _, hasSV := q["source_variable"]; hasSV {
		t.Errorf("source_variable should be omitted when only pattern given")
	}
	if q["where_condition"] == nil {
		t.Errorf("where_condition should be set")
	}
}

func TestBuildExplorationPatch_EmptyProducesEmptyBody(t *testing.T) {
	body, err := BuildExplorationPatch(ExplorationShorthand{})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "{}" {
		t.Errorf("empty shorthand = %q, want {}", body)
	}
}
