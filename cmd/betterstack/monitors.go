package betterstack

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/client"
	"github.com/tommymorgan/betterstack-cli/internal/config"
	"github.com/tommymorgan/betterstack-cli/internal/output"
)

var monitorsCmd = &cobra.Command{
	Use:   "monitors",
	Short: "Manage BetterStack monitors",
}

var monitorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List monitors",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		params := client.MonitorListParams{
			Status: status,
		}

		c := client.New(token, Version)
		results, err := c.ListMonitors(context.Background(), params, limit)
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, results)
		}
		return output.RenderMonitorsTable(w, results)
	},
}

var monitorsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single monitor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		result, err := c.GetMonitor(context.Background(), args[0])
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, result)
		}

		return output.RenderMonitorDetail(w, result)
	},
}

func init() {
	monitorsListCmd.Flags().String("status", "", "Filter by status (up, down, paused, pending, maintenance, validating)")
	monitorsListCmd.Flags().Int("limit", 0, "Maximum number of results")

	monitorsCmd.AddCommand(monitorsListCmd)
	monitorsCmd.AddCommand(monitorsGetCmd)
	rootCmd.AddCommand(monitorsCmd)
}
