package cmd

import (
	"fmt"
	"net"
	"strings"
)

func parseTags(tags []string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		parts := strings.SplitN(t, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

// validateWorkloadName checks kebab-case naming convention.
func validateWorkloadName(name string) error {
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if !kebabCaseRegex.MatchString(name) {
		return fmt.Errorf("--name must be kebab-case (lowercase alphanumeric with hyphens): %q", name)
	}
	return nil
}

// validateArchetype checks the archetype is one of corp, online, sandbox.
func validateArchetype(archetype string) error {
	if archetype == "" {
		return nil // optional
	}
	for _, a := range allowedArchetypes {
		if archetype == a {
			return nil
		}
	}
	return fmt.Errorf("--archetype must be one of %v, got %q", allowedArchetypes, archetype)
}

// validateAddressSpace validates CIDR notation.
func validateAddressSpace(addressSpace string) error {
	if addressSpace == "" {
		return nil // optional
	}
	_, _, err := net.ParseCIDR(addressSpace)
	if err != nil {
		return fmt.Errorf("--address-space must be valid CIDR notation: %w", err)
	}
	return nil
}
