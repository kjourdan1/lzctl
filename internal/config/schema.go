// Package config provides the configuration schema, loader, validator, and
// default values for lzctl.yaml — the single source of truth for a landing zone
// deployment.
package config

// LZConfig is the root struct matching lzctl.yaml.
type LZConfig struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"` // "lzctl/v1"
	Kind       string   `yaml:"kind" json:"kind"`             // "LandingZone"
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       Spec     `yaml:"spec" json:"spec"`
}

// Metadata holds top-level identification and region information.
type Metadata struct {
	Name            string `yaml:"name" json:"name"`
	Tenant          string `yaml:"tenant" json:"tenant"`
	PrimaryRegion   string `yaml:"primaryRegion" json:"primaryRegion"`
	SecondaryRegion string `yaml:"secondaryRegion,omitempty" json:"secondaryRegion,omitempty"`
}

// Spec contains the full landing zone specification.
type Spec struct {
	Platform     Platform      `yaml:"platform" json:"platform"`
	Governance   Governance    `yaml:"governance" json:"governance"`
	Naming       Naming        `yaml:"naming" json:"naming"`
	StateBackend StateBackend  `yaml:"stateBackend" json:"stateBackend"`
	LandingZones []LandingZone `yaml:"landingZones" json:"landingZones"`
	CICD         CICD          `yaml:"cicd" json:"cicd"`
	Testing      *Testing      `yaml:"testing,omitempty" json:"testing,omitempty"`
}

// Testing holds native Terraform test generation settings.
// When enabled, lzctl generates .tftest.hcl files for each platform layer
// and landing zone, validated by `terraform test` in plan mode.
type Testing struct {
	Enabled    bool            `yaml:"enabled" json:"enabled"`
	Assertions []TestAssertion `yaml:"assertions,omitempty" json:"assertions,omitempty"`
}

// TestAssertion defines a single assertion rendered as a `run` block in a .tftest.hcl file.
// Layer may be a platform layer name (e.g. "connectivity"), or "*" to apply to all layers.
type TestAssertion struct {
	Name         string `yaml:"name" json:"name"`
	Layer        string `yaml:"layer" json:"layer"`
	Condition    string `yaml:"condition" json:"condition"`
	ErrorMessage string `yaml:"errorMessage" json:"errorMessage"`
}

// Platform holds the platform-level configuration (MG, connectivity, identity, management).
type Platform struct {
	ManagementGroups ManagementGroupsConfig `yaml:"managementGroups" json:"managementGroups"`
	Connectivity     ConnectivityConfig     `yaml:"connectivity" json:"connectivity"`
	Identity         IdentityConfig         `yaml:"identity" json:"identity"`
	Management       ManagementConfig       `yaml:"management" json:"management"`
}

// ManagementGroupsConfig defines the management group hierarchy model.
type ManagementGroupsConfig struct {
	Model    string   `yaml:"model" json:"model"`                           // "caf-standard" | "caf-lite"
	Disabled []string `yaml:"disabled,omitempty" json:"disabled,omitempty"` // MG names to disable
}

// ConnectivityConfig defines the network connectivity model.
type ConnectivityConfig struct {
	Type string     `yaml:"type" json:"type"`                   // "hub-spoke" | "vwan" | "none"
	Hub  *HubConfig `yaml:"hub,omitempty" json:"hub,omitempty"` // required when type != "none"
}

// HubConfig holds hub network configuration.
type HubConfig struct {
	Region       string         `yaml:"region" json:"region"`
	AddressSpace string         `yaml:"addressSpace" json:"addressSpace"`
	Firewall     FirewallConfig `yaml:"firewall" json:"firewall"`
	DNS          DNSConfig      `yaml:"dns" json:"dns"`
	VPNGateway   GatewayConfig  `yaml:"vpnGateway" json:"vpnGateway"`
	ERGateway    GatewayConfig  `yaml:"expressRouteGateway" json:"expressRouteGateway"`
}

// FirewallConfig holds Azure Firewall settings.
type FirewallConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	SKU         string `yaml:"sku,omitempty" json:"sku,omitempty"`                 // "Standard" | "Premium"
	ThreatIntel string `yaml:"threatIntel,omitempty" json:"threatIntel,omitempty"` // "Alert" | "Deny" | "Off"
}

