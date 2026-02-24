package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/policy"
)

var policyDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Switch assignment to Default enforcement mode",
	Long: `Switches the specified policy assignment from DoNotEnforce (audit)
to Default (deny/enforce) mode. This is the final stage of the
Policy-as-Code workflow.

WARNING: This will enforce the policy and may block non-compliant
deployments. Ensure all remediation tasks have completed and
compliance rate is acceptable.

Updates the workflow state to 'deploy' in workflow.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		root, err := absRepoRoot()
		if err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")

		if dryRun {
			color.Yellow("⚡ [DRY-RUN] Assignment '%s' would be switched to Default enforcement", name)
			fmt.Printf("  Force:  %v\n", force)
			return nil
		}

		opts := policy.DeployOpts{
			RepoRoot: root,
			Name:     name,
			Force:    force,
		}

		err = policy.Deploy(opts)
		if err != nil {
			return fmt.Errorf("policy deploy failed: %w", err)
		}

		color.Green("✓ Assignment '%s' switched to Default enforcement", name)
		fmt.Println("\n  The policy is now enforcing compliance.")
		fmt.Println("  Non-compliant deployments will be denied.")
		return nil
	},
}
