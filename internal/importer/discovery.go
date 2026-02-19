package importer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kjourdan1/lzctl/internal/azure"
)

// DiscoveryOptions controls importer discovery filtering.
type DiscoveryOptions struct {
	Scope         string
	Subscription  string
	ResourceGroup string
	IncludeTypes  []string
	ExcludeTypes  []string
}

// Discovery discovers importable resources from Azure subscriptions.
type Discovery struct {
	cli azure.CLI
}

// NewDiscovery creates a discovery instance using the provided CLI.
func NewDiscovery(cli azure.CLI) *Discovery {
	if cli == nil {
		cli = azure.NewAzCLI()
	}
	return &Discovery{cli: cli}
}

// Discover returns resources discovered via az resource list with Terraform mappings.
func (d *Discovery) Discover(options DiscoveryOptions) ([]ImportableResource, error) {
	subscriptions, err := d.resolveSubscriptions(options)
	if err != nil {
		return nil, err
	}

	include := normalizeTypeSet(options.IncludeTypes)
	exclude := normalizeTypeSet(options.ExcludeTypes)

	resources := make([]ImportableResource, 0)
	for _, sub := range subscriptions {
		args := []string{"resource", "list", "--subscription", sub}
		if strings.TrimSpace(options.ResourceGroup) != "" {
			args = append(args, "--resource-group", strings.TrimSpace(options.ResourceGroup))
		}

		raw, runErr := d.cli.RunJSON(args...)
		if runErr != nil {
			return nil, fmt.Errorf("listing resources for subscription %s: %w", sub, runErr)
		}

		for _, item := range asSlice(raw) {
			resource := parseImportableResource(item)
			if resource.ID == "" || resource.AzureType == "" {
				continue
			}

			normalizedType := strings.ToLower(resource.AzureType)
			if len(include) > 0 && !include[normalizedType] {
				continue
			}
			if exclude[normalizedType] {
				continue
			}

			terraformType, supported := MapTerraformType(resource.AzureType)
			resource.Supported = supported
			if supported {
				resource.TerraformType = terraformType
			} else {
				resource.TerraformType = "unsupported"
				resource.Note = "unsupported — manual import required"
			}
			resources = append(resources, resource)
		}
	}

	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Subscription != resources[j].Subscription {
			return resources[i].Subscription < resources[j].Subscription
		}
		if resources[i].AzureType != resources[j].AzureType {
			return resources[i].AzureType < resources[j].AzureType
		}
		return resources[i].ID < resources[j].ID
	})

	return resources, nil
}

func (d *Discovery) resolveSubscriptions(options DiscoveryOptions) ([]string, error) {
	if strings.TrimSpace(options.Subscription) != "" {
		return []string{strings.TrimSpace(options.Subscription)}, nil
	}

	raw, err := d.cli.RunJSON("account", "list")
	if err != nil {
		return nil, fmt.Errorf("listing subscriptions: %w", err)
	}

	out := make([]string, 0)
	for _, item := range asSlice(raw) {
		entry := asMap(item)
		subscriptionID := asString(entry["id"])
		if subscriptionID != "" {
			out = append(out, subscriptionID)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no subscriptions returned by az account list")
	}

	return out, nil
}

func parseImportableResource(item any) ImportableResource {
	entry := asMap(item)
	return ImportableResource{
		ID:            asString(entry["id"]),
		AzureType:     strings.ToLower(asString(entry["type"])),
		Name:          asString(entry["name"]),
		Subscription:  asString(entry["subscriptionId"]),
		ResourceGroup: asString(entry["resourceGroup"]),
	}
}

func normalizeTypeSet(values []string) map[string]bool {
	normalized := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" {
			normalized[trimmed] = true
		}
	}
	return normalized
}

func asSlice(v any) []any {
	s, ok := v.([]any)
	if !ok {
		return []any{}
	}
	return s
}

func asMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return m
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}
