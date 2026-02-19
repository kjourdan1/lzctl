package bootstrap

import (
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E9-S3: ArgoCD federated credential tests

func TestArgoCDFederatedCredentialFromConfig_NoLandingZone(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			LandingZones: []config.LandingZone{},
		},
	}
	_, err := ArgoCDFederatedCredentialFromConfig(cfg, "missing-zone", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestArgoCDFederatedCredentialFromConfig_NotAKS(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			LandingZones: []config.LandingZone{
				{
					Name:      "paas-zone",
					Blueprint: &config.Blueprint{Type: "paas-secure"},
				},
			},
		},
	}
	_, err := ArgoCDFederatedCredentialFromConfig(cfg, "paas-zone", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aks-platform")
}

func TestArgoCDFederatedCredentialFromConfig_NilConfig(t *testing.T) {
	_, err := ArgoCDFederatedCredentialFromConfig(nil, "any-zone", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestArgoCDFederatedCredentialFromConfig_NoBlueprintOnLZ(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			LandingZones: []config.LandingZone{
				{Name: "aks-zone"},
			},
		},
	}
	_, err := ArgoCDFederatedCredentialFromConfig(cfg, "aks-zone", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aks-platform")
}

func TestArgoCDConstants(t *testing.T) {
	assert.Equal(t, "argocd-source-controller", ArgoCDFederatedCredentialName)
	assert.Equal(t, "system:serviceaccount:argocd:source-controller", ArgoCDSourceControllerSubject)
}
