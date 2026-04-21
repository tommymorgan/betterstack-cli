package betterstack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/bodyfile"
	"github.com/tommymorgan/betterstack-cli/internal/client"
	"github.com/tommymorgan/betterstack-cli/internal/errs"
	"github.com/tommymorgan/betterstack-cli/internal/output"
	"github.com/tommymorgan/betterstack-cli/internal/payload"
)

var logsAlertsCmd = &cobra.Command{
	Use:   "logs-alerts",
	Short: "Manage BetterStack logs alerts (threshold/anomaly alerts on explorations)",
	Long:  "Logs alerts attach to explorations. The exploration carries the query; the alert carries the firing condition. Alerts route through escalation policies; pick one via `policies list`, or use --body-file for ad-hoc direct-channel routing.",
}

var logsAlertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List logs alerts",
	Long:  "List logs alerts. Filters --exploration-id and --policy-id are applied client-side after full pagination; this is a design invariant (see plan).",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetInt("limit")
		explorationID, _ := cmd.Flags().GetString("exploration-id")
		policyID, _ := cmd.Flags().GetString("policy-id")

		results, err := c.ListAlerts(context.Background(), limit)
		if err != nil {
			return wrapAPIErr(err)
		}

		if explorationID != "" {
			results = filterAlertsByExploration(results, explorationID)
		}
		if policyID != "" {
			results = filterAlertsByPolicy(results, policyID)
		}

		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderListEnvelope(w, results)
		}
		return output.RenderAlertsTable(w, results)
	},
}

var logsAlertsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single logs alert by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		result, err := c.GetAlert(context.Background(), args[0])
		if err != nil {
			return wrapAPIErr(err)
		}
		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderJSON(w, result)
		}
		return output.RenderAlertDetail(w, result)
	},
}

var logsAlertsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a logs alert on an exploration",
	Long: `Create a threshold alert attached to an exploration. Shorthand flags produce a >= threshold alert routed through an escalation policy.

Shorthand requires --exploration-id, --name, --threshold, --window, and --policy-id.

Non-policy routing (team_id, user_id, schedule_id, or the "current_team" string sentinel) requires --body-file — pick one from "policies list" to use shorthand, or use --body-file for any other escalation_target shape.

The shorthand mode and --body-file are mutually exclusive. The --upsert flag (shorthand only) is non-atomic within the target exploration scope: a concurrent actor may create a duplicate between the list and the POST.

JSON output envelope: non-upsert emits the resource directly; --upsert wraps it as {"action": ..., "resource": ...}. Callers that need one selector across both can use '.id // .resource.id'.`,
	RunE: runLogsAlertsCreate,
}

func runLogsAlertsCreate(cmd *cobra.Command, _ []string) error {
	bodyFile, _ := cmd.Flags().GetString("body-file")
	explorationID, _ := cmd.Flags().GetString("exploration-id")
	name, _ := cmd.Flags().GetString("name")
	thresholdRaw, _ := cmd.Flags().GetString("threshold")
	window, _ := cmd.Flags().GetString("window")
	checkFlag, _ := cmd.Flags().GetString("check-period")
	queryFlag, _ := cmd.Flags().GetString("query-period")
	recoveryFlag, _ := cmd.Flags().GetString("recovery")
	policyID, _ := cmd.Flags().GetString("policy-id")
	pausedSet := cmd.Flags().Changed("paused") || cmd.Flags().Changed("no-paused")
	paused, _ := cmd.Flags().GetBool("paused")
	noPaused, _ := cmd.Flags().GetBool("no-paused")
	upsert, _ := cmd.Flags().GetBool("upsert")

	hasShorthand := name != "" || thresholdRaw != "" || window != "" ||
		checkFlag != "" || queryFlag != "" || recoveryFlag != "" ||
		policyID != "" || pausedSet

	if explorationID == "" {
		return errs.New(errs.ExitUserInput, "--exploration-id is required")
	}
	if bodyFile != "" && hasShorthand {
		return errs.New(errs.ExitUserInput,
			"--body-file and shorthand flags are mutually exclusive")
	}
	if bodyFile != "" && upsert {
		return errs.New(errs.ExitUserInput,
			"--upsert requires shorthand flags and is incompatible with --body-file")
	}

	c, err := resolveTelemetry(cmd)
	if err != nil {
		return err
	}

	if bodyFile != "" {
		body, err := bodyfile.Read(bodyFile, os.Stdin)
		if err != nil {
			return err
		}
		result, err := c.CreateAlert(context.Background(), explorationID, body)
		if err != nil {
			return wrapAPIErr(err)
		}
		return renderAlertResult(cmd, result)
	}

	shorthand, err := buildAlertShorthand(cmd, name, thresholdRaw, window, checkFlag, queryFlag, recoveryFlag, policyID, pausedSet, paused, noPaused)
	if err != nil {
		return err
	}

	if upsert {
		return runLogsAlertsUpsert(cmd, c, explorationID, shorthand)
	}

	body, err := payload.BuildAlertCreate(shorthand)
	if err != nil {
		return err
	}
	result, err := c.CreateAlert(context.Background(), explorationID, body)
	if err != nil {
		return wrapAPIErr(err)
	}
	return renderAlertResult(cmd, result)
}

var logsAlertsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a logs alert",
	Long:  "Update via shorthand flags (PATCH contains only provided fields) or --body-file (sends file contents verbatim). Use --body-file for nested shapes like anomaly_rrcf alerts, custom escalation_target shapes, or metadata objects.",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogsAlertsUpdate,
}

func runLogsAlertsUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]
	bodyFile, _ := cmd.Flags().GetString("body-file")
	name, _ := cmd.Flags().GetString("name")
	thresholdRaw, _ := cmd.Flags().GetString("threshold")
	window, _ := cmd.Flags().GetString("window")
	checkFlag, _ := cmd.Flags().GetString("check-period")
	queryFlag, _ := cmd.Flags().GetString("query-period")
	recoveryFlag, _ := cmd.Flags().GetString("recovery")
	policyID, _ := cmd.Flags().GetString("policy-id")
	pausedSet := cmd.Flags().Changed("paused") || cmd.Flags().Changed("no-paused")
	paused, _ := cmd.Flags().GetBool("paused")
	noPaused, _ := cmd.Flags().GetBool("no-paused")

	hasShorthand := name != "" || thresholdRaw != "" || window != "" ||
		checkFlag != "" || queryFlag != "" || recoveryFlag != "" ||
		policyID != "" || pausedSet

	if bodyFile != "" && hasShorthand {
		return errs.New(errs.ExitUserInput,
			"--body-file and shorthand flags are mutually exclusive")
	}
	if bodyFile == "" && !hasShorthand {
		return errs.New(errs.ExitUserInput,
			"update requires at least one shorthand flag or --body-file")
	}

	c, err := resolveTelemetry(cmd)
	if err != nil {
		return err
	}

	var body []byte
	if bodyFile != "" {
		body, err = bodyfile.Read(bodyFile, os.Stdin)
		if err != nil {
			return err
		}
	} else {
		sh, bErr := buildAlertShorthand(cmd, name, thresholdRaw, window, checkFlag, queryFlag, recoveryFlag, policyID, pausedSet, paused, noPaused)
		if bErr != nil {
			return bErr
		}
		body, err = payload.BuildAlertPatch(sh)
		if err != nil {
			return err
		}
	}

	result, err := c.UpdateAlert(context.Background(), id, body)
	if err != nil {
		return wrapAPIErr(err)
	}
	return renderAlertResult(cmd, result)
}

var logsAlertsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a logs alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if force, _ := cmd.Flags().GetBool("force"); force {
			return errs.New(errs.ExitUserInput,
				"--force is only valid on commands with dependency checks (explorations delete, policies delete)")
		}
		if err := RequireYes(cmd); err != nil {
			return err
		}
		c, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		if err := c.DeleteAlert(context.Background(), args[0]); err != nil {
			return wrapAPIErr(err)
		}
		return output.RenderDeleteEnvelope(cmd.OutOrStdout(), "logs-alert", args[0], jsonFlag(cmd))
	},
}

