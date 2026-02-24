package config

const (
	DefaultSoftDeleteDays   = 30
	DefaultLogRetentionDays = 90
	DefaultContainer        = "tfstate"
	DefaultNamingConvention = "caf"
	DefaultMainBranch       = "main"
	DefaultMGModel          = "caf-standard"
	DefaultConnectivityType = "none"
	DefaultIdentityType     = "workload-identity-federation"
)

// ApplyDefaults fills in default values for optional fields that were not
// specified in the YAML. It is called after parsing and before validation.
func ApplyDefaults(cfg *LZConfig) {
	if cfg.APIVersion == "" {
		cfg.APIVersion = "lzctl/v1"
	}
	if cfg.Kind == "" {
		cfg.Kind = "LandingZone"
	}

	// State backend defaults — state is a critical asset
	if cfg.Spec.StateBackend.Container == "" {
		cfg.Spec.StateBackend.Container = DefaultContainer
	}
	// Versioning and soft delete are enabled by default (PRD FR-2.9)
	if cfg.Spec.StateBackend.Versioning == nil {
		v := true
		cfg.Spec.StateBackend.Versioning = &v
	}
	if cfg.Spec.StateBackend.SoftDelete == nil {
		v := true
		cfg.Spec.StateBackend.SoftDelete = &v
	}
	if cfg.Spec.StateBackend.SoftDeleteDays == 0 {
		cfg.Spec.StateBackend.SoftDeleteDays = DefaultSoftDeleteDays
	}

	// Management defaults
	if cfg.Spec.Platform.Management.LogAnalytics.RetentionDays == 0 {
		cfg.Spec.Platform.Management.LogAnalytics.RetentionDays = DefaultLogRetentionDays
	}

	// Naming defaults
	if cfg.Spec.Naming.Convention == "" {
		cfg.Spec.Naming.Convention = DefaultNamingConvention
	}

	// CI/CD defaults
	if cfg.Spec.CICD.BranchPolicy.MainBranch == "" {
		cfg.Spec.CICD.BranchPolicy.MainBranch = DefaultMainBranch
	}

	// Management groups model default
	if cfg.Spec.Platform.ManagementGroups.Model == "" {
		cfg.Spec.Platform.ManagementGroups.Model = DefaultMGModel
	}

	// Connectivity type default
	if cfg.Spec.Platform.Connectivity.Type == "" {
		cfg.Spec.Platform.Connectivity.Type = DefaultConnectivityType
	}

	// Identity type default
	if cfg.Spec.Platform.Identity.Type == "" {
		cfg.Spec.Platform.Identity.Type = DefaultIdentityType
	}
}
