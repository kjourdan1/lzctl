package wizard

import (
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kjourdan1/lzctl/internal/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeImportPrompter implements Prompter for testing import wizard flows.
type fakeImportPrompter struct {
	inputs       map[string]string
	selects      map[string]string
	confirms     map[string]bool
	multiSelects map[string][]string
	calls        []string
}

func (f *fakeImportPrompter) Input(label, defaultValue string, _ survey.Validator) (string, error) {
	f.calls = append(f.calls, "input:"+label)
	if v, ok := f.inputs[label]; ok {
		return v, nil
	}
	return defaultValue, nil
}

func (f *fakeImportPrompter) Select(label string, _ []string, defaultValue string) (string, error) {
	f.calls = append(f.calls, "select:"+label)
	if v, ok := f.selects[label]; ok {
		return v, nil
	}
	return defaultValue, nil
}

func (f *fakeImportPrompter) Confirm(label string, defaultValue bool) (bool, error) {
	f.calls = append(f.calls, "confirm:"+label)
	if v, ok := f.confirms[label]; ok {
		return v, nil
	}
	return defaultValue, nil
}

func (f *fakeImportPrompter) MultiSelect(label string, options []string, defaults []string) ([]string, error) {
	f.calls = append(f.calls, "multiselect:"+label)
	if v, ok := f.multiSelects[label]; ok {
		return v, nil
	}
	return defaults, nil
}

func TestImportWizard_RunSubscriptionMode(t *testing.T) {
	p := &fakeImportPrompter{
		selects: map[string]string{
			"Import source": "subscription",
		},
		inputs: map[string]string{
			"Subscription ID (UUID)":                "a1b2c3d4-e5f6-1234-5678-abcdef012345",
			"Target directory for generated files":  "imports",
		},
	}

	wiz := NewImportWizard(p)
	cfg, err := wiz.Run()
	require.NoError(t, err)

	assert.Equal(t, "subscription", cfg.Source)
	assert.Equal(t, "a1b2c3d4-e5f6-1234-5678-abcdef012345", cfg.Subscription)
	assert.Equal(t, "imports", cfg.TargetDir)
}

func TestImportWizard_RunAuditReportMode(t *testing.T) {
	p := &fakeImportPrompter{
		selects: map[string]string{
			"Import source": "audit-report",
		},
		inputs: map[string]string{
			"Path to audit report JSON":            "my-audit.json",
			"Target directory for generated files": "output",
		},
	}

	wiz := NewImportWizard(p)
	cfg, err := wiz.Run()
	require.NoError(t, err)

	assert.Equal(t, "audit-report", cfg.Source)
	assert.Equal(t, "my-audit.json", cfg.AuditReportPath)
	assert.Equal(t, "output", cfg.TargetDir)
}

func TestImportWizard_RunResourceGroupMode(t *testing.T) {
	p := &fakeImportPrompter{
		selects: map[string]string{
			"Import source": "resource-group",
		},
		inputs: map[string]string{
			"Subscription ID (UUID)":               "a1b2c3d4-e5f6-1234-5678-abcdef012345",
			"Resource group name":                  "rg-app-prod",
			"Target directory for generated files": "",
		},
	}

	wiz := NewImportWizard(p)
	cfg, err := wiz.Run()
	require.NoError(t, err)

	assert.Equal(t, "resource-group", cfg.Source)
	assert.Equal(t, "rg-app-prod", cfg.ResourceGroup)
	assert.Equal(t, "imports", cfg.TargetDir) // default
}

func TestImportWizard_SelectResources(t *testing.T) {
	resources := []importer.ImportableResource{
		{
			ID: "/sub/rg/vnet1", AzureType: "microsoft.network/virtualnetworks",
			Name: "vnet-hub", TerraformType: "azurerm_virtual_network", Supported: true,
		},
		{
			ID: "/sub/rg/nsg1", AzureType: "microsoft.network/networksecuritygroups",
			Name: "nsg-default", TerraformType: "azurerm_network_security_group", Supported: true,
		},
		{
			ID: "/sub/rg/vm1", AzureType: "microsoft.compute/virtualmachines",
			Name: "vm01", TerraformType: "unsupported", Supported: false,
		},
	}

	// Simulate selecting only the first resource
	p := &fakeImportPrompter{
		multiSelects: map[string][]string{
			"Select resources to import": {
				"[✅] vnet-hub — microsoft.network/virtualnetworks (azurerm_virtual_network)",
			},
		},
	}

	wiz := NewImportWizard(p)
	selected, err := wiz.SelectResources(resources)
	require.NoError(t, err)

	assert.Len(t, selected, 1)
	assert.Equal(t, "vnet-hub", selected[0].Name)
}

func TestImportWizard_SelectResources_Empty(t *testing.T) {
	p := &fakeImportPrompter{}
	wiz := NewImportWizard(p)
	selected, err := wiz.SelectResources(nil)
	require.NoError(t, err)
	assert.Nil(t, selected)
}
