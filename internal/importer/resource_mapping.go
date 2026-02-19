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
	AVMSource     string `json:"avmSource,omitempty"` // AVM Registry source when applicable
	Supported     bool   `json:"supported"`
	Note          string `json:"note,omitempty"`
}

// mvpTerraformTypeByAzureType maps normalised Azure resource types to their
// canonical azurerm_* Terraform resource type.
var mvpTerraformTypeByAzureType = map[string]string{
	// Core infrastructure
	"microsoft.resources/resourcegroups":        "azurerm_resource_group",
	"microsoft.network/virtualnetworks":         "azurerm_virtual_network",
	"microsoft.network/virtualnetworks/subnets": "azurerm_subnet",
	"microsoft.network/networksecuritygroups":   "azurerm_network_security_group",
	"microsoft.network/routetables":             "azurerm_route_table",
	"microsoft.network/privateendpoints":        "azurerm_private_endpoint",
	"microsoft.network/privatednszones":         "azurerm_private_dns_zone",
	// Security & identity
	"microsoft.keyvault/vaults":                        "azurerm_key_vault",
	"microsoft.keyvault/managedhsms":                   "azurerm_key_vault_managed_hardware_security_module",
	"microsoft.storage/storageaccounts":                "azurerm_storage_account",
	"microsoft.managedidentity/userassignedidentities": "azurerm_user_assigned_identity",
	"microsoft.authorization/policyassignments":        "azurerm_subscription_policy_assignment",
	// App Service / Function App (paas-secure blueprint — AVM)
	"microsoft.web/sites":         "azurerm_linux_web_app",
	"microsoft.web/serverfarms":   "azurerm_service_plan",
	"microsoft.web/staticwebapps": "azurerm_static_web_app",
	// API Management (paas-secure blueprint — AVM)
	"microsoft.apimanagement/service": "azurerm_api_management",
	// Container platform (aks-platform / aca-platform blueprints — AVM)
	"microsoft.containerservice/managedclusters": "azurerm_kubernetes_cluster",
	"microsoft.containerregistry/registries":     "azurerm_container_registry",
	"microsoft.app/managedenvironments":          "azurerm_container_app_environment",
	"microsoft.app/containerapps":                "azurerm_container_app",
	// Virtual Desktop (avd-secure blueprint — AVM)
	"microsoft.desktopvirtualization/hostpools":         "azurerm_virtual_desktop_host_pool",
	"microsoft.desktopvirtualization/applicationgroups": "azurerm_virtual_desktop_application_group",
	"microsoft.desktopvirtualization/workspaces":        "azurerm_virtual_desktop_workspace",
}

// avmSourceByTerraformType maps azurerm_* types to their AVM Registry module
// source path (Azure/<module>/azurerm). Used to generate module import stubs
// instead of raw resource blocks when a suitable AVM module exists.
var avmSourceByTerraformType = map[string]string{
	"azurerm_key_vault":          "Azure/avm-res-keyvault-vault/azurerm",
	"azurerm_linux_web_app":      "Azure/avm-res-web-site/azurerm",
	"azurerm_service_plan":       "Azure/avm-res-web-serverfarm/azurerm",
	"azurerm_api_management":     "Azure/avm-res-apimanagement-service/azurerm",
	"azurerm_kubernetes_cluster": "Azure/avm-ptn-aks-production/azurerm",
	"azurerm_container_registry": "Azure/avm-res-containerregistry-registry/azurerm",
}

// MapTerraformType returns the Terraform type, AVM source hint, and support
// status for an Azure resource type. When an AVM module exists for the type,
// AVMSource is non-empty and callers should prefer a module stub over a raw
// resource block.
func MapTerraformType(azureType string) (terraformType string, supported bool) {
	normalized := strings.ToLower(strings.TrimSpace(azureType))
	terraformType, supported = mvpTerraformTypeByAzureType[normalized]
	return terraformType, supported
}

// AVMSource returns the AVM Registry source string for a given azurerm_*
// Terraform resource type, or an empty string when no AVM module is mapped.
func AVMSource(terraformType string) string {
	return avmSourceByTerraformType[strings.ToLower(strings.TrimSpace(terraformType))]
}

// IsBlueprintLayer reports whether a target layer path corresponds to a
// landing-zone blueprint directory (landing-zones/<name>/blueprint).
// Used by the import command to validate --layer values and emit correct
// import block paths.
func IsBlueprintLayer(layer string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(layer), "\\", "/"))
	parts := strings.Split(normalized, "/")
	return len(parts) == 3 &&
		parts[0] == "landing-zones" &&
		parts[2] == "blueprint"
}
