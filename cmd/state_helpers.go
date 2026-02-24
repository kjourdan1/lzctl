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
		label := strings.Join(args, " ")
		if len(args) > 2 {
			label = strings.Join(args[:2], " ")
		}
		return "", fmt.Errorf("az %s: %s", label, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
