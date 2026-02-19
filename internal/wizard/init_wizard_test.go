package wizard

import (
	"errors"
	"fmt"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPrompter struct {
	answers map[string]interface{}
	calls   []string
	errAt   string
}

func (m *mockPrompter) Input(label, _ string, _ survey.Validator) (string, error) {
	m.calls = append(m.calls, label)
	if m.errAt == label {
		return "", ErrCanceled
	}
	if v, ok := m.answers[label]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	return "", nil
}

func (m *mockPrompter) Select(label string, _ []string, _ string) (string, error) {
	m.calls = append(m.calls, label)
	if m.errAt == label {
		return "", ErrCanceled
	}
	if v, ok := m.answers[label]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	return "", nil
}

func (m *mockPrompter) Confirm(label string, _ bool) (bool, error) {
	m.calls = append(m.calls, label)
	if m.errAt == label {
		return false, ErrCanceled
	}
	if v, ok := m.answers[label]; ok {
		if b, ok := v.(bool); ok {
			return b, nil
		}
	}
	return false, nil
}

func (m *mockPrompter) MultiSelect(label string, options []string, defaults []string) ([]string, error) {
	m.calls = append(m.calls, label)
	if m.errAt == label {
		return nil, ErrCanceled
	}
	if v, ok := m.answers[label]; ok {
		if s, ok := v.([]string); ok {
			return s, nil
		}
	}
	return defaults, nil
}

func TestValidateTenantID(t *testing.T) {
	err := ValidateTenantID("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")
	require.NoError(t, err)

	err = ValidateTenantID("not-a-uuid")
	require.Error(t, err)
}

func TestInitWizardRun_OrderAndConditionalPrompts(t *testing.T) {
	mock := &mockPrompter{answers: map[string]interface{}{
		"Project name":                 "Contoso Platform",
		"Tenant ID (UUID)":             "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		"CI/CD platform":               "github-actions",
		"Management group model":       "caf-standard",
		"Connectivity model":           "hub-spoke",
		"Primary region":               "westeurope",
		"Secondary region":             "northeurope",
		"Identity model":               "workload-identity-federation",
		"State backend strategy":       "create-new",
		"Bootstrap state backend now?": true,
		"Firewall SKU":                 "Premium",
		"Enable VPN gateway?":          true,
		"Enable ExpressRoute gateway?": false,
		"Enable DNS private resolver?": true,
	}}

	cfg, err := NewInitWizard(mock).Run()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	expectedPrefix := []string{
		"Project name",
		"Tenant ID (UUID)",
		"CI/CD platform",
		"Management group model",
		"Connectivity model",
		"Primary region",
		"Secondary region",
		"Identity model",
		"State backend strategy",
		"Bootstrap state backend now?",
	}
	assert.GreaterOrEqual(t, len(mock.calls), len(expectedPrefix))
	assert.Equal(t, expectedPrefix, mock.calls[:len(expectedPrefix)])

	assert.Equal(t, "Premium", cfg.FirewallSKU)
	assert.True(t, cfg.VPNGatewayEnabled)
	assert.True(t, cfg.DNSPrivateResolver)
}

func TestInitWizardRun_NoConnectivitySkipsSubprompts(t *testing.T) {
	mock := &mockPrompter{answers: map[string]interface{}{
		"Project name":                 "Contoso Lite",
		"Tenant ID (UUID)":             "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		"CI/CD platform":               "azure-devops",
		"Management group model":       "caf-lite",
		"Connectivity model":           "none",
		"Primary region":               "francecentral",
		"Secondary region":             "",
		"Identity model":               "sp-secret",
		"State backend strategy":       "existing",
		"Bootstrap state backend now?": false,
	}}

	cfg, err := NewInitWizard(mock).Run()
	require.NoError(t, err)
	assert.Equal(t, "none", cfg.ConnectivityModel)

	for _, call := range mock.calls {
		assert.NotEqual(t, "Firewall SKU", call)
		assert.NotEqual(t, "Enable DNS private resolver?", call)
	}
}

func TestInitWizardRun_Canceled(t *testing.T) {
	mock := &mockPrompter{answers: map[string]interface{}{}, errAt: "Tenant ID (UUID)"}
	_, err := NewInitWizard(mock).Run()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCanceled))
}

func TestInitConfigToLZConfig(t *testing.T) {
	in := InitConfig{
		ProjectName:          "My Project",
		TenantID:             "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		CICDPlatform:         "github-actions",
		ManagementGroupModel: "caf-standard",
		ConnectivityModel:    "hub-spoke",
		PrimaryRegion:        "westeurope",
		SecondaryRegion:      "northeurope",
		IdentityModel:        "workload-identity-federation",
		StateBackendStrategy: "create-new",
		Bootstrap:            true,
		FirewallSKU:          "Standard",
		VPNGatewayEnabled:    true,
		ERGatewayEnabled:     false,
		DNSPrivateResolver:   true,
	}

	cfg := in.ToLZConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, "lzctl/v1", cfg.APIVersion)
	assert.Equal(t, "LandingZone", cfg.Kind)
	assert.Equal(t, "My Project", cfg.Metadata.Name)
	assert.Equal(t, in.TenantID, cfg.Metadata.Tenant)
	assert.Equal(t, "github-actions", cfg.Spec.CICD.Platform)
	assert.Equal(t, "caf-standard", cfg.Spec.Platform.ManagementGroups.Model)
	assert.Equal(t, "hub-spoke", cfg.Spec.Platform.Connectivity.Type)
	require.NotNil(t, cfg.Spec.Platform.Connectivity.Hub)
	assert.Equal(t, "Standard", cfg.Spec.Platform.Connectivity.Hub.Firewall.SKU)
}