// DNSConfig holds DNS resolution settings.
type DNSConfig struct {
	PrivateResolver bool     `yaml:"privateResolver" json:"privateResolver"`
	Forwarders      []string `yaml:"forwarders,omitempty" json:"forwarders,omitempty"`
}

// GatewayConfig holds VPN or ExpressRoute gateway settings.
type GatewayConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	SKU     string `yaml:"sku,omitempty" json:"sku,omitempty"`
}

// IdentityConfig defines the identity/authentication model.
type IdentityConfig struct {
	Type        string `yaml:"type" json:"type"`                                   // "workload-identity-federation" | "sp-federated" | "sp-secret"
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`               // managed identity resource name (set post-bootstrap)
	ClientID    string `yaml:"clientId,omitempty" json:"clientId,omitempty"`       // populated post-bootstrap
	PrincipalID string `yaml:"principalId,omitempty" json:"principalId,omitempty"` // populated post-bootstrap
}

// ManagementConfig holds management & monitoring settings.
type ManagementConfig struct {
	LogAnalytics      LogAnalyticsConfig `yaml:"logAnalytics" json:"logAnalytics"`
	AutomationAccount bool               `yaml:"automationAccount" json:"automationAccount"`
	Defender          DefenderConfig     `yaml:"defenderForCloud" json:"defenderForCloud"`
}

// LogAnalyticsConfig holds Log Analytics workspace settings.
type LogAnalyticsConfig struct {
	RetentionDays int      `yaml:"retentionDays" json:"retentionDays"`
	Solutions     []string `yaml:"solutions,omitempty" json:"solutions,omitempty"`
}

// DefenderConfig holds Microsoft Defender for Cloud settings.
type DefenderConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Plans   []string `yaml:"plans" json:"plans"`
}

// Governance holds policy-related configuration.
type Governance struct {
	Policies PolicyConfig `yaml:"policies" json:"policies"`
}

// PolicyConfig holds policy assignment and custom policy references.
type PolicyConfig struct {
	Assignments []string `yaml:"assignments" json:"assignments"`
	Custom      []string `yaml:"custom,omitempty" json:"custom,omitempty"`
}

// Naming holds naming convention configuration.
type Naming struct {
	Convention string            `yaml:"convention" json:"convention"` // "caf"
	Overrides  map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

// StateBackend holds Terraform remote state configuration.
// State is treated as a critical asset: versioning enables rollback,
// soft delete prevents accidental loss, and blob lease locking
// prevents concurrent writes (like DynamoDB locking in AWS).
type StateBackend struct {
	ResourceGroup  string `yaml:"resourceGroup" json:"resourceGroup"`
	StorageAccount string `yaml:"storageAccount" json:"storageAccount"`
	Container      string `yaml:"container" json:"container"`
	Subscription   string `yaml:"subscription" json:"subscription"`
	Versioning     *bool  `yaml:"versioning" json:"versioning"`                             // enable blob versioning for state history
	SoftDelete     *bool  `yaml:"softDelete" json:"softDelete"`                             // enable soft delete for accidental deletion protection
	SoftDeleteDays int    `yaml:"softDeleteDays,omitempty" json:"softDeleteDays,omitempty"` // retention days for soft-deleted blobs (default: 30)
}

// LandingZone represents one subscription-level landing zone.
type LandingZone struct {
	Name         string            `yaml:"name" json:"name"`
	Subscription string            `yaml:"subscription" json:"subscription"`
	Archetype    string            `yaml:"archetype" json:"archetype"` // "corp" | "online" | "sandbox"
	AddressSpace string            `yaml:"addressSpace" json:"addressSpace"`
	Connected    bool              `yaml:"connected" json:"connected"`
	Tags         map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Blueprint    *Blueprint        `yaml:"blueprint,omitempty" json:"blueprint,omitempty"`
}

// Blueprint defines an optional workload blueprint attached to a landing zone.
type Blueprint struct {
	Type      string         `yaml:"type" json:"type"` // "paas-secure" | "aks-platform" | "aca-platform" | "avd-secure"
	Overrides map[string]any `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

// AKSBlueprintConfig is the typed representation of the aks-platform blueprint
// overrides. Used by the template engine to render AKS-platform Terraform files
// with ArgoCD support (E9-S1). Decoded from Blueprint.Overrides via ParseAKSBlueprintConfig.
type AKSBlueprintConfig struct {
	AKS      AKSOverrideConfig      `yaml:"aks" json:"aks"`
	ACR      ACROverrideConfig      `yaml:"acr" json:"acr"`
	Defender DefenderOverrideConfig `yaml:"defender" json:"defender"`
	ArgoCD   ArgoCDConfig           `yaml:"argocd" json:"argocd"`
}

// AKSOverrideConfig holds AKS cluster-level overrides for aks-platform.
type AKSOverrideConfig struct {
	Version string `yaml:"version" json:"version"` // Kubernetes version e.g. "1.30"
}

// ACROverrideConfig holds Container Registry overrides for aks-platform.
type ACROverrideConfig struct {
	SKU string `yaml:"sku" json:"sku"` // "Basic" | "Standard" | "Premium"
}

// DefenderOverrideConfig holds Microsoft Defender for Containers overrides.
type DefenderOverrideConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// ArgoCDConfig is the opt-in ArgoCD configuration for aks-platform blueprints.
//
// When ArgoCD.Enabled = true, lzctl generates:
//   - argocd.tf (extension or helm block, E9-S2)
//   - landing-zones/<name>/blueprint/argocd/appset.yaml (ApplicationSet, E9-S4)
//   - landing-zones/<name>/blueprint/Makefile with argocd-login target (E9-S5)
//
// Mode values:
//   - "extension" (default/recommended): Microsoft Flux GitOps extension via Arc
//   - "helm": Argo CD helm chart (provides more version control)
type ArgoCDConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`               // default: false
	Mode           string `yaml:"mode" json:"mode"`                     // "extension" | "helm"
	RepoURL        string `yaml:"repoUrl" json:"repoUrl"`               // git repo URL for app-repo
	TargetRevision string `yaml:"targetRevision" json:"targetRevision"` // default: "HEAD"
	AppPath        string `yaml:"appPath" json:"appPath"`               // root path for ApplicationSets, default: "apps/"
	SSOEnabled     bool   `yaml:"ssoEnabled" json:"ssoEnabled"`         // Entra ID SSO integration
	ChartVersion   string `yaml:"chartVersion" json:"chartVersion"`     // pinned ArgoCD chart version (helm mode)
}

// ArgoCDMode validates the argocd.mode enum.
func (a *ArgoCDConfig) ArgoCDMode() string {
	switch a.Mode {
	case "helm":
		return "helm"
	default:
		return "extension"
	}
}

// EffectiveTargetRevision returns the target revision, defaulting to "HEAD".
func (a *ArgoCDConfig) EffectiveTargetRevision() string {
	if a.TargetRevision == "" {
		return "HEAD"
	}
	return a.TargetRevision
}

// EffectiveAppPath returns the app path, defaulting to "apps/".
func (a *ArgoCDConfig) EffectiveAppPath() string {
	if a.AppPath == "" {
		return "apps/"
	}
	return a.AppPath
}

// CICD holds CI/CD pipeline configuration.
type CICD struct {
	Platform     string       `yaml:"platform" json:"platform"`               // "github-actions" | "azure-devops"
	Model        string       `yaml:"model,omitempty" json:"model,omitempty"` // "push" (default) | "pull"
	Pull         *PullConfig  `yaml:"pull,omitempty" json:"pull,omitempty"`   // required when model == "pull"
	Repository   string       `yaml:"repository,omitempty" json:"repository,omitempty"`
	BranchPolicy BranchPolicy `yaml:"branchPolicy" json:"branchPolicy"`
}

// EffectiveModel returns the effective CI/CD model, defaulting to "push".
func (c *CICD) EffectiveModel() string {
	if c.Model == "pull" {
		return "pull"
	}
	return "push"
}

// PullConfig holds pull-mode (GitOps operator) configuration.
type PullConfig struct {
	Engine string `yaml:"engine" json:"engine"` // "atlantis" | "spacelift" | "tfcloud"
}

// BranchPolicy holds branch protection settings for CI/CD.
type BranchPolicy struct {
	MainBranch string `yaml:"mainBranch" json:"mainBranch"` // default: "main"
	RequirePR  bool   `yaml:"requirePR" json:"requirePR"`   // default: true
}
