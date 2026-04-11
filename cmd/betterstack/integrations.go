package betterstack

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/client"
	"github.com/tommymorgan/betterstack-cli/internal/config"
	"github.com/tommymorgan/betterstack-cli/internal/output"
)

var integrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "Manage BetterStack integrations",
}

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List integrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		integrationType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		results, err := c.ListIntegrations(context.Background(), integrationType, limit)
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, results)
		}
		return output.RenderIntegrationsTable(w, results)
	},
}

var integrationsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single integration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		integrationType, _ := cmd.Flags().GetString("type")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		result, err := c.GetIntegration(context.Background(), integrationType, args[0])
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, result)
		}
		return output.RenderIntegrationDetail(w, result)
	},
}

var integrationsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.ResolveToken()
		if err != nil {
			return err
		}

		integrationType, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		result, err := c.CreateIntegration(context.Background(), integrationType, name)
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		if jsonOutput {
			return output.RenderJSON(w, result)
		}
		return output.RenderIntegrationDetail(w, result)
	},
}

var integrationsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an integration",
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

		integrationType, _ := cmd.Flags().GetString("type")
		jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

		c := client.New(token, Version)
		if err := c.DeleteIntegration(context.Background(), integrationType, args[0]); err != nil {
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
	for _, cmd := range []*cobra.Command{integrationsListCmd, integrationsGetCmd, integrationsCreateCmd, integrationsDeleteCmd} {
		cmd.Flags().String("type", "", "Integration type (e.g., aws-cloudwatch)")
		cmd.MarkFlagRequired("type")
	}

	integrationsListCmd.Flags().Int("limit", 0, "Maximum number of results")
	integrationsCreateCmd.Flags().String("name", "", "Name for the integration")
	integrationsCreateCmd.MarkFlagRequired("name")
	integrationsDeleteCmd.Flags().Bool("yes", false, "Confirm destructive operation")

	integrationsCmd.AddCommand(integrationsListCmd)
	integrationsCmd.AddCommand(integrationsGetCmd)
	integrationsCmd.AddCommand(integrationsCreateCmd)
	integrationsCmd.AddCommand(integrationsDeleteCmd)
	rootCmd.AddCommand(integrationsCmd)
}
