package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
)

var workloadRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a landing zone from lzctl.yaml",
	Long: `Removes a landing zone entry from lzctl.yaml. This does NOT delete the
Azure subscription — it only removes the definition from the config file.

To clean up Azure resources, run plan + apply after removing.

Examples:
  lzctl workload remove --name old-app`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		cfgPath := localConfigPath()
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		found := false
		filtered := make([]config.LandingZone, 0, len(cfg.Spec.LandingZones))
		for _, lz := range cfg.Spec.LandingZones {
			if lz.Name == name {
				found = true
				continue
			}
			filtered = append(filtered, lz)
		}

		if !found {
			return fmt.Errorf("landing zone %q not found in lzctl.yaml", name)
		}

		if dryRun {
			color.Yellow("⚡ [DRY-RUN] Landing zone '%s' would be removed", name)
			return nil
		}

		cfg.Spec.LandingZones = filtered
		if err := config.Save(cfg, cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		color.Green("✓ Landing zone '%s' removed from lzctl.yaml", name)
		fmt.Println("\nThe Azure subscription has NOT been deleted.")
		fmt.Println("Next steps:")
		fmt.Println("  1. Run: lzctl plan   (to preview cleanup changes)")
		fmt.Println("  2. Run: lzctl apply  (to apply)")
		return nil
	},
}
