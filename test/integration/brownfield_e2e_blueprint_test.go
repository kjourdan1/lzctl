package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBrownfieldE2E_InitAdoptAddBlueprint tests the critical brownfield path:
//
//	lzctl init → lzctl workload adopt → lzctl add-blueprint → validate golden files
//
// This exercises E8-S3 (paas-secure rendering) + E8-S7 (add-blueprint command)
// + E8-S12 (pipeline matrix update) in a single end-to-end run.
func TestBrownfieldE2E_InitAdoptAddBlueprint(t *testing.T) {
	binPath := buildCLIForIntegration(t)
	fakeTerraformDir := installFakeTerraformForIntegration(t)
	repoDir := t.TempDir()

	env := append(os.Environ(),
		"CI=true",
		"PATH="+fakeTerraformDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	// ── Step 1: init ─────────────────────────────────────────────────
	out, err := runCLI(t, binPath, env,
		"init",
		"--ci",
		"--tenant-id", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		"--project-name", "brownfield-e2e",
		"--connectivity", "none",
		"--repo-root", repoDir,
	)
	require.NoError(t, err, "lzctl init failed: %s", out)

	lzctlYAML := filepath.Join(repoDir, "lzctl.yaml")
	_, statErr := os.Stat(lzctlYAML)
	require.NoError(t, statErr, "lzctl.yaml must be created by init")

	// ── Step 2: workload adopt (add landing zone) ─────────────────────
	out, err = runCLI(t, binPath, env,
		"workload", "adopt",
		"--ci",
		"--name", "contoso-paas", "--subscription", "cccccccc-dddd-4eee-8fff-000000000001", "--archetype", "corp",
		"--address-space", "10.10.0.0/24",
		"--connected=false",
		"--repo-root", repoDir,
	)
	require.NoError(t, err, "workload adopt failed: %s", out)

	// ── Step 3: add-blueprint (attach paas-secure to the zone) ────────
	out, err = runCLI(t, binPath, env,
		"add-blueprint",
		"--ci",
		"--landing-zone", "contoso-paas",
		"--type", "paas-secure",
		"--set", "appService.sku=P2v3",
		"--set", "apim.enabled=true",
		"--repo-root", repoDir,
	)
	require.NoError(t, err, "add-blueprint failed: %s", out)

	// ── Step 4: golden file assertions ───────────────────────────────
	blueprintDir := filepath.Join(repoDir, "landing-zones", "contoso-paas", "blueprint")

	goldenFiles := []string{
		filepath.Join(blueprintDir, "main.tf"),
		filepath.Join(blueprintDir, "variables.tf"),
		filepath.Join(blueprintDir, "blueprint.auto.tfvars"),
		filepath.Join(blueprintDir, "backend.hcl"),
	}
	for _, gf := range goldenFiles {
		_, statErr = os.Stat(gf)
		require.NoError(t, statErr, "golden file must exist: %s", gf)
	}

	// Validate main.tf content (secure-by-default assertions)
	mainTF, err := os.ReadFile(filepath.Join(blueprintDir, "main.tf"))
	require.NoError(t, err)
	mainContent := string(mainTF)
	assert.Contains(t, mainContent, "output \"workload_resource_group_id\"",
		"main.tf must export mandatory outputs")
	assert.Contains(t, mainContent, "privatelink.azurewebsites.net",
		"main.tf must reference the App Service private DNS zone")
	assert.Contains(t, mainContent, "public_network_access_enabled     = false",
		"secure-by-default: public network access must be disabled")

	// Validate blueprint.auto.tfvars reflects overrides
	tfvars, err := os.ReadFile(filepath.Join(blueprintDir, "blueprint.auto.tfvars"))
	require.NoError(t, err)
	assert.Contains(t, string(tfvars), `appservice_sku = "P2v3"`,
		"tfvars must reflect the --set appService.sku override")

	// ── Step 5: verify zone-matrix.json includes blueprint entry ──────
	matrixPath := filepath.Join(repoDir, ".lzctl", "zone-matrix.json")
	matrixData, err := os.ReadFile(matrixPath)
	require.NoError(t, err)
	matrixContent := string(matrixData)
	assert.Contains(t, matrixContent, "landing-zones/contoso-paas",
		"zone-matrix.json must include the landing zone dir")
	assert.Contains(t, matrixContent, "landing-zones/contoso-paas/blueprint",
		"zone-matrix.json must include the blueprint dir after add-blueprint (E8-S12)")

	// ── Step 6: final validate must pass ─────────────────────────────
	out, err = runCLI(t, binPath, env, "validate", "--repo-root", repoDir)
	require.NoError(t, err, "validate must pass after add-blueprint: %s", out)
}

// TestBrownfieldFixture_ValidSchema ensures the brownfield fixture file
// (test/fixtures/brownfield/lzctl.yaml) passes schema validation.
func TestBrownfieldFixture_ValidSchema(t *testing.T) {
	binPath := buildCLIForIntegration(t)
	fakeTerraformDir := installFakeTerraformForIntegration(t)

	fixtureDir := filepath.Clean(filepath.Join("..", "..", "test", "fixtures", "brownfield"))
	env := append(os.Environ(),
		"CI=true",
		"PATH="+fakeTerraformDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	out, err := runCLI(t, binPath, env, "validate", "--repo-root", fixtureDir)
	require.NoError(t, err, "brownfield fixture must pass schema validation: %s", out)
}
