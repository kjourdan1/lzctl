package importer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHCLGenerator_GenerateImportBlock_Supported(t *testing.T) {
	gen := NewHCLGenerator()
	r := ImportableResource{
		ID:            "/subscriptions/sub-001/resourceGroups/rg-core/providers/Microsoft.Network/virtualNetworks/vnet-hub",
		AzureType:     "microsoft.network/virtualnetworks",
		Name:          "vnet-hub",
		TerraformType: "azurerm_virtual_network",
		Supported:     true,
	}

	block := gen.GenerateImportBlock(r)
	assert.Contains(t, block, "import {")
	assert.Contains(t, block, "to = azurerm_virtual_network.vnet_hub")
	assert.Contains(t, block, r.ID)
}

func TestHCLGenerator_GenerateImportBlock_Unsupported(t *testing.T) {
	gen := NewHCLGenerator()
	r := ImportableResource{
		ID:            "/subscriptions/sub-001/.../vm01",
		AzureType:     "microsoft.compute/virtualmachines",
		Name:          "vm01",
		TerraformType: "unsupported",
		Supported:     false,
	}

	block := gen.GenerateImportBlock(r)
	assert.Contains(t, block, "# TODO: manual import required")
	assert.Contains(t, block, "vm01")
}

func TestHCLGenerator_GenerateResourceBlock_VNet(t *testing.T) {
	gen := NewHCLGenerator()
	r := ImportableResource{
		ID:            "/subscriptions/sub-001/resourceGroups/rg-core/providers/Microsoft.Network/virtualNetworks/vnet-hub",
		AzureType:     "microsoft.network/virtualnetworks",
		Name:          "vnet-hub",
		ResourceGroup: "rg-core",
		TerraformType: "azurerm_virtual_network",
		Supported:     true,
	}

	block := gen.GenerateResourceBlock(r)
	assert.Contains(t, block, `resource "azurerm_virtual_network" "vnet_hub"`)
	assert.Contains(t, block, `name     = "vnet-hub"`)
	assert.Contains(t, block, `resource_group_name = "rg-core"`)
	assert.Contains(t, block, "address_space")
}

func TestHCLGenerator_GenerateResourceBlock_ResourceGroup(t *testing.T) {
	gen := NewHCLGenerator()
	r := ImportableResource{
		ID:            "/subscriptions/sub-001/resourceGroups/rg-core",
		AzureType:     "microsoft.resources/resourcegroups",
		Name:          "rg-core",
		TerraformType: "azurerm_resource_group",
		Supported:     true,
	}

	block := gen.GenerateResourceBlock(r)
	assert.Contains(t, block, `resource "azurerm_resource_group" "rg_core"`)
	assert.NotContains(t, block, "resource_group_name") // RG doesn't reference itself
	assert.Contains(t, block, "location")
}

func TestHCLGenerator_GenerateAll_GroupsByLayer(t *testing.T) {
	gen := NewHCLGenerator()
	resources := []ImportableResource{
		{
			ID:        "/sub/rg/providers/Microsoft.Network/virtualNetworks/vnet1",
			AzureType: "microsoft.network/virtualnetworks", Name: "vnet1",
			TerraformType: "azurerm_virtual_network", Supported: true,
		},
		{
			ID:        "/sub/resourceGroups/rg1",
			AzureType: "microsoft.resources/resourcegroups", Name: "rg1",
			TerraformType: "azurerm_resource_group", Supported: true,
		},
	}

	files := gen.GenerateAll(resources, "imports")
	require.NotEmpty(t, files)

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}

	// Network resources go to connectivity layer
	assert.Contains(t, paths, "imports/connectivity/import.tf")
	assert.Contains(t, paths, "imports/connectivity/resources.tf")

	// Resource groups go to general layer
	assert.Contains(t, paths, "imports/general/import.tf")
	assert.Contains(t, paths, "imports/general/resources.tf")
}

func TestHCLGenerator_GenerateAll_Empty(t *testing.T) {
	gen := NewHCLGenerator()
	files := gen.GenerateAll(nil, "imports")
	assert.Nil(t, files)
}

func TestHCLGenerator_InferLayer(t *testing.T) {
	gen := NewHCLGenerator()
	tests := []struct {
		azureType string
		expected  string
	}{
		{"microsoft.network/virtualnetworks", "connectivity"},
		{"microsoft.authorization/policyassignments", "governance"},
		{"microsoft.managedidentity/userassignedidentities", "identity"},
		{"microsoft.resources/resourcegroups", "general"},
		{"microsoft.compute/virtualmachines", "general"},
	}
	for _, tt := range tests {
		t.Run(tt.azureType, func(t *testing.T) {
			layer := gen.inferLayer(tt.azureType)
			assert.Equal(t, tt.expected, layer)
		})
	}
}

