package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuleGOV001_MissingHierarchy(t *testing.T) {
	snapshot := &TenantSnapshot{ManagementGroups: []ManagementGroup{{Name: "root", DisplayName: "Tenant Root Group"}}}
	findings := ruleGOV001().Evaluate(snapshot)
	assert.NotEmpty(t, findings)
	assert.Equal(t, "GOV-001", findings[0].ID)
}

func TestRuleGOV001_OK(t *testing.T) {
	snapshot := &TenantSnapshot{ManagementGroups: []ManagementGroup{{Name: "platform"}, {Name: "landing-zones"}}}
	findings := ruleGOV001().Evaluate(snapshot)
	assert.Empty(t, findings)
}
