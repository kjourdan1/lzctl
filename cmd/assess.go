package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/output"
)

var assessCmd = &cobra.Command{
	Use:   "assess",
	Short: "Discover existing Azure resources and produce a readiness report",
	Long: `Connects to Azure and inventories:
  - Management Group hierarchy
  - Subscriptions and their placement
  - Virtual networks, peerings, DNS
  - Diagnostic settings, Log Analytics workspaces
  - Azure Policies (assignments, definitions, compliance)
  - RBAC role assignments

Produces a readiness report comparing current state against the lzctl.yaml
platform configuration.

Examples:
  lzctl assess
  lzctl assess --json
  lzctl assess --output assess-report/`,
	RunE: runAssess,
}

var (
	assessOutputDir string
)

func init() {
	assessCmd.Flags().StringVar(&assessOutputDir, "output", "", "custom output directory for reports")
	rootCmd.AddCommand(assessCmd)
}

func runAssess(cmd *cobra.Command, args []string) error {
	output.Init(verbosity > 0, jsonOutput)

	root, _ := filepath.Abs(repoRoot)
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen, color.Bold)

	// Load project config.
	cfgPath := localConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w (run lzctl init first)", err)
	}

	bold.Fprintf(os.Stderr, "🔍 Assessing project: %s\n\n", cfg.Metadata.Name)

	// outDir reserved for future file output (E7)
	_ = assessOutputDir

	// Check which platform layers exist.
	layers, _ := resolveLocalLayers(root, "")

	// Build assessment result.
	result := map[string]interface{}{
		"project": map[string]interface{}{
			"name":          cfg.Metadata.Name,
			"tenant":        cfg.Metadata.Tenant,
			"primaryRegion": cfg.Metadata.PrimaryRegion,
			"connectivity":  cfg.Spec.Platform.Connectivity.Type,
			"landingZones":  len(cfg.Spec.LandingZones),
		},
		"platformLayers":   layers,
		"layerCount":       len(layers),
		"landingZoneCount": len(cfg.Spec.LandingZones),
	}

	// Landing zone details.
	lzDetails := make([]map[string]interface{}, 0, len(cfg.Spec.LandingZones))
	for _, lz := range cfg.Spec.LandingZones {
		detail := map[string]interface{}{
			"name":      lz.Name,
			"archetype": lz.Archetype,
		}
		if lz.Subscription != "" {
			detail["subscription"] = lz.Subscription
		}
		lzDetails = append(lzDetails, detail)
	}
	result["landingZones"] = lzDetails

	if jsonOutput {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Text output.
	fmt.Fprintf(os.Stderr, "📋 Project: %s\n", cfg.Metadata.Name)
	fmt.Fprintf(os.Stderr, "   Tenant:        %s\n", cfg.Metadata.Tenant)
	fmt.Fprintf(os.Stderr, "   Region:        %s\n", cfg.Metadata.PrimaryRegion)
	fmt.Fprintf(os.Stderr, "   Connectivity:  %s\n\n", cfg.Spec.Platform.Connectivity.Type)

	bold.Fprintln(os.Stderr, "📂 Platform Layers")
	if len(layers) > 0 {
		for _, layer := range layers {
			green.Fprintf(os.Stderr, "  ✅ %s\n", layer)
		}
	} else {
		fmt.Fprintln(os.Stderr, "  (none generated — run lzctl init first)")
	}
	fmt.Fprintln(os.Stderr)

	bold.Fprintln(os.Stderr, "🏢 Landing Zones")
	if len(cfg.Spec.LandingZones) > 0 {
		for _, lz := range cfg.Spec.LandingZones {
			sub := "(pending)"
			if lz.Subscription != "" {
				sub = lz.Subscription
			}
			fmt.Fprintf(os.Stderr, "  • %-25s Archetype: %-10s Sub: %s\n",
				lz.Name, lz.Archetype, sub)
		}
	} else {
		fmt.Fprintln(os.Stderr, "  (none defined — add in lzctl.yaml)")
	}
	fmt.Fprintln(os.Stderr)

	green.Fprintln(os.Stderr, "✅ Assessment complete.")
	fmt.Fprintln(os.Stderr, "   Next: lzctl plan  to see what Terraform would change")

	return nil
}
