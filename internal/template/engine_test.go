package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleConfig() *config.LZConfig {
	cfg := &config.LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Metadata: config.Metadata{
			Name:          "contoso-alz",
			Tenant:        "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
			PrimaryRegion: "westeurope",
		},
		Spec: config.Spec{
			Platform: config.Platform{
				ManagementGroups: config.ManagementGroupsConfig{Model: "caf-standard"},
				Connectivity:     config.ConnectivityConfig{Type: "none"},
				Identity:         config.IdentityConfig{Type: "workload-identity-federation"},
				Management: config.ManagementConfig{
					LogAnalytics: config.LogAnalyticsConfig{RetentionDays: 90},
					Defender:     config.DefenderConfig{Enabled: true, Plans: []string{"VirtualMachines"}},
				},
			},
			Governance: config.Governance{Policies: config.PolicyConfig{Assignments: []string{"deploy-mdfc-config"}}},
			Naming:     config.Naming{Convention: "caf"},
			StateBackend: config.StateBackend{
				ResourceGroup:  "rg-lzctl-state",
				StorageAccount: "stlzctlstate",
				Container:      "tfstate",
				Subscription:   "00000000-0000-0000-0000-000000000000",
			},
			CICD: config.CICD{
				Platform: "github-actions",
				BranchPolicy: config.BranchPolicy{
					MainBranch: "main",
					RequirePR:  true,
				},
			},
		},
	}
	config.ApplyDefaults(cfg)
	return cfg
}

func TestRenderAll(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	files, err := engine.RenderAll(sampleConfig())
	require.NoError(t, err)
	assert.Len(t, files, 22)

	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
		assert.NotEmpty(t, f.Content)
	}

	assert.True(t, paths["lzctl.yaml"])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "shared", "backend.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "shared", "backend.hcl"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "shared", "providers.tf"))])
	assert.True(t, paths[".gitignore"])
	assert.True(t, paths["README.md"])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "management-groups", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "management", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "governance", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "governance", "policies", "caf-default.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "identity", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join(".github", "workflows", "validate.yml"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join(".github", "workflows", "deploy.yml"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join(".github", "workflows", "drift.yml"))])
}

func TestRenderAll_ConnectivityVariants(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	t.Run("vwan", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.Platform.Connectivity.Type = "vwan"
		cfg.Spec.Platform.Connectivity.Hub = nil

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		paths := map[string]bool{}
		for _, f := range files {
			paths[f.Path] = true
		}
		assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "connectivity", "main.tf"))])
	})

	t.Run("hub_spoke_fw", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.Platform.Connectivity.Type = "hub-spoke"
		cfg.Spec.Platform.Connectivity.Hub = &config.HubConfig{
			Region:       "westeurope",
			AddressSpace: "10.0.0.0/16",
			Firewall:     config.FirewallConfig{Enabled: true, SKU: "Standard"},
		}

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		found := false
		for _, f := range files {
			if f.Path == filepath.ToSlash(filepath.Join("platform", "connectivity", "main.tf")) {
				found = true
				assert.Contains(t, f.Content, "azure_firewall")
			}
		}
		assert.True(t, found)
	})

	t.Run("azure_devops_pipeline", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.CICD.Platform = "azure-devops"

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		paths := map[string]bool{}
		for _, f := range files {
			paths[f.Path] = true
		}

		assert.True(t, paths[filepath.ToSlash(filepath.Join(".azuredevops", "pipelines", "validate.yml"))])
		assert.True(t, paths[filepath.ToSlash(filepath.Join(".azuredevops", "pipelines", "deploy.yml"))])
		assert.True(t, paths[filepath.ToSlash(filepath.Join(".azuredevops", "pipelines", "drift.yml"))])
	})

	t.Run("landing_zone_archetypes", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.LandingZones = []config.LandingZone{
			{Name: "corp-zone", Archetype: "corp", AddressSpace: "10.10.0.0/24", Connected: true, Tags: map[string]string{"env": "prod"}},
			{Name: "online-zone", Archetype: "online", AddressSpace: "10.20.0.0/24", Connected: true},
			{Name: "sandbox-zone", Archetype: "sandbox", AddressSpace: "10.30.0.0/24", Connected: false},
		}

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		paths := map[string]string{}
		for _, f := range files {
			paths[f.Path] = f.Content
		}

		assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "corp-zone", "main.tf")))
		assert.Contains(t, paths[filepath.ToSlash(filepath.Join("landing-zones", "corp-zone", "main.tf"))], "corp_vnet")

		assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "online-zone", "main.tf")))
		assert.Contains(t, paths[filepath.ToSlash(filepath.Join("landing-zones", "online-zone", "main.tf"))], "allow-https-inbound")

		assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "sandbox-zone", "main.tf")))
		assert.NotContains(t, paths[filepath.ToSlash(filepath.Join("landing-zones", "sandbox-zone", "main.tf"))], "virtual_network_peering")
	})
}

func TestWriteAll_DryRun(t *testing.T) {
	writer := Writer{DryRun: true}
	dir := t.TempDir()
	files := []RenderedFile{{Path: "a/b.txt", Content: "hello"}}

	paths, err := writer.WriteAll(files, dir)
	require.NoError(t, err)
	assert.Len(t, paths, 1)

	_, statErr := os.Stat(filepath.Join(dir, "a", "b.txt"))
	assert.Error(t, statErr)
}

func TestWriteAll_Write(t *testing.T) {
	writer := Writer{DryRun: false}
	dir := t.TempDir()
	files := []RenderedFile{{Path: "a/b.txt", Content: "hello"}}

	paths, err := writer.WriteAll(files, dir)
	require.NoError(t, err)
	assert.Len(t, paths, 1)

	data, readErr := os.ReadFile(filepath.Join(dir, "a", "b.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "hello", string(data))
}
