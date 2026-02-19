package upgrade

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.2.3", "1.2.3"},
		{"~> 0.4.0", "0.4.0"},
		{">= 1.0.0", "1.0.0"},
		{"<= 2.0.0", "2.0.0"},
		{"> 1.0.0", "1.0.0"},
		{"= 1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractVersion(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScanFile(t *testing.T) {
	content := `
module "hub_network" {
  source  = "Azure/avm-res-network-virtualnetwork/azurerm"
  version = "~> 0.4.0"

  name                = "vnet-hub"
  resource_group_name = azurerm_resource_group.hub.name
  location            = var.location
}

module "firewall" {
  source  = "Azure/avm-res-network-azurefirewall/azurerm"
  version = ">= 0.3.0"

  name = "fw-hub"
}
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o644))

	pins, err := scanFile(tfFile)
	require.NoError(t, err)
	require.Len(t, pins, 2)

	assert.Equal(t, "Azure", pins[0].Ref.Namespace)
	assert.Equal(t, "avm-res-network-virtualnetwork", pins[0].Ref.Name)
	assert.Equal(t, "azurerm", pins[0].Ref.Provider)
	assert.Equal(t, "0.4.0", pins[0].Version)

	assert.Equal(t, "avm-res-network-azurefirewall", pins[1].Ref.Name)
	assert.Equal(t, "0.3.0", pins[1].Version)
}

func TestScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	content := `
module "nsg" {
  source  = "Azure/avm-res-network-networksecuritygroup/azurerm"
  version = "0.2.0"
}
`
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "platform", "connectivity"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "platform", "connectivity", "main.tf"), []byte(content), 0o644))

	pins, err := ScanDirectory(tmpDir)
	require.NoError(t, err)
	require.Len(t, pins, 1)
	assert.Equal(t, "avm-res-network-networksecuritygroup", pins[0].Ref.Name)
}

func TestScanDirectory_IncludesBlueprintSubdirs(t *testing.T) {
	tmpDir := t.TempDir()

	content := `
module "kv" {
  source  = "Azure/avm-res-keyvault-vault/azurerm"
  version = "0.9.0"
}
`
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "landing-zones", "corp-lz", "blueprint"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "landing-zones", "corp-lz", "blueprint", "main.tf"), []byte(content), 0o644))

	pins, err := ScanDirectory(tmpDir)
	require.NoError(t, err)
	require.Len(t, pins, 1)
	assert.Equal(t, "avm-res-keyvault-vault", pins[0].Ref.Name)
}

func TestScanDirectory_SkipsTerraformDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Module in .terraform should be skipped.
	tfDir := filepath.Join(tmpDir, ".terraform", "modules")
	require.NoError(t, os.MkdirAll(tfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte(`
module "x" {
  source  = "Azure/something/azurerm"
  version = "1.0.0"
}
`), 0o644))

	pins, err := ScanDirectory(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, pins)
}

func TestUpdateVersionInFile(t *testing.T) {
	content := `
module "hub_network" {
  source  = "Azure/avm-res-network-virtualnetwork/azurerm"
  version = "~> 0.4.0"

  name = "vnet-hub"
}
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o644))

	pin := ModulePin{
		Ref:      ModuleRef{Namespace: "Azure", Name: "avm-res-network-virtualnetwork", Provider: "azurerm"},
		Version:  "0.4.0",
		FilePath: tfFile,
		Line:     3,
	}

	err := UpdateVersionInFile(pin, "0.5.0")
	require.NoError(t, err)

	updated, err := os.ReadFile(tfFile)
	require.NoError(t, err)
	assert.Contains(t, string(updated), `version = "~> 0.5.0"`)
	assert.NotContains(t, string(updated), "0.4.0")
}
