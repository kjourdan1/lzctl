package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapTerraformType_MappedType(t *testing.T) {
	typeName, supported := MapTerraformType("Microsoft.Network/virtualNetworks")
	assert.True(t, supported)
	assert.Equal(t, "azurerm_virtual_network", typeName)
}

func TestMapTerraformType_UnsupportedType(t *testing.T) {
	typeName, supported := MapTerraformType("Microsoft.Compute/virtualMachines")
	assert.False(t, supported)
	assert.Equal(t, "", typeName)
}

// E8-S11 — AVM resource type mappings

func TestMapTerraformType_WebSites(t *testing.T) {
	tfType, supported := MapTerraformType("Microsoft.Web/sites")
	assert.True(t, supported)
	assert.Equal(t, "azurerm_linux_web_app", tfType)
}

func TestMapTerraformType_APIM(t *testing.T) {
	tfType, supported := MapTerraformType("Microsoft.ApiManagement/service")
	assert.True(t, supported)
	assert.Equal(t, "azurerm_api_management", tfType)
}

func TestMapTerraformType_AKS(t *testing.T) {
	tfType, supported := MapTerraformType("Microsoft.ContainerService/managedClusters")
	assert.True(t, supported)
	assert.Equal(t, "azurerm_kubernetes_cluster", tfType)
}

func TestMapTerraformType_ACR(t *testing.T) {
	tfType, supported := MapTerraformType("Microsoft.ContainerRegistry/registries")
	assert.True(t, supported)
	assert.Equal(t, "azurerm_container_registry", tfType)
}

func TestAVMSource_KeyVault(t *testing.T) {
	src := AVMSource("azurerm_key_vault")
	assert.Equal(t, "Azure/avm-res-keyvault-vault/azurerm", src)
}

func TestAVMSource_AKS(t *testing.T) {
	src := AVMSource("azurerm_kubernetes_cluster")
	assert.Equal(t, "Azure/avm-ptn-aks-production/azurerm", src)
}

func TestAVMSource_UnknownType(t *testing.T) {
	src := AVMSource("azurerm_resource_group")
	assert.Equal(t, "", src)
}

func TestIsBlueprintLayer(t *testing.T) {
	assert.True(t, IsBlueprintLayer("landing-zones/contoso-app/blueprint"))
	assert.True(t, IsBlueprintLayer("landing-zones/my-zone/blueprint"))
	// Not blueprint layers
	assert.False(t, IsBlueprintLayer("platform/connectivity"))
	assert.False(t, IsBlueprintLayer("landing-zones/my-zone"))
	assert.False(t, IsBlueprintLayer("imports"))
	assert.False(t, IsBlueprintLayer("landing-zones/my/zone/blueprint"))
}
