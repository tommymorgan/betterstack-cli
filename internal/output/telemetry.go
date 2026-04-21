package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/tommymorgan/betterstack-cli/internal/duration"
)

// Shared types for telemetry-host (logs) resources. Each resource's shape
// follows BetterStack's common envelope: {"id": "...", "attributes": {...}}.

type Exploration struct {
	ID         string `json:"id"`
	Attributes struct {
		Name          string `json:"name"`
		DateRangeFrom string `json:"date_range_from"`
		DateRangeTo   string `json:"date_range_to"`
		Chart         struct {
			ChartType string `json:"chart_type"`
		} `json:"chart"`
		Queries []struct {
			QueryType      string `json:"query_type"`
			WhereCondition string `json:"where_condition"`
			SourceVariable string `json:"source_variable"`
		} `json:"queries"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	} `json:"attributes"`
}

func RenderExplorationsTable(w io.Writer, data []json.RawMessage) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tDATE RANGE\tUPDATED")
	for _, raw := range data {
		var e Exploration
		if err := json.Unmarshal(raw, &e); err != nil {
			fmt.Fprintln(tw, "-\t(parse error)\t-\t-")
			continue
		}
		dateRange := fmt.Sprintf("%s → %s", orDash(e.Attributes.DateRangeFrom), orDash(e.Attributes.DateRangeTo))
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			e.ID,
			orDash(e.Attributes.Name),
			dateRange,
			formatTimeOrDash(e.Attributes.UpdatedAt),
		)
	}
	return tw.Flush()
}

func RenderExplorationDetail(w io.Writer, data json.RawMessage) error {
	var e Exploration
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("failed to parse exploration: %w", err)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", e.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", orDash(e.Attributes.Name))
	fmt.Fprintf(tw, "Chart Type:\t%s\n", orDash(e.Attributes.Chart.ChartType))
	fmt.Fprintf(tw, "Date Range From:\t%s\n", orDash(e.Attributes.DateRangeFrom))
	fmt.Fprintf(tw, "Date Range To:\t%s\n", orDash(e.Attributes.DateRangeTo))
	if len(e.Attributes.Queries) > 0 {
		q := e.Attributes.Queries[0]
		fmt.Fprintf(tw, "Query Type:\t%s\n", orDash(q.QueryType))
		fmt.Fprintf(tw, "Where Condition:\t%s\n", orDash(q.WhereCondition))
		fmt.Fprintf(tw, "Source:\t%s\n", orDash(q.SourceVariable))
	}
	fmt.Fprintf(tw, "Created At:\t%s\n", formatTimeOrDash(e.Attributes.CreatedAt))
	fmt.Fprintf(tw, "Updated At:\t%s\n", formatTimeOrDash(e.Attributes.UpdatedAt))
	return tw.Flush()
}

type LogsAlert struct {
	ID         string `json:"id"`
	Attributes struct {
		Name             string `json:"name"`
		AlertType        string `json:"alert_type"`
		Operator         string `json:"operator"`
		Value            int    `json:"value"`
		CheckPeriod      int    `json:"check_period"`
		QueryPeriod      int    `json:"query_period"`
		RecoveryPeriod   int    `json:"recovery_period"`
		Paused           bool   `json:"paused"`
		ExplorationID    string `json:"exploration_id"`
		EscalationTarget struct {
			PolicyID   json.Number `json:"policy_id"`
			TeamID     json.Number `json:"team_id"`
			UserID     json.Number `json:"user_id"`
			ScheduleID json.Number `json:"schedule_id"`
		} `json:"escalation_target"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	} `json:"attributes"`
}

func RenderAlertsTable(w io.Writer, data []json.RawMessage) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tCONDITION\tCHECK/WINDOW\tPAUSED")
	for _, raw := range data {
		var a LogsAlert
		if err := json.Unmarshal(raw, &a); err != nil {
			fmt.Fprintln(tw, "-\t(parse error)\t-\t-\t-")
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			a.ID,
			orDash(a.Attributes.Name),
			renderAlertCondition(a),
			duration.Human(a.Attributes.CheckPeriod),
			yesNo(a.Attributes.Paused),
		)
	}
	return tw.Flush()
}

func RenderAlertDetail(w io.Writer, data json.RawMessage) error {
	var a LogsAlert
	if err := json.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("failed to parse alert: %w", err)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", a.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", orDash(a.Attributes.Name))
	fmt.Fprintf(tw, "Exploration ID:\t%s\n", orDash(a.Attributes.ExplorationID))
	fmt.Fprintf(tw, "Condition:\t%s\n", renderAlertCondition(a))
	fmt.Fprintf(tw, "Check Period:\t%s\n", duration.Human(a.Attributes.CheckPeriod))
	fmt.Fprintf(tw, "Query Period:\t%s\n", duration.Human(a.Attributes.QueryPeriod))
	if a.Attributes.RecoveryPeriod > 0 {
		fmt.Fprintf(tw, "Recovery Period:\t%s\n", duration.Human(a.Attributes.RecoveryPeriod))
	}
	fmt.Fprintf(tw, "Paused:\t%s\n", yesNo(a.Attributes.Paused))
	fmt.Fprintf(tw, "Escalation Target:\t%s\n", renderEscalationTarget(a))
	fmt.Fprintf(tw, "Created At:\t%s\n", formatTimeOrDash(a.Attributes.CreatedAt))
	fmt.Fprintf(tw, "Updated At:\t%s\n", formatTimeOrDash(a.Attributes.UpdatedAt))
	return tw.Flush()
}

