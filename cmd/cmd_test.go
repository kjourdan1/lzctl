package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kjourdan1/lzctl/internal/azauth"
	_ "github.com/kjourdan1/lzctl/schemas" // ensure JSON schema is loaded
)

// executeCommand runs a CLI command and captures output.
func executeCommand(args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)

	// Reset all flag defaults to avoid state leaking between tests.
	resetFlags := func(cmd *cobra.Command) {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
	resetFlags(rootCmd)
	for _, sub := range rootCmd.Commands() {
		resetFlags(sub)
	}

	err := rootCmd.Execute()

	return stdout.String(), stderr.String(), err
}

// ── Root command ────────────────────────────────────────────

func TestRootCmd_Help(t *testing.T) {
	stdout, _, err := executeCommand("--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lzctl")
	assert.Contains(t, stdout, "Azure Landing Zones")
}

// ── Version command ─────────────────────────────────────────

func TestVersionCmd(t *testing.T) {
	stdout, _, err := executeCommand("version")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lzctl version")
}

// ── Init command flags ──────────────────────────────────────

func TestInitCmd_MissingTenantID(t *testing.T) {
	// With auto-detection, --tenant-id is optional when az CLI is available.
	// Mock the command runner to simulate az CLI not being available.
	original := azauth.GetCommandRunner()
	defer azauth.SetCommandRunner(original)
	azauth.SetCommandRunner(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("exec: az not found")
	})

	_, _, err := executeCommand("init", "test-tenant")
	assert.Error(t, err) // fails because --tenant-id not provided and az CLI unavailable
}

func TestInitCmd_Help(t *testing.T) {
	stdout, _, err := executeCommand("init", "--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "init")
	assert.Contains(t, stdout, "lzctl.yaml")
	assert.Contains(t, stdout, "--project-name")
	assert.Contains(t, stdout, "--mg-model")
	assert.Contains(t, stdout, "--connectivity")
	assert.Contains(t, stdout, "--cicd-platform")
	assert.Contains(t, stdout, "--state-strategy")
}

// ── Init command in temp dir ────────────────────────────────

func TestInitCmd_WithTenantID(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := executeCommand(
		"init",
		"--tenant-id", "00000000-0000-0000-0000-000000000001",
		"--repo-root", tmpDir,
	)
	require.NoError(t, err)
}

// ── Select command ──────────────────────────────────────────

func TestSelectCmd_Help(t *testing.T) {
	stdout, _, err := executeCommand("select", "--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "profiles")
}

// ── Plan command ────────────────────────────────────────────

func TestPlanCmd_Help(t *testing.T) {
	stdout, _, err := executeCommand("plan", "--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "plan")
}

// ── Apply command ───────────────────────────────────────────

func TestApplyCmd_Help(t *testing.T) {
	stdout, _, err := executeCommand("apply", "--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "apply")
}

// ── Assess command ──────────────────────────────────────────

func TestAssessCmd_Help(t *testing.T) {
	stdout, _, err := executeCommand("assess", "--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "assess")
	assert.Contains(t, stdout, "readiness")
}

// ── Docs command ────────────────────────────────────────────

func TestDocsCmd_Help(t *testing.T) {
	stdout, _, err := executeCommand("docs", "--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "docs")
}

// ── Global mode flag ────────────────────────────────────────

func TestGlobalVerboseFlag_ShowsInHelp(t *testing.T) {
	stdout, _, err := executeCommand("--help")
	require.NoError(t, err)
	assert.Contains(t, stdout, "--verbose")
	assert.Contains(t, stdout, "--ci")
}

func TestAuditCommandRegistered(t *testing.T) {
	found := false
	for _, command := range rootCmd.Commands() {
		if command.Name() == "audit" {
			found = true
			break
		}
	}
	assert.True(t, found, "audit command should be registered")
}

	func TestAddBlueprintCommandRegistered(t *testing.T) {
		found := false
		for _, c := range rootCmd.Commands() {
			if c.Use == "add-blueprint" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected add-blueprint command to be registered")
	}
