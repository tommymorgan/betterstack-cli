package betterstack

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/client"
	"github.com/tommymorgan/betterstack-cli/internal/config"
	"github.com/tommymorgan/betterstack-cli/internal/output"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage BetterStack sources (webhook integrations)",
}

var sourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		results, err := c.ListSources(context.Background(), limit)
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, results)
		}
		return output.RenderSourcesTable(w, results)
	},
}

var sourcesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		result, err := c.GetSource(context.Background(), args[0])
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, result)
		}
		return output.RenderSourceDetail(w, result)
	},
}

var sourcesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new source",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		sourceType, _ := cmd.Flags().GetString("type")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		result, err := c.CreateSource(context.Background(), name, sourceType)
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, result)
		}
		return output.RenderSourceDetail(w, result)
	},
}

var sourcesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			return fmt.Errorf("refusing to delete without confirmation; pass --yes to confirm")
		}

		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		if err := c.DeleteSource(context.Background(), args[0]); err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderDeleteConfirmationJSON(w, args[0])
		}
		return output.RenderDeleteConfirmation(w, args[0])
	},
}

func init() {
	sourcesListCmd.Flags().Int("limit", 0, "Maximum number of results")

	sourcesCreateCmd.Flags().String("name", "", "Name for the source")
	sourcesCreateCmd.Flags().String("type", "", "Source type (e.g., amazon_cloudwatch, datadog)")
	sourcesCreateCmd.MarkFlagRequired("name")
	sourcesCreateCmd.MarkFlagRequired("type")

	sourcesDeleteCmd.Flags().Bool("yes", false, "Confirm destructive operation")

	sourcesCmd.AddCommand(sourcesListCmd)
	sourcesCmd.AddCommand(sourcesGetCmd)
	sourcesCmd.AddCommand(sourcesCreateCmd)
	sourcesCmd.AddCommand(sourcesDeleteCmd)
	rootCmd.AddCommand(sourcesCmd)
}
