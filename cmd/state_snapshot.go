package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/output"
	"github.com/kjourdan1/lzctl/internal/state"
	"github.com/spf13/cobra"
)

var (
	snapshotLayer string
	snapshotTag   string
	snapshotAll   bool
)

var stateSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create point-in-time backup of state files",
	Long: `Create a blob snapshot of Terraform state files before mutations.

Snapshots leverage Azure Storage blob versioning to create an immutable
point-in-time copy. This is the safety net for terraform apply — if
something goes wrong, you can restore a previous version.

Use --all to snapshot every state file, or --layer to target a specific
layer. Snapshots are tagged with a label for easy identification.

Examples:
  lzctl state snapshot --all --tag "pre-apply-sprint-5"
  lzctl state snapshot --layer connectivity --tag "before-firewall-change"`,
	RunE: runStateSnapshot,
}

func init() {
	stateSnapshotCmd.Flags().StringVar(&snapshotLayer, "layer", "", "snapshot a specific layer's state only")
	stateSnapshotCmd.Flags().StringVar(&snapshotTag, "tag", "", "label for this snapshot (default: auto-generated timestamp)")
	stateSnapshotCmd.Flags().BoolVar(&snapshotAll, "all", false, "snapshot all state files")
	stateCmd.AddCommand(stateSnapshotCmd)
}

func runStateSnapshot(cmd *cobra.Command, _ []string) error {
	output.Init(verbosity > 0, jsonOutput)

	if !snapshotAll && snapshotLayer == "" {
		return fmt.Errorf("specify --all to snapshot all states, or --layer <name> for a specific layer")
	}

	cfg, err := config.Load(localConfigPath())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if snapshotTag == "" {
		snapshotTag = fmt.Sprintf("lzctl-%s", time.Now().UTC().Format("20060102-150405"))
	}

	mgr := state.NewManager(cfg, &azCLIAdapter{})

	if snapshotAll {
		snapshots, err := mgr.SnapshotAll(snapshotTag)
		if err != nil {
			return err
		}
		if jsonOutput {
			output.JSON(snapshots)
			return nil
		}
		fmt.Printf("✅ Created %d snapshot(s) with tag %q\n", len(snapshots), snapshotTag)
		for _, s := range snapshots {
			fmt.Printf("  • %s → version %s\n", s.Key, s.VersionID)
		}
		return nil
	}

	// Single layer snapshot
	stateKey := layerToStateKey(snapshotLayer)
	snap, err := mgr.CreateSnapshot(stateKey, snapshotTag)
	if err != nil {
		return err
	}

	if jsonOutput {
		output.JSON(snap)
		return nil
	}
	fmt.Printf("✅ Snapshot created: %s → version %s (tag: %s)\n", snap.Key, snap.VersionID, snap.Tag)
	return nil
}

// layerToStateKey converts a layer name to its state file key.
func layerToStateKey(layer string) string {
	layer = strings.TrimSpace(layer)
	if strings.HasSuffix(layer, ".tfstate") {
		return layer
	}
	return "platform-" + layer + ".tfstate"
}
