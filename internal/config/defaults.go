package config

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
		cfg.Spec.StateBackend.Container = "tfstate"
	}
	// Versioning and soft delete are enabled by default (PRD FR-2.9)
	if !cfg.Spec.StateBackend.Versioning {
		cfg.Spec.StateBackend.Versioning = true
	}
	if !cfg.Spec.StateBackend.SoftDelete {
		cfg.Spec.StateBackend.SoftDelete = true
	}
	if cfg.Spec.StateBackend.SoftDeleteDays == 0 {
		cfg.Spec.StateBackend.SoftDeleteDays = 30
	}

	// Management defaults
	if cfg.Spec.Platform.Management.LogAnalytics.RetentionDays == 0 {
		cfg.Spec.Platform.Management.LogAnalytics.RetentionDays = 90
	}

	// Naming defaults
	if cfg.Spec.Naming.Convention == "" {
		cfg.Spec.Naming.Convention = "caf"
	}

	// CI/CD defaults
	if cfg.Spec.CICD.BranchPolicy.MainBranch == "" {
		cfg.Spec.CICD.BranchPolicy.MainBranch = "main"
	}

	// Management groups model default
	if cfg.Spec.Platform.ManagementGroups.Model == "" {
		cfg.Spec.Platform.ManagementGroups.Model = "caf-standard"
	}

	// Connectivity type default
	if cfg.Spec.Platform.Connectivity.Type == "" {
		cfg.Spec.Platform.Connectivity.Type = "none"
	}

	// Identity type default
	if cfg.Spec.Platform.Identity.Type == "" {
		cfg.Spec.Platform.Identity.Type = "workload-identity-federation"
	}
}
