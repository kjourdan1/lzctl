package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// azCLIAdapter implements state.AzCLIRunner using the local az CLI binary.
type azCLIAdapter struct{}

func (a *azCLIAdapter) Run(args ...string) (string, error) {
	cmd := exec.Command("az", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("az %s: %s", strings.Join(args[:2], " "), strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
