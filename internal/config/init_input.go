package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type InitInput struct {
	TenantID      string                 `yaml:"tenantId"`
	ProjectName   string                 `yaml:"projectName"`
	MGModel       string                 `yaml:"mgModel"`
	Connectivity  string                 `yaml:"connectivity"`
	PrimaryRegion string                 `yaml:"primaryRegion"`
	CICDPlatform  string                 `yaml:"cicdPlatform"`
	StateStrategy string                 `yaml:"stateStrategy"`
	LandingZones  []InitInputLandingZone `yaml:"landingZones"`
}

type InitInputLandingZone struct {
	Name           string `yaml:"name"`
	Archetype      string `yaml:"archetype"`
	SubscriptionID string `yaml:"subscriptionId"`
	AddressSpace   string `yaml:"addressSpace"`
}

func LoadInitInput(path string) (*InitInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading init input file %s: %w", path, err)
	}

	var in InitInput
	if err := yaml.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("parsing init input YAML: %w", err)
	}

	if err := in.Validate(); err != nil {
		return nil, err
	}

	return &in, nil
}

func (in *InitInput) Validate() error {
	if in == nil {
		return fmt.Errorf("init input cannot be nil")
	}

	if strings.TrimSpace(in.TenantID) == "" {
		return fmt.Errorf("tenantId is required")
	}
	if strings.TrimSpace(in.ProjectName) == "" {
		return fmt.Errorf("projectName is required")
	}
	if strings.TrimSpace(in.PrimaryRegion) == "" {
		return fmt.Errorf("primaryRegion is required")
	}

	if err := validateOneOf("mgModel", in.MGModel, []string{"caf-standard", "caf-lite"}); err != nil {
		return err
	}
	if err := validateOneOf("connectivity", in.Connectivity, []string{"hub-spoke", "vwan", "none"}); err != nil {
		return err
	}
	if err := validateOneOf("cicdPlatform", in.CICDPlatform, []string{"github-actions", "azure-devops"}); err != nil {
		return err
	}
	if err := validateOneOf("stateStrategy", in.StateStrategy, []string{"create-new", "existing", "terraform-cloud"}); err != nil {
		return err
	}

	for i, lz := range in.LandingZones {
		if strings.TrimSpace(lz.Name) == "" {
			return fmt.Errorf("landingZones[%d].name is required", i)
		}
		if err := validateOneOf(fmt.Sprintf("landingZones[%d].archetype", i), lz.Archetype, []string{"corp", "online", "sandbox"}); err != nil {
			return err
		}
		if strings.TrimSpace(lz.AddressSpace) == "" {
			return fmt.Errorf("landingZones[%d].addressSpace is required", i)
		}
	}

	return nil
}

func (in *InitInput) ToLZConfig() (*LZConfig, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	cfg := &LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Metadata: Metadata{
			Name:          strings.TrimSpace(in.ProjectName),
			Tenant:        strings.TrimSpace(in.TenantID),
			PrimaryRegion: strings.TrimSpace(in.PrimaryRegion),
		},
		Spec: Spec{
			Platform: Platform{
				ManagementGroups: ManagementGroupsConfig{Model: strings.TrimSpace(in.MGModel)},
				Connectivity:     ConnectivityConfig{Type: strings.TrimSpace(in.Connectivity)},
				Identity:         IdentityConfig{Type: "workload-identity-federation"},
				Management: ManagementConfig{
					LogAnalytics:      LogAnalyticsConfig{RetentionDays: 90},
					AutomationAccount: true,
					Defender:          DefenderConfig{Enabled: true, Plans: []string{"VirtualMachines", "StorageAccounts"}},
				},
			},
			Governance: Governance{
				Policies: PolicyConfig{Assignments: []string{"deploy-mdfc-config"}},
			},
			Naming: Naming{Convention: "caf"},
			StateBackend: StateBackend{
				ResourceGroup:  "rg-" + slugifyValue(in.ProjectName) + "-tfstate",
				StorageAccount: storageAccountNameValue(in.ProjectName + "-tfstate"),
				Container:      "tfstate",
				Subscription:   "<subscription-id>",
			},
			CICD: CICD{
				Platform: strings.TrimSpace(in.CICDPlatform),
				BranchPolicy: BranchPolicy{
					MainBranch: "main",
					RequirePR:  true,
				},
			},
		},
	}

	if cfg.Spec.Platform.Connectivity.Type != "none" {
		cfg.Spec.Platform.Connectivity.Hub = &HubConfig{
			Region:       cfg.Metadata.PrimaryRegion,
			AddressSpace: "10.0.0.0/16",
			Firewall: FirewallConfig{
				Enabled:     cfg.Spec.Platform.Connectivity.Type == "hub-spoke",
				SKU:         "Standard",
				ThreatIntel: "Alert",
			},
			DNS:        DNSConfig{PrivateResolver: cfg.Spec.Platform.Connectivity.Type == "hub-spoke"},
			VPNGateway: GatewayConfig{Enabled: false, SKU: "VpnGw2"},
			ERGateway:  GatewayConfig{Enabled: false, SKU: "ErGw1AZ"},
		}
	}

	for _, lz := range in.LandingZones {
		sub := strings.TrimSpace(lz.SubscriptionID)
		if sub == "" {
			sub = "<subscription-id>"
		}
		cfg.Spec.LandingZones = append(cfg.Spec.LandingZones, LandingZone{
			Name:         strings.TrimSpace(lz.Name),
			Subscription: sub,
			Archetype:    strings.TrimSpace(lz.Archetype),
			AddressSpace: strings.TrimSpace(lz.AddressSpace),
			Connected:    strings.TrimSpace(in.Connectivity) != "none",
		})
	}

	ApplyDefaults(cfg)
	checks, err := ValidateCross(cfg, "")
	if err != nil {
		return nil, err
	}
	for _, c := range checks {
		if c.Status == "error" {
			return nil, fmt.Errorf("invalid init input: %s", c.Message)
		}
	}

	return cfg, nil
}

func validateOneOf(name, value string, allowed []string) error {
	v := strings.TrimSpace(value)
	if v == "" {
		return fmt.Errorf("%s is required", name)
	}
	for _, candidate := range allowed {
		if strings.EqualFold(v, candidate) {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s", name, strings.Join(allowed, ", "))
}

func slugifyValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.Join(strings.Fields(value), "-")
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "landing-zone"
	}
	return value
}

func storageAccountNameValue(value string) string {
	v := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.ToLower(value))
	if len(v) > 24 {
		v = v[:24]
	}
	if v == "" {
		return "stlzctlstate"
	}
	if !strings.HasPrefix(v, "st") {
		v = "st" + v
		if len(v) > 24 {
			v = v[:24]
		}
	}
	return v
}
