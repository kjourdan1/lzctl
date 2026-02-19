package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave_RoundTrip(t *testing.T) {
	cfg := &LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Metadata: Metadata{
			Name:          "test-tenant",
			Tenant:        "test",
			PrimaryRegion: "westeurope",
		},
		Spec: Spec{
			LandingZones: []LandingZone{
				{Name: "zone-a", Archetype: "corp", AddressSpace: "10.0.0.0/24", Connected: true},
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "lzctl.yaml")

	err := Save(cfg, path)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, cfg.Metadata.Name, loaded.Metadata.Name)
	assert.Len(t, loaded.Spec.LandingZones, 1)
	assert.Equal(t, "zone-a", loaded.Spec.LandingZones[0].Name)
}

func TestSave_NilConfig(t *testing.T) {
	err := Save(nil, "/tmp/test.yaml")
	assert.Error(t, err)
}

func TestSave_InvalidPath(t *testing.T) {
	cfg := &LZConfig{APIVersion: "lzctl/v1"}
	err := Save(cfg, filepath.Join(string(os.PathSeparator), "nonexistent", "deep", "path", "test.yaml"))
	assert.Error(t, err)
}

func TestAddLandingZone_Success(t *testing.T) {
	cfg := &LZConfig{
		Spec: Spec{
			LandingZones: []LandingZone{
				{Name: "zone-a", Archetype: "corp"},
			},
		},
	}

	zone := LandingZone{Name: "zone-b", Archetype: "online", AddressSpace: "10.1.0.0/24"}
	err := AddLandingZone(cfg, zone)
	require.NoError(t, err)
	assert.Len(t, cfg.Spec.LandingZones, 2)
}

func TestAddLandingZone_Duplicate(t *testing.T) {
	cfg := &LZConfig{
		Spec: Spec{
			LandingZones: []LandingZone{
				{Name: "zone-a", Archetype: "corp"},
			},
		},
	}

	zone := LandingZone{Name: "zone-a", Archetype: "online"}
	err := AddLandingZone(cfg, zone)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddLandingZone_NilConfig(t *testing.T) {
	zone := LandingZone{Name: "zone-a"}
	err := AddLandingZone(nil, zone)
	assert.Error(t, err)
}
