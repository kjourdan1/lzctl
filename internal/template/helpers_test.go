package template

import (
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCAFName(t *testing.T) {
	assert.Equal(t, "rg-platform-weu", CAFName("rg", "platform", "weu"))
}

func TestRegionShort(t *testing.T) {
	assert.Equal(t, "weu", RegionShort("westeurope"))
	assert.Equal(t, "abc", RegionShort("abc-region"))
}

func TestCIDRSubnet(t *testing.T) {
	subnet, err := CIDRSubnet("10.0.0.0/16", 24, 0)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", subnet)

	subnet, err = CIDRSubnet("10.0.0.0/16", 24, 1)
	require.NoError(t, err)
	assert.Equal(t, "10.0.1.0/24", subnet)
}

func TestSlugify(t *testing.T) {
	assert.Equal(t, "my-project", Slugify("My Project"))
	assert.Equal(t, "default", Slugify("***"))
}

func TestStorageAccountName(t *testing.T) {
	assert.Equal(t, "contosoplatformtfstate", StorageAccountName("contoso-platform-tfstate"))
	assert.Len(t, StorageAccountName("averyveryverylongstorageaccountnametoolong"), 24)
}

func TestToJSON(t *testing.T) {
	result := ToJSON([]string{"a", "b"})
	assert.Equal(t, `["a","b"]`, result)
}

func TestDNSZoneRef(t *testing.T) {
	assert.Equal(t, "privatelink.azurewebsites.net", DNSZoneRef("appService"))
	assert.Equal(t, "privatelink.vaultcore.azure.net", DNSZoneRef("keyVault"))
	assert.Equal(t, "privatelink.azure-api.net", DNSZoneRef("apim"))
	assert.Equal(t, "", DNSZoneRef("unknown-service"))
}

func TestConnectivityRemoteState(t *testing.T) {
	cfg := &config.LZConfig{Spec: config.Spec{StateBackend: config.StateBackend{
		ResourceGroup:  "rg-lz-state",
		StorageAccount: "stlzstate",
		Container:      "tfstate",
		Subscription:   "00000000-0000-0000-0000-000000000001",
	}}}

	block := ConnectivityRemoteState(cfg)
	assert.Contains(t, block, `data "terraform_remote_state" "connectivity"`)
	assert.Contains(t, block, `key                  = "platform-connectivity.tfstate"`)
	assert.Contains(t, block, `resource_group_name  = "rg-lz-state"`)
}
