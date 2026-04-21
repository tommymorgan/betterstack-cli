package payload

import (
	"encoding/json"

	"github.com/tommymorgan/betterstack-cli/internal/pattern"
)

// ExplorationShorthand captures the CLI shorthand flags for creating a
// count-matching exploration. Empty fields are omitted from the payload.
type ExplorationShorthand struct {
	Name     string
	SourceID string
	Pattern  string
}

// BuildExplorationCreate builds the POST body for `explorations create` from
// shorthand flags. The chart is a number_chart with a tail_query that matches
// the pattern literally in the message field.
func BuildExplorationCreate(s ExplorationShorthand) ([]byte, error) {
	body := map[string]any{
		"name": s.Name,
		"chart": map[string]any{
			"chart_type": "number_chart",
		},
		"queries": []map[string]any{
			{
				"query_type":      "tail_query",
				"where_condition": pattern.WhereCondition(s.Pattern),
				"source_variable": s.SourceID,
			},
		},
		"date_range_from": "now-1h",
		"date_range_to":   "now",
	}
	return json.Marshal(body)
}

// BuildExplorationPatch builds the PATCH body for `explorations update`
// containing only the fields that correspond to user-provided shorthand flags.
// Unspecified flags (empty strings for Name/SourceID/Pattern) are omitted.
func BuildExplorationPatch(s ExplorationShorthand) ([]byte, error) {
	body := map[string]any{}
	if s.Name != "" {
		body["name"] = s.Name
	}
	if s.Pattern != "" || s.SourceID != "" {
		q := map[string]any{
			"query_type": "tail_query",
		}
		if s.Pattern != "" {
			q["where_condition"] = pattern.WhereCondition(s.Pattern)
		}
		if s.SourceID != "" {
			q["source_variable"] = s.SourceID
		}
		body["queries"] = []map[string]any{q}
	}
	return json.Marshal(body)
}
