package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/audit"
	"github.com/kjourdan1/lzctl/internal/azure"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Analyze current Azure tenant against CAF baseline",
	Long: `Scans the current Azure context and computes a CAF gap analysis.

Outputs:
  - Markdown report (default)
  - JSON report with --json

The command is read-only and does not modify Azure resources.`,
	RunE: runAudit,
}

var (
	auditScope  string
	auditOutput string
)

func init() {
	auditCmd.Flags().StringVar(&auditScope, "scope", "", "optional management group scope")
	auditCmd.Flags().StringVar(&auditOutput, "output", "", "optional output file path")

	rootCmd.AddCommand(auditCmd)
}

func runAudit(cmd *cobra.Command, args []string) error {
	_ = args
	root, err := absRepoRoot()
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)

	bold.Fprintln(os.Stderr, "🔍 Running Azure audit...")

	scanner := azure.NewScanner(nil, auditScope)
	snapshot, warnings, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("audit scan failed: %w", err)
	}

	engine := audit.NewComplianceEngine()
	report := engine.Evaluate(snapshot)

	if len(warnings) > 0 {
		yellow.Fprintf(os.Stderr, "⚠️  %d scan warning(s) encountered\n", len(warnings))
	}

	fmt.Fprintf(os.Stderr, "\n📊 CAF Alignment Score: %d/100\n", report.Score.Overall)
	fmt.Fprintf(os.Stderr, "   Findings: critical=%d high=%d medium=%d low=%d\n",
		report.Summary.Critical,
		report.Summary.High,
		report.Summary.Medium,
		report.Summary.Low,
	)

	if jsonOutput {
		payload, marshalErr := audit.RenderJSON(report)
		if marshalErr != nil {
			return fmt.Errorf("rendering JSON report: %w", marshalErr)
		}

		if auditOutput != "" {
			path := auditOutput
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}
			if writeErr := os.WriteFile(path, payload, 0o644); writeErr != nil {
				return fmt.Errorf("writing audit report: %w", writeErr)
			}
			green.Fprintf(os.Stderr, "\n✅ JSON report written: %s\n", path)
			return nil
		}

		fmt.Fprintln(os.Stdout, string(payload))
		return nil
	}

	markdown := audit.RenderMarkdown(report)
	if auditOutput != "" {
		path := auditOutput
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if writeErr := os.WriteFile(path, []byte(markdown), 0o644); writeErr != nil {
			return fmt.Errorf("writing audit report: %w", writeErr)
		}
		green.Fprintf(os.Stderr, "\n✅ Markdown report written: %s\n", path)
		return nil
	}

	fmt.Fprintln(os.Stdout, markdown)
	green.Fprintln(os.Stderr, "\n✅ Audit complete")
	return nil
}
