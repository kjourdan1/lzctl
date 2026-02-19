package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/output"
	lztemplate "github.com/kjourdan1/lzctl/internal/template"
	"github.com/kjourdan1/lzctl/internal/wizard"
)

var (
	addZoneName         string
	addZoneArchetype    string
	addZoneSubscription string
	addZoneAddressSpace string
	addZoneConnected    bool
	addZoneTags         []string
	addZoneForce        bool
)

var addZoneCmd = &cobra.Command{
	Use:   "add-zone",
	Short: "Add a new landing zone to the project",
	Long: `Adds a new landing zone entry to lzctl.yaml and generates the corresponding
Terraform configuration from the archetype template (corp, online, or sandbox).

The command performs address-space overlap detection against the hub network
and all existing landing zones before proceeding.

If run without flags, an interactive wizard guides you through the setup.

Examples:
  lzctl add-zone
  lzctl add-zone --name app-prod --archetype corp --subscription-id <uuid> --address-space 10.2.0.0/24
  lzctl add-zone --name sandbox-dev --archetype sandbox --address-space 10.99.0.0/24 --no-connect
  lzctl add-zone --name app-prod --archetype corp --subscription-id <uuid> --address-space 10.2.0.0/24 --dry-run`,
	RunE: runAddZone,
}

func init() {
	addZoneCmd.Flags().StringVar(&addZoneName, "name", "", "landing zone name")
	addZoneCmd.Flags().StringVar(&addZoneArchetype, "archetype", "corp", "archetype: corp, online, sandbox")
	addZoneCmd.Flags().StringVar(&addZoneSubscription, "subscription-id", "", "Azure subscription ID (UUID)")
	addZoneCmd.Flags().StringVar(&addZoneAddressSpace, "address-space", "", "CIDR address space (e.g. 10.2.0.0/24)")
	addZoneCmd.Flags().BoolVar(&addZoneConnected, "connected", true, "connect spoke to hub network")
	addZoneCmd.Flags().StringSliceVar(&addZoneTags, "tag", nil, "tags as key=value pairs (repeatable)")
	addZoneCmd.Flags().BoolVar(&addZoneForce, "force", false, "overwrite existing landing zone files")

	rootCmd.AddCommand(addZoneCmd)
}

func runAddZone(cmd *cobra.Command, args []string) error {
	output.Init(verbosity > 0, jsonOutput)

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	// Load existing config.
	cfgPath := filepath.Join(absRoot, "lzctl.yaml")
	if cfgFile != "" {
		cfgPath = cfgFile
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	var zone config.LandingZone

	// If name flag not provided, run interactive wizard.
	if addZoneName == "" {
		wizCfg, wizErr := wizard.NewAddZoneWizard(nil, cfg.Spec.LandingZones).Run()
		if wizErr != nil {
			if errors.Is(wizErr, wizard.ErrCancelled) {
				output.Warn("add-zone wizard cancelled")
				return nil
			}
			return fmt.Errorf("add-zone wizard: %w", wizErr)
		}
		zone = wizCfg.ToLandingZone()
	} else {
		zone = config.LandingZone{
			Name:         addZoneName,
			Archetype:    addZoneArchetype,
			Subscription: addZoneSubscription,
			AddressSpace: addZoneAddressSpace,
			Connected:    addZoneConnected,
			Tags:         parseTags(addZoneTags),
		}
	}

	// Validate archetype.
	arch := strings.ToLower(strings.TrimSpace(zone.Archetype))
	if arch == "" {
		arch = "corp"
	}
	validArchetypes := map[string]bool{"corp": true, "online": true, "sandbox": true}
	if !validArchetypes[arch] {
		return fmt.Errorf("invalid archetype %q; must be one of: corp, online, sandbox", zone.Archetype)
	}
	zone.Archetype = arch

	// Dry-run output.
	if dryRun {
		return addZoneDryRun(zone)
	}

	// Add zone to config (checks duplicate name).
	if err := config.AddLandingZone(cfg, zone); err != nil {
		if addZoneForce {
			output.Warn(fmt.Sprintf("overwriting existing zone: %s", err))
			// Remove existing and re-add.
			removeZone(cfg, zone.Name)
			_ = config.AddLandingZone(cfg, zone)
		} else {
			return fmt.Errorf("add zone: %w (use --force to overwrite)", err)
		}
	}

	// Cross-validate (IP overlap check).
	checks, _ := config.ValidateCross(cfg, absRoot)
	for _, c := range checks {
		if c.Status == "error" && strings.Contains(c.Name, "overlap") {
			return fmt.Errorf("validation failed: %s — %s", c.Name, c.Message)
		}
	}

	// Save updated config.
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Render landing zone templates.
	engine, err := lztemplate.NewEngine()
	if err != nil {
		return fmt.Errorf("creating template engine: %w", err)
	}

	files, err := engine.RenderZone(cfg, zone)
	if err != nil {
		return fmt.Errorf("rendering landing zone templates: %w", err)
	}

	writer := lztemplate.Writer{DryRun: false}
	written, err := writer.WriteAll(files, absRoot)
	if err != nil {
		return fmt.Errorf("writing landing zone files: %w", err)
	}

	// JSON output mode.
	if jsonOutput {
		output.JSON(map[string]interface{}{
			"status":    "ok",
			"zone":      zone,
			"files":     written,
			"archetype": arch,
		})
		return nil
	}

	// Human-readable output.
	color.Green("✓ Landing zone '%s' added successfully", zone.Name)
	fmt.Printf("\n  Archetype:     %s\n", zone.Archetype)
	fmt.Printf("  Subscription:  %s\n", zone.Subscription)
	fmt.Printf("  Address Space: %s\n", zone.AddressSpace)
	fmt.Printf("  Connected:     %v\n", zone.Connected)
	fmt.Println()
	fmt.Println("Generated files:")
	for _, f := range written {
		fmt.Printf("  • %s\n", f)
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Review generated config: landing-zones/%s/\n", lztemplate.Slugify(zone.Name))
	fmt.Printf("  2. Run: lzctl validate\n")
	fmt.Printf("  3. Run: lzctl plan\n")
	fmt.Printf("  4. Run: lzctl apply\n")
	return nil
}

func addZoneDryRun(zone config.LandingZone) error {
	if jsonOutput {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"status":  "dry-run",
			"zone":    zone,
			"message": "no files were written",
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	color.Yellow("⚡ [DRY-RUN] Landing zone '%s' would be added", zone.Name)
	fmt.Printf("\n  Archetype:     %s\n", zone.Archetype)
	fmt.Printf("  Subscription:  %s\n", zone.Subscription)
	fmt.Printf("  Address Space: %s\n", zone.AddressSpace)
	fmt.Printf("  Connected:     %v\n", zone.Connected)
	fmt.Println("\n  No config or Terraform files were written.")
	return nil
}

func removeZone(cfg *config.LZConfig, name string) {
	zones := make([]config.LandingZone, 0, len(cfg.Spec.LandingZones))
	for _, z := range cfg.Spec.LandingZones {
		if z.Name != name {
			zones = append(zones, z)
		}
	}
	cfg.Spec.LandingZones = zones
}
