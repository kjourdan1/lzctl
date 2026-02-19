package cmd

import (
	"path/filepath"
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddBlueprintCmd_Succeeds(t *testing.T) {
	repo := t.TempDir()
	_, _, err := executeCommand("init", "--tenant-id", "00000000-0000-0000-0000-000000000001", "--repo-root", repo)
	require.NoError(t, err)
	_, _, err = executeCommand("workload", "adopt", "--name", "corp-lz", "--subscription", "11111111-1111-4111-8111-111111111111", "--address-space", "10.1.0.0/24", "--repo-root", repo)
	require.NoError(t, err)

	_, _, err = executeCommand(
		"add-blueprint",
		"--repo-root", repo,
		"--landing-zone", "corp-lz",
		"--type", "paas-secure",
		"--set", "apim.enabled=false",
		"--set", "appService.sku=P2v3",
	)
	require.NoError(t, err)

	cfg, err := config.Load(filepath.Join(repo, "lzctl.yaml"))
	require.NoError(t, err)

	var zone *config.LandingZone
	for i := range cfg.Spec.LandingZones {
		if cfg.Spec.LandingZones[i].Name == "corp-lz" {
			zone = &cfg.Spec.LandingZones[i]
			break
		}
	}
	require.NotNil(t, zone)
	require.NotNil(t, zone.Blueprint)
	assert.Equal(t, "paas-secure", zone.Blueprint.Type)

	assert.FileExists(t, filepath.Join(repo, "landing-zones", "corp-lz", "blueprint", "main.tf"))
	assert.FileExists(t, filepath.Join(repo, "landing-zones", "corp-lz", "blueprint", "variables.tf"))
	assert.FileExists(t, filepath.Join(repo, "landing-zones", "corp-lz", "blueprint", "blueprint.auto.tfvars"))
	assert.FileExists(t, filepath.Join(repo, "landing-zones", "corp-lz", "blueprint", "backend.hcl"))
}

func TestAddBlueprintCmd_InvalidType(t *testing.T) {
	repo := t.TempDir()
	_, _, err := executeCommand("init", "--tenant-id", "00000000-0000-0000-0000-000000000001", "--repo-root", repo)
	require.NoError(t, err)
	_, _, err = executeCommand("workload", "adopt", "--name", "corp-lz", "--subscription", "11111111-1111-4111-8111-111111111111", "--address-space", "10.1.0.0/24", "--repo-root", repo)
	require.NoError(t, err)

	_, _, err = executeCommand(
		"add-blueprint",
		"--repo-root", repo,
		"--landing-zone", "corp-lz",
		"--type", "unknown",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid blueprint type")
}

func TestAddBlueprintCmd_OverwriteRequired(t *testing.T) {
	repo := t.TempDir()
	_, _, err := executeCommand("init", "--tenant-id", "00000000-0000-0000-0000-000000000001", "--repo-root", repo)
	require.NoError(t, err)
	_, _, err = executeCommand("workload", "adopt", "--name", "corp-lz", "--subscription", "11111111-1111-4111-8111-111111111111", "--address-space", "10.1.0.0/24", "--repo-root", repo)
	require.NoError(t, err)

	_, _, err = executeCommand(
		"add-blueprint",
		"--repo-root", repo,
		"--landing-zone", "corp-lz",
		"--type", "paas-secure",
	)
	require.NoError(t, err)

	_, _, err = executeCommand(
		"add-blueprint",
		"--repo-root", repo,
		"--landing-zone", "corp-lz",
		"--type", "paas-secure",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already has a blueprint")

	_, _, err = executeCommand(
		"add-blueprint",
		"--repo-root", repo,
		"--landing-zone", "corp-lz",
		"--type", "paas-secure",
		"--overwrite",
	)
	require.NoError(t, err)
}
