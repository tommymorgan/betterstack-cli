package betterstack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/bodyfile"
	"github.com/tommymorgan/betterstack-cli/internal/errs"
	"github.com/tommymorgan/betterstack-cli/internal/output"
)

var policiesCmd = &cobra.Command{
	Use:   "policies",
	Short: "Manage BetterStack escalation policies (uptime host)",
	Long:  "Escalation policies route alerts through notification steps. Policies have nested structures (steps, per-step channels, on-call rotations) that shorthand flags can't express, so create/update require --body-file. Delete performs a refcount precheck and refuses if any logs-alerts reference the policy (non-atomic; concurrent creation races may leave orphans).",
}

var policiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List escalation policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveUptime()
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetInt("limit")
		results, err := c.ListPolicies(context.Background(), limit)
		if err != nil {
			return wrapAPIErr(err)
		}
		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderListEnvelope(w, results)
		}
		return output.RenderPoliciesTable(w, results)
	},
}

var policiesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single escalation policy by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveUptime()
		if err != nil {
			return err
		}
		result, err := c.GetPolicy(context.Background(), args[0])
		if err != nil {
			return wrapAPIErr(err)
		}
		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderJSON(w, result)
		}
		return output.RenderPolicyDetail(w, result)
	},
}

var policiesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an escalation policy from a JSON body file",
	Long:  "Create an escalation policy. Policies require --body-file because their nested step shape can't be expressed as shorthand flags. The --upsert flag is not supported for policies (upsert requires shorthand; policies require --body-file).",
	RunE: func(cmd *cobra.Command, _ []string) error {
		bodyFile, _ := cmd.Flags().GetString("body-file")
		upsert, _ := cmd.Flags().GetBool("upsert")

		if upsert {
			return errs.New(errs.ExitUserInput,
				"--upsert is not supported for policies: policies require --body-file (no shorthand exists), and the \"shorthand OR body\" rule precludes body-file upsert")
		}
		if bodyFile == "" {
			return errs.New(errs.ExitUserInput,
				"--body-file is required for policies create (no shorthand flags exist)")
		}

		body, err := bodyfile.Read(bodyFile, os.Stdin)
		if err != nil {
			return err
		}

		c, err := resolveUptime()
		if err != nil {
			return err
		}
		result, err := c.CreatePolicy(context.Background(), body)
		if err != nil {
			return wrapAPIErr(err)
		}
		return renderPolicyResult(cmd, result)
	},
}

var policiesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an escalation policy from a JSON body file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return errs.New(errs.ExitUserInput,
				"--body-file is required for policies update (no shorthand flags exist)")
		}
		body, err := bodyfile.Read(bodyFile, os.Stdin)
		if err != nil {
			return err
		}
		c, err := resolveUptime()
		if err != nil {
			return err
		}
		result, err := c.UpdatePolicy(context.Background(), args[0], body)
		if err != nil {
			return wrapAPIErr(err)
		}
		return renderPolicyResult(cmd, result)
	},
}

var policiesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an escalation policy (refuses if any logs-alerts reference it; --force bypasses)",
	Long: `Delete an escalation policy. Requires --yes to confirm.

By default, this paginates /api/v2/alerts on the telemetry host and refuses if any alert's escalation_target.policy_id matches; pass --force to skip the precheck.

The precheck is best-effort: a concurrent actor creating a dependent alert between the precheck and the DELETE will surface the dependency_appeared_after_precheck error prefix.`,
	Args: cobra.ExactArgs(1),
	RunE: runPoliciesDelete,
}

func runPoliciesDelete(cmd *cobra.Command, args []string) error {
	id := args[0]
	if err := RequireYes(cmd); err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")

	if !force {
		// Paginate alerts via the telemetry host, short-circuiting on first match.
		telemetry, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		found, err := telemetry.IterateAlerts(context.Background(), func(item json.RawMessage) bool {
			return alertReferencesPolicy(item, id)
		})
		if err != nil {
			return wrapAPIErr(err)
		}
		if found != nil {
			return EmitDependencyConflict(cmd, DependencyConflict{
				ResourceType:   "policy",
				ResourceID:     id,
				Count:          "at least 1",
				InspectCommand: fmt.Sprintf("logs-alerts list --policy-id %s", id),
			})
		}
	}

	uptime, err := resolveUptime()
	if err != nil {
		return err
	}
	if err := uptime.DeletePolicy(context.Background(), id); err != nil {
		return handleDeleteRace(cmd, err, fmt.Sprintf("logs-alerts list --policy-id %s", id))
	}
	return output.RenderDeleteEnvelope(cmd.OutOrStdout(), "policy", id, jsonFlag(cmd))
}

// alertReferencesPolicy reports whether a raw alert's escalation_target
// references the given policy ID. Uses string comparison on json.Number so
// both integer and string representations match.
func alertReferencesPolicy(raw json.RawMessage, policyID string) bool {
	var e alertEnvelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return false
	}
	return string(e.Attributes.EscalationTarget.PolicyID) == policyID
}

func renderPolicyResult(cmd *cobra.Command, result json.RawMessage) error {
	w := cmd.OutOrStdout()
	if jsonFlag(cmd) {
		return output.RenderJSON(w, result)
	}
	return output.RenderPolicyDetail(w, result)
}

func init() {
	policiesListCmd.Flags().Int("limit", 0, "Maximum number of results")
	for _, c := range []*cobra.Command{policiesCreateCmd, policiesUpdateCmd} {
		c.Flags().String("body-file", "", "Path to a JSON file with the full request body (or - for stdin)")
	}
	policiesCreateCmd.Flags().Bool("upsert", false, "Not supported for policies (will error)")
	policiesDeleteCmd.Flags().Bool("yes", false, "Confirm destructive operation")
	policiesDeleteCmd.Flags().Bool("force", false, "Skip dependency-refcount precheck")

	policiesCmd.AddCommand(policiesListCmd)
	policiesCmd.AddCommand(policiesGetCmd)
	policiesCmd.AddCommand(policiesCreateCmd)
	policiesCmd.AddCommand(policiesUpdateCmd)
	policiesCmd.AddCommand(policiesDeleteCmd)
	rootCmd.AddCommand(policiesCmd)
}
