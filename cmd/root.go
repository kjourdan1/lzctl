// Package cmd implements the Cobra-based CLI for lzctl.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	repoRoot   string
	verbosity  int
	dryRun     bool
	jsonOutput bool // --json flag for machine-readable output
	ciMode     bool
)

// rootCmd is the top-level command for lzctl.
var rootCmd = &cobra.Command{
	Use:   "lzctl",
	Short: "Landing Zone Factory CLI – Azure landing zones orchestrator",
	Long: `lzctl is a stateless CLI that orchestrates Azure Landing Zones following the
Cloud Adoption Framework (CAF) design areas.

It uses Terraform with Azure Verified Modules (AVM) to deploy platform
layers in CAF dependency order:
  1. management-groups   (Resource Organisation)
  2. identity            (Identity & Access)
  3. management          (Management & Monitoring)
  4. governance          (Azure Policies)
  5. connectivity        (Hub-Spoke or vWAN)

Project configuration lives in lzctl.yaml (source of truth).
Templates generate Terraform configs under platform/ and landing-zones/.
Compatible with Azure DevOps Services and GitHub Actions CI/CD.

Workflow: init → validate → plan → apply → audit`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: lzctl.yaml)")
	rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "path to project repo root")
	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "increase verbosity (-v, -vv, -vvv)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate actions without changing Azure resources")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output results as JSON (machine-readable)")
	rootCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "strict non-interactive mode (fails when required inputs are missing)")

	_ = viper.BindPFlag("repo_root", rootCmd.PersistentFlags().Lookup("repo-root"))
	_ = viper.BindPFlag("dry_run", rootCmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("ci", rootCmd.PersistentFlags().Lookup("ci"))
}

func effectiveCIMode() bool {
	if ciMode {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("CI")), "true")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("lzctl")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}
	viper.SetEnvPrefix("LZCTL")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil && verbosity > 0 {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
