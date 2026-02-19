package state

import (
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCLI implements AzCLIRunner for testing.
type mockCLI struct {
	responses map[string]string
	errors    map[string]error
	calls     [][]string
}

func newMockCLI() *mockCLI {
	return &mockCLI{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockCLI) Run(args ...string) (string, error) {
	m.calls = append(m.calls, args)
	key := args[0]
	if len(args) > 1 {
		key = args[0] + " " + args[1]
	}
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return "{}", nil
}

func testConfig() *config.LZConfig {
	return &config.LZConfig{
		Metadata: config.Metadata{Name: "test-lz", Tenant: "tenant-id"},
		Spec: config.Spec{
			StateBackend: config.StateBackend{
				ResourceGroup:  "rg-tfstate",
				StorageAccount: "stlzctltfstate",
				Container:      "tfstate",
				Subscription:   "sub-id",
			},
		},
	}
}

func TestStateKeyToLayer(t *testing.T) {
	tests := []struct {
		key    string
		expect string
	}{
		{"platform-connectivity.tfstate", "connectivity"},
		{"platform-management-groups.tfstate", "management-groups"},
		{"landing-zones-app1.tfstate", "lz:app1"},
		{"custom.tfstate", "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.expect, stateKeyToLayer(tt.key))
		})
	}
}

func TestListStates(t *testing.T) {
	cli := newMockCLI()
	cli.responses["storage blob"] = `[
		{"name": "platform-connectivity.tfstate", "properties": {"contentLength": 1234, "lastModified": "", "leaseStatus": "unlocked"}, "versionId": "v1"},
		{"name": "platform-management-groups.tfstate", "properties": {"contentLength": 567, "lastModified": "", "leaseStatus": "locked"}, "versionId": "v2"},
		{"name": "some-other-file.json", "properties": {"contentLength": 100, "lastModified": "", "leaseStatus": "unlocked"}, "versionId": ""}
	]`

	mgr := NewManager(testConfig(), cli)
	states, err := mgr.ListStates()
	require.NoError(t, err)
	assert.Len(t, states, 2) // only .tfstate files
	assert.Equal(t, "connectivity", states[0].Layer)
	assert.Equal(t, "locked", states[1].LeaseStatus)
}

func TestCreateSnapshot(t *testing.T) {
	cli := newMockCLI()
	cli.responses["storage blob"] = `{"snapshot": "2026-02-19T10:00:00Z", "versionId": "snap-v1"}`

	mgr := NewManager(testConfig(), cli)
	snap, err := mgr.CreateSnapshot("platform-connectivity.tfstate", "pre-apply")
	require.NoError(t, err)
	assert.Equal(t, "snap-v1", snap.VersionID)
	assert.Equal(t, "pre-apply", snap.Tag)
	assert.Equal(t, "platform-connectivity.tfstate", snap.Key)
}

func TestCheckHealth_AllPass(t *testing.T) {
	cli := newMockCLI()
	cli.responses["storage account"] = `{
		"properties": {
			"supportsHttpsTrafficOnly": true,
			"minimumTlsVersion": "TLS1_2",
			"encryption": {"requireInfrastructureEncryption": true}
		}
	}`
	cli.responses["storage account"] = `{
		"properties": {
			"supportsHttpsTrafficOnly": true,
			"minimumTlsVersion": "TLS1_2",
			"encryption": {"requireInfrastructureEncryption": true}
		}
	}`

	mgr := NewManager(testConfig(), cli)
	health, err := mgr.CheckHealth()
	require.NoError(t, err)
	assert.Equal(t, "stlzctltfstate", health.StorageAccount)
}

func TestNewManager(t *testing.T) {
	cfg := testConfig()
	cli := newMockCLI()
	mgr := NewManager(cfg, cli)
	assert.NotNil(t, mgr)
	assert.Equal(t, cfg, mgr.cfg)
}
