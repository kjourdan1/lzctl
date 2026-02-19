package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCross_OverlappingCIDR(t *testing.T) {
	cfg := &LZConfig{
		Metadata: Metadata{Name: "contoso", Tenant: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", PrimaryRegion: "westeurope"},
		Spec: Spec{
			Platform: Platform{
				Connectivity: ConnectivityConfig{Type: "hub-spoke", Hub: &HubConfig{AddressSpace: "10.0.0.0/16"}},
			},
			LandingZones: []LandingZone{{Name: "corp", AddressSpace: "10.0.1.0/24", Subscription: "11111111-1111-4111-8111-111111111111"}},
			StateBackend: StateBackend{Subscription: "00000000-0000-4000-8000-000000000000"},
		},
	}

	checks, err := ValidateCross(cfg, "")
	require.NoError(t, err)
	assert.True(t, hasCrossStatus(checks, "error"))
	assert.True(t, hasCrossName(checks, "address-space-overlap"))
}

func TestValidateCross_CustomPolicyPath(t *testing.T) {
	repo := t.TempDir()
	policyPath := filepath.Join(repo, "policies", "custom", "policy.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(policyPath), 0o755))
	require.NoError(t, os.WriteFile(policyPath, []byte("{}"), 0o644))

	cfg := &LZConfig{
		Metadata: Metadata{Name: "contoso", Tenant: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", PrimaryRegion: "westeurope"},
		Spec: Spec{
			Governance:   Governance{Policies: PolicyConfig{Custom: []string{"policies/custom/policy.json"}}},
			StateBackend: StateBackend{Subscription: "00000000-0000-4000-8000-000000000000"},
		},
	}

	checks, err := ValidateCross(cfg, repo)
	require.NoError(t, err)
	assert.False(t, hasCrossStatus(checks, "error"))
}

func hasCrossStatus(checks []CrossCheck, status string) bool {
	for _, c := range checks {
		if c.Status == status {
			return true
		}
	}
	return false
}

func hasCrossName(checks []CrossCheck, name string) bool {
	for _, c := range checks {
		if c.Name == name {
			return true
		}
	}
	return false
}
