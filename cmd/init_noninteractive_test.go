package cmd

import (
	"path/filepath"
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCmd_CIMode_RequiresTenant(t *testing.T) {
	_, _, err := executeCommand("--ci", "init", "--repo-root", t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--ci mode requires --tenant-id")
}

func TestInitCmd_CIEnv_RequiresTenant(t *testing.T) {
	t.Setenv("CI", "true")
	_, _, err := executeCommand("init", "--repo-root", t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--ci mode requires --tenant-id")
}

func TestInitCmd_CIMode_WithTenant_Succeeds(t *testing.T) {
	repo := t.TempDir()
	_, _, err := executeCommand("--ci", "init", "--tenant-id", "00000000-0000-0000-0000-000000000001", "--repo-root", repo)
	require.NoError(t, err)
}

func TestInitCmd_InvalidConnectivityEnum_ReturnsError(t *testing.T) {
	repo := t.TempDir()
	_, _, err := executeCommand(
		"init",
		"--tenant-id", "00000000-0000-0000-0000-000000000001",
		"--connectivity", "foobar",
		"--repo-root", repo,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value for --connectivity")
}

func TestInitCmd_NonInteractive_FlagCombinations(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectMG     string
		expectConn   string
		expectCICD   string
		expectTenant string
		secondary    string
	}{
		{
			name: "caf-lite none",
			args: []string{"--mg-model", "caf-lite", "--connectivity", "none"},
			expectMG: "caf-lite", expectConn: "none", expectCICD: "github-actions", expectTenant: "00000000-0000-0000-0000-000000000001",
		},
		{
			name: "caf-standard hub-spoke",
			args: []string{"--mg-model", "caf-standard", "--connectivity", "hub-spoke", "--cicd-platform", "azure-devops"},
			expectMG: "caf-standard", expectConn: "hub-spoke", expectCICD: "azure-devops", expectTenant: "00000000-0000-0000-0000-000000000001",
		},
		{
			name: "caf-standard vwan",
			args: []string{"--mg-model", "caf-standard", "--connectivity", "vwan", "--secondary-region", "northeurope"},
			expectMG: "caf-standard", expectConn: "vwan", expectCICD: "github-actions", expectTenant: "00000000-0000-0000-0000-000000000001", secondary: "northeurope",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			base := []string{
				"init",
				"--tenant-id", tc.expectTenant,
				"--project-name", "contoso-platform",
				"--identity", "workload-identity-federation",
				"--state-strategy", "create-new",
				"--primary-region", "westeurope",
				"--repo-root", repo,
			}
			base = append(base, tc.args...)

			_, _, err := executeCommand(base...)
			require.NoError(t, err)

			cfg, loadErr := config.Load(filepath.Join(repo, "lzctl.yaml"))
			require.NoError(t, loadErr)

			assert.Equal(t, tc.expectTenant, cfg.Metadata.Tenant)
			assert.Equal(t, tc.expectMG, cfg.Spec.Platform.ManagementGroups.Model)
			assert.Equal(t, tc.expectConn, cfg.Spec.Platform.Connectivity.Type)
			assert.Equal(t, tc.expectCICD, cfg.Spec.CICD.Platform)
			if tc.secondary != "" {
				assert.Equal(t, tc.secondary, cfg.Metadata.SecondaryRegion)
			}
		})
	}
}

func TestInitCmd_EnvVars_NonInteractive(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("LZCTL_TENANT_ID", "11111111-1111-1111-1111-111111111111")
	t.Setenv("LZCTL_PROJECT_NAME", "env-project")
	t.Setenv("LZCTL_MG_MODEL", "caf-lite")
	t.Setenv("LZCTL_CONNECTIVITY", "none")
	t.Setenv("LZCTL_CICD_PLATFORM", "azure-devops")

	_, _, err := executeCommand("init", "--repo-root", repo)
	require.NoError(t, err)

	cfg, loadErr := config.Load(filepath.Join(repo, "lzctl.yaml"))
	require.NoError(t, loadErr)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", cfg.Metadata.Tenant)
	assert.Equal(t, "env-project", cfg.Metadata.Name)
	assert.Equal(t, "caf-lite", cfg.Spec.Platform.ManagementGroups.Model)
	assert.Equal(t, "none", cfg.Spec.Platform.Connectivity.Type)
	assert.Equal(t, "azure-devops", cfg.Spec.CICD.Platform)
}

func TestInitCmd_FlagOverridesEnv(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("LZCTL_TENANT_ID", "11111111-1111-1111-1111-111111111111")

	_, _, err := executeCommand(
		"init",
		"--tenant-id", "22222222-2222-2222-2222-222222222222",
		"--repo-root", repo,
	)
	require.NoError(t, err)

	cfg, loadErr := config.Load(filepath.Join(repo, "lzctl.yaml"))
	require.NoError(t, loadErr)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", cfg.Metadata.Tenant)
}
