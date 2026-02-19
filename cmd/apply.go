package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/exitcode"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply infrastructure changes across platform layers",
	Long: `Apply planned infrastructure changes.

Runs 'terraform apply' across platform layers in CAF dependency order:
  1. management-groups   (Resource Organization)
  2. identity            (Identity & Access)
  3. management          (Management & Monitoring)
  4. governance          (Azure Policies)
  5. connectivity        (Hub-Spoke or vWAN)

Use --layer to apply a single layer, or omit to apply all activated layers.
Use --auto-approve to skip the interactive confirmation (for CI/CD).`,
	RunE: runApply,
}

var (
	applyLayer       string
	applyAutoApprove bool
)

func init() {
	applyCmd.Flags().StringVar(&applyLayer, "layer", "", "specific layer to apply")
	applyCmd.Flags().StringVar(&applyLayer, "target", "", "alias for --layer")
	applyCmd.Flags().BoolVar(&applyAutoApprove, "auto-approve", false, "skip confirmation (CI only)")

	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	root, _ := filepath.Abs(repoRoot)

	if effectiveCIMode() && !applyAutoApprove && !dryRun {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("--ci mode requires --auto-approve for apply"))
	}

	if err := ensureTerraformInstalled(); err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	layers, err := resolveLocalLayers(root, applyLayer)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	// Interactive confirmation unless --auto-approve or --dry-run
	if !applyAutoApprove && !dryRun {
		yellow := color.New(color.FgYellow, color.Bold)
		yellow.Fprintln(os.Stderr, "⚠️  You are about to apply platform layer changes")
		fmt.Fprintf(os.Stderr, "   Layers: %s\n", strings.Join(layers, ", "))
		fmt.Fprintf(os.Stderr, "\n   Type 'yes' to proceed: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "yes" {
			fmt.Fprintln(os.Stderr, "\n❌ Apply canceled.")
			return nil
		}
		fmt.Fprintln(os.Stderr)
	}

	bold := color.New(color.Bold)
	bold.Fprintf(os.Stderr, "🚀 Applying platform layers\n")
	fmt.Fprintf(os.Stderr, "   Layers: %s\n\n", strings.Join(layers, ", "))

	for _, layer := range layers {
		dir := filepath.Join(root, "platform", layer)
		if _, initErr := runTerraformCmd(cmd.Context(), dir, "init", "-input=false", "-no-color"); initErr != nil {
			return exitcode.Wrap(exitcode.Terraform, fmt.Errorf("layer %s: terraform init failed", layer))
		}

		if dryRun {
			out, planErr := runTerraformCmd(cmd.Context(), dir, "plan", "-input=false", "-detailed-exitcode", "-no-color")
			add, change, destroy := parsePlanSummary(out)
			fmt.Fprintf(os.Stderr, "   ⚡ %-20s +%d ~%d -%d (dry-run)\n", layer, add, change, destroy)
			if planErr != nil && strings.Contains(out, "Error:") {
				return exitcode.Wrap(exitcode.Terraform, fmt.Errorf("layer %s: terraform plan failed", layer))
			}
			continue
		}

		if _, applyErr := runTerraformCmd(cmd.Context(), dir, "apply", "-auto-approve", "-input=false", "-no-color"); applyErr != nil {
			color.New(color.FgRed).Fprintf(os.Stderr, "   ❌ %-20s failed\n", layer)
			return exitcode.Wrap(exitcode.Terraform, fmt.Errorf("layer %s: terraform apply failed", layer))
		}
		color.New(color.FgGreen).Fprintf(os.Stderr, "   ✅ %-20s applied\n", layer)
	}

	fmt.Fprintln(os.Stderr)
	if dryRun {
		color.New(color.FgYellow, color.Bold).Fprintln(os.Stderr, "⚡ [DRY-RUN] Simulation complete. No infrastructure changes were applied.")
	} else {
		color.New(color.FgGreen, color.Bold).Fprintln(os.Stderr, "✅ Apply complete. Run: lzctl audit  to validate conformity.")
	}

	return nil
}