func TestHCLGenerator_LocalName(t *testing.T) {
	gen := NewHCLGenerator()
	tests := []struct {
		name     string
		expected string
	}{
		{"vnet-hub", "vnet_hub"},
		{"rg-core-001", "rg_core_001"},
		{"My Resource", "my_resource"},
		{"My_Resource", "my_resource"},
		{"  PROD VNet#1  ", "prod_vnet1"},
		{"@@@", "imported"},
		{"", "imported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ImportableResource{Name: tt.name}
			result := gen.localName(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHCLGenerator_GenerateAll_OnlyUnsupported(t *testing.T) {
	gen := NewHCLGenerator()
	resources := []ImportableResource{
		{
			ID:            "/sub/rg/providers/Microsoft.Compute/virtualMachines/vm1",
			AzureType:     "microsoft.compute/virtualmachines",
			Name:          "vm1",
			TerraformType: "unsupported",
			Supported:     false,
		},
	}

	files := gen.GenerateAll(resources, "imports")
	require.Len(t, files, 2)

	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
		assert.Contains(t, f.Content, "# TODO: manual import required")
	}

	assert.True(t, paths["imports/general/import.tf"])
	assert.True(t, paths["imports/general/resources.tf"])
}

func TestHCLGenerator_GenerateAll_MultipleLayersNoCollision(t *testing.T) {
	gen := NewHCLGenerator()
	resources := []ImportableResource{
		{
			ID:            "/sub/rg/providers/Microsoft.Network/virtualNetworks/vnet1",
			AzureType:     "microsoft.network/virtualnetworks",
			Name:          "vnet1",
			TerraformType: "azurerm_virtual_network",
			Supported:     true,
		},
		{
			ID:            "/sub/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/uai1",
			AzureType:     "microsoft.managedidentity/userassignedidentities",
			Name:          "uai1",
			TerraformType: "azurerm_user_assigned_identity",
			Supported:     true,
		},
		{
			ID:            "/sub/rg/providers/Microsoft.Authorization/policyAssignments/p1",
			AzureType:     "microsoft.authorization/policyassignments",
			Name:          "p1",
			TerraformType: "azurerm_policy_assignment",
			Supported:     true,
		},
	}

	files := gen.GenerateAll(resources, "imports")
	require.Len(t, files, 6)

	seen := map[string]struct{}{}
	for _, f := range files {
		if _, exists := seen[f.Path]; exists {
			t.Fatalf("duplicate generated path: %s", f.Path)
		}
		seen[f.Path] = struct{}{}
	}

	expected := []string{
		"imports/connectivity/import.tf",
		"imports/connectivity/resources.tf",
		"imports/identity/import.tf",
		"imports/identity/resources.tf",
		"imports/governance/import.tf",
		"imports/governance/resources.tf",
	}
	for _, p := range expected {
		if _, ok := seen[p]; !ok {
			t.Fatalf("expected generated file not found: %s", p)
		}
	}
}

func TestHCLGenerator_GenerateAll_GeneralLayerPathAlwaysNamespaced(t *testing.T) {
	gen := NewHCLGenerator()
	resources := []ImportableResource{
		{
			ID:            "/sub/resourceGroups/rg1",
			AzureType:     "microsoft.resources/resourcegroups",
			Name:          "rg1",
			TerraformType: "azurerm_resource_group",
			Supported:     true,
		},
	}

	files := gen.GenerateAll(resources, "imports")
	require.Len(t, files, 2)

	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}

	assert.True(t, paths["imports/general/import.tf"])
	assert.True(t, paths["imports/general/resources.tf"])
	assert.False(t, paths["imports/import.tf"], fmt.Sprintf("unexpected legacy non-namespaced path in %+v", paths))
}

// E8-S11: AVM module stub generation for blueprint resource types.

func TestHCLGenerator_GenerateResourceBlock_KeyVault_AVMStub(t *testing.T) {
	gen := NewHCLGenerator()
	r := ImportableResource{
		ID:            "/subscriptions/sub/resourceGroups/rg-app/providers/Microsoft.KeyVault/vaults/kv-app",
		AzureType:     "microsoft.keyvault/vaults",
		Name:          "kv-app",
		ResourceGroup: "rg-app",
		TerraformType: "azurerm_key_vault",
		Supported:     true,
	}

	block := gen.GenerateResourceBlock(r)
	// Must emit a module block, not a resource block
	assert.Contains(t, block, `module "kv_app"`)
	assert.Contains(t, block, `source  = "Azure/avm-res-keyvault-vault/azurerm"`)
	assert.NotContains(t, block, `resource "azurerm_key_vault"`)
}

func TestHCLGenerator_GenerateResourceBlock_AKS_AVMStub(t *testing.T) {
	gen := NewHCLGenerator()
	r := ImportableResource{
		ID:            "/subscriptions/sub/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/aks-prod",
		AzureType:     "microsoft.containerservice/managedclusters",
		Name:          "aks-prod",
		ResourceGroup: "rg-aks",
		TerraformType: "azurerm_kubernetes_cluster",
		Supported:     true,
	}

	block := gen.GenerateResourceBlock(r)
	assert.Contains(t, block, `module "aks_prod"`)
	assert.Contains(t, block, `source  = "Azure/avm-ptn-aks-production/azurerm"`)
	assert.Contains(t, block, `private_cluster_enabled = true`)
}

func TestHCLGenerator_InferLayer_BlueprintTypes(t *testing.T) {
	gen := NewHCLGenerator()
	assert.Equal(t, "blueprint-paas", gen.inferLayer("microsoft.web/sites"))
	assert.Equal(t, "blueprint-paas", gen.inferLayer("microsoft.apimanagement/service"))
	assert.Equal(t, "blueprint-aks", gen.inferLayer("microsoft.containerservice/managedclusters"))
	assert.Equal(t, "blueprint-aks", gen.inferLayer("microsoft.containerregistry/registries"))
	assert.Equal(t, "blueprint-avd", gen.inferLayer("microsoft.desktopvirtualization/hostpools"))
}
