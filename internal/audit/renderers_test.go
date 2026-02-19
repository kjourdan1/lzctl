package audit

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdown_Basic(t *testing.T) {
	report := &AuditReport{
		TenantID:  "tenant-123",
		ScannedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		Score: AuditScore{
			Overall: 78,
		},
		Summary: AuditSummary{
			Critical: 1,
			High:     1,
			Medium:   0,
			Low:      0,
		},
		Findings: []AuditFinding{
			{
				ID:            "GOV-001",
				Title:         "Management group hierarchy incomplete",
				Discipline:    "governance",
				Severity:      "high",
				CurrentState:  "2 management groups",
				ExpectedState: "4 management groups",
				Remediation:   "Add required baseline management groups",
			},
		},
	}

	output := RenderMarkdown(report)
	assert.Contains(t, output, "# Azure Audit Report")
	assert.Contains(t, output, "tenant-123")
	assert.Contains(t, output, "**78/100**")
	assert.Contains(t, output, "## Governance")
	assert.Contains(t, output, "GOV-001")
	assert.True(t, strings.Contains(output, "Remediation"))
}

func TestRenderJSON_Basic(t *testing.T) {
	report := &AuditReport{
		TenantID: "tenant-456",
		Score: AuditScore{
			Overall: 92,
		},
	}

	bytes, err := RenderJSON(report)
	require.NoError(t, err)
	jsonOutput := string(bytes)
	assert.Contains(t, jsonOutput, "tenant-456")
	assert.Contains(t, jsonOutput, "\"overall\": 92")
}
