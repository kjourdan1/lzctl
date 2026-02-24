package config

import (
	"fmt"
	"net"
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

	if err := os.WriteFile(path, data, 0o600); err != nil {
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

	// Check duplicate name and address space overlaps.
	for _, z := range cfg.Spec.LandingZones {
		if z.Name == zone.Name {
			return fmt.Errorf("landing zone %q already exists", zone.Name)
		}
		// Check address space overlap if both zones have an address space.
		if z.AddressSpace != "" && zone.AddressSpace != "" {
			_, existingNet, err1 := net.ParseCIDR(z.AddressSpace)
			_, newNet, err2 := net.ParseCIDR(zone.AddressSpace)
			if err1 == nil && err2 == nil && overlaps(existingNet, newNet) {
				return fmt.Errorf("landing zone %q address space %s overlaps with existing zone %q (%s)",
					zone.Name, zone.AddressSpace, z.Name, z.AddressSpace)
			}
		}
	}

	cfg.Spec.LandingZones = append(cfg.Spec.LandingZones, zone)
	return nil
}
