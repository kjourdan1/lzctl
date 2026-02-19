package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreFindings(t *testing.T) {
	findings := []AuditFinding{
		{ID: "GOV-001", Discipline: "governance", Severity: "high"},
		{ID: "NET-003", Discipline: "connectivity", Severity: "critical"},
		{ID: "MGT-002", Discipline: "management", Severity: "medium"},
	}

	score := scoreFindings(findings)
	assert.Less(t, score.Governance, 100)
	assert.Less(t, score.Connectivity, 100)
	assert.Less(t, score.Management, 100)
	assert.Equal(t, 100, score.Identity)
	assert.Equal(t, 100, score.Security)
	assert.True(t, score.Overall >= 0 && score.Overall <= 100)
}

func TestComplianceEngineEvaluate(t *testing.T) {
	engine := NewComplianceEngineWithRules([]ComplianceRule{ruleGOV001(), ruleNET001()})
	report := engine.Evaluate(&TenantSnapshot{})
	assert.NotNil(t, report)
	assert.NotEmpty(t, report.Findings)
	assert.Equal(t, len(report.Findings), report.Summary.TotalFindings)
}
