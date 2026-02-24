package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const (
	// DefaultCLITimeout is the maximum duration for a single az CLI invocation.
	DefaultCLITimeout = 120 * time.Second
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
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("%s %v: timed out after %s", name, args, DefaultCLITimeout)
	}
	return out, err
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
