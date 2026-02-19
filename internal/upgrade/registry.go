// Package upgrade provides Terraform module version checking and upgrading
// capabilities via the Terraform Registry API.
package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// RegistryBaseURL is the Terraform public registry API endpoint.
const RegistryBaseURL = "https://registry.terraform.io/v1/modules"

// ModuleVersions holds the response from the registry versions endpoint.
type ModuleVersions struct {
	Modules []ModuleEntry `json:"modules"`
}

// ModuleEntry is a single module in the versions response.
type ModuleEntry struct {
	Versions []VersionEntry `json:"versions"`
}

// VersionEntry is a single version of a module.
type VersionEntry struct {
	Version string `json:"version"`
}

// ModuleRef identifies a Terraform Registry module.
type ModuleRef struct {
	Namespace string // e.g. "Azure"
	Name      string // e.g. "avm-res-network-virtualnetwork"
	Provider  string // e.g. "azurerm"
}

// String returns the registry path format.
func (m ModuleRef) String() string {
	return fmt.Sprintf("%s/%s/%s", m.Namespace, m.Name, m.Provider)
}

// UpgradeInfo holds the result of a version check for one module.
type UpgradeInfo struct {
	Module         ModuleRef `json:"module"`
	CurrentVersion string    `json:"currentVersion"`
	LatestVersion  string    `json:"latestVersion"`
	UpgradeAvail   bool      `json:"upgradeAvailable"`
	Error          string    `json:"error,omitempty"`
}

// Client queries the Terraform Registry API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a registry client with sensible defaults.
func NewClient() *Client {
	return &Client{
		BaseURL: RegistryBaseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LatestVersion fetches the latest version of a module from the registry.
func (c *Client) LatestVersion(mod ModuleRef) (string, error) {
	url := fmt.Sprintf("%s/%s/versions", c.BaseURL, mod.String())

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching versions for %s: %w", mod, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("registry returned %d for %s: %s", resp.StatusCode, mod, string(body))
	}

	var result ModuleVersions
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding registry response for %s: %w", mod, err)
	}

	if len(result.Modules) == 0 || len(result.Modules[0].Versions) == 0 {
		return "", fmt.Errorf("no versions found for %s", mod)
	}

	versions := result.Modules[0].Versions
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})

	return versions[0].Version, nil
}

// CheckUpgrades checks multiple modules against the registry and returns
// upgrade information for each.
func CheckUpgrades(client *Client, modules []ModulePin) []UpgradeInfo {
	results := make([]UpgradeInfo, 0, len(modules))

	for _, pin := range modules {
		info := UpgradeInfo{
			Module:         pin.Ref,
			CurrentVersion: pin.Version,
		}

		latest, err := client.LatestVersion(pin.Ref)
		if err != nil {
			info.Error = err.Error()
			results = append(results, info)
			continue
		}

		info.LatestVersion = latest
		info.UpgradeAvail = compareVersions(latest, pin.Version) > 0
		results = append(results, info)
	}

	return results
}

// compareVersions compares two semver-ish version strings.
// Returns >0 if a > b, <0 if a < b, 0 if equal.
func compareVersions(a, b string) int {
	aParts := splitVersion(a)
	bParts := splitVersion(b)

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] > bParts[i] {
			return 1
		}
		if aParts[i] < bParts[i] {
			return -1
		}
	}
	return len(aParts) - len(bParts)
}

func splitVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		// Parse numeric part, ignoring pre-release suffixes.
		num := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				num = num*10 + int(c-'0')
			} else {
				break
			}
		}
		result[i] = num
	}
	return result
}
