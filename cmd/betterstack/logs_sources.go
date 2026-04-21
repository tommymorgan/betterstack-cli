package betterstack

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/output"
)

var logsSourcesCmd = &cobra.Command{
	Use:   "logs-sources",
	Short: "List and get log sources on the telemetry host",
	Long:  "Read-only access to log sources. Create/delete are intentionally out of scope; users run the BetterStack UI to provision sources.",
}

var logsSourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List log sources",
	RunE: func(cmd *cobra.Command, _ []string) error {
		c, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetInt("limit")
		results, err := c.ListSources(context.Background(), limit)
		if err != nil {
			return wrapAPIErr(err)
		}
		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderListEnvelope(w, results)
		}
		return output.RenderSourcesTable(w, results)
	},
}

var logsSourcesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single log source by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveTelemetry(cmd)
		if err != nil {
			return err
		}
		result, err := c.GetSource(context.Background(), args[0])
		if err != nil {
			return wrapAPIErr(err)
		}
		w := cmd.OutOrStdout()
		if jsonFlag(cmd) {
			return output.RenderJSON(w, result)
		}
		return output.RenderSourceDetail(w, result)
	},
}

func init() {
	logsSourcesListCmd.Flags().Int("limit", 0, "Maximum number of results")
	logsSourcesCmd.AddCommand(logsSourcesListCmd)
	logsSourcesCmd.AddCommand(logsSourcesGetCmd)
	rootCmd.AddCommand(logsSourcesCmd)
}
