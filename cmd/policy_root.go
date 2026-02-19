package cmd

import (
	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Policy-as-Code lifecycle management",
	Long: `Manage Azure Policy definitions, initiatives, assignments, and exemptions
following the Enterprise Policy as Code (EPAC) methodology.

Workflow stages:
  1. create    → Define policy in JSON, scaffold files
  2. test      → Deploy assignment in Audit mode (DoNotEnforce)
  3. verify    → Check compliance state, review non-compliant resources
  4. remediate → Create remediation tasks for existing resources
  5. deploy    → Switch to Deny/Enforce mode (Default enforcement)

All policy artifacts are stored in the policies/ directory and tracked
in policies/workflow.yaml for lifecycle state management.`,
}

func init() {
	rootCmd.AddCommand(policyCmd)

	policyCmd.AddCommand(policyCreateCmd)
	policyCmd.AddCommand(policyTestCmd)
	policyCmd.AddCommand(policyVerifyCmd)
	policyCmd.AddCommand(policyRemediateCmd)
	policyCmd.AddCommand(policyDeployCmd)
	policyCmd.AddCommand(policyStatusCmd)
	policyCmd.AddCommand(policyDiffCmd)

	policyCreateCmd.Flags().StringP("type", "t", "definition", "Policy type: definition, initiative, assignment, exemption")
	policyCreateCmd.Flags().StringP("name", "n", "", "Policy name (kebab-case)")
	policyCreateCmd.Flags().StringP("category", "c", "General", "Policy category")
	policyCreateCmd.Flags().StringP("scope", "s", "", "Assignment scope (management group name)")
	policyCreateCmd.Flags().String("initiative", "", "Initiative name (for assignment type)")

	policyTestCmd.Flags().StringP("name", "n", "", "Assignment name")
	_ = policyTestCmd.MarkFlagRequired("name")

	policyVerifyCmd.Flags().StringP("name", "n", "", "Assignment name")
	policyVerifyCmd.Flags().StringP("output", "o", "", "Output report path")
	_ = policyVerifyCmd.MarkFlagRequired("name")

	policyRemediateCmd.Flags().StringP("name", "n", "", "Assignment name")
	_ = policyRemediateCmd.MarkFlagRequired("name")

	policyDeployCmd.Flags().StringP("name", "n", "", "Assignment name")
	policyDeployCmd.Flags().Bool("force", false, "Skip compliance check before enforcing")
	_ = policyDeployCmd.MarkFlagRequired("name")
}
