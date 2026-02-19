package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E9-S1: AKSBlueprintConfig parsing + ArgoCD validation

func TestParseAKSBlueprintConfig_Empty(t *testing.T) {
	cfg, err := ParseAKSBlueprintConfig(nil)
	require.NoError(t, err)
	assert.Equal(t, AKSBlueprintConfig{}, cfg)
}

func TestParseAKSBlueprintConfig_AKSVersion(t *testing.T) {
	overrides := map[string]any{
		"aks": map[string]any{"version": "1.30"},
	}
	cfg, err := ParseAKSBlueprintConfig(overrides)
	require.NoError(t, err)
	assert.Equal(t, "1.30", cfg.AKS.Version)
}

func TestParseAKSBlueprintConfig_ArgoCD(t *testing.T) {
	overrides := map[string]any{
		"argocd": map[string]any{
			"enabled":        true,
			"mode":           "extension",
			"repoUrl":        "https://github.com/org/app-repo",
			"targetRevision": "HEAD",
			"appPath":        "apps/",
		},
	}
	cfg, err := ParseAKSBlueprintConfig(overrides)
	require.NoError(t, err)
	assert.True(t, cfg.ArgoCD.Enabled)
	assert.Equal(t, "extension", cfg.ArgoCD.Mode)
	assert.Equal(t, "https://github.com/org/app-repo", cfg.ArgoCD.RepoURL)
	assert.Equal(t, "HEAD", cfg.ArgoCD.TargetRevision)
}

func TestParseAKSBlueprintConfig_ACRSku(t *testing.T) {
	overrides := map[string]any{
		"acr": map[string]any{"sku": "Premium"},
	}
	cfg, err := ParseAKSBlueprintConfig(overrides)
	require.NoError(t, err)
	assert.Equal(t, "Premium", cfg.ACR.SKU)
}

func TestValidateArgoCDConfig_DisabledAlwaysValid(t *testing.T) {
	err := ValidateArgoCDConfig(ArgoCDConfig{Enabled: false})
	assert.NoError(t, err)
}

func TestValidateArgoCDConfig_ExtensionModeValid(t *testing.T) {
	err := ValidateArgoCDConfig(ArgoCDConfig{
		Enabled: true,
		Mode:    "extension",
		RepoURL: "https://github.com/org/app-repo",
	})
	assert.NoError(t, err)
}

func TestValidateArgoCDConfig_HelmModeValid(t *testing.T) {
	err := ValidateArgoCDConfig(ArgoCDConfig{
		Enabled: true,
		Mode:    "helm",
		RepoURL: "https://github.com/org/app-repo",
	})
	assert.NoError(t, err)
}

func TestValidateArgoCDConfig_InvalidMode(t *testing.T) {
	err := ValidateArgoCDConfig(ArgoCDConfig{
		Enabled: true,
		Mode:    "flux",
		RepoURL: "https://github.com/org/app-repo",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "argocd.mode")
}

func TestValidateArgoCDConfig_MissingRepoURL(t *testing.T) {
	err := ValidateArgoCDConfig(ArgoCDConfig{
		Enabled: true,
		Mode:    "extension",
		RepoURL: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "argocd.repoUrl")
}

func TestArgoCDConfig_Defaults(t *testing.T) {
	a := ArgoCDConfig{Enabled: true, Mode: "extension", RepoURL: "https://github.com/org/repo"}
	assert.Equal(t, "extension", a.ArgoCDMode())
	assert.Equal(t, "HEAD", a.EffectiveTargetRevision())
	assert.Equal(t, "apps/", a.EffectiveAppPath())
}

func TestArgoCDConfig_HelmMode(t *testing.T) {
	a := ArgoCDConfig{Mode: "helm"}
	assert.Equal(t, "helm", a.ArgoCDMode())
}

func TestArgoCDConfig_EmptyModeDefaultsToExtension(t *testing.T) {
	a := ArgoCDConfig{}
	assert.Equal(t, "extension", a.ArgoCDMode())
}
