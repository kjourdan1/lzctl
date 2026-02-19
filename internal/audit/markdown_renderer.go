package audit

import (
	"fmt"
	"strings"
)

// RenderMarkdown renders a human-readable audit report in Markdown format.
func RenderMarkdown(report *AuditReport) string {
	if report == nil {
		return "# Azure Audit Report\n\nNo data available."
	}

	b := &strings.Builder{}
	fmt.Fprintf(b, "# Azure Audit Report\n\n")
	fmt.Fprintf(b, "- Tenant: `%s`\n", report.TenantID)
	fmt.Fprintf(b, "- Scanned At: `%s`\n", report.ScannedAt.UTC().Format("2006-01-02 15:04:05Z"))
	fmt.Fprintf(b, "- Overall Score: **%d/100**\n", report.Score.Overall)
	fmt.Fprintf(b, "- Findings: critical=%d, high=%d, medium=%d, low=%d\n\n",
		report.Summary.Critical, report.Summary.High, report.Summary.Medium, report.Summary.Low)

	disciplines := []string{"governance", "identity", "management", "connectivity", "security"}

	for _, discipline := range disciplines {
		section := findingsForDiscipline(report.Findings, discipline)
		if len(section) == 0 {
			continue
		}
		fmt.Fprintf(b, "## %s\n\n", titleCase(discipline))
		for _, f := range section {
			fmt.Fprintf(b, "### %s — %s\n\n", f.ID, f.Title)
			fmt.Fprintf(b, "- Severity: **%s**\n", strings.ToUpper(f.Severity))
			fmt.Fprintf(b, "- Current: %s\n", f.CurrentState)
			fmt.Fprintf(b, "- Expected: %s\n", f.ExpectedState)
			fmt.Fprintf(b, "- Remediation: %s\n", f.Remediation)
			fmt.Fprintf(b, "- Auto-fixable: %t\n\n", f.AutoFixable)
		}
	}

	return b.String()
}

func findingsForDiscipline(findings []AuditFinding, discipline string) []AuditFinding {
	filtered := make([]AuditFinding, 0)
	for _, f := range findings {
		if strings.EqualFold(strings.TrimSpace(f.Discipline), discipline) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func titleCase(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) == 1 {
		return strings.ToUpper(trimmed)
	}
	return strings.ToUpper(trimmed[:1]) + strings.ToLower(trimmed[1:])
}
