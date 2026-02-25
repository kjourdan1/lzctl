package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
)

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "List available CAF design-area profiles",
	Long: `Displays the catalog of available CAF profiles (platform layers)
that can be enabled in lzctl.yaml.

Available platform layers (AVM modules):
  resource-org         – Management Group hierarchy, subscription placement
  identity-access      – RBAC groups, role assignments, PIM/JIT
  management-logs      – Log Analytics, diagnostics, monitoring
  connectivity-hubspoke – Hub-spoke network topology
  connectivity-vwan    – Virtual WAN topology
  security             – Defender, Key Vaults, baselines
  governance           – Azure Policy assignments & remediations

Examples:
  lzctl select --list`,
	RunE: runSelect,
}

var selectList bool

func init() {
	selectCmd.Flags().BoolVar(&selectList, "list", false, "list available profiles")
	rootCmd.AddCommand(selectCmd)
}

// platformProfile describes a CAF design-area layer.
type platformProfile struct {
	Name        string
	Description string
}

var catalogProfiles = []platformProfile{
	{"resource-org", "Management Group hierarchy, subscription placement"},
	{"identity-access", "RBAC groups, role assignments, PIM/JIT"},
	{"management-logs", "Log Analytics, diagnostics, monitoring"},
	{"connectivity-hubspoke", "Hub-spoke network topology (AVM)"},
	{"connectivity-vwan", "Virtual WAN topology (AVM)"},
	{"security", "Defender for Cloud, Key Vaults, baselines"},
	{"governance", "Azure Policy assignments & remediations"},
	{"subscription-vending", "Landing zone subscription vending (AVM)"},
	{"policy-as-code", "EPAC policy definitions & initiatives"},
}

func runSelect(cmd *cobra.Command, args []string) error {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	// Show current project profile if lzctl.yaml exists.
	cfg, _ := configCache()

	bold.Fprintln(os.Stderr, "📋 Available CAF Platform Layers:")
	fmt.Fprintln(os.Stderr)
	for _, p := range catalogProfiles {
		active := " "
		if cfg != nil && isLayerActive(cfg, p.Name) {
			active = "✅"
		}
		fmt.Fprintf(os.Stderr, "  %s %-25s %s\n", active, p.Name, p.Description)
	}
	fmt.Fprintln(os.Stderr)

	if cfg != nil {
		green.Fprintf(os.Stderr, "  Connectivity type: %s\n", cfg.Spec.Platform.Connectivity.Type)
		fmt.Fprintf(os.Stderr, "  Landing zones:     %d\n\n", len(cfg.Spec.LandingZones))
	}

	fmt.Fprintln(os.Stderr, "To enable/disable layers, edit lzctl.yaml and run: lzctl plan")

	return nil
}

func isLayerActive(cfg *config.LZConfig, name string) bool {
	switch name {
	case "connectivity-hubspoke":
		return cfg.Spec.Platform.Connectivity.Type == "hub-spoke"
	case "connectivity-vwan":
		return cfg.Spec.Platform.Connectivity.Type == "vwan"
	case "resource-org":
		return cfg.Metadata.Tenant != ""
	case "governance":
		return len(cfg.Spec.Governance.Policies.Assignments) > 0
	default:
		return true // Core layers are always active
	}
}
