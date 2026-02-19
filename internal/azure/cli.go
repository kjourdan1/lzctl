package azure

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// CLI abstracts az command execution to make scanner testable.
type CLI interface {
	RunJSON(args ...string) (any, error)
}

// CommandRunner executes external commands.
type CommandRunner func(name string, args ...string) ([]byte, error)

// AzCLI is the default implementation using Azure CLI.
type AzCLI struct {
	runner CommandRunner
}

// NewAzCLI returns a default Azure CLI wrapper.
func NewAzCLI() *AzCLI {
	return &AzCLI{runner: defaultRunner}
}

// NewAzCLIWithRunner returns an Azure CLI wrapper with injected runner.
func NewAzCLIWithRunner(runner CommandRunner) *AzCLI {
	if runner == nil {
		runner = defaultRunner
	}
	return &AzCLI{runner: runner}
}

func defaultRunner(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// RunJSON executes az and decodes JSON output.
func (a *AzCLI) RunJSON(args ...string) (any, error) {
	fullArgs := append(args, "--output", "json")
	out, err := a.runner("az", fullArgs...)
	if err != nil {
		return nil, fmt.Errorf("az %v: %w", args, err)
	}
	var data any
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("invalid az json output: %w", err)
	}
	return data, nil
}
