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
