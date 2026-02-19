package cmd

import (
	"fmt"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/output"
	"github.com/kjourdan1/lzctl/internal/state"
	"github.com/spf13/cobra"
)

var stateHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Validate state backend security posture",
	Long: `Check that the Terraform state backend follows security best practices.

Validates:
  • Blob versioning enabled (audit trail + rollback capability)
  • Soft delete enabled (protection against accidental deletion)
  • HTTPS-only enforced (encryption in transit)
  • TLS 1.2 minimum (no legacy protocols)
  • Infrastructure encryption (double encryption at rest)
  • Container soft delete (protection against container deletion)

Each failing check includes a remediation command.`,
	RunE: runStateHealth,
}

func init() {
	stateCmd.AddCommand(stateHealthCmd)
}

func runStateHealth(cmd *cobra.Command, _ []string) error {
	output.Init(verbosity > 0, jsonOutput)

	cfg, err := config.Load(localConfigPath())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	mgr := state.NewManager(cfg, &azCLIAdapter{})
	health, err := mgr.CheckHealth()
	if err != nil {
		return err
	}

	if jsonOutput {
		output.JSON(health)
		return nil
	}

	fmt.Printf("State Backend Health — %s/%s\n", health.StorageAccount, health.Container)
	fmt.Println("─────────────────────────────────────────")
	for _, c := range health.Checks {
		icon := "✅"
		switch c.Status {
		case "fail":
			icon = "❌"
		case "warn":
			icon = "⚠️"
		}
		fmt.Printf("  %s  %s\n", icon, c.Message)
		if c.Fix != "" && c.Status != "pass" {
			fmt.Printf("       💡 %s\n", c.Fix)
		}
	}

	fmt.Println()
	if health.Healthy {
		fmt.Println("✅ State backend is healthy")
	} else {
		fmt.Println("❌ State backend has issues — resolve the findings above")
	}
	return nil
}