// buildAlertShorthand parses the raw CLI flag strings into an AlertShorthand.
func buildAlertShorthand(cmd *cobra.Command, name, thresholdRaw, window, check, query, recovery, policyID string, pausedSet, paused, noPaused bool) (payload.AlertShorthand, error) {
	_ = cmd
	var sh payload.AlertShorthand
	sh.Name = name
	sh.PolicyID = policyID

	if thresholdRaw != "" {
		n, err := parseIntFlag("--threshold", thresholdRaw)
		if err != nil {
			return sh, err
		}
		sh.Threshold = &n
	}

	w, c, q, r, err := payload.ParseDurationFlags(window, check, query, recovery)
	if err != nil {
		return sh, err
	}
	sh.WindowSecs = w
	sh.CheckSecs = c
	sh.QuerySecs = q
	sh.RecoverySecs = r

	if pausedSet {
		var b bool
		switch {
		case noPaused:
			b = false
		case paused:
			b = true
		}
		sh.Paused = &b
	}

	return sh, nil
}

// parseIntFlag validates an integer flag value.
func parseIntFlag(flag, s string) (int, error) {
	n := 0
	if s == "" {
		return 0, errs.New(errs.ExitUserInput, "%s is empty", flag)
	}
	neg := false
	i := 0
	if s[0] == '-' {
		neg = true
		i = 1
	}
	if i == len(s) {
		return 0, errs.New(errs.ExitUserInput, "%s %q is not an integer", flag, s)
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, errs.New(errs.ExitUserInput, "%s %q must be an integer", flag, s)
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		return -n, nil
	}
	return n, nil
}

func renderAlertResult(cmd *cobra.Command, result json.RawMessage) error {
	w := cmd.OutOrStdout()
	if jsonFlag(cmd) {
		return output.RenderJSON(w, result)
	}
	return output.RenderAlertDetail(w, result)
}

// runLogsAlertsUpsert implements upsert scoped to the target exploration.
// Matching is by name among alerts that also belong to the same exploration.
func runLogsAlertsUpsert(cmd *cobra.Command, c *client.TelemetryClient, explorationID string, s payload.AlertShorthand) error {
	ctx := context.Background()
	all, err := c.ListAlerts(ctx, 0)
	if err != nil {
		return wrapAPIErr(err)
	}
	matches, parseErr := findAlertsByNameAndExploration(all, s.Name, explorationID)
	if parseErr != nil {
		return errs.Wrap(errs.ExitUpstream,
			fmt.Errorf("upsert cannot match alerts: malformed alert envelope: %w", parseErr))
	}

	switch len(matches) {
	case 0:
		body, err := payload.BuildAlertCreate(s)
		if err != nil {
			return err
		}
		result, err := c.CreateAlert(ctx, explorationID, body)
		if err != nil {
			return wrapAPIErr(err)
		}
		return renderUpsertAlert(cmd, "created", result)
	case 1:
		existing := matches[0]
		equal, parseErr := alertShorthandEquals(existing.raw, s)
		if parseErr != nil {
			return errs.Wrap(errs.ExitUpstream,
				fmt.Errorf("could not verify match for upsert: malformed alert envelope: %w", parseErr))
		}
		if equal {
			return renderUpsertAlert(cmd, "unchanged", existing.raw)
		}
		body, err := payload.BuildAlertPatch(s)
		if err != nil {
			return err
		}
		result, err := c.UpdateAlert(ctx, existing.id, body)
		if err != nil {
			if isNotFound(err) {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: upsert_match_deleted_after_precheck: logs-alerts get %s\n", existing.id)
				return errs.SilentExit(errs.ExitUpstream)
			}
			return wrapAPIErr(err)
		}
		return renderUpsertAlert(cmd, "updated", result)
	default:
		return errs.New(errs.ExitUserInput,
			"%d alerts match name %q within exploration %s; use a unique --name or rename duplicates via `logs-alerts update`",
			len(matches), s.Name, explorationID)
	}
}

