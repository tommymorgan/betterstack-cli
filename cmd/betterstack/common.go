package betterstack

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/client"
	"github.com/tommymorgan/betterstack-cli/internal/config"
	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

// jsonFlag returns whether --json is set on the root command.
func jsonFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("json")
	return v
}

// quietFlag returns whether --quiet is set on the root command. Also honors
// BETTERSTACK_QUIET.
func quietFlag(cmd *cobra.Command) bool {
	if v, _ := cmd.Root().PersistentFlags().GetBool("quiet"); v {
		return true
	}
	return envTruthy("BETTERSTACK_QUIET")
}

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch v {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// isTTY reports whether stderr is a terminal. Used to decide whether to emit
// the telemetry-fallback notice. Kept as a var so tests can stub it.
var isTTY = func() bool {
	info, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ensure io is referenced (helps keep imports alive if lint strips unused)
var _ io.Writer = (*strings.Builder)(nil)

var telemetryNoticeOnce sync.Once

// resolveTelemetry returns a TelemetryClient, emitting the one-time TTY
// fallback notice when the telemetry token is missing but the uptime token
// is available.
func resolveTelemetry(cmd *cobra.Command) (*client.TelemetryClient, error) {
	token, source, err := config.ResolveTelemetryToken()
	if err != nil {
		return nil, errs.Wrap(errs.ExitAuth, err)
	}
	if source == config.SourceUptimeFallback && isTTY() && !quietFlag(cmd) {
		w := cmd.ErrOrStderr()
		telemetryNoticeOnce.Do(func() {
			fmt.Fprintln(w, "note: BETTERSTACK_TELEMETRY_TOKEN unset; sent BETTERSTACK_API_TOKEN to telemetry.betterstack.com (works only for global tokens)")
		})
	}
	return client.NewTelemetry(token, Version), nil
}

// resolveUptime returns the uptime Client.
func resolveUptime() (*client.Client, error) {
	token, err := config.ResolveToken()
	if err != nil {
		return nil, errs.Wrap(errs.ExitAuth, err)
	}
	return client.New(token, Version), nil
}

// apiErrorExitCode maps *client.APIError into the exit-code taxonomy:
// 401/403 → ExitAuth, 4xx → ExitUpstream (unexpected), 5xx → ExitUpstream.
func apiErrorExitCode(err error) errs.ExitCode {
	apiErr, ok := err.(*client.APIError)
	if !ok {
		return errs.ExitGeneric
	}
	switch apiErr.StatusCode {
	case 401, 403:
		return errs.ExitAuth
	default:
		return errs.ExitUpstream
	}
}

// wrapAPIErr maps an API error to a CLIError carrying the right exit code.
func wrapAPIErr(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*errs.CLIError); ok {
		return err
	}
	return errs.Wrap(apiErrorExitCode(err), err)
}

// DEPENDENCIES_EXIST error emission. Both --json and human modes use the same
// fields; the shape switches on the --json flag.
type DependencyConflict struct {
	ResourceType   string
	ResourceID     string
	Count          string // e.g. "at least 1"
	InspectCommand string // literal command the user should run
}

// EmitDependencyConflict writes the error in the correct format for the
// current --json setting and returns a silent CLIError with exit 4.
func EmitDependencyConflict(cmd *cobra.Command, dc DependencyConflict) error {
	errW := cmd.ErrOrStderr()
	if jsonFlag(cmd) {
		payload := map[string]any{
			"error": map[string]any{
				"code":            "DEPENDENCIES_EXIST",
				"count":           dc.Count,
				"inspect_command": dc.InspectCommand,
				"resource_type":   dc.ResourceType,
				"resource_id":     dc.ResourceID,
			},
		}
		enc := json.NewEncoder(errW)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
	} else {
		fmt.Fprintf(errW,
			"error: cannot delete %s %s: %s dependent alert. Run `%s` to inspect, or pass --force to override.\n",
			dc.ResourceType, dc.ResourceID, dc.Count, dc.InspectCommand,
		)
	}
	return errs.SilentExit(errs.ExitDependencyConflict)
}

// RequireYes returns a user-input error when --yes was not passed.
func RequireYes(cmd *cobra.Command) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		return errs.New(errs.ExitUserInput, "refusing to perform destructive operation without --yes")
	}
	return nil
}
