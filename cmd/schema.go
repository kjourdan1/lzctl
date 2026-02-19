package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/exitcode"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Export or validate the lzctl.yaml JSON Schema",
	Long: `Schema tooling for the lzctl.yaml configuration file.

Examples:
  lzctl schema export                      # print schema to stdout
  lzctl schema export --output schema.json # write to file
  lzctl schema validate                    # validate lzctl.yaml against schema`,
}

var schemaExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the lzctl.yaml JSON Schema",
	RunE:  runSchemaExport,
}

var schemaValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate lzctl.yaml against the JSON Schema",
	RunE:  runSchemaValidate,
}

var schemaOutputFile string

func init() {
	schemaExportCmd.Flags().StringVarP(&schemaOutputFile, "output", "o", "", "write schema to file instead of stdout")

	schemaCmd.AddCommand(schemaExportCmd)
	schemaCmd.AddCommand(schemaValidateCmd)
	rootCmd.AddCommand(schemaCmd)
}

func runSchemaExport(cmd *cobra.Command, args []string) error {
	data := config.GetSchema()
	if len(data) == 0 {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("no embedded schema available"))
	}

	if schemaOutputFile != "" {
		root, _ := filepath.Abs(repoRoot)
		outPath := schemaOutputFile
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(root, outPath)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return exitcode.Wrap(exitcode.Validation, err)
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return exitcode.Wrap(exitcode.Validation, err)
		}
		color.New(color.FgGreen).Fprintf(os.Stderr, "✅ Schema written to %s\n", outPath)
		return nil
	}

	fmt.Println(string(data))
	return nil
}

func runSchemaValidate(cmd *cobra.Command, args []string) error {
	cfgPath := localConfigPath()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("load config: %w", err))
	}

	result, err := config.Validate(cfg)
	if err != nil {
		return exitcode.Wrap(exitcode.Validation, err)
	}
	if !result.Valid {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "❌ %s: %s\n", e.Field, e.Description)
		}
		return exitcode.Wrap(exitcode.Validation, fmt.Errorf("schema validation failed with %d error(s)", len(result.Errors)))
	}

	color.New(color.FgGreen, color.Bold).Fprintf(os.Stderr, "✅ %s is valid.\n", cfgPath)
	return nil
}
