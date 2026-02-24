package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CrossCheck is a validation result entry for semantic/cross-field checks.
type CrossCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pass | warning | error
	Message string `json:"message"`
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// ValidateCross runs semantic checks that are not fully covered by JSON schema.
func ValidateCross(cfg *LZConfig, repoRoot string) ([]CrossCheck, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	checks := make([]CrossCheck, 0, 12)
	add := func(name, status, message string) {
		checks = append(checks, CrossCheck{Name: name, Status: status, Message: message})
	}

	if cfg.Spec.StateBackend.Subscription != "" && !isPlaceholder(cfg.Spec.StateBackend.Subscription) {
		if !uuidRE.MatchString(strings.TrimSpace(cfg.Spec.StateBackend.Subscription)) {
			add("state-backend-subscription", "error", "state backend subscription must be a valid UUID")
		} else {
			add("state-backend-subscription", "pass", "state backend subscription format is valid")
		}
	}

	// State life management checks — state is a critical asset
	if cfg.Spec.StateBackend.Versioning != nil && !*cfg.Spec.StateBackend.Versioning {
		add("state-versioning", "warning", "blob versioning is disabled — state history and rollback will not be available; set stateBackend.versioning: true")
	} else {
		add("state-versioning", "pass", "blob versioning is enabled for state history and rollback")
	}
	if cfg.Spec.StateBackend.SoftDelete != nil && !*cfg.Spec.StateBackend.SoftDelete {
		add("state-soft-delete", "warning", "soft delete is disabled — accidental state deletion will be unrecoverable; set stateBackend.softDelete: true")
	} else {
		add("state-soft-delete", "pass", "soft delete is enabled for state protection")
	}
	if cfg.Spec.StateBackend.StorageAccount != "" && !isPlaceholder(cfg.Spec.StateBackend.StorageAccount) {
		name := cfg.Spec.StateBackend.StorageAccount
		if len(name) < 3 || len(name) > 24 {
			add("state-storage-name", "error", fmt.Sprintf("storage account name %q must be 3-24 characters", name))
		}
	}

	for i, zone := range cfg.Spec.LandingZones {
		if zone.Subscription == "" || isPlaceholder(zone.Subscription) {
			continue
		}
		if !uuidRE.MatchString(strings.TrimSpace(zone.Subscription)) {
			add(fmt.Sprintf("landing-zone-subscription-%d", i+1), "error", fmt.Sprintf("landing zone %q subscription must be a valid UUID", zone.Name))
		}
	}

	type cidrScope struct {
		Name string
		CIDR string
		Net  *net.IPNet
	}

	networks := make([]cidrScope, 0, len(cfg.Spec.LandingZones)+1)
	if hub := cfg.Spec.Platform.Connectivity.Hub; hub != nil && strings.TrimSpace(hub.AddressSpace) != "" {
		_, ipn, err := net.ParseCIDR(strings.TrimSpace(hub.AddressSpace))
		if err != nil {
			add("hub-address-space", "error", fmt.Sprintf("invalid hub address space: %v", err))
		} else {
			networks = append(networks, cidrScope{Name: "hub", CIDR: hub.AddressSpace, Net: ipn})
			if prefixTooSmall(ipn) {
				add("hub-address-space-size", "warning", fmt.Sprintf("hub address space %s may be too small", hub.AddressSpace))
			}
		}
	}

	for _, zone := range cfg.Spec.LandingZones {
		cidr := strings.TrimSpace(zone.AddressSpace)
		if cidr == "" {
			continue
		}
		_, ipn, err := net.ParseCIDR(cidr)
		if err != nil {
			add("landing-zone-address-space", "error", fmt.Sprintf("landing zone %q has invalid address space: %v", zone.Name, err))
			continue
		}
		networks = append(networks, cidrScope{Name: "landing-zone:" + zone.Name, CIDR: cidr, Net: ipn})
		if prefixTooSmall(ipn) {
			add("landing-zone-address-space-size", "warning", fmt.Sprintf("landing zone %q address space %s may be too small", zone.Name, cidr))
		}
	}

	for i := 0; i < len(networks); i++ {
		for j := i + 1; j < len(networks); j++ {
			if overlaps(networks[i].Net, networks[j].Net) {
				add("address-space-overlap", "error", fmt.Sprintf("%s (%s) overlaps %s (%s)", networks[i].Name, networks[i].CIDR, networks[j].Name, networks[j].CIDR))
			}
		}
	}

	if len(cfg.Spec.Governance.Policies.Custom) == 0 {
		add("custom-policy-paths", "pass", "no custom policy paths defined")
	} else {
		for _, rel := range cfg.Spec.Governance.Policies.Custom {
			if strings.TrimSpace(rel) == "" {
				continue
			}
			if repoRoot == "" {
				add("custom-policy-paths", "warning", "cannot verify custom policy paths without repo root")
				break
			}
			candidate := filepath.Join(repoRoot, filepath.FromSlash(rel))
			if !fileExists(candidate) {
				add("custom-policy-paths", "error", fmt.Sprintf("custom policy path not found: %s", rel))
			}
		}
	}

	if !hasStatus(checks, "error") {
		add("cross-check-summary", "pass", "cross checks passed")
	}

	return checks, nil
}

func isPlaceholder(value string) bool {
	v := strings.TrimSpace(value)
	return v == "" || strings.HasPrefix(v, "<")
}

func prefixTooSmall(ipn *net.IPNet) bool {
	if ipn == nil {
		return false
	}
	one, bits := ipn.Mask.Size()
	if bits != 32 {
		return false
	}
	return one > 24
}

func overlaps(a, b *net.IPNet) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Contains(b.IP) || b.Contains(a.IP)
}

func hasStatus(checks []CrossCheck, status string) bool {
	for _, c := range checks {
		if c.Status == status {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, statErr := os.Stat(path)
	return statErr == nil && !info.IsDir()
}
