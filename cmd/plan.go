package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/exitcode"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Preview infrastructure changes across platform layers",
	Long: `Preview changes before applying.

Runs 'terraform plan' across platform layers in CAF dependency order:
  1. management-groups   (Resource Organization)
  2. identity            (Identity & Access)
  3. management          (Management & Monitoring)
  4. governance          (Azure Policies)
  5. connectivity        (Hub-Spoke or vWAN)

Use --layer to plan a single layer, or omit to plan all activated layers.

The plan summary can be saved to a file with --out for CI/CD PR comments.`,
	RunE: runPlan,
}

var (
	planLayer string
	planOut   string
)

func init() {
	planCmd.Flags().StringVar(&planLayer, "layer", "", "specific layer to plan (default: all activated)")
	planCmd.Flags().StringVar(&planLayer, "target", "", "alias for --layer")
	planCmd.Flags().StringVar(&planOut, "out", "", "write plan output summary to file")

	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, args []string) error {
	root, _ := filepath.Abs(repoRoot)

	if err := ensureTerraformInstalled(); err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	layers, err := resolveLocalLayers(root, planLayer)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	bold := color.New(color.Bold)
	bold.Fprintf(os.Stderr, "📐 Planning platform layers\n")
	fmt.Fprintf(os.Stderr, "   Layers: %s\n\n", strings.Join(layers, ", "))

	totalAdd, totalChange, totalDestroy := 0, 0, 0
	combined := strings.Builder{}

	for _, layer := range layers {
		dir := filepath.Join(root, "platform", layer)
		if _, initErr := runTerraformCmd(cmd.Context(), dir, "init", "-input=false", "-no-color"); initErr != nil {
			return exitcode.Wrap(exitcode.Terraform, fmt.Errorf("layer %s: terraform init failed", layer))
		}

		out, planErr := runTerraformCmd(cmd.Context(), dir, "plan", "-input=false", "-detailed-exitcode", "-no-color")
		add, change, destroy := parsePlanSummary(out)
		totalAdd += add
		totalChange += change
		totalDestroy += destroy

		combined.WriteString("## ")
		combined.WriteString(layer)
		combined.WriteString("\n")
		combined.WriteString(out)
		combined.WriteString("\n\n")

		if planErr != nil {
			// terraform plan detailed-exitcode returns 2 when changes are present.
			if strings.Contains(out, "Error:") {
				return exitcode.Wrap(exitcode.Terraform, fmt.Errorf("layer %s: terraform plan failed", layer))
			}
		}

		// Per-layer summary
		icon := "✅"
		if add > 0 || change > 0 || destroy > 0 {
			icon = "📝"
		}
		fmt.Fprintf(os.Stderr, "   %s %-20s +%d ~%d -%d\n", icon, layer, add, change, destroy)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "📊 Plan Summary:\n")
	fmt.Fprintf(os.Stderr, "   Resources to add    : %d\n", totalAdd)
	fmt.Fprintf(os.Stderr, "   Resources to change : %d\n", totalChange)
	fmt.Fprintf(os.Stderr, "   Resources to destroy: %d\n\n", totalDestroy)

	if totalDestroy > 0 {
		color.New(color.FgRed, color.Bold).Fprintln(os.Stderr, "   ⚠️  WARNING: Plan includes resource destruction!")
		fmt.Fprintln(os.Stderr)
	}

	if strings.TrimSpace(planOut) != "" {
		if writeErr := os.WriteFile(planOut, []byte(combined.String()), 0o644); writeErr != nil {
			return exitcode.Wrap(exitcode.Generic, fmt.Errorf("writing --out file: %w", writeErr))
		}
		fmt.Fprintf(os.Stderr, "📁 Plan output written to: %s\n\n", planOut)
	}

	color.New(color.FgGreen, color.Bold).Fprintln(os.Stderr, "✅ Plan complete. Review changes and run: lzctl apply")

	return nil
}
