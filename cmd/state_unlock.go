package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kjourdan1/lzctl/internal/output"
	"github.com/kjourdan1/lzctl/internal/state"
	"github.com/spf13/cobra"
)

var unlockStateKey string

var stateUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Force-release a stuck state lock",
	Long: `Break a stuck blob lease on a Terraform state file.

Azure Storage uses blob leases for state locking (equivalent to DynamoDB
locking in AWS). If a pipeline fails mid-apply, the lease may remain,
blocking subsequent operations.

This command force-breaks the lease. Use with caution — only when you
are certain no terraform operation is in progress.

Example:
  lzctl state unlock --key platform-connectivity.tfstate`,
	RunE: runStateUnlock,
}

func init() {
	stateUnlockCmd.Flags().StringVar(&unlockStateKey, "key", "", "state file key to unlock (e.g. platform-connectivity.tfstate)")
	_ = stateUnlockCmd.MarkFlagRequired("key")
	stateCmd.AddCommand(stateUnlockCmd)
}

func runStateUnlock(cmd *cobra.Command, _ []string) error {
	output.Init(verbosity > 0, jsonOutput)

	// M5: Require confirmation before breaking lease (destructive operation)
	if !effectiveCIMode() {
		fmt.Printf("⚠️  You are about to force-break the lease on %q.\n", unlockStateKey)
		fmt.Print("Type 'yes' to confirm: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(answer) != "yes" {
			return fmt.Errorf("canceled by user")
		}
	}

	cfg, err := configCache()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	mgr := state.NewManager(cfg, &azCLIAdapter{})
	if err := mgr.BreakLease(unlockStateKey); err != nil {
		return err
	}

	fmt.Printf("✅ Lease broken on %s — state is now unlocked\n", unlockStateKey)
	return nil
}
