package template

import (
	"testing"

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
