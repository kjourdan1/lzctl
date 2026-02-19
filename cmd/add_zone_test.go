package cmd

import (
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestRemoveZone(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			LandingZones: []config.LandingZone{
				{Name: "zone-a", Archetype: "corp"},
				{Name: "zone-b", Archetype: "online"},
				{Name: "zone-c", Archetype: "sandbox"},
			},
		},
	}

	removeZone(cfg, "zone-b")
	assert.Len(t, cfg.Spec.LandingZones, 2)
	assert.Equal(t, "zone-a", cfg.Spec.LandingZones[0].Name)
	assert.Equal(t, "zone-c", cfg.Spec.LandingZones[1].Name)
}

func TestRemoveZone_NotFound(t *testing.T) {
	cfg := &config.LZConfig{
		Spec: config.Spec{
			LandingZones: []config.LandingZone{
				{Name: "zone-a", Archetype: "corp"},
			},
		},
	}

	removeZone(cfg, "nonexistent")
	assert.Len(t, cfg.Spec.LandingZones, 1)
}

func TestAddZoneDryRun_JSON(t *testing.T) {
	// Verify dry-run doesn't panic with JSON output.
	zone := config.LandingZone{
		Name:         "test-zone",
		Archetype:    "corp",
		Subscription: "11111111-2222-3333-4444-555555555555",
		AddressSpace: "10.2.0.0/24",
		Connected:    true,
	}

	// Set global for JSON mode.
	origJSON := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = origJSON }()

	err := addZoneDryRun(zone)
	assert.NoError(t, err)
}
