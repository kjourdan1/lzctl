package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/policy"
)

var policyVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Check compliance state and generate report",
	Long: `Queries Azure Policy compliance state for the specified assignment
and generates a compliance report. Shows non-compliant resources grouped
by policy definition.

Updates the workflow state to 'verify' in workflow.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		root, err := absRepoRoot()
		if err != nil {
			return err
		}
		outputFile, _ := cmd.Flags().GetString("output")

		opts := policy.VerifyOpts{
			RepoRoot: root,
			Name:     name,
			Output:   outputFile,
		}

		report, err := policy.Verify(opts)
		if err != nil {
			return fmt.Errorf("policy verify failed: %w", err)
		}

		bold := color.New(color.Bold)
		bold.Printf("\n📋 Compliance Report: %s\n\n", name)

		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)
		yellow := color.New(color.FgYellow)

		fmt.Printf("  Evaluated:     %d resources\n", report.Evaluated)
		green.Printf("  Compliant:     %d\n", report.Compliant)
		if report.NonCompliant > 0 {
			red.Printf("  Non-Compliant: %d\n", report.NonCompliant)
		} else {
			green.Printf("  Non-Compliant: %d\n", report.NonCompliant)
		}
		if report.Exempt > 0 {
			yellow.Printf("  Exempt:        %d\n", report.Exempt)
		}

		var complianceRate float64
		if report.Evaluated > 0 {
			complianceRate = float64(report.Compliant) / float64(report.Evaluated) * 100
		}
		fmt.Printf("\n  Compliance Rate: %.1f%%\n", complianceRate)

		if report.NonCompliant > 0 {
			fmt.Println("\nNon-compliant resources by policy:")
			for _, group := range report.NonCompliantGroups {
				red.Printf("  ✗ %s (%d resources)\n", group.PolicyName, group.Count)
				for _, res := range group.Resources {
					fmt.Printf("    - %s\n", res)
				}
			}
			fmt.Println("\nNext steps:")
			fmt.Printf("  1. Review non-compliant resources above\n")
			fmt.Printf("  2. Run: lzctl policy remediate --name %s\n", name)
		} else {
			green.Println("\n✓ All resources are compliant!")
			fmt.Println("\nNext steps:")
			fmt.Printf("  Run: lzctl policy deploy --name %s\n", name)
		}

		if outputFile != "" {
			color.Cyan("\n  Report saved to: %s", outputFile)
		}

		return nil
	},
}
