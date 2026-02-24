package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/policy"
)

var policyDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local policy definitions with deployed state",
	Long: `Compares local policy definitions, initiatives, and assignments in the
repository with the deployed state in Azure. Shows which policies need
to be created, updated, or deleted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := absRepoRoot()
		if err != nil {
			return err
		}

		opts := policy.DiffOpts{
			RepoRoot: root,
		}

		diff, err := policy.Diff(opts)
		if err != nil {
			return fmt.Errorf("policy diff failed: %w", err)
		}

		bold := color.New(color.Bold)
		bold.Println("\n📊 Policy Diff: Local vs Deployed")
		fmt.Println(strings.Repeat("─", 60))

		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)
		yellow := color.New(color.FgYellow)

		if len(diff.ToCreate) > 0 {
			green.Printf("\n+ To Create (%d):\n", len(diff.ToCreate))
			for _, p := range diff.ToCreate {
				green.Printf("  + %s (%s)\n", p.Name, p.Type)
			}
		}

		if len(diff.ToUpdate) > 0 {
			yellow.Printf("\n~ To Update (%d):\n", len(diff.ToUpdate))
			for _, p := range diff.ToUpdate {
				yellow.Printf("  ~ %s (%s)\n", p.Name, p.Type)
			}
		}

		if len(diff.ToDelete) > 0 {
			red.Printf("\n- To Delete (%d):\n", len(diff.ToDelete))
			for _, p := range diff.ToDelete {
				red.Printf("  - %s (%s)\n", p.Name, p.Type)
			}
		}

		if len(diff.Unchanged) > 0 {
			fmt.Printf("\n= Unchanged (%d)\n", len(diff.Unchanged))
		}

		if len(diff.ToCreate) == 0 && len(diff.ToUpdate) == 0 && len(diff.ToDelete) == 0 {
			color.Green("\n✓ Local and deployed state are in sync")
		}
		return nil
	},
}
