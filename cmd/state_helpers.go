package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const azCLIAdapterTimeout = 120 * time.Second

// azCLIAdapter implements state.AzCLIRunner using the local az CLI binary.
type azCLIAdapter struct{}

func (a *azCLIAdapter) Run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), azCLIAdapterTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "az", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		label := strings.Join(args, " ")
		if len(args) > 2 {
			label = strings.Join(args[:2], " ")
		}
		return "", fmt.Errorf("az %s: %s", label, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
