package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildCLIForIntegration(t *testing.T) string {
	t.Helper()
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	binDir := t.TempDir()
	binName := "lzctl"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(binDir, binName)

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))

	return binPath
}

func installFakeTerraformForIntegration(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()

	if runtime.GOOS == "windows" {
		script := "@echo off\r\n" +
			"if \"%1\"==\"init\" ( echo Terraform initialized & exit /b 0 )\r\n" +
			"if \"%1\"==\"validate\" ( echo Success! The configuration is valid. & exit /b 0 )\r\n" +
			"if \"%1\"==\"plan\" ( echo Plan: 0 to add, 0 to change, 0 to destroy & exit /b 0 )\r\n" +
			"exit /b 0\r\n"
		require.NoError(t, os.WriteFile(filepath.Join(binDir, "terraform.bat"), []byte(script), 0o644))
	} else {
		script := "#!/usr/bin/env sh\n" +
			"case \"$1\" in\n" +
			"  init) echo \"Terraform initialized\"; exit 0 ;;\n" +
			"  validate) echo \"Success! The configuration is valid.\"; exit 0 ;;\n" +
			"  plan) echo \"Plan: 0 to add, 0 to change, 0 to destroy\"; exit 0 ;;\n" +
			"esac\n" +
			"exit 0\n"
		path := filepath.Join(binDir, "terraform")
		require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	}

	return binDir
}

func runCLI(t *testing.T, binPath string, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestHeadlessInit_FullWorkflow(t *testing.T) {
	binPath := buildCLIForIntegration(t)
	fakeTerraformDir := installFakeTerraformForIntegration(t)
	repoDir := t.TempDir()

	env := append(os.Environ(), "CI=true", "PATH="+fakeTerraformDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := runCLI(
		t,
		binPath,
		env,
		"init",
		"--ci",
		"--tenant-id", "00000000-0000-0000-0000-000000000001",
		"--project-name", "headless-e2e",
		"--connectivity", "none",
		"--repo-root", repoDir,
	)
	require.NoError(t, err, out)

	_, statErr := os.Stat(filepath.Join(repoDir, "lzctl.yaml"))
	require.NoError(t, statErr)
	_, statErr = os.Stat(filepath.Join(repoDir, "platform"))
	require.NoError(t, statErr)

	out, err = runCLI(t, binPath, env, "validate", "--repo-root", repoDir)
	require.NoError(t, err, out)

	out, err = runCLI(t, binPath, env, "plan", "--repo-root", repoDir)
	require.NoError(t, err, out)

	assert.NotContains(t, out, "Type 'yes' to proceed")
}
