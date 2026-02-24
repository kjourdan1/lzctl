package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/policy"
)

var policyRemediateCmd = &cobra.Command{
	Use:   "remediate",
	Short: "Create remediation tasks for non-compliant resources",
	Long: `Creates Azure Policy remediation tasks for resources that are
non-compliant with the specified assignment. Remediation tasks will
bring existing resources into compliance.

Only policies with 'deployIfNotExists' or 'modify' effects can be remediated.
Policies with 'audit' or 'deny' effects require manual remediation.

Updates the workflow state to 'remediate' in workflow.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		root, err := absRepoRoot()
		if err != nil {
			return err
		}

		opts := policy.RemediateOpts{
			RepoRoot: root,
			Name:     name,
			DryRun:   dryRun,
		}

		result, err := policy.Remediate(opts)
		if err != nil {
			return fmt.Errorf("policy remediate failed: %w", err)
		}

		if dryRun {
			color.Yellow("⚡ Dry run – remediation tasks not created")
			fmt.Printf("\n  Would create %d remediation tasks\n", result.TaskCount)
		} else {
			color.Green("✓ Created %d remediation tasks", result.TaskCount)
		}

		for _, task := range result.Tasks {
			fmt.Printf("\n  Task: %s\n", task.Name)
			fmt.Printf("    Policy:    %s\n", task.PolicyName)
			fmt.Printf("    Resources: %d\n", task.ResourceCount)
			fmt.Printf("    Status:    %s\n", task.Status)
		}

		if !dryRun {
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Monitor remediation tasks in Azure Portal")
			fmt.Printf("  2. Run: lzctl policy verify --name %s  (after tasks complete)\n", name)
			fmt.Printf("  3. Run: lzctl policy deploy --name %s  (when compliant)\n", name)
		}

		return nil
	},
}
