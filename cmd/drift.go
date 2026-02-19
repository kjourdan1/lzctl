package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/exitcode"
)

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect configuration drift between desired state and actual Azure state",
	Long: `Runs 'terraform plan' across platform layers to detect drift
(manual changes made outside of IaC).

Reports per-layer:
  - Resources to add (missing from Azure)
  - Resources to change (modified outside Terraform)
  - Resources to destroy (unexpected in state)

Use --layer to check a single layer, or omit to check all.
Use --json for machine-readable output.`,
	RunE: runDrift,
}

var (
	driftLayer string
)

func init() {
	driftCmd.Flags().StringVar(&driftLayer, "layer", "", "specific layer to check")

	rootCmd.AddCommand(driftCmd)
}

func runDrift(cmd *cobra.Command, args []string) error {
	root, _ := filepath.Abs(repoRoot)

	if err := ensureTerraformInstalled(); err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	layers, err := resolveLocalLayers(root, driftLayer)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	bold := color.New(color.Bold)
	if !jsonOutput {
		bold.Fprintf(os.Stderr, "🔄 Detecting drift across platform layers\n\n")
	}

	type layerDrift struct {
		Layer   string `json:"layer"`
		Add     int    `json:"add"`
		Change  int    `json:"change"`
		Destroy int    `json:"destroy"`
		Total   int    `json:"total"`
		Error   string `json:"error,omitempty"`
	}
	results := make([]layerDrift, 0, len(layers))
	totalDrift := 0

	for _, layer := range layers {
		dir := filepath.Join(root, "platform", layer)
		ld := layerDrift{Layer: layer}

		if _, initErr := runTerraformCmd(cmd.Context(), dir, "init", "-input=false", "-no-color"); initErr != nil {
			ld.Error = "terraform init failed"
			results = append(results, ld)
			if !jsonOutput {
				color.New(color.FgRed).Fprintf(os.Stderr, "   ❌ %-20s init failed\n", layer)
			}
			continue
		}

		out, planErr := runTerraformCmd(cmd.Context(), dir, "plan", "-input=false", "-detailed-exitcode", "-no-color")
		add, change, destroy := parsePlanSummary(out)
		ld.Add = add
		ld.Change = change
		ld.Destroy = destroy
		ld.Total = add + change + destroy
		totalDrift += ld.Total

		if planErr != nil && strings.Contains(out, "Error:") {
			ld.Error = "terraform plan failed"
		}

		results = append(results, ld)

		if !jsonOutput {
			if ld.Total == 0 {
				color.New(color.FgGreen).Fprintf(os.Stderr, "   ✅ %-20s no drift\n", layer)
			} else {
				color.New(color.FgYellow).Fprintf(os.Stderr, "   ⚠️  %-20s +%d ~%d -%d\n", layer, add, change, destroy)
			}
		}
	}

	if jsonOutput {
		status := "ok"
		if totalDrift > 0 {
			status = "drift-detected"
		}
		data, _ := json.MarshalIndent(map[string]interface{}{
			"status":     status,
			"totalDrift": totalDrift,
			"layers":     results,
		}, "", "  ")
		fmt.Fprintln(os.Stdout, string(data))
	} else {
		fmt.Fprintln(os.Stderr)
		if totalDrift == 0 {
			color.New(color.FgGreen, color.Bold).Fprintln(os.Stderr, "✅ No drift detected!")
		} else {
			color.New(color.FgYellow, color.Bold).Fprintf(os.Stderr,
				"⚠️  %d drift item(s) detected across %d layer(s)\n", totalDrift, len(layers))
			fmt.Fprintln(os.Stderr, "   Run: lzctl apply  to reconcile drift")
		}
	}

	if totalDrift > 0 {
		return exitcode.Wrap(exitcode.Drift, fmt.Errorf("%d drift item(s) detected", totalDrift))
	}

	return nil
}
