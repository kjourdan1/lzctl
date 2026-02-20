package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installFakeTerraform(t *testing.T, planLine string, planExitCode int) string {
	t.Helper()

	binDir := t.TempDir()
	var scriptPath string

	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(binDir, "terraform.bat")
		script := "@echo off\r\n" +
			"if \"%1\"==\"init\" (\r\n" +
			"  echo Terraform has been successfully initialized!\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"if \"%1\"==\"validate\" (\r\n" +
			"  echo Success! The configuration is valid.\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"if \"%1\"==\"plan\" (\r\n" +
			"  echo " + planLine + "\r\n" +
			"  exit /b " + strconv.Itoa(planExitCode) + "\r\n" +
			")\r\n" +
			"if \"%1\"==\"apply\" (\r\n" +
			"  echo Apply complete! Resources: 0 added, 0 changed, 0 destroyed.\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"exit /b 0\r\n"
		require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o644))
	} else {
		scriptPath = filepath.Join(binDir, "terraform")
		script := "#!/bin/sh\n" +
			"case \"$1\" in\n" +
			"  init)\n" +
			"    echo \"Terraform has been successfully initialized!\"\n" +
			"    exit 0\n" +
			"    ;;\n" +
			"  validate)\n" +
			"    echo \"Success! The configuration is valid.\"\n" +
			"    exit 0\n" +
			"    ;;\n" +
			"  plan)\n" +
			"    echo \"" + planLine + "\"\n" +
			"    exit " + strconv.Itoa(planExitCode) + "\n" +
			"    ;;\n" +
			"  apply)\n" +
			"    echo \"Apply complete! Resources: 0 added, 0 changed, 0 destroyed.\"\n" +
			"    exit 0\n" +
			"    ;;\n" +
			"esac\n" +
			"exit 0\n"
		require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))
	}

	t.Setenv("PATH", binDir)
	return scriptPath
}

func initRepoForCommandTests(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	_, _, err := executeCommand("init", "--tenant-id", "00000000-0000-0000-0000-000000000001", "--repo-root", repo)
	require.NoError(t, err)
	return repo
}

func executeCommandWithProcessIO(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = wOut
	os.Stderr = wErr

	// Read pipes concurrently to prevent deadlock on Windows where pipe
	// buffers are small (4 KB) and synchronous writes can block the command.
	type result struct {
		data []byte
		err  error
	}
	outCh := make(chan result, 1)
	errCh := make(chan result, 1)
	go func() {
		b, readErr := io.ReadAll(rOut)
		outCh <- result{b, readErr}
	}()
	go func() {
		b, readErr := io.ReadAll(rErr)
		errCh <- result{b, readErr}
	}()

	_, _, runErr := executeCommand(args...)

	require.NoError(t, wOut.Close())
	require.NoError(t, wErr.Close())

	outRes := <-outCh
	errRes := <-errCh

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	require.NoError(t, outRes.err)
	require.NoError(t, errRes.err)

	return string(outRes.data), string(errRes.data), runErr
}

func TestValidateCmd_MissingConfig_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, err := executeCommand("validate", "--repo-root", tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading config")
}

func TestPlanCmd_WithoutTerraform_ReturnsError(t *testing.T) {
	t.Setenv("PATH", "")
	tmpDir := t.TempDir()

	_, _, err := executeCommand("plan", "--repo-root", tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terraform not found")
}

func TestApplyCmd_WithoutTerraform_ReturnsError(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("CI", "") // prevent GitHub Actions CI=true from triggering --auto-approve gate
	tmpDir := t.TempDir()

	_, _, err := executeCommand("apply", "--repo-root", tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terraform not found")
}

func TestDriftCmd_WithoutTerraform_ReturnsError(t *testing.T) {
	t.Setenv("PATH", "")
	tmpDir := t.TempDir()

	_, _, err := executeCommand("drift", "--repo-root", tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terraform not found")
}

func TestHistoryCmd_NoAuditEvents(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, stderr, err := executeCommandWithProcessIO(t, "history")
	assert.NoError(t, err)
	assert.Contains(t, stderr, "No audit events found")
}

func TestValidateCmd_JsonOutput_OnInitializedRepo(t *testing.T) {
	installFakeTerraform(t, "Plan: 0 to add, 0 to change, 0 to destroy", 0)
	repo := initRepoForCommandTests(t)

	stdout, _, err := executeCommandWithProcessIO(t, "validate", "--repo-root", repo, "--json")
	require.NoError(t, err)

	type validateJSON struct {
		Status string `json:"status"`
		Data   struct {
			Errors float64 `json:"errors"`
		} `json:"data"`
	}
	var payload validateJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	assert.Equal(t, "ok", payload.Status)
	assert.Equal(t, float64(0), payload.Data.Errors)
}

func TestPlanCmd_TargetLayer_AndOutFile(t *testing.T) {
	installFakeTerraform(t, "Plan: 1 to add, 0 to change, 0 to destroy", 2)
	repo := initRepoForCommandTests(t)
	outPath := filepath.Join(t.TempDir(), "plan.out")

	_, stderr, err := executeCommandWithProcessIO(t, "plan", "--repo-root", repo, "--layer", "management-groups", "--out", outPath)
	require.NoError(t, err)
	assert.Contains(t, stderr, "management-groups")

	b, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(b), "## management-groups")
}

func TestApplyCmd_DryRun_TargetLayer(t *testing.T) {
	installFakeTerraform(t, "Plan: 1 to add, 0 to change, 0 to destroy", 2)
	repo := initRepoForCommandTests(t)

	_, stderr, err := executeCommandWithProcessIO(t, "--dry-run", "apply", "--repo-root", repo, "--layer", "management-groups")
	require.NoError(t, err)
	assert.Contains(t, stderr, "management-groups")
	assert.Contains(t, stderr, "(dry-run)")
}

func TestDriftCmd_JsonOutput_WithDetectedDrift(t *testing.T) {
	installFakeTerraform(t, "Plan: 2 to add, 1 to change, 0 to destroy", 2)
	repo := initRepoForCommandTests(t)

	stdout, _, err := executeCommandWithProcessIO(t, "drift", "--repo-root", repo, "--layer", "management-groups", "--json")
	assert.Error(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	assert.Equal(t, "drift-detected", payload["status"])
	assert.Equal(t, float64(3), payload["totalDrift"])
}

func TestRollbackCmd_DryRun_TargetLayer(t *testing.T) {
	installFakeTerraform(t, "Plan: 0 to add, 1 to change, 0 to destroy", 0)
	repo := initRepoForCommandTests(t)

	_, stderr, err := executeCommandWithProcessIO(t, "--dry-run", "rollback", "--repo-root", repo, "--layer", "management-groups", "--auto-approve")
	require.NoError(t, err)
	assert.Contains(t, stderr, "management-groups")
	assert.Contains(t, stderr, "[DRY-RUN]")
}

func TestApplyCmd_CIMode_RequiresAutoApprove(t *testing.T) {
	t.Setenv("CI", "true")
	repo := t.TempDir()

	_, _, err := executeCommand("apply", "--repo-root", repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--ci mode requires --auto-approve for apply")
}

func TestRollbackCmd_CIMode_RequiresAutoApprove(t *testing.T) {
	t.Setenv("CI", "true")
	repo := t.TempDir()

	_, _, err := executeCommand("rollback", "--repo-root", repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--ci mode requires --auto-approve for rollback")
}
