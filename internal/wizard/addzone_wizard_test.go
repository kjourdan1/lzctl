package wizard

import (
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type addZoneMockPrompter struct {
	inputResponses   []string
	selectResponses  []string
	confirmResponses []bool
	inputIdx         int
	selectIdx        int
	confirmIdx       int
}

func (m *addZoneMockPrompter) Input(label, defaultValue string, _ interface{}) (string, error) {
	if m.inputIdx >= len(m.inputResponses) {
		return defaultValue, nil
	}
	val := m.inputResponses[m.inputIdx]
	m.inputIdx++
	return val, nil
}

func (m *addZoneMockPrompter) Select(label string, options []string, defaultValue string) (string, error) {
	if m.selectIdx >= len(m.selectResponses) {
		return defaultValue, nil
	}
	val := m.selectResponses[m.selectIdx]
	m.selectIdx++
	return val, nil
}

func (m *addZoneMockPrompter) Confirm(label string, defaultValue bool) (bool, error) {
	if m.confirmIdx >= len(m.confirmResponses) {
		return defaultValue, nil
	}
	val := m.confirmResponses[m.confirmIdx]
	m.confirmIdx++
	return val, nil
}

func (m *addZoneMockPrompter) MultiSelect(label string, options []string, defaults []string) ([]string, error) {
	return defaults, nil
}

func TestAddZoneWizard_Success(t *testing.T) {
	existing := []config.LandingZone{
		{Name: "existing-zone", AddressSpace: "10.0.0.0/24"},
	}

	mock := &addZoneMockPrompter{
		inputResponses:   []string{"new-zone", "11111111-2222-3333-4444-555555555555", "10.1.0.0/24"},
		selectResponses:  []string{"corp"},
		confirmResponses: []bool{true},
	}

	wiz := NewAddZoneWizard(mock, existing)
	result, err := wiz.Run()
	require.NoError(t, err)

	assert.Equal(t, "new-zone", result.Name)
	assert.Equal(t, "corp", result.Archetype)
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", result.Subscription)
	assert.Equal(t, "10.1.0.0/24", result.AddressSpace)
	assert.True(t, result.Connected)
}

func TestAddZoneWizard_DuplicateName(t *testing.T) {
	existing := []config.LandingZone{
		{Name: "my-zone", AddressSpace: "10.0.0.0/24"},
	}

	mock := &addZoneMockPrompter{
		inputResponses:   []string{"my-zone"},
		selectResponses:  []string{"corp"},
		confirmResponses: []bool{true},
	}

	wiz := NewAddZoneWizard(mock, existing)
	_, err := wiz.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddZoneWizard_OverlappingCIDR(t *testing.T) {
	existing := []config.LandingZone{
		{Name: "existing-zone", AddressSpace: "10.1.0.0/16"},
	}

	mock := &addZoneMockPrompter{
		inputResponses:   []string{"new-zone", "11111111-2222-3333-4444-555555555555", "10.1.0.0/24"},
		selectResponses:  []string{"corp"},
		confirmResponses: []bool{true},
	}

	wiz := NewAddZoneWizard(mock, existing)
	_, err := wiz.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overlaps")
}

func TestAddZoneConfig_ToLandingZone(t *testing.T) {
	cfg := &AddZoneConfig{
		Name:         "test-zone",
		Archetype:    "online",
		Subscription: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		AddressSpace: "10.5.0.0/24",
		Connected:    false,
		Tags:         map[string]string{"env": "dev"},
	}

	lz := cfg.ToLandingZone()
	assert.Equal(t, "test-zone", lz.Name)
	assert.Equal(t, "online", lz.Archetype)
	assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", lz.Subscription)
	assert.Equal(t, "10.5.0.0/24", lz.AddressSpace)
	assert.False(t, lz.Connected)
	assert.Equal(t, "dev", lz.Tags["env"])
}

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"10.0.0.0/24", true},
		{"192.168.1.0/16", true},
		{"invalid", false},
		{"", false},
		{"10.0.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateCIDR(tt.input)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
