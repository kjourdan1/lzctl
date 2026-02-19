package importer

import "strings"

// ImportableResource represents one Azure resource candidate for Terraform import.
type ImportableResource struct {
	ID            string `json:"id"`
	AzureType     string `json:"azureType"`
	Name          string `json:"name"`
	Subscription  string `json:"subscription,omitempty"`
	ResourceGroup string `json:"resourceGroup,omitempty"`
	TerraformType string `json:"terraformType"`
	Supported     bool   `json:"supported"`
	Note          string `json:"note,omitempty"`
}

var mvpTerraformTypeByAzureType = map[string]string{
	"microsoft.resources/resourcegroups":               "azurerm_resource_group",
	"microsoft.network/virtualnetworks":                "azurerm_virtual_network",
	"microsoft.network/virtualnetworks/subnets":        "azurerm_subnet",
	"microsoft.network/networksecuritygroups":          "azurerm_network_security_group",
	"microsoft.network/routetables":                    "azurerm_route_table",
	"microsoft.keyvault/vaults":                        "azurerm_key_vault",
	"microsoft.storage/storageaccounts":                "azurerm_storage_account",
	"microsoft.managedidentity/userassignedidentities": "azurerm_user_assigned_identity",
	"microsoft.authorization/policyassignments":        "azurerm_subscription_policy_assignment",
}

// MapTerraformType returns Terraform type and support status for an Azure resource type.
func MapTerraformType(azureType string) (terraformType string, supported bool) {
	normalized := strings.ToLower(strings.TrimSpace(azureType))
	terraformType, supported = mvpTerraformTypeByAzureType[normalized]
	return terraformType, supported
}
