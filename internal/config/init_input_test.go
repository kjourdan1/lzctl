package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadInitInput_AndToLZConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "lzctl-init-input.yaml")

	content, marshalErr := yaml.Marshal(InitInput{
		TenantID:      "00000000-0000-0000-0000-000000000001",
		ProjectName:   "contoso-platform",
		MGModel:       "caf-standard",
		Connectivity:  "hub-spoke",
		PrimaryRegion: "westeurope",
		CICDPlatform:  "github-actions",
		StateStrategy: "create-new",
		LandingZones: []InitInputLandingZone{
			{Name: "corp-prod", Archetype: "corp", SubscriptionID: "11111111-1111-4111-8111-111111111111", AddressSpace: "10.10.0.0/16"},
			{Name: "online-dev", Archetype: "online", SubscriptionID: "22222222-2222-4222-8222-222222222222", AddressSpace: "10.20.0.0/16"},
		},
	})
	require.NoError(t, marshalErr)
	require.NoError(t, os.WriteFile(path, content, 0o644))

	in, err := LoadInitInput(path)
	require.NoError(t, err)

	cfg, err := in.ToLZConfig()
	require.NoError(t, err)
	assert.Equal(t, "contoso-platform", cfg.Metadata.Name)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", cfg.Metadata.Tenant)
	assert.Equal(t, "caf-standard", cfg.Spec.Platform.ManagementGroups.Model)
	assert.Equal(t, "hub-spoke", cfg.Spec.Platform.Connectivity.Type)
	assert.Equal(t, "github-actions", cfg.Spec.CICD.Platform)
	assert.Len(t, cfg.Spec.LandingZones, 2)
	assert.Equal(t, "corp-prod", cfg.Spec.LandingZones[0].Name)
	assert.Equal(t, "online-dev", cfg.Spec.LandingZones[1].Name)
}

func TestLoadInitInput_InvalidEnum(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.yaml")

	content, marshalErr := yaml.Marshal(InitInput{
		TenantID:      "00000000-0000-0000-0000-000000000001",
		ProjectName:   "contoso-platform",
		MGModel:       "invalid",
		Connectivity:  "hub-spoke",
		PrimaryRegion: "westeurope",
		CICDPlatform:  "github-actions",
		StateStrategy: "create-new",
	})
	require.NoError(t, marshalErr)
	require.NoError(t, os.WriteFile(path, content, 0o644))

	_, err := LoadInitInput(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mgModel")
}

func TestInitInput_ToLZConfig_DetectsAddressOverlap(t *testing.T) {
	in := &InitInput{
		TenantID:      "00000000-0000-0000-0000-000000000001",
		ProjectName:   "contoso-platform",
		MGModel:       "caf-standard",
		Connectivity:  "hub-spoke",
		PrimaryRegion: "westeurope",
		CICDPlatform:  "github-actions",
		StateStrategy: "create-new",
		LandingZones: []InitInputLandingZone{
			{Name: "lz1", Archetype: "corp", SubscriptionID: "11111111-1111-4111-8111-111111111111", AddressSpace: "10.10.0.0/16"},
			{Name: "lz2", Archetype: "online", SubscriptionID: "22222222-2222-4222-8222-222222222222", AddressSpace: "10.10.0.0/24"},
		},
	}

	_, err := in.ToLZConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "overlaps")
}
