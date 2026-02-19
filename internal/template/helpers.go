package template

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
	texttemplate "text/template"

	"github.com/kjourdan1/lzctl/internal/config"
)

// HelperFuncMap returns template helper functions.
func HelperFuncMap() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"cafName":        CAFName,
		"regionShort":    RegionShort,
		"cidrSubnet":     CIDRSubnet,
		"slugify":        Slugify,
		"storageAccName": StorageAccountName,
		"toJSON":         ToJSON,
		"dnsZoneRef":     DNSZoneRef,
	}
}

// DNSZoneRef returns the central Private DNS zone name for a known Azure service.
func DNSZoneRef(serviceType string) string {
	s := strings.ToLower(strings.TrimSpace(serviceType))
	switch s {
	case "appservice", "function", "functions", "azurewebsites":
		return "privatelink.azurewebsites.net"
	case "keyvault", "vault":
		return "privatelink.vaultcore.azure.net"
	case "apim", "api-management":
		return "privatelink.azure-api.net"
	case "acr", "containerregistry":
		return "privatelink.azurecr.io"
	case "aks":
		return "privatelink.<region>.azmk8s.io"
	default:
		return ""
	}
}

// ConnectivityRemoteState returns an azurerm remote state block targeting connectivity state.
func ConnectivityRemoteState(cfg *config.LZConfig) string {
	if cfg == nil {
		return ""
	}
	return fmt.Sprintf(`data "terraform_remote_state" "connectivity" {
  backend = "azurerm"
  config = {
    resource_group_name  = %q
    storage_account_name = %q
    container_name       = %q
    key                  = "platform-connectivity.tfstate"
    subscription_id      = %q
    use_azuread_auth     = true
  }
}
`, cfg.Spec.StateBackend.ResourceGroup, cfg.Spec.StateBackend.StorageAccount, cfg.Spec.StateBackend.Container, cfg.Spec.StateBackend.Subscription)
}

// ToJSON marshals a value to a compact JSON string for template rendering.
func ToJSON(value interface{}) string {
	b, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// CAFName builds a simple CAF-style name from parts.
func CAFName(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(strings.ToLower(p))
		if s != "" {
			clean = append(clean, s)
		}
	}
	return strings.Join(clean, "-")
}

var regionCodes = map[string]string{
	"westeurope":         "weu",
	"northeurope":        "neu",
	"francecentral":      "frc",
	"eastus":             "eus",
	"eastus2":            "eus2",
	"westus2":            "wus2",
	"uksouth":            "uks",
	"swedencentral":      "swc",
	"germanywestcentral": "gwc",
	"canadacentral":      "cac",
}

// RegionShort maps an Azure region to its short code.
func RegionShort(region string) string {
	norm := strings.TrimSpace(strings.ToLower(region))
	if code, ok := regionCodes[norm]; ok {
		return code
	}
	if len(norm) >= 3 {
		return norm[:3]
	}
	return norm
}

// CIDRSubnet returns the indexed subnet for a parent CIDR.
func CIDRSubnet(parent string, newPrefix, index int) (string, error) {
	ip, ipNet, err := net.ParseCIDR(parent)
	if err != nil {
		return "", fmt.Errorf("invalid parent cidr: %w", err)
	}

	ones, bits := ipNet.Mask.Size()
	if newPrefix < ones || newPrefix > bits {
		return "", fmt.Errorf("invalid new prefix %d for %s", newPrefix, parent)
	}

	subnetCount := 1 << (newPrefix - ones)
	if index < 0 || index >= subnetCount {
		return "", fmt.Errorf("subnet index %d out of range [0,%d)", index, subnetCount)
	}

	base := ip.To4()
	if base == nil {
		return "", fmt.Errorf("only IPv4 is supported")
	}

	size := 1 << (bits - newPrefix)
	offset := index * size

	addr := uint32(base[0])<<24 | uint32(base[1])<<16 | uint32(base[2])<<8 | uint32(base[3])
	addr += uint32(offset)

	result := net.IPv4(byte(addr>>24), byte(addr>>16), byte(addr>>8), byte(addr)).String()
	return fmt.Sprintf("%s/%d", result, newPrefix), nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9-]+`)

// Slugify normalizes a string into kebab-case.
func Slugify(value string) string {
	s := strings.ToLower(strings.TrimSpace(value))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.Join(strings.Fields(s), "-")
	s = nonSlug.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if s == "" {
		return "default"
	}
	return s
}

// StorageAccountName returns a valid Azure storage account name (<=24 chars, lowercase alnum).
func StorageAccountName(value string) string {
	s := strings.ToLower(value)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, s)
	if len(s) > 24 {
		s = s[:24]
	}
	if s == "" {
		return "stlzctlstate"
	}
	return s
}
