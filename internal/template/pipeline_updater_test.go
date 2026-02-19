package template

import (
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateZoneMatrix_Empty(t *testing.T) {
	cfg := &config.LZConfig{}
	result := GenerateZoneMatrix(cfg)
	assert.Equal(t, "[]", result)
}

func TestGenerateZoneMatrix_Nil(t *testing.T) {
	result := GenerateZoneMatrix(nil)
	assert.Equal(t, "[]", result)
}

func TestGenerateZoneMatrix_MultipleZones(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			LandingZones: []config.LandingZone{
				{Name: "app-prod", Archetype: "corp"},
				{Name: "sandbox-dev", Archetype: "sandbox"},
			},
		},
	}

	result := GenerateZoneMatrix(cfg)
	assert.Contains(t, result, `"name": "app-prod"`)
	assert.Contains(t, result, `"dir": "landing-zones/app-prod"`)
	assert.Contains(t, result, `"archetype": "corp"`)
	assert.Contains(t, result, `"name": "sandbox-dev"`)
	assert.Contains(t, result, `"archetype": "sandbox"`)
}

func TestWriteLandingZoneMatrix(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			LandingZones: []config.LandingZone{
				{Name: "zone-a", Archetype: "corp"},
			},
		},
	}

	tmpDir := t.TempDir()
	path, err := WriteLandingZoneMatrix(cfg, tmpDir)
	require.NoError(t, err)
	assert.Contains(t, path, "zone-matrix.json")
}

func TestPipelineUpdater_DryRun(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			CICD: config.CICD{
				Platform: "github-actions",
				BranchPolicy: config.BranchPolicy{
					MainBranch: "main",
				},
			},
		},
	}

	updater, err := NewPipelineUpdater()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	paths, err := updater.UpdatePipelinesDryRun(cfg, tmpDir)
	require.NoError(t, err)
	assert.NotEmpty(t, paths)
}
