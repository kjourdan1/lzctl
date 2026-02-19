package audit

import (
	"sort"
	"strings"
	"time"
)

type ComplianceRule interface {
	ID() string
	Discipline() string
	Evaluate(snapshot *TenantSnapshot) []AuditFinding
}

type ComplianceEngine struct {
	rules []ComplianceRule
}

func NewComplianceEngine() *ComplianceEngine {
	return &ComplianceEngine{rules: defaultRules()}
}

func NewComplianceEngineWithRules(rules []ComplianceRule) *ComplianceEngine {
	return &ComplianceEngine{rules: rules}
}

func (e *ComplianceEngine) Evaluate(snapshot *TenantSnapshot) *AuditReport {
	if snapshot == nil {
		snapshot = &TenantSnapshot{}
	}
	if snapshot.ScannedAt.IsZero() {
		snapshot.ScannedAt = time.Now().UTC()
	}

	findings := make([]AuditFinding, 0)
	for _, rule := range e.rules {
		findings = append(findings, rule.Evaluate(snapshot)...)
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Discipline == findings[j].Discipline {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].Discipline < findings[j].Discipline
	})

	summary := summarize(findings)
	score := scoreFindings(findings)

	return &AuditReport{
		TenantID:  strings.TrimSpace(snapshot.TenantID),
		ScannedAt: snapshot.ScannedAt,
		Score:     score,
		Findings:  findings,
		Summary:   summary,
	}
}

func summarize(findings []AuditFinding) AuditSummary {
	out := AuditSummary{TotalFindings: len(findings)}
	for _, f := range findings {
		switch strings.ToLower(f.Severity) {
		case "critical":
			out.Critical++
		case "high":
			out.High++
		case "medium":
			out.Medium++
		case "low":
			out.Low++
		}
		if f.AutoFixable {
			out.AutoFixable++
		}
	}
	return out
}

func scoreFindings(findings []AuditFinding) AuditScore {
	byDisciplinePenalty := map[string]int{
		"governance":   0,
		"identity":     0,
		"management":   0,
		"connectivity": 0,
		"security":     0,
	}

	for _, f := range findings {
		penalty := severityPenalty(f.Severity)
		disc := strings.ToLower(strings.TrimSpace(f.Discipline))
		if _, ok := byDisciplinePenalty[disc]; ok {
			byDisciplinePenalty[disc] += penalty
		}
	}

	gov := disciplineScore(byDisciplinePenalty["governance"])
	idt := disciplineScore(byDisciplinePenalty["identity"])
	mgt := disciplineScore(byDisciplinePenalty["management"])
	net := disciplineScore(byDisciplinePenalty["connectivity"])
	sec := disciplineScore(byDisciplinePenalty["security"])

	return AuditScore{
		Overall:      (gov + idt + mgt + net + sec) / 5,
		Governance:   gov,
		Identity:     idt,
		Management:   mgt,
		Connectivity: net,
		Security:     sec,
	}
}

func severityPenalty(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 20
	case "high":
		return 12
	case "medium":
		return 6
	case "low":
		return 3
	default:
		return 0
	}
}

func disciplineScore(penalty int) int {
	score := 100 - penalty
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