func renderAlertCondition(a LogsAlert) string {
	op := a.Attributes.Operator
	symbol := op
	switch op {
	case "higher_than_or_equal":
		symbol = ">="
	case "higher_than":
		symbol = ">"
	case "lower_than_or_equal":
		symbol = "<="
	case "lower_than":
		symbol = "<"
	case "equal":
		symbol = "=="
	}
	return fmt.Sprintf("%s %d", symbol, a.Attributes.Value)
}

func renderEscalationTarget(a LogsAlert) string {
	et := a.Attributes.EscalationTarget
	switch {
	case et.PolicyID != "":
		return "policy " + et.PolicyID.String()
	case et.TeamID != "":
		return "team " + et.TeamID.String()
	case et.UserID != "":
		return "user " + et.UserID.String()
	case et.ScheduleID != "":
		return "schedule " + et.ScheduleID.String()
	}
	return dash
}

type Policy struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string           `json:"name"`
		Steps     []map[string]any `json:"steps"`
		UpdatedAt string           `json:"updated_at"`
		CreatedAt string           `json:"created_at"`
	} `json:"attributes"`
}

func RenderPoliciesTable(w io.Writer, data []json.RawMessage) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tSTEPS\tUPDATED")
	for _, raw := range data {
		var p Policy
		if err := json.Unmarshal(raw, &p); err != nil {
			fmt.Fprintln(tw, "-\t(parse error)\t-\t-")
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n",
			p.ID,
			orDash(p.Attributes.Name),
			len(p.Attributes.Steps),
			formatTimeOrDash(p.Attributes.UpdatedAt),
		)
	}
	return tw.Flush()
}

func RenderPolicyDetail(w io.Writer, data json.RawMessage) error {
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("failed to parse policy: %w", err)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", p.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", orDash(p.Attributes.Name))
	fmt.Fprintf(tw, "Steps:\t%d\n", len(p.Attributes.Steps))
	for i, step := range p.Attributes.Steps {
		stepJSON, _ := json.Marshal(step)
		fmt.Fprintf(tw, "  Step %d:\t%s\n", i+1, stepJSON)
	}
	fmt.Fprintf(tw, "Created At:\t%s\n", formatTimeOrDash(p.Attributes.CreatedAt))
	fmt.Fprintf(tw, "Updated At:\t%s\n", formatTimeOrDash(p.Attributes.UpdatedAt))
	return tw.Flush()
}

type LogsSource struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		Platform  string `json:"platform"`
		TeamName  string `json:"team_name"`
		Retention int    `json:"retention"`
	} `json:"attributes"`
}

func RenderSourcesTable(w io.Writer, data []json.RawMessage) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tPLATFORM\tTEAM")
	for _, raw := range data {
		var s LogsSource
		if err := json.Unmarshal(raw, &s); err != nil {
			fmt.Fprintln(tw, "-\t(parse error)\t-\t-")
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			s.ID,
			orDash(s.Attributes.Name),
			orDash(s.Attributes.Platform),
			orDash(s.Attributes.TeamName),
		)
	}
	return tw.Flush()
}

func RenderSourceDetail(w io.Writer, data json.RawMessage) error {
	var s LogsSource
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", s.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", orDash(s.Attributes.Name))
	fmt.Fprintf(tw, "Platform:\t%s\n", orDash(s.Attributes.Platform))
	fmt.Fprintf(tw, "Team Name:\t%s\n", orDash(s.Attributes.TeamName))
	retention := dash
	if s.Attributes.Retention > 0 {
		retention = fmt.Sprintf("%d days", s.Attributes.Retention)
	}
	fmt.Fprintf(tw, "Retention:\t%s\n", retention)
	return tw.Flush()
}

// RenderDeleteEnvelope emits a human confirmation line or the JSON envelope
// described in the plan.
func RenderDeleteEnvelope(w io.Writer, resourceType, id string, asJSON bool) error {
	if asJSON {
		return RenderJSON(w, map[string]any{"id": id, "deleted": true})
	}
	_, err := fmt.Fprintf(w, "Deleted %s %s\n", resourceType, id)
	return err
}

// RenderUpsertEnvelope emits the JSON envelope for upsert commands:
// {"action": "...", "resource": {...}}
func RenderUpsertEnvelope(w io.Writer, action string, resource json.RawMessage, asJSON bool) error {
	if asJSON {
		return RenderJSON(w, map[string]any{"action": action, "resource": resource})
	}
	_, err := fmt.Fprintf(w, "%s\n", action)
	if err != nil {
		return err
	}
	return nil
}

// RenderListEnvelope emits {"items": [...]} in JSON mode.
func RenderListEnvelope(w io.Writer, items []json.RawMessage) error {
	return RenderJSON(w, map[string]any{"items": items})
}

// yesNo returns "yes" or "no" for a boolean, matching existing rendering.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// formatTimeOrDash is like formatTime but returns a dash for empty input.
func formatTimeOrDash(t string) string {
	if t == "" {
		return dash
	}
	return formatTime(t)
}
