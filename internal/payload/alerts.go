package payload

import (
	"encoding/json"

	"github.com/tommymorgan/betterstack-cli/internal/duration"
	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

// AlertShorthand captures the CLI shorthand flags for creating a threshold
// logs-alert that routes through an escalation policy. Zero-valued fields are
// omitted (used on PATCH). PolicyID is expected to be an integer-valued
// string; the CLI renders it as a JSON number in the request body.
type AlertShorthand struct {
	Name        string
	Threshold   *int
	WindowSecs  int
	CheckSecs   int
	QuerySecs   int
	RecoverySecs int
	PolicyID    string
	Paused      *bool
}

// BuildAlertCreate builds the POST body for `logs-alerts create` from
// shorthand flags. It produces a threshold alert with higher_than_or_equal
// operator and an escalation_target of {"policy_id": N}.
func BuildAlertCreate(s AlertShorthand) ([]byte, error) {
	if s.Threshold == nil {
		return nil, errs.New(errs.ExitUserInput, "--threshold is required when not using --body-file")
	}
	if s.WindowSecs <= 0 {
		return nil, errs.New(errs.ExitUserInput, "--window is required when not using --body-file")
	}
	if s.PolicyID == "" {
		return nil, errs.New(errs.ExitUserInput, "--policy-id is required when not using --body-file")
	}
	if s.Name == "" {
		return nil, errs.New(errs.ExitUserInput, "--name is required when not using --body-file")
	}

	policyID, err := parseIntID(s.PolicyID)
	if err != nil {
		return nil, err
	}

	// check_period defaults to window when not explicitly set
	checkSecs := s.CheckSecs
	if checkSecs == 0 {
		checkSecs = s.WindowSecs
	}
	querySecs := s.QuerySecs
	if querySecs == 0 {
		querySecs = s.WindowSecs
	}

	body := map[string]any{
		"name":         s.Name,
		"alert_type":   "threshold",
		"operator":     "higher_than_or_equal",
		"value":        *s.Threshold,
		"check_period": checkSecs,
		"query_period": querySecs,
		"escalation_target": map[string]any{
			"policy_id": policyID,
		},
	}
	if s.RecoverySecs > 0 {
		body["recovery_period"] = s.RecoverySecs
	}
	if s.Paused != nil {
		body["paused"] = *s.Paused
	}

	return json.Marshal(body)
}

// BuildAlertPatch builds the PATCH body for `logs-alerts update`, containing
// only the fields corresponding to user-provided shorthand flags.
//
// Field ordering matters: --window is a shortcut that sets BOTH check_period
// and query_period, but --check-period and --query-period (when explicitly
// provided) override those individual fields. This mirrors the create
// semantics where --window is the fallback for each period.
func BuildAlertPatch(s AlertShorthand) ([]byte, error) {
	body := map[string]any{}
	if s.Name != "" {
		body["name"] = s.Name
	}
	if s.Threshold != nil {
		body["value"] = *s.Threshold
		body["operator"] = "higher_than_or_equal"
		body["alert_type"] = "threshold"
	}
	// --window broad stroke: sets both periods; overridden below by explicit
	// --check-period / --query-period if the user gave those too.
	if s.WindowSecs > 0 {
		body["check_period"] = s.WindowSecs
		body["query_period"] = s.WindowSecs
	}
	if s.CheckSecs > 0 {
		body["check_period"] = s.CheckSecs
	}
	if s.QuerySecs > 0 {
		body["query_period"] = s.QuerySecs
	}
	if s.RecoverySecs > 0 {
		body["recovery_period"] = s.RecoverySecs
	}
	if s.PolicyID != "" {
		policyID, err := parseIntID(s.PolicyID)
		if err != nil {
			return nil, err
		}
		body["escalation_target"] = map[string]any{"policy_id": policyID}
	}
	if s.Paused != nil {
		body["paused"] = *s.Paused
	}
	return json.Marshal(body)
}

// parseIntID converts a numeric-string ID to an integer for use as a JSON number.
// The BetterStack API treats policy_id as an integer, so the body must contain
// a number rather than a string.
func parseIntID(s string) (int, error) {
	var n int
	var sign = 1
	if len(s) == 0 {
		return 0, errs.New(errs.ExitUserInput, "id is empty")
	}
	i := 0
	if s[0] == '-' {
		sign = -1
		i = 1
	}
	if i == len(s) {
		return 0, errs.New(errs.ExitUserInput, "id %q is not numeric", s)
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, errs.New(errs.ExitUserInput, "id %q must be numeric (digits only)", s)
		}
		n = n*10 + int(c-'0')
	}
	return sign * n, nil
}

// ParseDurationFlags converts individual duration-flag strings to the integer
// seconds fields on an AlertShorthand. Blank strings leave the corresponding
// field unset (which is how "not provided" is expressed on patches).
func ParseDurationFlags(window, check, query, recovery string) (win, chk, qry, rec int, err error) {
	if window != "" {
		win, err = duration.ParseSeconds(window)
		if err != nil {
			return
		}
	}
	if check != "" {
		chk, err = duration.ParseSeconds(check)
		if err != nil {
			return
		}
	}
	if query != "" {
		qry, err = duration.ParseSeconds(query)
		if err != nil {
			return
		}
	}
	if recovery != "" {
		rec, err = duration.ParseSeconds(recovery)
		if err != nil {
			return
		}
	}
	return
}
