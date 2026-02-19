// lzctl – Landing Zone Factory CLI
// Stateless orchestrator: reads/writes lzctl.yaml in a local Git repo,
// drives Terraform + AVM platform layers, and produces documentation.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kjourdan1/lzctl/cmd"
	"github.com/kjourdan1/lzctl/internal/audit"
	"github.com/kjourdan1/lzctl/internal/exitcode"
	_ "github.com/kjourdan1/lzctl/schemas"
)

func main() {
	start := time.Now()
	if err := cmd.Execute(); err != nil {
		code := exitcode.Of(err)
		event := audit.BuildEvent(os.Args, "failure", code, time.Since(start))
		_ = audit.Write(event)
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(code)
	}

	event := audit.BuildEvent(os.Args, "success", exitcode.OK, time.Since(start))
	_ = audit.Write(event)
}
