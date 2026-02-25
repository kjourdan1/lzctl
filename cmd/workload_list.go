package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var workloadListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all landing zones defined in lzctl.yaml",
	Long: `Reads lzctl.yaml and displays a table of landing zones with their
archetype, subscription, and connectivity status.

Examples:
  lzctl workload list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := configCache()
		if err != nil {
			return fmt.Errorf("load config: %w (run lzctl init first)", err)
		}

		if len(cfg.Spec.LandingZones) == 0 {
			fmt.Println("No landing zones defined.")
			fmt.Println("\nRun: lzctl workload add --name <name> --archetype corp")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tARCHETYPE\tSUBSCRIPTION\tCONNECTED\tADDRESS SPACE")
		fmt.Fprintln(w, "────\t─────────\t────────────\t─────────\t─────────────")
		for _, lz := range cfg.Spec.LandingZones {
			sub := "(pending)"
			if lz.Subscription != "" {
				sub = lz.Subscription
			}
			connected := "no"
			if lz.Connected {
				connected = "yes"
			}
			addr := "-"
			if lz.AddressSpace != "" {
				addr = lz.AddressSpace
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				lz.Name, lz.Archetype, sub, connected, addr)
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("flush output: %w", err)
		}
		fmt.Printf("\nTotal: %d landing zone(s)\n", len(cfg.Spec.LandingZones))
		return nil
	},
}
