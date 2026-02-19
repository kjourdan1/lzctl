package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/exitcode"
	"github.com/kjourdan1/lzctl/internal/output"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate project configuration and Terraform layers",
	Long: `Runs a comprehensive validation suite:

  1. lzctl.yaml schema validation
  2. Cross-validation (referenced files, consistency checks)
  3. Terraform validate per platform layer (if terraform is installed)

Used in CI as the first gate before plan.`,
	RunE: runValidate,
}

var (
	validateStrict bool
)

func init() {
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "fail on warnings")

	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	_ = cmd
	root, _ := filepath.Abs(repoRoot)
	configPath := localConfigPath()

	output.Init(verbosity > 0, jsonOutput)

	cfg, err := config.Load(configPath)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("loading config %q: %w", configPath, err))
	}

	schemaResult, err := config.Validate(cfg)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("schema validation error: %w", err))
	}

	type check struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	checks := make([]check, 0, 24)

	if schemaResult.Valid {
		checks = append(checks, check{Name: "schema", Status: "pass", Message: "lzctl.yaml matches schema"})
	} else {
		for _, e := range schemaResult.Errors {
			checks = append(checks, check{Name: "schema", Status: "error", Message: fmt.Sprintf("%s: %s", e.Field, e.Description)})
		}
	}

	crossChecks, err := config.ValidateCross(cfg, root)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("cross validation failed: %w", err))
	}
	for _, c := range crossChecks {
		checks = append(checks, check{Name: c.Name, Status: c.Status, Message: c.Message})
	}

	if err := ensureTerraformInstalled(); err != nil {
		checks = append(checks, check{Name: "terraform", Status: "warning", Message: err.Error()})
	} else {
		layers, layerErr := resolveLocalLayers(root, "")
		if layerErr != nil {
			checks = append(checks, check{Name: "terraform-layers", Status: "warning", Message: layerErr.Error()})
		} else {
			for _, layer := range layers {
				dir := filepath.Join(root, "platform", layer)
				if !fileExistsLocal(filepath.Join(dir, "main.tf")) {
					checks = append(checks, check{Name: "terraform-" + layer, Status: "warning", Message: "skipped (no main.tf)"})
					continue
				}

				if _, initErr := runTerraformCmd(nil, dir, "init", "-backend=false", "-input=false", "-no-color"); initErr != nil {
					checks = append(checks, check{Name: "terraform-" + layer, Status: "error", Message: "terraform init failed"})
					continue
				}
				if _, valErr := runTerraformCmd(nil, dir, "validate", "-no-color"); valErr != nil {
					checks = append(checks, check{Name: "terraform-" + layer, Status: "error", Message: "terraform validate failed"})
				} else {
					checks = append(checks, check{Name: "terraform-" + layer, Status: "pass", Message: "terraform validate passed"})
				}
			}
		}
	}

	errorsCount := 0
	warningsCount := 0
	for _, c := range checks {
		switch c.Status {
		case "error":
			errorsCount++
		case "warning":
			warningsCount++
		}
	}

	if jsonOutput {
		output.JSON(map[string]interface{}{
			"config":   configPath,
			"checks":   checks,
			"errors":   errorsCount,
			"warnings": warningsCount,
		})
	} else {
		fmt.Fprintf(os.Stderr, "🔎 Validating: %s\n\n", configPath)
		for _, c := range checks {
			icon := "✅"
			if c.Status == "warning" {
				icon = "⚠️"
			}
			if c.Status == "error" {
				icon = "❌"
			}
			fmt.Fprintf(os.Stderr, "  %s %s: %s\n", icon, c.Name, c.Message)
		}
		fmt.Fprintln(os.Stderr)
	}

	if errorsCount > 0 {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("%d validation error(s) found", errorsCount))
	}
	if warningsCount > 0 && validateStrict {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("%d warning(s) found (strict mode)", warningsCount))
	}

	if !jsonOutput {
		color.New(color.FgGreen, color.Bold).Fprintf(os.Stderr, "✅ Validation passed (%d checks, %d warnings)\n", len(checks), warningsCount)
	}

	return nil
}
