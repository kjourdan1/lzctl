package cmd

import (
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Terraform state lifecycle management",
	Long: `Manage Terraform state files as critical assets.

State is the single source of truth for your infrastructure. This command
provides visibility, protection, and recovery capabilities:

  snapshot   Create point-in-time backups before mutations
  list       Enumerate state files and their lock status
  health     Validate backend security posture (versioning, encryption, locking)
  unlock     Force-release a stuck blob lease

Best practices enforced by lzctl:
  • Remote state in Azure Storage with blob lease locking
  • Blob versioning enabled for audit trail and rollback
  • Soft delete for protection against accidental deletion
  • HTTPS-only + TLS 1.2 for encryption in transit
  • Automated snapshots before apply (via CI/CD pipelines)`,
}

func init() {
	rootCmd.AddCommand(stateCmd)
}
