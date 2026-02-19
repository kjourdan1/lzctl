package importer

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCLI struct {
	responses map[string]any
	errors    map[string]error
}

func (f *fakeCLI) RunJSON(args ...string) (any, error) {
	key := strings.Join(args, " ")
	if err, ok := f.errors[key]; ok {
		return nil, err
	}
	if response, ok := f.responses[key]; ok {
		return response, nil
	}
	return []any{}, nil
}

func TestDiscovery_Discover_AllSubscriptions(t *testing.T) {
	cli := &fakeCLI{
		responses: map[string]any{
			"account list": []any{
				map[string]any{"id": "sub-001"},
			},
			"resource list --subscription sub-001": []any{
				map[string]any{
					"id":             "/subscriptions/sub-001/resourceGroups/rg-core/providers/Microsoft.Network/virtualNetworks/vnet-core",
					"type":           "Microsoft.Network/virtualNetworks",
					"name":           "vnet-core",
					"subscriptionId": "sub-001",
					"resourceGroup":  "rg-core",
				},
				map[string]any{
					"id":             "/subscriptions/sub-001/resourceGroups/rg-core/providers/Microsoft.Compute/virtualMachines/vm01",
					"type":           "Microsoft.Compute/virtualMachines",
					"name":           "vm01",
					"subscriptionId": "sub-001",
					"resourceGroup":  "rg-core",
				},
			},
		},
		errors: map[string]error{},
	}

	d := NewDiscovery(cli)
	resources, err := d.Discover(DiscoveryOptions{})
	require.NoError(t, err)
	require.Len(t, resources, 2)

	assert.Equal(t, "azurerm_virtual_network", resources[1].TerraformType)
	assert.True(t, resources[1].Supported)
	assert.Equal(t, "unsupported", resources[0].TerraformType)
	assert.False(t, resources[0].Supported)
	assert.Equal(t, "unsupported — manual import required", resources[0].Note)
}

func TestDiscovery_Discover_WithFilters(t *testing.T) {
	cli := &fakeCLI{
		responses: map[string]any{
			"resource list --subscription sub-123 --resource-group rg-app": []any{
				map[string]any{
					"id":             "/subscriptions/sub-123/resourceGroups/rg-app/providers/Microsoft.Network/virtualNetworks/vnet-app",
					"type":           "Microsoft.Network/virtualNetworks",
					"name":           "vnet-app",
					"subscriptionId": "sub-123",
					"resourceGroup":  "rg-app",
				},
				map[string]any{
					"id":             "/subscriptions/sub-123/resourceGroups/rg-app/providers/Microsoft.Storage/storageAccounts/stapp",
					"type":           "Microsoft.Storage/storageAccounts",
					"name":           "stapp",
					"subscriptionId": "sub-123",
					"resourceGroup":  "rg-app",
				},
			},
		},
		errors: map[string]error{},
	}

	d := NewDiscovery(cli)
	resources, err := d.Discover(DiscoveryOptions{
		Subscription:  "sub-123",
		ResourceGroup: "rg-app",
		IncludeTypes:  []string{"microsoft.network/virtualnetworks", "microsoft.storage/storageaccounts"},
		ExcludeTypes:  []string{"microsoft.storage/storageaccounts"},
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "microsoft.network/virtualnetworks", resources[0].AzureType)
	assert.Equal(t, "azurerm_virtual_network", resources[0].TerraformType)
}

func TestDiscovery_Discover_ResourceListError(t *testing.T) {
	cli := &fakeCLI{
		responses: map[string]any{},
		errors: map[string]error{
			"resource list --subscription sub-err": errors.New("boom"),
		},
	}

	d := NewDiscovery(cli)
	resources, err := d.Discover(DiscoveryOptions{Subscription: "sub-err"})
	assert.Nil(t, resources)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listing resources for subscription sub-err")
}
