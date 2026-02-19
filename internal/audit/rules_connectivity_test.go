package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuleNET003_Overlap(t *testing.T) {
	snapshot := &TenantSnapshot{
		VirtualNetworks: []VirtualNetwork{
			{Name: "vnet1", AddressSpaces: []string{"10.0.0.0/16"}},
			{Name: "vnet2", AddressSpaces: []string{"10.0.1.0/24"}},
		},
	}
	findings := ruleNET003().Evaluate(snapshot)
	assert.NotEmpty(t, findings)
	assert.Equal(t, "NET-003", findings[0].ID)
}

func TestRuleNET003_NoOverlap(t *testing.T) {
	snapshot := &TenantSnapshot{
		VirtualNetworks: []VirtualNetwork{
			{Name: "vnet1", AddressSpaces: []string{"10.0.0.0/16"}},
			{Name: "vnet2", AddressSpaces: []string{"10.1.0.0/16"}},
		},
	}
	findings := ruleNET003().Evaluate(snapshot)
	assert.Empty(t, findings)
}