func renderUpsertAlert(cmd *cobra.Command, action string, resource json.RawMessage) error {
	w := cmd.OutOrStdout()
	if jsonFlag(cmd) {
		return output.RenderJSON(w, map[string]any{
			"action":   action,
			"resource": resource,
		})
	}
	fmt.Fprintf(w, "%s:\n", action)
	return output.RenderAlertDetail(w, resource)
}

type alertSummary struct {
	id            string
	name          string
	explorationID string
	raw           json.RawMessage
}

// alertEnvelope covers either top-level exploration_id or
// relationships.exploration.data.id. ExplorationID is json.Number because the
// server returns it as a JSON number (not a string). EscalationTarget is raw
// because the server returns either an object or a string sentinel
// ("current_team"); binding it to a narrow struct made every alert in a
// mixed-shape account fail to unmarshal.
type alertEnvelope struct {
	ID         string `json:"id"`
	Attributes struct {
		Name             string          `json:"name"`
		ExplorationID    json.Number     `json:"exploration_id"`
		Value            int             `json:"value"`
		Operator         string          `json:"operator"`
		CheckPeriod      int             `json:"check_period"`
		QueryPeriod      int             `json:"query_period"`
		EscalationTarget json.RawMessage `json:"escalation_target"`
	} `json:"attributes"`
	Relationships struct {
		Exploration struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"exploration"`
	} `json:"relationships"`
}

// alertPolicyID extracts the policy ID from an alert's raw escalation_target,
// returning ("", false) when the target is a string sentinel (e.g.
// "current_team") or an object without a policy_id.
func alertPolicyID(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var obj struct {
		PolicyID json.Number `json:"policy_id"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", false
	}
	if obj.PolicyID == "" {
		return "", false
	}
	return obj.PolicyID.String(), true
}

func parseAlert(raw json.RawMessage) (alertSummary, bool) {
	var e alertEnvelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return alertSummary{}, false
	}
	return alertSummary{
		id:            e.ID,
		name:          e.Attributes.Name,
		explorationID: alertExplorationID(e),
		raw:           raw,
	}, true
}

// alertExplorationID returns the parent exploration ID as a string, preferring
// attributes.exploration_id (json.Number) and falling back to relationships.
func alertExplorationID(e alertEnvelope) string {
	if e.Attributes.ExplorationID != "" {
		return e.Attributes.ExplorationID.String()
	}
	return e.Relationships.Exploration.Data.ID
}

// findAlertsByNameAndExploration filters alerts that match both name and
// parent exploration. Returns an error on envelope-parse failure so upsert
// does not silently POST a duplicate whose match was unreadable.
func findAlertsByNameAndExploration(list []json.RawMessage, name, explorationID string) ([]alertSummary, error) {
	var out []alertSummary
	for _, item := range list {
		a, err := parseAlertStrict(item)
		if err != nil {
			return nil, err
		}
		if a.name == name && a.explorationID == explorationID {
			out = append(out, a)
		}
	}
	return out, nil
}

// parseAlertStrict is the error-propagating variant used on paths where a
// parse failure must not silently drop the alert (upsert matching, precheck).
func parseAlertStrict(raw json.RawMessage) (alertSummary, error) {
	var e alertEnvelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return alertSummary{}, err
	}
	return alertSummary{
		id:            e.ID,
		name:          e.Attributes.Name,
		explorationID: alertExplorationID(e),
		raw:           raw,
	}, nil
}

func filterAlertsByExploration(list []json.RawMessage, explorationID string) []json.RawMessage {
	out := make([]json.RawMessage, 0)
	for _, item := range list {
		a, ok := parseAlert(item)
		if !ok {
			continue
		}
		if a.explorationID == explorationID {
			out = append(out, item)
		}
	}
	return out
}

