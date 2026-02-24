package cmd

import (
	"fmt"
	"regexp"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
)

var (
	kebabCaseRegex    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	allowedArchetypes = []string{"corp", "online", "sandbox"}
)

var workloadAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new landing zone definition to lzctl.yaml",
	Long: `Adds a new landing zone entry to lzctl.yaml under spec.landingZones.
The entry defines a subscription to be vended via the AVM lz-vending module.

Examples:
  lzctl workload add --name app-frontend --archetype corp
  lzctl workload add --name app-frontend --archetype corp \
    --address-space 10.1.0.0/24 --tag env=prod --tag team=frontend`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
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

		cfgPath := localConfigPath()
		cfg, err := config.Load(cfgPath)
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
			Archetype:    archetype,
			AddressSpace: addressSpace,
			Connected:    connected,
			Tags:         tags,
		}

		if dryRun {
			color.Yellow("⚡ [DRY-RUN] Landing zone '%s' would be added", name)
			fmt.Printf("  Archetype:     %s\n", archetype)
			if addressSpace != "" {
				fmt.Printf("  Address Space: %s\n", addressSpace)
			}
			fmt.Printf("  Connected:     %v\n", connected)
			return nil
		}

		cfg.Spec.LandingZones = append(cfg.Spec.LandingZones, lz)
		if err := config.Save(cfg, cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		color.Green("✓ Landing zone '%s' added to lzctl.yaml", name)
		fmt.Printf("  Archetype: %s\n", archetype)
		if addressSpace != "" {
			fmt.Printf("  Address Space: %s\n", addressSpace)
		}
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Run: lzctl plan   (to preview Terraform changes)")
		fmt.Println("  2. Run: lzctl apply  (to deploy)")
		return nil
	},
}
