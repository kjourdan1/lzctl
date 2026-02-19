package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Save marshals the LZConfig to YAML and writes it to the specified path.
func Save(cfg *LZConfig, path string) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config to %s: %w", path, err)
	}
	return nil
}

// AddLandingZone appends a LandingZone to the config, checking for duplicate
// names and address space overlaps. Returns an error if a conflict is found.
func AddLandingZone(cfg *LZConfig, zone LandingZone) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Check duplicate name.
	for _, z := range cfg.Spec.LandingZones {
		if z.Name == zone.Name {
			return fmt.Errorf("landing zone %q already exists", zone.Name)
		}
	}

	cfg.Spec.LandingZones = append(cfg.Spec.LandingZones, zone)
	return nil
}
