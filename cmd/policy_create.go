package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/policy"
)

var policyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Scaffold a new policy definition or initiative",
	Long: `Creates a new policy definition or initiative JSON file from a template.
The file is placed in the correct directory under policies/.

Examples:
  lzctl policy create --type definition --name deny-rdp-internet --category Network
  lzctl policy create --type initiative --name compliance-baseline --category Security
  lzctl policy create --type assignment --name baseline-root --scope root-mg --initiative security-baseline`,
	RunE: func(cmd *cobra.Command, args []string) error {
		policyType, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")
		category, _ := cmd.Flags().GetString("category")
		scope, _ := cmd.Flags().GetString("scope")
		initiative, _ := cmd.Flags().GetString("initiative")
		root, err := absRepoRoot()
		if err != nil {
			return err
		}

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		opts := policy.CreateOpts{
			RepoRoot:   root,
			Type:       policyType,
			Name:       name,
			Category:   category,
			Scope:      scope,
			Initiative: initiative,
		}

		outPath, err := policy.Create(opts)
		if err != nil {
			return fmt.Errorf("policy create failed: %w", err)
		}

		color.Green("✓ Created %s: %s", policyType, outPath)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Edit the generated JSON file with your policy logic")
		fmt.Println("  2. Run: lzctl validate --all")
		fmt.Printf("  3. Run: lzctl policy test --name %s\n", name)
		return nil
	},
}
