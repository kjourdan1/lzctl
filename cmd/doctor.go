package cmd

import (
	"context"
	"os"

	"github.com/kjourdan1/lzctl/internal/doctor"
	"github.com/kjourdan1/lzctl/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check prerequisites and environment readiness",
	Long: `Verify that all required tools (terraform, az, git), Azure session,
and resource providers are correctly configured.

Each check reports ✅ (pass), ❌ (fail), or ⚠️ (warning) with an
actionable fix suggestion.

Exit code 0 if all critical checks pass, 1 otherwise.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	output.Init(verbosity > 0, jsonOutput)

	ctx := context.Background()
	executor := doctor.NewRealExecutor()
	summary := doctor.RunAll(ctx, executor)

	doctor.PrintResults(summary)

	if summary.HasFailure {
		os.Exit(1)
	}
	return nil
}
