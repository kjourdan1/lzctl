package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
)

var workloadAdoptCmd = &cobra.Command{
	Use:   "adopt",
	Short: "Adopt an existing subscription as a landing zone",
	Long: `Adds an existing subscription to lzctl.yaml under spec.landingZones.
The subscription will be brought under management via Terraform import.

Examples:
  lzctl workload adopt --name legacy-app --subscription <sub-id>
  lzctl workload adopt --name legacy-app --subscription <sub-id> --archetype corp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		subscriptionID, _ := cmd.Flags().GetString("subscription")
		archetype, _ := cmd.Flags().GetString("archetype")
		addressSpace, _ := cmd.Flags().GetString("address-space")
		connected, _ := cmd.Flags().GetBool("connected")
		tagList, _ := cmd.Flags().GetStringSlice("tag")

		// Validate inputs
		if err := validateWorkloadName(name); err != nil {
			return err
		}
		if err := validateArchetype(archetype); err != nil {
			return err
		}
		if err := validateAddressSpace(addressSpace); err != nil {
			return err
		}

		tags := parseTags(tagList)

		cfg, err := configCache()
		if err != nil {
			return fmt.Errorf("load config: %w (run lzctl init first)", err)
		}

		// Check for duplicates.
		for _, lz := range cfg.Spec.LandingZones {
			if lz.Name == name {
				return fmt.Errorf("landing zone %q already exists in lzctl.yaml", name)
			}
		}

		lz := config.LandingZone{
			Name:         name,
			Subscription: subscriptionID,
			Archetype:    archetype,
			AddressSpace: addressSpace,
			Connected:    connected,
			Tags:         tags,
		}

		if dryRun {
			color.Yellow("⚡ [DRY-RUN] Landing zone '%s' would be adopted (subscription: %s)", name, subscriptionID)
			return nil
		}

		cfg.Spec.LandingZones = append(cfg.Spec.LandingZones, lz)
		if err := config.Save(cfg, localConfigPath()); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		color.Green("✓ Landing zone '%s' adopted (subscription: %s)", name, subscriptionID)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Run: lzctl plan   (Terraform will generate import blocks)")
		fmt.Println("  2. Run: lzctl apply  (to bring under management)")
		return nil
	},
}
