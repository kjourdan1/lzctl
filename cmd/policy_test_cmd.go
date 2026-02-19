package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/policy"
)

var policyTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Deploy policy assignment in audit mode (DoNotEnforce)",
	Long: `Deploys the specified policy assignment with enforcementMode=DoNotEnforce.
This allows you to see compliance data without blocking any deployments.

The assignment's workflow state is updated to 'test' in workflow.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		root, _ := filepath.Abs(repoRoot)

		opts := policy.TestOpts{
			RepoRoot: root,
			Name:     name,
			DryRun:   dryRun,
		}

		result, err := policy.Test(opts)
		if err != nil {
			return fmt.Errorf("policy test failed: %w", err)
		}

		if dryRun {
			color.Yellow("⚡ Dry run – no changes applied")
		} else {
			color.Green("✓ Assignment '%s' deployed in DoNotEnforce mode", name)
		}

		fmt.Printf("\n  Scope:      %s\n", result.Scope)
		fmt.Printf("  Initiative: %s\n", result.Initiative)
		fmt.Printf("  State:      test (audit-only)\n")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Wait 24h for compliance data to populate")
		fmt.Printf("  2. Run: lzctl policy verify --name %s\n", name)
		return nil
	},
}
