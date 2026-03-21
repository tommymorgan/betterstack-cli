package betterstack

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/client"
	"github.com/tommymorgan/betterstack-cli/internal/config"
	"github.com/tommymorgan/betterstack-cli/internal/output"
)

var incidentsCmd = &cobra.Command{
	Use:   "incidents",
	Short: "Manage BetterStack incidents",
}

var incidentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List incidents",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		monitorID, _ := cmd.Flags().GetString("monitor-id")
		resolved, _ := cmd.Flags().GetBool("resolved")
		resolvedSet := cmd.Flags().Changed("resolved")
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		params := client.IncidentListParams{
			From:      from,
			To:        to,
			MonitorID: monitorID,
		}
		if resolvedSet {
			params.Resolved = &resolved
		}

		c := client.New(token, Version)
		results, err := c.ListIncidents(context.Background(), params, limit)
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, results)
		}
		return output.RenderIncidentsTable(w, results)
	},
}

var incidentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		result, err := c.GetIncident(context.Background(), args[0])
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, result)
		}

		return output.RenderIncidentDetail(w, result)
	},
}

func init() {
	incidentsListCmd.Flags().String("from", "", "Start date (YYYY-MM-DD)")
	incidentsListCmd.Flags().String("to", "", "End date (YYYY-MM-DD)")
	incidentsListCmd.Flags().String("monitor-id", "", "Filter by monitor ID")
	incidentsListCmd.Flags().Bool("resolved", false, "Filter by resolution status")
	incidentsListCmd.Flags().Int("limit", 0, "Maximum number of results")

	incidentsCmd.AddCommand(incidentsListCmd)
	incidentsCmd.AddCommand(incidentsGetCmd)
	rootCmd.AddCommand(incidentsCmd)
}
