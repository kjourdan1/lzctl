package wizard

import (
	"fmt"
	"net"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/kjourdan1/lzctl/internal/config"
)

// AddZoneConfig holds the result of the add-zone wizard.
type AddZoneConfig struct {
	Name         string
	Archetype    string
	Subscription string
	AddressSpace string
	Connected    bool
	Tags         map[string]string
}

// AddZoneWizard guides the user through creating a new landing zone.
type AddZoneWizard struct {
	prompter Prompter
	existing []config.LandingZone
}

// NewAddZoneWizard creates a wizard pre-loaded with existing zones for overlap checks.
func NewAddZoneWizard(p Prompter, existingZones []config.LandingZone) *AddZoneWizard {
	if p == nil {
		p = NewSurveyPrompter()
	}
	return &AddZoneWizard{prompter: p, existing: existingZones}
}

// Run executes the interactive wizard and returns the collected config.
func (w *AddZoneWizard) Run() (*AddZoneConfig, error) {
	name, err := w.prompter.Input("Landing zone name", "", survey.Required)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name.
	for _, z := range w.existing {
		if strings.EqualFold(z.Name, name) {
			return nil, fmt.Errorf("landing zone %q already exists", name)
		}
	}

	archetype, err := w.prompter.Select(
		"Archetype",
		[]string{"corp", "online", "sandbox"},
		"corp",
	)
	if err != nil {
		return nil, err
	}

	subscription, err := w.prompter.Input(
		"Subscription ID (UUID)",
		"",
		ValidateTenantID, // UUID validator works for subscription IDs too
	)
	if err != nil {
		return nil, err
	}

	addressSpace, err := w.prompter.Input(
		"Address space (CIDR, e.g. 10.1.0.0/24)",
		"",
		validateCIDR,
	)
	if err != nil {
		return nil, err
	}

	// Check for IP overlap with existing zones.
	if err := w.checkOverlap(addressSpace); err != nil {
		return nil, err
	}

	connected := true
	if archetype == "sandbox" {
		connected = false
	}
	connPrompt, err := w.prompter.Confirm("Connect to hub network?", connected)
	if err != nil {
		return nil, err
	}

	return &AddZoneConfig{
		Name:         strings.TrimSpace(name),
		Archetype:    archetype,
		Subscription: strings.TrimSpace(subscription),
		AddressSpace: strings.TrimSpace(addressSpace),
		Connected:    connPrompt,
		Tags:         map[string]string{},
	}, nil
}

// ToLandingZone converts the wizard result to a config.LandingZone.
func (c *AddZoneConfig) ToLandingZone() config.LandingZone {
	return config.LandingZone{
		Name:         c.Name,
		Subscription: c.Subscription,
		Archetype:    c.Archetype,
		AddressSpace: c.AddressSpace,
		Connected:    c.Connected,
		Tags:         c.Tags,
	}
}

func (w *AddZoneWizard) checkOverlap(newCIDR string) error {
	_, newNet, err := net.ParseCIDR(strings.TrimSpace(newCIDR))
	if err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", newCIDR, err)
	}

	for _, z := range w.existing {
		cidr := strings.TrimSpace(z.AddressSpace)
		if cidr == "" {
			continue
		}
		_, existNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if existNet.Contains(newNet.IP) || newNet.Contains(existNet.IP) {
			return fmt.Errorf("address space %s overlaps with existing zone %q (%s)", newCIDR, z.Name, cidr)
		}
	}
	return nil
}

func validateCIDR(val interface{}) error {
	v := strings.TrimSpace(fmt.Sprintf("%v", val))
	if v == "" {
		return fmt.Errorf("address space is required")
	}
	_, _, err := net.ParseCIDR(v)
	if err != nil {
		return fmt.Errorf("invalid CIDR notation: %w", err)
	}
	return nil
}
