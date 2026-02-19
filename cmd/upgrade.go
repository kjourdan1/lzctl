package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/output"
	"github.com/kjourdan1/lzctl/internal/upgrade"
)

var (
	upgradeApply  bool
	upgradeModule string
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check and apply Terraform module version upgrades",
	Long: `Scans all .tf files for module source/version pins, queries the
Terraform Registry for latest versions, and reports available upgrades.

By default, reports only. Use --apply to update version pins in-place.

Examples:
  lzctl upgrade                           # check all modules
  lzctl upgrade --module Azure/avm-res-network-virtualnetwork/azurerm
  lzctl upgrade --apply                   # apply all upgrades
  lzctl upgrade --apply --dry-run         # preview changes without writing
  lzctl upgrade --json                    # machine-readable output`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeApply, "apply", false, "apply version upgrades to .tf files")
	upgradeCmd.Flags().StringVar(&upgradeModule, "module", "", "check a specific module (namespace/name/provider)")

	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	output.Init(verbosity > 0, jsonOutput)

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	// Scan for module pins.
	pins, err := upgrade.ScanDirectory(absRoot)
	if err != nil {
		return fmt.Errorf("scanning modules: %w", err)
	}

	if len(pins) == 0 {
		if jsonOutput {
			output.JSON(map[string]interface{}{
				"status":  "ok",
				"modules": []interface{}{},
				"message": "no module pins found",
			})
			return nil
		}
		color.Yellow("No Terraform module pins found in %s", absRoot)
		return nil
	}

	// Filter by specific module if requested.
	if upgradeModule != "" {
		filtered := make([]upgrade.ModulePin, 0)
		for _, p := range pins {
			if p.Ref.String() == upgradeModule {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("module %q not found in any .tf files", upgradeModule)
		}
		pins = filtered
	}

	// Deduplicate modules (same module may appear in multiple files).
	uniquePins := deduplicatePins(pins)

	// Check for upgrades via registry.
	client := upgrade.NewClient()
	results := upgrade.CheckUpgrades(client, uniquePins)

	// Apply upgrades if requested.
	if upgradeApply {
		return applyUpgrades(pins, results)
	}

	// Output results.
	if jsonOutput {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"status":   "ok",
			"upgrades": results,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output.
	hasUpgrades := false
	for _, r := range results {
		if r.Error != "" {
			color.Yellow("⚠ %s: %s", r.Module, r.Error)
			continue
		}
		if r.UpgradeAvail {
			hasUpgrades = true
			color.Cyan("↑ %s: %s → %s", r.Module, r.CurrentVersion, r.LatestVersion)
		} else {
			color.Green("✓ %s: %s (up to date)", r.Module, r.CurrentVersion)
		}
	}

	if hasUpgrades {
		fmt.Println()
		color.Yellow("Run 'lzctl upgrade --apply' to update version pins.")
	}

	return nil
}

func applyUpgrades(allPins []upgrade.ModulePin, results []upgrade.UpgradeInfo) error {
	applied := 0
	for _, r := range results {
		if !r.UpgradeAvail || r.Error != "" {
			continue
		}

		// Find all pins for this module across files.
		for _, pin := range allPins {
			if pin.Ref.String() != r.Module.String() {
				continue
			}

			if dryRun {
				color.Yellow("⚡ [DRY-RUN] Would update %s in %s: %s → %s",
					pin.Ref, pin.FilePath, pin.Version, r.LatestVersion)
				applied++
				continue
			}

			if err := upgrade.UpdateVersionInFile(pin, r.LatestVersion); err != nil {
				color.Red("✗ Failed to update %s in %s: %v", pin.Ref, pin.FilePath, err)
				continue
			}

			color.Green("✓ Updated %s in %s: %s → %s",
				pin.Ref, pin.FilePath, pin.Version, r.LatestVersion)
			applied++
		}
	}

	if applied == 0 {
		color.Green("All modules are up to date.")
	} else if dryRun {
		fmt.Printf("\n%d version pin(s) would be updated.\n", applied)
	} else {
		fmt.Printf("\n%d version pin(s) updated. Run 'lzctl validate' to verify.\n", applied)
	}

	return nil
}

func deduplicatePins(pins []upgrade.ModulePin) []upgrade.ModulePin {
	seen := make(map[string]bool)
	unique := make([]upgrade.ModulePin, 0, len(pins))
	for _, p := range pins {
		key := p.Ref.String() + "@" + p.Version
		if !seen[key] {
			seen[key] = true
			unique = append(unique, p)
		}
	}
	return unique
}