func filterAlertsByPolicy(list []json.RawMessage, policyID string) []json.RawMessage {
	out := make([]json.RawMessage, 0)
	for _, item := range list {
		var e alertEnvelope
		if err := json.Unmarshal(item, &e); err != nil {
			continue
		}
		pid, ok := alertPolicyID(e.Attributes.EscalationTarget)
		if !ok {
			continue
		}
		if pid == policyID {
			out = append(out, item)
		}
	}
	return out
}

// alertShorthandEquals reports whether the existing alert already matches
// every shorthand field the user provided. Returns (false, err) on parse
// failure so upsert does not silently treat an unreadable match as
// "unchanged" and clobber its fields on the next run.
func alertShorthandEquals(raw json.RawMessage, s payload.AlertShorthand) (bool, error) {
	var e alertEnvelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return false, err
	}
	if s.Name != "" && e.Attributes.Name != s.Name {
		return false, nil
	}
	if s.Threshold != nil && e.Attributes.Value != *s.Threshold {
		return false, nil
	}
	if s.WindowSecs > 0 {
		// --window sets both check_period and query_period in create; equality
		// must mirror that semantic.
		if e.Attributes.CheckPeriod != s.WindowSecs || e.Attributes.QueryPeriod != s.WindowSecs {
			return false, nil
		}
	}
	if s.CheckSecs > 0 && e.Attributes.CheckPeriod != s.CheckSecs {
		return false, nil
	}
	if s.QuerySecs > 0 && e.Attributes.QueryPeriod != s.QuerySecs {
		return false, nil
	}
	if s.PolicyID != "" {
		pid, ok := alertPolicyID(e.Attributes.EscalationTarget)
		if !ok || pid != s.PolicyID {
			return false, nil
		}
	}
	return true, nil
}

func init() {
	for _, c := range []*cobra.Command{logsAlertsCreateCmd, logsAlertsUpdateCmd} {
		c.Flags().String("body-file", "", "Path to a JSON file with the full request body (or - for stdin)")
		c.Flags().String("name", "", "Alert name")
		c.Flags().String("threshold", "", "Firing threshold (integer; >= is the implied operator)")
		c.Flags().String("window", "", "Query window as a Go duration (e.g., 5m, 30s, 2h); max unit: h")
		c.Flags().String("check-period", "", "Check period (Go duration; defaults to --window)")
		c.Flags().String("query-period", "", "Query period (Go duration; defaults to --window)")
		c.Flags().String("recovery", "", "Recovery period (Go duration)")
		c.Flags().String("policy-id", "", "Escalation policy ID (numeric; required for shorthand — use --body-file for team/user/schedule/\"current_team\" routing)")
		c.Flags().Bool("paused", false, "Create alert in paused state")
		c.Flags().Bool("no-paused", false, "Create/update alert in unpaused state")
	}
	logsAlertsCreateCmd.Flags().String("exploration-id", "", "Parent exploration ID (required)")
	logsAlertsCreateCmd.Flags().Bool("upsert", false, "Create if no alert with this name on this exploration exists, otherwise PATCH the sole match (non-atomic)")

	logsAlertsListCmd.Flags().Int("limit", 0, "Maximum number of results")
	logsAlertsListCmd.Flags().String("exploration-id", "", "Filter by parent exploration ID (client-side)")
	logsAlertsListCmd.Flags().String("policy-id", "", "Filter by escalation policy ID (client-side)")

	logsAlertsDeleteCmd.Flags().Bool("yes", false, "Confirm destructive operation")
	logsAlertsDeleteCmd.Flags().Bool("force", false, "Not supported on logs-alerts delete (will error)")

	logsAlertsCmd.AddCommand(logsAlertsListCmd)
	logsAlertsCmd.AddCommand(logsAlertsGetCmd)
	logsAlertsCmd.AddCommand(logsAlertsCreateCmd)
	logsAlertsCmd.AddCommand(logsAlertsUpdateCmd)
	logsAlertsCmd.AddCommand(logsAlertsDeleteCmd)
	rootCmd.AddCommand(logsAlertsCmd)
}
