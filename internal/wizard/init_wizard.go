package wizard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kjourdan1/lzctl/internal/config"
)

// InitConfig captures all inputs collected by the init wizard.
type InitConfig struct {
	ProjectName          string
	TenantID             string
	CICDPlatform         string
	ManagementGroupModel string
	ConnectivityModel    string
	PrimaryRegion        string
	SecondaryRegion      string
	IdentityModel        string
	StateBackendStrategy string
	Bootstrap            bool

	FirewallSKU        string
	VPNGatewayEnabled  bool
	ERGatewayEnabled   bool
	DNSPrivateResolver bool
}

// ToLZConfig converts wizard input to the v2 config model.
func (c InitConfig) ToLZConfig() *config.LZConfig {
	name := strings.TrimSpace(c.ProjectName)
	if name == "" {
		name = "landing-zone"
	}

	cfg := &config.LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Metadata: config.Metadata{
			Name:          name,
			Tenant:        c.TenantID,
			PrimaryRegion: c.PrimaryRegion,
		},
		Spec: config.Spec{
			Platform: config.Platform{
				ManagementGroups: config.ManagementGroupsConfig{Model: c.ManagementGroupModel},
				Connectivity:     config.ConnectivityConfig{Type: c.ConnectivityModel},
				Identity:         config.IdentityConfig{Type: c.IdentityModel},
				Management: config.ManagementConfig{
					LogAnalytics:      config.LogAnalyticsConfig{RetentionDays: 90},
					AutomationAccount: true,
					Defender:          config.DefenderConfig{Enabled: true, Plans: []string{"VirtualMachines", "StorageAccounts"}},
				},
			},
			Governance: config.Governance{
				Policies: config.PolicyConfig{Assignments: []string{"deploy-mdfc-config"}},
			},
			Naming: config.Naming{Convention: "caf"},
			StateBackend: config.StateBackend{
				ResourceGroup:  "rg-" + slugify(name) + "-tfstate",
				StorageAccount: storageAccountName(name + "-tfstate"),
				Container:      "tfstate",
				Subscription:   "<subscription-id>",
			},
			CICD: config.CICD{
				Platform:   c.CICDPlatform,
				Repository: "<owner>/<repo>",
				BranchPolicy: config.BranchPolicy{
					MainBranch: "main",
					RequirePR:  true,
				},
			},
		},
	}

	if c.SecondaryRegion != "" {
		cfg.Metadata.SecondaryRegion = c.SecondaryRegion
	}

	if c.ConnectivityModel != "none" {
		cfg.Spec.Platform.Connectivity.Hub = &config.HubConfig{
			Region:       c.PrimaryRegion,
			AddressSpace: "10.0.0.0/16",
			Firewall: config.FirewallConfig{
				Enabled:     c.ConnectivityModel == "hub-spoke",
				SKU:         c.FirewallSKU,
				ThreatIntel: "Alert",
			},
			DNS: config.DNSConfig{
				PrivateResolver: c.DNSPrivateResolver,
			},
			VPNGateway: config.GatewayConfig{Enabled: c.VPNGatewayEnabled, SKU: "VpnGw2"},
			ERGateway:  config.GatewayConfig{Enabled: c.ERGatewayEnabled, SKU: "ErGw1AZ"},
		}
	}

	config.ApplyDefaults(cfg)
	return cfg
}

// InitWizard drives the interactive init flow.
type InitWizard struct {
	prompter Prompter
}

// NewInitWizard returns an init wizard; if p is nil, survey is used.
func NewInitWizard(p Prompter) *InitWizard {
	if p == nil {
		p = NewSurveyPrompter()
	}
	return &InitWizard{prompter: p}
}

// Run collects wizard input in the required order.
func (w *InitWizard) Run() (*InitConfig, error) {
	cfg := &InitConfig{}
	var err error

	cfg.ProjectName, err = w.prompter.Input("Project name", "landing-zone", survey.ComposeValidators(ValidateNonEmpty))
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.TenantID, err = w.prompter.Input("Tenant ID (UUID)", "", survey.ComposeValidators(ValidateTenantID))
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.CICDPlatform, err = w.prompter.Select("CI/CD platform", []string{"github-actions", "azure-devops"}, "github-actions")
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.ManagementGroupModel, err = w.prompter.Select("Management group model", []string{"caf-standard", "caf-lite"}, "caf-standard")
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.ConnectivityModel, err = w.prompter.Select("Connectivity model", []string{"hub-spoke", "vwan", "none"}, "hub-spoke")
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.PrimaryRegion, err = w.prompter.Select("Primary region", CommonAzureRegions, "westeurope")
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.SecondaryRegion, err = w.prompter.Select("Secondary region", append([]string{""}, CommonAzureRegions...), "")
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.IdentityModel, err = w.prompter.Select("Identity model", []string{"workload-identity-federation", "sp-federated", "sp-secret"}, "workload-identity-federation")
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.StateBackendStrategy, err = w.prompter.Select("State backend strategy", []string{"create-new", "existing", "terraform-cloud"}, "create-new")
	if err != nil {
		return nil, handlePromptErr(err)
	}

	cfg.Bootstrap, err = w.prompter.Confirm("Bootstrap state backend now?", true)
	if err != nil {
		return nil, handlePromptErr(err)
	}

	if cfg.ConnectivityModel != "none" {
		cfg.FirewallSKU, err = w.prompter.Select("Firewall SKU", []string{"Standard", "Premium"}, "Standard")
		if err != nil {
			return nil, handlePromptErr(err)
		}
		cfg.VPNGatewayEnabled, err = w.prompter.Confirm("Enable VPN gateway?", false)
		if err != nil {
			return nil, handlePromptErr(err)
		}
		cfg.ERGatewayEnabled, err = w.prompter.Confirm("Enable ExpressRoute gateway?", false)
		if err != nil {
			return nil, handlePromptErr(err)
		}
	}

	if cfg.ConnectivityModel == "hub-spoke" {
		cfg.DNSPrivateResolver, err = w.prompter.Confirm("Enable DNS private resolver?", true)
		if err != nil {
			return nil, handlePromptErr(err)
		}
	}

	return cfg, nil
}

func handlePromptErr(err error) error {
	if errors.Is(err, ErrCancelled) {
		return fmt.Errorf("wizard cancelled: %w", ErrCancelled)
	}
	return err
}

func slugify(value string) string {
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

func storageAccountName(value string) string {
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
