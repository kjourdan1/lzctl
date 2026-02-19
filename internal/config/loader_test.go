package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func fixturesDir() string {
	return filepath.Join("..", "..", "test", "fixtures", "configs")
}

// --- Loader tests ---

func TestLoad_StandardHubSpoke(t *testing.T) {
	cfg, err := Load(filepath.Join(fixturesDir(), "standard-hub-spoke.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "lzctl/v1", cfg.APIVersion)
	assert.Equal(t, "LandingZone", cfg.Kind)
	assert.Equal(t, "contoso-alz", cfg.Metadata.Name)
	assert.Equal(t, "contoso.onmicrosoft.com", cfg.Metadata.Tenant)
	assert.Equal(t, "westeurope", cfg.Metadata.PrimaryRegion)
	assert.Equal(t, "northeurope", cfg.Metadata.SecondaryRegion)

	// Platform
	assert.Equal(t, "caf-standard", cfg.Spec.Platform.ManagementGroups.Model)
	assert.Equal(t, "hub-spoke", cfg.Spec.Platform.Connectivity.Type)
	require.NotNil(t, cfg.Spec.Platform.Connectivity.Hub)
	assert.Equal(t, "10.0.0.0/16", cfg.Spec.Platform.Connectivity.Hub.AddressSpace)
	assert.True(t, cfg.Spec.Platform.Connectivity.Hub.Firewall.Enabled)
	assert.Equal(t, "Premium", cfg.Spec.Platform.Connectivity.Hub.Firewall.SKU)

	// Management
	assert.Equal(t, 90, cfg.Spec.Platform.Management.LogAnalytics.RetentionDays)
	assert.True(t, cfg.Spec.Platform.Management.Defender.Enabled)
	assert.Len(t, cfg.Spec.Platform.Management.Defender.Plans, 6)

	// Landing zones
	assert.Len(t, cfg.Spec.LandingZones, 2)
	assert.Equal(t, "lz-corp-app1", cfg.Spec.LandingZones[0].Name)
	assert.Equal(t, "corp", cfg.Spec.LandingZones[0].Archetype)
	assert.True(t, cfg.Spec.LandingZones[0].Connected)

	// CI/CD
	assert.Equal(t, "github-actions", cfg.Spec.CICD.Platform)
	assert.Equal(t, "main", cfg.Spec.CICD.BranchPolicy.MainBranch)
}

func TestLoad_LiteNoConnectivity(t *testing.T) {
	cfg, err := Load(filepath.Join(fixturesDir(), "lite-no-connectivity.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "startup-lite", cfg.Metadata.Name)
	assert.Equal(t, "caf-lite", cfg.Spec.Platform.ManagementGroups.Model)
	assert.Equal(t, "none", cfg.Spec.Platform.Connectivity.Type)
	assert.Nil(t, cfg.Spec.Platform.Connectivity.Hub)
	assert.Equal(t, "sp-secret", cfg.Spec.Platform.Identity.Type)
	assert.Equal(t, 30, cfg.Spec.Platform.Management.LogAnalytics.RetentionDays)
	assert.Empty(t, cfg.Spec.LandingZones)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{{{not yaml"), 0644))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config YAML")
}

// --- Defaults tests ---

func TestApplyDefaults_FillsMissingFields(t *testing.T) {
	cfg := &LZConfig{
		Metadata: Metadata{Name: "test"},
	}
	ApplyDefaults(cfg)

	assert.Equal(t, "lzctl/v1", cfg.APIVersion)
	assert.Equal(t, "LandingZone", cfg.Kind)
	assert.Equal(t, "caf", cfg.Spec.Naming.Convention)
	assert.Equal(t, "main", cfg.Spec.CICD.BranchPolicy.MainBranch)
	assert.Equal(t, 90, cfg.Spec.Platform.Management.LogAnalytics.RetentionDays)
	assert.Equal(t, "tfstate", cfg.Spec.StateBackend.Container)
	assert.Equal(t, "caf-standard", cfg.Spec.Platform.ManagementGroups.Model)
	assert.Equal(t, "none", cfg.Spec.Platform.Connectivity.Type)
	assert.Equal(t, "workload-identity-federation", cfg.Spec.Platform.Identity.Type)
}

func TestApplyDefaults_DoesNotOverrideExisting(t *testing.T) {
	cfg := &LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Spec: Spec{
			Platform: Platform{
				ManagementGroups: ManagementGroupsConfig{Model: "caf-lite"},
				Connectivity:     ConnectivityConfig{Type: "hub-spoke"},
				Identity:         IdentityConfig{Type: "sp-secret"},
				Management: ManagementConfig{
					LogAnalytics: LogAnalyticsConfig{RetentionDays: 365},
				},
			},
			Naming:       Naming{Convention: "custom"},
			StateBackend: StateBackend{Container: "mycontainer"},
			CICD:         CICD{BranchPolicy: BranchPolicy{MainBranch: "develop"}},
		},
	}
	ApplyDefaults(cfg)

	assert.Equal(t, "caf-lite", cfg.Spec.Platform.ManagementGroups.Model)
	assert.Equal(t, "hub-spoke", cfg.Spec.Platform.Connectivity.Type)
	assert.Equal(t, "sp-secret", cfg.Spec.Platform.Identity.Type)
	assert.Equal(t, 365, cfg.Spec.Platform.Management.LogAnalytics.RetentionDays)
	assert.Equal(t, "custom", cfg.Spec.Naming.Convention)
	assert.Equal(t, "mycontainer", cfg.Spec.StateBackend.Container)
	assert.Equal(t, "develop", cfg.Spec.CICD.BranchPolicy.MainBranch)
}

// --- Round-trip test ---

func TestRoundTrip_MarshalUnmarshal(t *testing.T) {
	original, err := Load(filepath.Join(fixturesDir(), "standard-hub-spoke.yaml"))
	require.NoError(t, err)

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	// Parse back
	parsed, err := Parse(data)
	require.NoError(t, err)

	assert.Equal(t, original.APIVersion, parsed.APIVersion)
	assert.Equal(t, original.Kind, parsed.Kind)
	assert.Equal(t, original.Metadata.Name, parsed.Metadata.Name)
	assert.Equal(t, original.Metadata.Tenant, parsed.Metadata.Tenant)
	assert.Equal(t, original.Spec.Platform.Connectivity.Type, parsed.Spec.Platform.Connectivity.Type)
	assert.Equal(t, original.Spec.Platform.Connectivity.Hub.AddressSpace, parsed.Spec.Platform.Connectivity.Hub.AddressSpace)
	assert.Len(t, parsed.Spec.LandingZones, len(original.Spec.LandingZones))
	assert.Equal(t, original.Spec.CICD.Platform, parsed.Spec.CICD.Platform)
}
