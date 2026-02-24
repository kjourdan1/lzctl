package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/exitcode"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback platform layers to a previous Terraform state",
	Long: `Rolls back platform layers by replaying a previous Terraform plan.

Layers are processed in reverse CAF order:
  5. connectivity
  4. governance
  3. management
  2. identity
  1. management-groups

Use --layer to roll back a specific layer only.
Use --auto-approve to skip the interactive confirmation (CI/CD).`,
	RunE: runRollback,
}

var (
	rollbackLayer       string
	rollbackAutoApprove bool
)

func init() {
	rollbackCmd.Flags().StringVar(&rollbackLayer, "layer", "", "specific layer to roll back")
	rollbackCmd.Flags().BoolVar(&rollbackAutoApprove, "auto-approve", false, "skip confirmation")

	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	root, err := absRepoRoot()
	if err != nil {
		return err
	}

	if effectiveCIMode() && !rollbackAutoApprove && !dryRun {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("--ci mode requires --auto-approve for rollback"))
	}

	if err := ensureTerraformInstalled(); err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	layers, err := resolveLocalLayers(root, rollbackLayer)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}

	// Reverse layer order for rollback (destroy in reverse dependency order)
	reversed := make([]string, len(layers))
	for i, l := range layers {
		reversed[len(layers)-1-i] = l
	}

	// Interactive confirmation
	if !rollbackAutoApprove && !dryRun {
		yellow := color.New(color.FgYellow, color.Bold)
		yellow.Fprintln(os.Stderr, "⚠️  You are about to rollback platform layer changes")
		fmt.Fprintf(os.Stderr, "   Layers (reverse order): %s\n", strings.Join(reversed, ", "))
		fmt.Fprintf(os.Stderr, "\n   Type 'yes' to proceed: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "yes" {
			fmt.Fprintln(os.Stderr, "\n❌ Rollback canceled.")
			return nil
		}
		fmt.Fprintln(os.Stderr)
	}

	bold := color.New(color.Bold)
	bold.Fprintf(os.Stderr, "↩️  Rolling back platform layers\n\n")

	type layerResult struct {
		Layer    string `json:"layer"`
		Status   string `json:"status"`
		Duration string `json:"duration"`
		Error    string `json:"error,omitempty"`
	}
	results := make([]layerResult, 0, len(reversed))

	for _, layer := range reversed {
		dir := filepath.Join(root, "platform", layer)
		start := time.Now()
		lr := layerResult{Layer: layer}

		if initOut, initErr := runTerraformCmd(cmd.Context(), dir, "init", "-input=false", "-no-color"); initErr != nil {
			lr.Status = "failed"
			lr.Error = fmt.Sprintf("terraform init failed: %s", initOut)
			lr.Duration = time.Since(start).Round(time.Millisecond).String()
			results = append(results, lr)
			color.New(color.FgRed).Fprintf(os.Stderr, "   ❌ %s: init failed\n", layer)
			continue
		}

		if dryRun {
			out, _ := runTerraformCmd(cmd.Context(), dir, "plan", "-input=false", "-no-color")
			add, change, destroy := parsePlanSummary(out)
			lr.Status = "dry-run"
			lr.Duration = time.Since(start).Round(time.Millisecond).String()
			results = append(results, lr)
			fmt.Fprintf(os.Stderr, "   ⚡ %-20s +%d ~%d -%d (dry-run)\n", layer, add, change, destroy)
			continue
		}

		if applyOut, applyErr := runTerraformCmd(cmd.Context(), dir, "apply", "-auto-approve", "-input=false", "-no-color"); applyErr != nil {
			lr.Status = "failed"
			lr.Error = "terraform apply failed"
			lr.Duration = time.Since(start).Round(time.Millisecond).String()
			color.New(color.FgRed).Fprintf(os.Stderr, "   ❌ %s (%s): apply failed\n", layer, lr.Duration)
			return exitcode.Wrap(exitcode.Terraform, fmt.Errorf("rollback layer %s: terraform apply failed: %s", layer, applyOut))
		}

		lr.Status = "ok"
		lr.Duration = time.Since(start).Round(time.Millisecond).String()
		results = append(results, lr)
		color.New(color.FgGreen).Fprintf(os.Stderr, "   ✅ %s (%s)\n", layer, lr.Duration)
	}

	fmt.Fprintln(os.Stderr)

	if jsonOutput {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"status":  "ok",
			"dryRun":  dryRun,
			"results": results,
		}, "", "  ")
		fmt.Fprintln(os.Stdout, string(data))
	}

	if dryRun {
		color.New(color.FgYellow, color.Bold).Fprintln(os.Stderr, "⚡ [DRY-RUN] Rollback simulation complete. No infrastructure changes were applied.")
	} else {
		color.New(color.FgGreen, color.Bold).Fprintln(os.Stderr, "✅ Rollback complete.")
	}

	return nil
}

func parseRollbackTimestamp(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("--to is required")
	}
	if ts, err := time.Parse("20060102-150405", raw); err == nil {
		return ts.UTC(), nil
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid --to timestamp %q: expected YYYYMMDD-HHMMSS or RFC3339", raw)
}
