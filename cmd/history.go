package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/audit"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show CLI audit history",
	Long: `Displays audit events written by lzctl in JSONL format.

By default, reads ~/.lzctl/audit.log and prints the latest events.
Use --tenant to filter on a specific tenant.`,
	RunE: runHistory,
}

var (
	historyTenant string
	historyLimit  int
)

func init() {
	historyCmd.Flags().StringVar(&historyTenant, "tenant", "", "filter by tenant")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "max number of events to display")
	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	events, err := audit.ReadUserAudit()
	if err != nil {
		return fmt.Errorf("history failed: %w", err)
	}
	if len(events) == 0 {
		fmt.Fprintln(os.Stderr, "No audit events found.")
		return nil
	}

	filtered := make([]audit.Event, 0, len(events))
	for _, event := range events {
		if historyTenant != "" && event.Tenant != historyTenant {
			continue
		}
		filtered = append(filtered, event)
	}
	if len(filtered) == 0 {
		fmt.Fprintln(os.Stderr, "No matching audit events.")
		return nil
	}

	start := 0
	if historyLimit > 0 && len(filtered) > historyLimit {
		start = len(filtered) - historyLimit
	}

	bold := color.New(color.Bold)
	bold.Fprintln(os.Stderr, "📜 lzctl history")
	for _, event := range filtered[start:] {
		status := color.New(color.FgGreen)
		if event.Result != "success" {
			status = color.New(color.FgRed)
		}
		status.Fprintf(os.Stderr, "  %s", event.Result)
		fmt.Fprintf(os.Stderr, "  %s  op=%s", event.Timestamp, event.Operation)
		if event.Tenant != "" {
			fmt.Fprintf(os.Stderr, "  tenant=%s", event.Tenant)
		}
		if event.Ring != "" {
			fmt.Fprintf(os.Stderr, "  ring=%s", event.Ring)
		}
		fmt.Fprintf(os.Stderr, "  exit=%d  duration=%dms\n", event.ExitCode, event.DurationMs)
	}

	return nil
}
