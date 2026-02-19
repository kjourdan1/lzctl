package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/output"
	"github.com/kjourdan1/lzctl/internal/state"
	"github.com/spf13/cobra"
)

var stateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Terraform state files in the backend",
	Long: `Enumerate all .tfstate blobs in the Azure Storage container.

Shows each state file's layer name, size, last modification time, and
lock status (locked = a terraform operation is in progress).`,
	RunE: runStateList,
}

func init() {
	stateCmd.AddCommand(stateListCmd)
}

func runStateList(cmd *cobra.Command, _ []string) error {
	output.Init(verbosity > 0, jsonOutput)

	cfg, err := config.Load(localConfigPath())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	mgr := state.NewManager(cfg, &azCLIAdapter{})
	states, err := mgr.ListStates()
	if err != nil {
		return err
	}

	if jsonOutput {
		output.JSON(states)
		return nil
	}

	if len(states) == 0 {
		fmt.Println("No state files found in the backend.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "LAYER\tKEY\tSIZE\tLAST MODIFIED\tLOCK")
	for _, s := range states {
		mod := "—"
		if !s.LastModified.IsZero() {
			mod = s.LastModified.Format(time.RFC3339)
		}
		lock := "🔓"
		if s.LeaseStatus == "locked" {
			lock = "🔒"
		}
		fmt.Fprintf(w, "%s\t%s\t%d B\t%s\t%s\n", s.Layer, s.Key, s.Size, mod, lock)
	}
	w.Flush()
	return nil
}
