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

var explorationsCmd = &cobra.Command{
	Use:   "explorations",
	Short: "Manage BetterStack log explorations (saved queries)",
	Long:  "Explorations are saved queries on the telemetry (Logs) host. An exploration carries the query; a logs-alert attached to it carries the firing condition.",
}

var explorationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List explorations",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetInt("limit")
		results, err := c.ListExplorations(context.Background(), limit)
		if err != nil {
			return wrapAPIErr(err)
		}
		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderListEnvelope(w, results)
		}
		return output.RenderExplorationsTable(w, results)
	},
}

var explorationsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single exploration by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		result, err := c.GetExploration(context.Background(), args[0])
		if err != nil {
			return wrapAPIErr(err)
		}
		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderJSON(w, result)
		}
		return output.RenderExplorationDetail(w, result)
	},
}

var explorationsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an exploration",
	Long: `Create an exploration via shorthand flags (--source-id, --pattern, --name) for a count-matching query, or via --body-file for a full payload.

The shorthand mode and --body-file are mutually exclusive.
The --upsert flag (shorthand only) is non-atomic: a concurrent actor may create a resource with the same name between the list and the POST, producing duplicates.`,
	RunE: runExplorationsCreate,
}

func runExplorationsCreate(cmd *cobra.Command, _ []string) error {
	bodyFile, _ := cmd.Flags().GetString("body-file")
	sourceID, _ := cmd.Flags().GetString("source-id")
	pattern, _ := cmd.Flags().GetString("pattern")
	name, _ := cmd.Flags().GetString("name")
	upsert, _ := cmd.Flags().GetBool("upsert")

	hasShorthand := sourceID != "" || pattern != "" || name != ""

	if bodyFile != "" && hasShorthand {
		return errs.New(errs.ExitUserInput,
			"--body-file and shorthand flags (--source-id/--pattern/--name) are mutually exclusive")
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
		result, err := c.CreateExploration(context.Background(), body)
		if err != nil {
			return wrapAPIErr(err)
		}
		return renderExplorationResult(cmd, result)
	}

	// Shorthand mode requires at minimum a name
	if name == "" {
		return errs.New(errs.ExitUserInput, "--name is required when using shorthand flags")
	}
	if sourceID == "" {
		return errs.New(errs.ExitUserInput, "--source-id is required when using shorthand flags")
	}
	if pattern == "" {
		return errs.New(errs.ExitUserInput, "--pattern is required when using shorthand flags")
	}

	shorthand := payload.ExplorationShorthand{
		Name: name, SourceID: sourceID, Pattern: pattern,
	}

	if upsert {
		return runExplorationsUpsert(cmd, c, shorthand)
	}

	body, err := payload.BuildExplorationCreate(shorthand)
	if err != nil {
		return err
	}
	result, err := c.CreateExploration(context.Background(), body)
	if err != nil {
		return wrapAPIErr(err)
	}
	return renderExplorationResult(cmd, result)
}

func renderExplorationResult(cmd *cobra.Command, result json.RawMessage) error {
	w := cmd.OutOrStdout()
	if jsonFlag(cmd) {
		return output.RenderJSON(w, result)
	}
	return output.RenderExplorationDetail(w, result)
}

var explorationsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an exploration",
	Long:  "Update via shorthand flags (sends PATCH with only provided fields) or --body-file (sends full file contents). Shorthand and --body-file are mutually exclusive.",
	Args:  cobra.ExactArgs(1),
	RunE:  runExplorationsUpdate,
}

func runExplorationsUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]
	bodyFile, _ := cmd.Flags().GetString("body-file")
	sourceID, _ := cmd.Flags().GetString("source-id")
	pattern, _ := cmd.Flags().GetString("pattern")
	name, _ := cmd.Flags().GetString("name")

	hasShorthand := sourceID != "" || pattern != "" || name != ""
	if bodyFile != "" && hasShorthand {
		return errs.New(errs.ExitUserInput,
			"--body-file and shorthand flags (--source-id/--pattern/--name) are mutually exclusive")
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
		body, err = payload.BuildExplorationPatch(payload.ExplorationShorthand{
			Name: name, SourceID: sourceID, Pattern: pattern,
		})
		if err != nil {
			return err
		}
	}

	result, err := c.UpdateExploration(context.Background(), id, body)
	if err != nil {
		return wrapAPIErr(err)
	}
	return renderExplorationResult(cmd, result)
}

var explorationsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an exploration (refuses if dependent alerts exist; --force bypasses)",
	Long: `Delete an exploration. Requires --yes to confirm.

By default, this runs a single refcount precheck via GET /api/v2/explorations/{id}/alerts?per_page=1 and refuses to delete if any dependent alerts exist. Pass --force to skip this precheck.

The precheck is best-effort: a concurrent actor may create a dependent alert between the precheck and the DELETE, producing a "dependency_appeared_after_precheck" error.`,
	Args: cobra.ExactArgs(1),
	RunE: runExplorationsDelete,
}

func runExplorationsDelete(cmd *cobra.Command, args []string) error {
	id := args[0]
	if err := RequireYes(cmd); err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")

	c, err := resolveTelemetry(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()

	if !force {
		alerts, err := c.ExplorationAlertsFirstPage(ctx, id, 1)
		if err != nil {
			return wrapAPIErr(err)
		}
		if len(alerts) > 0 {
			return EmitDependencyConflict(cmd, DependencyConflict{
				ResourceType:   "exploration",
				ResourceID:     id,
				Count:          "at least 1",
				InspectCommand: fmt.Sprintf("logs-alerts list --exploration-id %s", id),
			})
		}
	}

	if err := c.DeleteExploration(ctx, id); err != nil {
		return handleDeleteRace(cmd, err, fmt.Sprintf("logs-alerts list --exploration-id %s", id))
	}

	w := cmd.OutOrStdout()
	return output.RenderDeleteEnvelope(w, "exploration", id, jsonFlag(cmd))
}

// handleDeleteRace maps a 409/422 on DELETE (after a precheck passed) into
// the documented machine-matchable error prefix and exit code 4.
func handleDeleteRace(cmd *cobra.Command, err error, inspectCommand string) error {
	apiErr, ok := err.(*client.APIError)
	if ok && (apiErr.StatusCode == 409 || apiErr.StatusCode == 422) {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: dependency_appeared_after_precheck: %s\n", inspectCommand)
		return errs.SilentExit(errs.ExitDependencyConflict)
	}
	return wrapAPIErr(err)
}

func init() {
	// Shared flags for create/update
	for _, c := range []*cobra.Command{explorationsCreateCmd, explorationsUpdateCmd} {
		c.Flags().String("body-file", "", "Path to a JSON file with the full request body (or - for stdin)")
		c.Flags().String("source-id", "", "Log source ID (for shorthand count-matching exploration)")
		c.Flags().String("pattern", "", "Literal message substring to match (for shorthand)")
		c.Flags().String("name", "", "Exploration name (for shorthand)")
	}
	explorationsCreateCmd.Flags().Bool("upsert", false, "Create if no exploration with this name exists, otherwise PATCH the sole match (non-atomic)")

	explorationsListCmd.Flags().Int("limit", 0, "Maximum number of results")

	explorationsDeleteCmd.Flags().Bool("yes", false, "Confirm destructive operation")
	explorationsDeleteCmd.Flags().Bool("force", false, "Skip dependency-refcount precheck")

	explorationsCmd.AddCommand(explorationsListCmd)
	explorationsCmd.AddCommand(explorationsGetCmd)
	explorationsCmd.AddCommand(explorationsCreateCmd)
	explorationsCmd.AddCommand(explorationsUpdateCmd)
	explorationsCmd.AddCommand(explorationsDeleteCmd)
	rootCmd.AddCommand(explorationsCmd)
}
