package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeInitInputFixture(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "lzctl-init-input.yaml")
	content, marshalErr := yaml.Marshal(config.InitInput{
		TenantID:      "00000000-0000-0000-0000-000000000001",
		ProjectName:   "contoso-platform",
		MGModel:       "caf-standard",
		Connectivity:  "none",
		PrimaryRegion: "westeurope",
		CICDPlatform:  "github-actions",
		StateStrategy: "create-new",
		LandingZones: []config.InitInputLandingZone{
			{Name: "corp-prod", Archetype: "corp", SubscriptionID: "11111111-1111-4111-8111-111111111111", AddressSpace: "10.10.0.0/16"},
			{Name: "online-dev", Archetype: "online", SubscriptionID: "22222222-2222-4222-8222-222222222222", AddressSpace: "10.20.0.0/16"},
		},
	})
	require.NoError(t, marshalErr)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func TestInitCmd_FromFile_Succeeds(t *testing.T) {
	repo := t.TempDir()
	inputPath := writeInitInputFixture(t, t.TempDir())

	_, _, err := executeCommand("init", "--from-file", inputPath, "--repo-root", repo)
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(repo, "lzctl.yaml"))
	require.NoError(t, statErr)
	_, statErr = os.Stat(filepath.Join(repo, "landing-zones", "corp-prod", "main.tf"))
	require.NoError(t, statErr)
	_, statErr = os.Stat(filepath.Join(repo, "landing-zones", "online-dev", "main.tf"))
	require.NoError(t, statErr)
}

func TestInitCmd_FromFile_DryRun_PrintsManifestAndDoesNotWrite(t *testing.T) {
	repo := t.TempDir()
	inputPath := writeInitInputFixture(t, t.TempDir())

	stdout, _, err := executeCommandWithProcessIO(t, "--dry-run", "init", "--from-file", inputPath, "--repo-root", repo)
	require.NoError(t, err)
	assert.Contains(t, stdout, "apiVersion: lzctl/v1")
	assert.Contains(t, stdout, "kind: LandingZone")

	_, statErr := os.Stat(filepath.Join(repo, "lzctl.yaml"))
	assert.Error(t, statErr)
}

func TestInitCmd_FromFile_ExistingConfigWithoutForce_Fails(t *testing.T) {
	repo := t.TempDir()
	inputPath := writeInitInputFixture(t, t.TempDir())
	require.NoError(t, os.WriteFile(filepath.Join(repo, "lzctl.yaml"), []byte("existing"), 0o644))

	_, _, err := executeCommand("init", "--from-file", inputPath, "--repo-root", repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lzctl.yaml already exists")
}

func TestInitCmd_FromFile_WithConfig_Fails(t *testing.T) {
	repo := t.TempDir()
	inputPath := writeInitInputFixture(t, t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "lzctl.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("apiVersion: lzctl/v1\nkind: LandingZone\nmetadata:\n  name: x\n  tenant: x\n  primaryRegion: westeurope\nspec:\n  platform:\n    managementGroups:\n      model: caf-standard\n    connectivity:\n      type: none\n    identity:\n      type: workload-identity-federation\n    management:\n      logAnalytics:\n        retentionDays: 90\n      automationAccount: true\n      defenderForCloud:\n        enabled: true\n        plans: []\n  governance:\n    policies:\n      assignments: []\n  naming:\n    convention: caf\n  stateBackend:\n    resourceGroup: rg\n    storageAccount: stabc\n    container: tfstate\n    subscription: <subscription-id>\n  landingZones: []\n  cicd:\n    platform: github-actions\n    branchPolicy:\n      mainBranch: main\n      requirePR: true\n"), 0o644))

	_, _, err := executeCommand("init", "--from-file", inputPath, "--config", cfgPath, "--repo-root", repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file cannot be combined with --config")
}

func TestInitCmd_CIMode_FromFile_WithoutTenantFlag_Succeeds(t *testing.T) {
	repo := t.TempDir()
	inputPath := writeInitInputFixture(t, t.TempDir())

	_, _, err := executeCommand("--ci", "init", "--from-file", inputPath, "--repo-root", repo)
	require.NoError(t, err)
}
