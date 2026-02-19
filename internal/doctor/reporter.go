package doctor

import (
	"fmt"
	"strings"

	"github.com/kjourdan1/lzctl/internal/output"
)

// StatusIcon returns the emoji/icon for a check status.
func StatusIcon(s Status) string {
	if output.NoColor() {
		switch s {
		case StatusPass:
			return "[PASS]"
		case StatusFail:
			return "[FAIL]"
		case StatusWarn:
			return "[WARN]"
		case StatusSkip:
			return "[SKIP]"
		default:
			return "[????]"
		}
	}
	switch s {
	case StatusPass:
		return "✅"
	case StatusFail:
		return "❌"
	case StatusWarn:
		return "⚠️"
	case StatusSkip:
		return "⏭️"
	default:
		return "❓"
	}
}

// PrintResults prints check results to stderr using the output package.
// Returns nothing; the caller should check summary.HasFailure for exit code.
func PrintResults(summary Summary) {
	if output.JSONMode {
		output.JSON(summary)
		return
	}

	output.Info("Running prerequisite checks...\n")

	lastCategory := ""
	for i, r := range summary.Results {
		checks := AllChecks()
		cat := ""
		if i < len(checks) {
			cat = checks[i].Category
		}
		if cat != lastCategory {
			printCategoryHeader(cat)
			lastCategory = cat
		}
		printCheckResult(r)
	}

	fmt.Println() // blank line before summary
	printSummaryLine(summary)
}

func printCategoryHeader(cat string) {
	var label string
	switch cat {
	case "tool":
		label = "Required Tools"
	case "auth":
		label = "Authentication"
	case "azure":
		label = "Azure Permissions"
	case "state":
		label = "State Backend"
	default:
		label = strings.Title(cat) //nolint:staticcheck
	}
	fmt.Println()
	if output.NoColor() {
		fmt.Printf("--- %s ---\n", label)
	} else {
		fmt.Printf("━━ %s ━━\n", label)
	}
}

func printCheckResult(r CheckResult) {
	icon := StatusIcon(r.Status)
	fmt.Printf("  %s  %s\n", icon, r.Message)
	if r.Fix != "" && r.Status != StatusPass {
		if output.NoColor() {
			fmt.Printf("       Fix: %s\n", r.Fix)
		} else {
			fmt.Printf("       💡 %s\n", r.Fix)
		}
	}
}

func printSummaryLine(s Summary) {
	parts := []string{}
	if s.TotalPass > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", s.TotalPass))
	}
	if s.TotalWarn > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", s.TotalWarn))
	}
	if s.TotalFail > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", s.TotalFail))
	}

	line := strings.Join(parts, ", ")

	if s.HasFailure {
		output.Fail(fmt.Sprintf("Doctor found issues: %s", line))
	} else if s.TotalWarn > 0 {
		output.Warn(fmt.Sprintf("Doctor completed with warnings: %s", line))
	} else {
		output.Success(fmt.Sprintf("All checks passed (%d)", s.TotalPass))
	}
}
