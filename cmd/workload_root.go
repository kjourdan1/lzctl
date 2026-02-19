package cmd

import "github.com/spf13/cobra"

var workloadCmd = &cobra.Command{
	Use:   "workload",
	Short: "Manage landing zone subscriptions in lzctl.yaml",
	Long: `Manage workload subscriptions within an Azure Landing Zone.

Landing zones are defined in lzctl.yaml under spec.landingZones.
Each landing zone maps to an AVM subscription-vending module invocation.

Sub-commands:
  add    → Add a new landing zone definition
  adopt  → Adopt an existing subscription as a landing zone
  list   → List all defined landing zones
  remove → Remove a landing zone definition`,
}

func init() {
	rootCmd.AddCommand(workloadCmd)

	workloadCmd.AddCommand(workloadAddCmd)
	workloadCmd.AddCommand(workloadAdoptCmd)
	workloadCmd.AddCommand(workloadListCmd)
	workloadCmd.AddCommand(workloadRemoveCmd)

	workloadAddCmd.Flags().StringP("name", "n", "", "Landing zone name (kebab-case)")
	workloadAddCmd.Flags().String("archetype", "corp", "Archetype: corp, online, sandbox")
	workloadAddCmd.Flags().String("address-space", "", "VNet address space (e.g. 10.1.0.0/24)")
	workloadAddCmd.Flags().Bool("connected", true, "Enable VNet peering to hub network")
	workloadAddCmd.Flags().StringSlice("tag", nil, "Tags in key=value format (repeatable)")
	_ = workloadAddCmd.MarkFlagRequired("name")

	workloadAdoptCmd.Flags().StringP("name", "n", "", "Landing zone name (kebab-case)")
	workloadAdoptCmd.Flags().String("subscription", "", "Existing Azure subscription ID")
	workloadAdoptCmd.Flags().String("archetype", "corp", "Archetype: corp, online, sandbox")
	workloadAdoptCmd.Flags().String("address-space", "", "VNet address space")
	workloadAdoptCmd.Flags().Bool("connected", true, "Enable VNet peering to hub network")
	workloadAdoptCmd.Flags().StringSlice("tag", nil, "Tags in key=value format (repeatable)")
	_ = workloadAdoptCmd.MarkFlagRequired("name")
	_ = workloadAdoptCmd.MarkFlagRequired("subscription")

	workloadListCmd.Flags().StringP("output", "o", "table", "Output format: table, yaml")

	workloadRemoveCmd.Flags().StringP("name", "n", "", "Landing zone name")
	_ = workloadRemoveCmd.MarkFlagRequired("name")
}
