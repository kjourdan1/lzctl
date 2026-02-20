package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/kjourdan1/lzctl/internal/azauth"
	stateboot "github.com/kjourdan1/lzctl/internal/bootstrap"
	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/output"
	lztemplate "github.com/kjourdan1/lzctl/internal/template"
	"github.com/kjourdan1/lzctl/internal/wizard"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new landing zone project",
	Long: `Initializes the project by generating Terraform configurations from lzctl.yaml.

If --config is provided, loads the configuration file directly. Otherwise,
launches an interactive wizard to build the configuration.

Generated structure:
  platform/
    management-groups/    (Resource Organization)
    identity/             (Identity & Access)
    management/           (Management & Monitoring)
    governance/           (Azure Policies)
    connectivity/         (Hub-Spoke or vWAN)
  landing-zones/          (Workload subscriptions)
  pipelines/              (CI/CD pipeline definitions)

This command is idempotent: it will not overwrite existing files unless --force is specified.
It never pushes to any remote.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd)
	},
}

var (
	initTenantID        string
	initSubscriptionID  string
	initFromFile        string
	initProjectName     string
	initMGModel         string
	initConnectivity    string
	initIdentity        string
	initPrimaryRegion   string
	initSecondaryRegion string
	initCICDPlatform    string
	initStateStrategy   string
	initForce           bool
	initNoBootstrap     bool
)

func init() {
	initCmd.Flags().StringVar(&initTenantID, "tenant-id", "", "Azure AD tenant ID (auto-detected from Azure CLI if omitted)")
	initCmd.Flags().StringVar(&initSubscriptionID, "subscription-id", "", "Azure Subscription ID (auto-detected from Azure CLI if omitted)")
	initCmd.Flags().StringVar(&initFromFile, "from-file", "", "path to one-shot init input YAML (converted to lzctl.yaml)")
	initCmd.Flags().StringVar(&initProjectName, "project-name", "landing-zone", "project name")
	initCmd.Flags().StringVar(&initMGModel, "mg-model", "caf-standard", "management group model (caf-standard|caf-lite)")
	initCmd.Flags().StringVar(&initConnectivity, "connectivity", "hub-spoke", "connectivity model (hub-spoke|vwan|none)")
	initCmd.Flags().StringVar(&initIdentity, "identity", "workload-identity-federation", "identity model (workload-identity-federation|sp-federated|sp-secret)")
	initCmd.Flags().StringVar(&initPrimaryRegion, "primary-region", "westeurope", "primary Azure region")
	initCmd.Flags().StringVar(&initSecondaryRegion, "secondary-region", "", "secondary Azure region (optional)")
	initCmd.Flags().StringVar(&initCICDPlatform, "cicd-platform", "github-actions", "CI/CD platform (github-actions|azure-devops)")
	initCmd.Flags().StringVar(&initStateStrategy, "state-strategy", "create-new", "state backend strategy (create-new|existing|terraform-cloud)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing files")
	initCmd.Flags().BoolVar(&initNoBootstrap, "no-bootstrap", false, "skip automatic state backend provisioning")

	_ = bindInitEnv()

	rootCmd.AddCommand(initCmd)
}

func bindInitEnv() error {
	binding := []struct {
		key     string
		flag    string
		envName string
	}{
		{key: "tenant_id", flag: "tenant-id", envName: "LZCTL_TENANT_ID"},
		{key: "subscription_id", flag: "subscription-id", envName: "LZCTL_SUBSCRIPTION_ID"},
		{key: "from_file", flag: "from-file", envName: "LZCTL_FROM_FILE"},
		{key: "project_name", flag: "project-name", envName: "LZCTL_PROJECT_NAME"},
		{key: "mg_model", flag: "mg-model", envName: "LZCTL_MG_MODEL"},
		{key: "connectivity", flag: "connectivity", envName: "LZCTL_CONNECTIVITY"},
		{key: "identity", flag: "identity", envName: "LZCTL_IDENTITY"},
		{key: "primary_region", flag: "primary-region", envName: "LZCTL_PRIMARY_REGION"},
		{key: "secondary_region", flag: "secondary-region", envName: "LZCTL_SECONDARY_REGION"},
		{key: "cicd_platform", flag: "cicd-platform", envName: "LZCTL_CICD_PLATFORM"},
		{key: "state_strategy", flag: "state-strategy", envName: "LZCTL_STATE_STRATEGY"},
	}

	for _, b := range binding {
		if err := bindPFlagAndEnv(initCmd, b.key, b.flag, b.envName); err != nil {
			return err
		}
	}
	return nil
}

func bindPFlagAndEnv(c *cobra.Command, key, flag, env string) error {
	if err := viper.BindPFlag(key, c.Flags().Lookup(flag)); err != nil {
		return err
	}
	if err := viper.BindEnv(key, env); err != nil {
		return err
	}
	return nil
}

func resolveInitValue(cmd *cobra.Command, flagName, flagValue, envName, defaultValue string) string {
	if cmd != nil && cmd.Flags().Changed(flagName) {
		return strings.TrimSpace(flagValue)
	}
	if envValue := strings.TrimSpace(os.Getenv(envName)); envValue != "" {
		return envValue
	}
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	return defaultValue
}

func validateInitEnum(name, value string, allowed []string) error {
	v := strings.TrimSpace(strings.ToLower(value))
	for _, candidate := range allowed {
		if v == candidate {
			return nil
		}
	}
	return fmt.Errorf("invalid value for --%s: %q (allowed: %s)", name, value, strings.Join(allowed, ", "))
}

func validateInitInputs(mgModel, connectivity, identity, cicdPlatform, stateStrategy string) error {
	if err := validateInitEnum("mg-model", mgModel, []string{"caf-standard", "caf-lite"}); err != nil {
		return err
	}
	if err := validateInitEnum("connectivity", connectivity, []string{"hub-spoke", "vwan", "none"}); err != nil {
		return err
	}
	if err := validateInitEnum("identity", identity, []string{"workload-identity-federation", "sp-federated", "sp-secret"}); err != nil {
		return err
	}
	if err := validateInitEnum("cicd-platform", cicdPlatform, []string{"github-actions", "azure-devops"}); err != nil {
		return err
	}
	if err := validateInitEnum("state-strategy", stateStrategy, []string{"create-new", "existing", "terraform-cloud"}); err != nil {
		return err
	}
	return nil
}

func runInit(cmd *cobra.Command) error {
	output.Init(verbosity > 0, jsonOutput)

	fromFile := resolveInitValue(cmd, "from-file", initFromFile, "LZCTL_FROM_FILE", "")
	tenantID := resolveInitValue(cmd, "tenant-id", initTenantID, "LZCTL_TENANT_ID", "")
	subscriptionID := resolveInitValue(cmd, "subscription-id", initSubscriptionID, "LZCTL_SUBSCRIPTION_ID", "")
	projectName := resolveInitValue(cmd, "project-name", initProjectName, "LZCTL_PROJECT_NAME", "landing-zone")
	mgModel := resolveInitValue(cmd, "mg-model", initMGModel, "LZCTL_MG_MODEL", "caf-standard")
	connectivity := resolveInitValue(cmd, "connectivity", initConnectivity, "LZCTL_CONNECTIVITY", "hub-spoke")
	identity := resolveInitValue(cmd, "identity", initIdentity, "LZCTL_IDENTITY", "workload-identity-federation")
	primaryRegion := resolveInitValue(cmd, "primary-region", initPrimaryRegion, "LZCTL_PRIMARY_REGION", "westeurope")
	secondaryRegion := resolveInitValue(cmd, "secondary-region", initSecondaryRegion, "LZCTL_SECONDARY_REGION", "")
	cicdPlatform := resolveInitValue(cmd, "cicd-platform", initCICDPlatform, "LZCTL_CICD_PLATFORM", "github-actions")
	stateStrategy := resolveInitValue(cmd, "state-strategy", initStateStrategy, "LZCTL_STATE_STRATEGY", "create-new")

	initTenantID = tenantID
	initSubscriptionID = subscriptionID

	if effectiveCIMode() && cfgFile == "" && strings.TrimSpace(fromFile) == "" && tenantID == "" {
		return fmt.Errorf("--ci mode requires --tenant-id (or LZCTL_TENANT_ID)")
	}

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}
	if strings.TrimSpace(fromFile) != "" {
		if strings.TrimSpace(cfgFile) != "" {
			return fmt.Errorf("--from-file cannot be combined with --config")
		}
		configPath := filepath.Join(absRoot, "lzctl.yaml")
		if fileExistsLocal(configPath) && !initForce {
			return fmt.Errorf("lzctl.yaml already exists, use --force to overwrite")
		}
	}

	var cfg *config.LZConfig
	var wizardCfg *wizard.InitConfig
	if strings.TrimSpace(fromFile) != "" {
		inputCfg, loadErr := config.LoadInitInput(fromFile)
		if loadErr != nil {
			return fmt.Errorf("loading init input from --from-file %q: %w", fromFile, loadErr)
		}
		cfg, err = inputCfg.ToLZConfig()
		if err != nil {
			return fmt.Errorf("converting --from-file input to lzctl config: %w", err)
		}
	} else if cfgFile != "" {
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config from --config %q: %w", cfgFile, err)
		}
	} else if tenantID != "" {
		if err := validateInitInputs(mgModel, connectivity, identity, cicdPlatform, stateStrategy); err != nil {
			return err
		}

		// Non-interactive mode: --tenant-id or LZCTL_TENANT_ID was supplied.
		wizardCfg = &wizard.InitConfig{
			ProjectName:          projectName,
			TenantID:             tenantID,
			CICDPlatform:         cicdPlatform,
			ManagementGroupModel: mgModel,
			ConnectivityModel:    connectivity,
			PrimaryRegion:        primaryRegion,
			SecondaryRegion:      secondaryRegion,
			IdentityModel:        identity,
			StateBackendStrategy: stateStrategy,
			Bootstrap:            false, // never auto-bootstrap in non-interactive mode
			FirewallSKU:          "Standard",
		}
		cfg = wizardCfg.ToLZConfig()
	} else {
		if effectiveCIMode() {
			return fmt.Errorf("--ci mode requires --tenant-id (or LZCTL_TENANT_ID)")
		}

		var runErr error
		wizardCfg, runErr = wizard.NewInitWizard(nil).Run()
		if runErr != nil {
			if errors.Is(runErr, wizard.ErrCanceled) {
				output.Warn("init wizard canceled")
				return nil
			}
			return fmt.Errorf("running init wizard: %w", runErr)
		}
		cfg = wizardCfg.ToLZConfig()
	}

	if wizardCfg != nil && !dryRun && !initNoBootstrap && wizardCfg.Bootstrap && strings.EqualFold(wizardCfg.StateBackendStrategy, "create-new") {
		if err := runInitBootstrap(cmd, cfg); err != nil {
			output.Warn("state backend bootstrap failed", "error", err.Error())
		}
	}

	validation, err := config.Validate(cfg)
	if err != nil {
		return fmt.Errorf("validating config: %w", err)
	}
	if !validation.Valid {
		if jsonOutput {
			output.JSON(validation)
			return fmt.Errorf("config validation failed")
		}
		for _, v := range validation.Errors {
			output.Error(fmt.Sprintf("%s: %s", v.Field, v.Description))
		}
		return fmt.Errorf("config validation failed")
	}

	if dryRun && strings.TrimSpace(fromFile) != "" && !jsonOutput {
		manifestYAML, marshalErr := yaml.Marshal(cfg)
		if marshalErr == nil {
			fmt.Fprintln(os.Stdout, string(manifestYAML))
		}
	}

	engine, err := lztemplate.NewEngine()
	if err != nil {
		return fmt.Errorf("creating template engine: %w", err)
	}

	files, err := engine.RenderAll(cfg)
	if err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}

	writer := lztemplate.Writer{DryRun: dryRun}
	written, err := writer.WriteAll(files, absRoot)
	if err != nil {
		return fmt.Errorf("writing rendered files: %w", err)
	}

	if jsonOutput {
		output.JSON(map[string]interface{}{
			"status": "ok",
			"dryRun": dryRun,
			"files":  written,
		})
		return nil
	}

	if dryRun {
		output.Success(fmt.Sprintf("Dry-run complete: %d files would be generated", len(written)))
	} else {
		output.Success(fmt.Sprintf("Init complete: generated %d files", len(written)))
	}
	for _, path := range written {
		output.Info("generated", "file", path)
	}

	_ = cmd
	return nil
}

func runInitBootstrap(cmd *cobra.Command, cfg *config.LZConfig) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	subscriptionID := strings.TrimSpace(initSubscriptionID)
	if subscriptionID == "" && !isPlaceholderValue(cfg.Spec.StateBackend.Subscription) {
		subscriptionID = strings.TrimSpace(cfg.Spec.StateBackend.Subscription)
	}
	if subscriptionID == "" {
		detected, err := azauth.DetectSubscriptionID()
		if err == nil {
			subscriptionID = detected
		}
	}
	if subscriptionID == "" {
		return fmt.Errorf("missing subscription ID for bootstrap (use --subscription-id or az login)")
	}

	tenantID := strings.TrimSpace(initTenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(cfg.Metadata.Tenant)
	}
	if tenantID == "" || isPlaceholderValue(tenantID) {
		detected, err := azauth.DetectTenantID()
		if err == nil {
			tenantID = detected
		}
	}

	region := strings.TrimSpace(cfg.Metadata.PrimaryRegion)
	if region == "" {
		region = "westeurope"
	}

	ghOrg := ""
	ghRepo := ""
	platform := strings.ToLower(strings.TrimSpace(cfg.Spec.CICD.Platform))
	if platform == "github-actions" || platform == "github" {
		if out, err := runGitOutput(cmd.Context(), "config", "--get", "remote.origin.url"); err == nil {
			ghOrg, ghRepo = parseGitHubRemote(strings.TrimSpace(out))
		}
	}

	result, err := stateboot.StateBackend(stateboot.Options{
		TenantName:     cfg.Metadata.Name,
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		Region:         region,
		GitHubOrg:      ghOrg,
		GitHubRepo:     ghRepo,
		Verbosity:      verbosity,
	})
	if err != nil {
		return err
	}

	cfg.Spec.StateBackend.ResourceGroup = result.ResourceGroupName
	cfg.Spec.StateBackend.StorageAccount = result.StorageAccountName
	cfg.Spec.StateBackend.Container = result.ContainerName
	cfg.Spec.StateBackend.Subscription = subscriptionID
	if result.SPNAppID != "" {
		cfg.Spec.Platform.Identity.ClientID = result.SPNAppID
	}
	if result.SPNObjectID != "" {
		cfg.Spec.Platform.Identity.PrincipalID = result.SPNObjectID
	}

	return nil
}

func isPlaceholderValue(value string) bool {
	v := strings.TrimSpace(value)
	return v == "" || strings.HasPrefix(v, "<")
}

// runGitOutput runs a git command and returns stdout.
func runGitOutput(ctx context.Context, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	return string(out), err
}

// parseGitHubRemote extracts org and repo from a GitHub remote URL.
// Supports both HTTPS and SSH formats.
func parseGitHubRemote(remote string) (string, string) {
	// SSH: git@github.com:org/repo.git
	if strings.Contains(remote, "github.com:") {
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) == 2 {
			path := strings.TrimSuffix(parts[1], ".git")
			segments := strings.SplitN(path, "/", 2)
			if len(segments) == 2 {
				return segments[0], segments[1]
			}
		}
	}
	// HTTPS: https://github.com/org/repo.git
	if strings.Contains(remote, "github.com/") {
		idx := strings.Index(remote, "github.com/")
		path := remote[idx+len("github.com/"):]
		path = strings.TrimSuffix(path, ".git")
		segments := strings.SplitN(path, "/", 2)
		if len(segments) == 2 {
			return segments[0], segments[1]
		}
	}
	return "", ""
}
