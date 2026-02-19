package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	lztemplate "github.com/kjourdan1/lzctl/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderAndWrite_FullPlatformWithArchetypes(t *testing.T) {
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
				Connectivity: config.ConnectivityConfig{Type: "hub-spoke", Hub: &config.HubConfig{
					Region:       "westeurope",
					AddressSpace: "10.0.0.0/16",
					Firewall:     config.FirewallConfig{Enabled: true, SKU: "Standard"},
				}},
				Identity: config.IdentityConfig{Type: "workload-identity-federation"},
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
			LandingZones: []config.LandingZone{
				{Name: "corp-zone", Archetype: "corp", AddressSpace: "10.10.0.0/24", Connected: true, Tags: map[string]string{"env": "prod"}},
				{Name: "online-zone", Archetype: "online", AddressSpace: "10.20.0.0/24", Connected: true},
				{Name: "sandbox-zone", Archetype: "sandbox", AddressSpace: "10.30.0.0/24", Connected: false},
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

	engine, err := lztemplate.NewEngine()
	require.NoError(t, err)

	files, err := engine.RenderAll(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	outDir := t.TempDir()
	writer := lztemplate.Writer{DryRun: false}
	_, err = writer.WriteAll(files, outDir)
	require.NoError(t, err)

	mustExist := []string{
		"platform/management-groups/main.tf",
		"platform/connectivity/main.tf",
		"landing-zones/corp-zone/main.tf",
		"landing-zones/online-zone/main.tf",
		"landing-zones/sandbox-zone/main.tf",
		".github/workflows/validate.yml",
	}

	for _, rel := range mustExist {
		_, statErr := os.Stat(filepath.Join(outDir, filepath.FromSlash(rel)))
		assert.NoError(t, statErr, rel)
	}
}
