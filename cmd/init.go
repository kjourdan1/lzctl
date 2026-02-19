package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

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
    management-groups/    (Resource Organisation)
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
	initTenantID       string
	initSubscriptionID string
	initForce          bool
	initNoBootstrap    bool
)

func init() {
	initCmd.Flags().StringVar(&initTenantID, "tenant-id", "", "Azure AD tenant ID (auto-detected from Azure CLI if omitted)")
	initCmd.Flags().StringVar(&initSubscriptionID, "subscription-id", "", "Azure Subscription ID (auto-detected from Azure CLI if omitted)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing files")
	initCmd.Flags().BoolVar(&initNoBootstrap, "no-bootstrap", false, "skip automatic state backend provisioning")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command) error {
	output.Init(verbosity > 0, jsonOutput)

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	var cfg *config.LZConfig
	var wizardCfg *wizard.InitConfig
	if cfgFile != "" {
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config from --config %q: %w", cfgFile, err)
		}
	} else {
		var runErr error
		wizardCfg, runErr = wizard.NewInitWizard(nil).Run()
		if runErr != nil {
			if errors.Is(runErr, wizard.ErrCancelled) {
				output.Warn("init wizard cancelled")
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
