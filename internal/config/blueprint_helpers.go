package config

import (
	"encoding/json"
	"fmt"
)

// ParseAKSBlueprintConfig decodes a Blueprint.Overrides map into a typed
// AKSBlueprintConfig struct. Unknown keys are silently ignored, making it safe
// to call on overrides that contain non-AKS keys.
func ParseAKSBlueprintConfig(overrides map[string]any) (AKSBlueprintConfig, error) {
	var cfg AKSBlueprintConfig

	if len(overrides) == 0 {
		return cfg, nil
	}

	// Round-trip through JSON to handle map[string]any → typed struct.
	b, err := json.Marshal(overrides)
	if err != nil {
		return cfg, fmt.Errorf("marshalling aks blueprint overrides: %w", err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing aks blueprint overrides: %w", err)
	}

	return cfg, nil
}

// ValidateArgoCDConfig returns an error when the argocd configuration is
// semantically invalid (e.g. unknown mode, missing repoUrl when enabled).
func ValidateArgoCDConfig(a ArgoCDConfig) error {
	if !a.Enabled {
		return nil
	}
	switch a.Mode {
	case "", "extension", "helm":
		// valid
	default:
		return fmt.Errorf("argocd.mode %q is invalid: must be \"extension\" or \"helm\"", a.Mode)
	}
	if a.RepoURL == "" {
		return fmt.Errorf("argocd.repoUrl is required when argocd.enabled = true")
	}
	return nil
}
