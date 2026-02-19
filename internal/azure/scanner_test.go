package azure

import (
	"fmt"
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
	if v, ok := f.responses[key]; ok {
		return v, nil
	}
	return []any{}, nil
}

func TestScannerScan(t *testing.T) {
	cli := &fakeCLI{
		responses: map[string]any{
			"account management-group list":                                   []any{map[string]any{"id": "mg1", "name": "platform", "displayName": "Platform"}},
			"account list":                                                    []any{map[string]any{"id": "sub1", "name": "Sub One", "state": "Enabled", "managementGroupId": "/providers/Microsoft.Management/managementGroups/platform"}},
			"policy assignment list --all":                                    []any{map[string]any{"id": "pa1", "name": "deploy-mdfc-config", "scope": "/subscriptions/sub1", "policyDefinitionId": "caf-baseline"}},
			"role assignment list --all":                                      []any{map[string]any{"id": "ra1", "principalId": "p1", "principalType": "ServicePrincipal", "roleDefinitionName": "Contributor", "scope": "/subscriptions/sub1", "hasFederatedCredential": true}},
			"network vnet list --subscription sub1":                           []any{map[string]any{"id": "vnet1", "name": "hub-vnet", "addressSpace": map[string]any{"addressPrefixes": []any{"10.0.0.0/16"}}, "virtualNetworkPeerings": []any{map[string]any{"name": "peer1", "peeringState": "Connected", "remoteVirtualNetwork": map[string]any{"id": "vnet2"}}}}},
			"monitor diagnostic-settings list --resource /subscriptions/sub1": []any{map[string]any{"id": "diag1", "name": "default", "workspaceId": "law1"}},
			"security pricing list --subscription sub1":                       []any{map[string]any{"name": "VirtualMachines", "pricingTier": "Standard"}},
		},
		errors: map[string]error{},
	}

	scanner := NewScanner(cli, "")
	snapshot, warnings, err := scanner.Scan()
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	assert.Empty(t, warnings)
	assert.Len(t, snapshot.ManagementGroups, 1)
	assert.Len(t, snapshot.Subscriptions, 1)
	assert.Len(t, snapshot.VirtualNetworks, 1)
	assert.Len(t, snapshot.Peerings, 1)
	assert.Len(t, snapshot.DefenderPlans, 1)
}

func TestScannerScan_SubscriptionError(t *testing.T) {
	cli := &fakeCLI{
		responses: map[string]any{},
		errors: map[string]error{
			"account list": fmt.Errorf("boom"),
		},
	}

	scanner := NewScanner(cli, "")
	snapshot, warnings, err := scanner.Scan()
	assert.Error(t, err)
	assert.Nil(t, snapshot)
	assert.NotNil(t, warnings)
}
