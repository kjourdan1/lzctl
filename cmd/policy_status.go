package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/policy"
)

var policyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show policy workflow state for all assignments",
	Long: `Displays the current workflow state of all policy assignments,
including compliance data, remediation task status, and exemptions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := absRepoRoot()
		if err != nil {
			return err
		}

		opts := policy.StatusOpts{
			RepoRoot: root,
		}

		status, err := policy.Status(opts)
		if err != nil {
			return fmt.Errorf("policy status failed: %w", err)
		}

		bold := color.New(color.Bold)
		bold.Println("\n📊 Policy Workflow Status")
		fmt.Println(strings.Repeat("─", 72))

		bold.Printf("\nDefinitions: %d deployed\n", len(status.Definitions))
		for _, d := range status.Definitions {
			fmt.Printf("  ✓ %s\n", d.Name)
		}

		bold.Printf("\nInitiatives: %d deployed\n", len(status.Initiatives))
		for _, i := range status.Initiatives {
			fmt.Printf("  ✓ %s\n", i.Name)
		}

		bold.Printf("\nAssignments:\n")
		for _, a := range status.Assignments {
			stateColor := color.FgWhite
			stateIcon := "○"
			switch a.State {
			case "created":
				stateColor = color.FgWhite
				stateIcon = "○"
			case "test":
				stateColor = color.FgYellow
				stateIcon = "◐"
			case "verify":
				stateColor = color.FgCyan
				stateIcon = "◑"
			case "remediate":
				stateColor = color.FgMagenta
				stateIcon = "◕"
			case "deploy":
				stateColor = color.FgGreen
				stateIcon = "●"
			}

			c := color.New(stateColor)
			c.Printf("  %s %-35s [%s]\n", stateIcon, a.Name, a.State)
			fmt.Printf("    Scope: %s | Enforcement: %s\n", a.Scope, a.EnforcementMode)
			if a.Compliance.Evaluated > 0 {
				rate := float64(a.Compliance.Compliant) / float64(a.Compliance.Evaluated) * 100
				fmt.Printf("    Compliance: %.1f%% (%d/%d) | Non-Compliant: %d | Exempt: %d\n",
					rate, a.Compliance.Compliant, a.Compliance.Evaluated,
					a.Compliance.NonCompliant, a.Compliance.Exempt)
			}
		}

		if len(status.Exemptions) > 0 {
			bold.Printf("\nExemptions: %d active\n", len(status.Exemptions))
			for _, e := range status.Exemptions {
				fmt.Printf("  ⚠ %s (expires: %s)\n", e.Name, e.ExpiresOn)
				fmt.Printf("    Assignment: %s | Ticket: %s\n", e.Assignment, e.TicketRef)
			}
		}

		fmt.Println()
		return nil
	},
}
