package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kjourdan1/lzctl/internal/audit"
	"github.com/kjourdan1/lzctl/internal/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBrownfieldWorkflow_AuditToImport tests the end-to-end brownfield flow:
//
//	audit (mock snapshot) → produce report → import from report → validate generated files.
func TestBrownfieldWorkflow_AuditToImport(t *testing.T) {
	// ----------------------------------------------------------------
	// Step 1: Build a mock tenant snapshot with known gaps
	// ----------------------------------------------------------------
	snapshot := &audit.TenantSnapshot{
		TenantID: "a1b2c3d4-e5f6-1234-5678-abcdef012345",
		ManagementGroups: []audit.ManagementGroup{
			{ID: "/providers/Microsoft.Management/managementGroups/mg-root", Name: "mg-root", DisplayName: "Root"},
			// Missing intermediate MG → should produce a governance finding
		},
		Subscriptions: []audit.Subscription{
			{ID: "sub-001", DisplayName: "Production", State: "Enabled", ManagementGroup: "mg-root"},
		},
		VirtualNetworks: []audit.VirtualNetwork{
			{ID: "/subscriptions/sub-001/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet-hub",
				Name: "vnet-hub", SubscriptionID: "sub-001", AddressSpaces: []string{"10.0.0.0/16"}},
		},
		DefenderPlans: []audit.DefenderPlan{
			{SubscriptionID: "sub-001", Name: "VirtualMachines", PricingTier: "Standard"},
		},
	}

	// ----------------------------------------------------------------
	// Step 2: Run the compliance engine to produce an audit report
	// ----------------------------------------------------------------
	engine := audit.NewComplianceEngine()
	report := engine.Evaluate(snapshot)

	assert.NotNil(t, report)
	assert.Greater(t, report.Summary.TotalFindings, 0, "expected at least 1 finding for an incomplete tenant")

	// ----------------------------------------------------------------
	// Step 3: Serialize the report to JSON (simulates --output --json)
	// ----------------------------------------------------------------
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "audit-report.json")

	reportJSON, err := audit.RenderJSON(report)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(reportPath, reportJSON, 0o644))

	// Verify round-trip: re-read and parse
	rawReport, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	var parsedReport audit.AuditReport
	require.NoError(t, json.Unmarshal(rawReport, &parsedReport))
	assert.Equal(t, report.TenantID, parsedReport.TenantID)
	assert.Equal(t, report.Score.Overall, parsedReport.Score.Overall)

	// ----------------------------------------------------------------
	// Step 4: Simulate import from audit report (extracting auto-fixable resources)
	// ----------------------------------------------------------------
	importableResources := extractAutoFixableResources(parsedReport)

	// Also test direct discovery-based import with mock CLI
	mockResources := []importer.ImportableResource{
		{
			ID:            "/subscriptions/sub-001/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet-hub",
			AzureType:     "microsoft.network/virtualnetworks",
			Name:          "vnet-hub",
			Subscription:  "sub-001",
			ResourceGroup: "rg-net",
			TerraformType: "azurerm_virtual_network",
			Supported:     true,
		},
		{
			ID:            "/subscriptions/sub-001/resourceGroups/rg-net/providers/Microsoft.Network/networkSecurityGroups/nsg-default",
			AzureType:     "microsoft.network/networksecuritygroups",
			Name:          "nsg-default",
			Subscription:  "sub-001",
			ResourceGroup: "rg-net",
			TerraformType: "azurerm_network_security_group",
			Supported:     true,
		},
		{
			ID:            "/subscriptions/sub-001/resourceGroups/rg-app/providers/Microsoft.Compute/virtualMachines/vm-legacy",
			AzureType:     "microsoft.compute/virtualmachines",
			Name:          "vm-legacy",
			Subscription:  "sub-001",
			ResourceGroup: "rg-app",
			TerraformType: "unsupported",
			Supported:     false,
			Note:          "unsupported — manual import required",
		},
	}

	// Merge: use mock resources if audit report had none auto-fixable
	allResources := append(importableResources, mockResources...)

	// ----------------------------------------------------------------
	// Step 5: Generate HCL import blocks
	// ----------------------------------------------------------------
	gen := importer.NewHCLGenerator()
	targetDir := filepath.Join(tmpDir, "imports")
	files := gen.GenerateAll(allResources, targetDir)

	require.NotEmpty(t, files, "expected generated import files")

	// ----------------------------------------------------------------
	// Step 6: Write generated files and validate their content
	// ----------------------------------------------------------------
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(f.Path, []byte(f.Content), 0o644))
	}

	// Verify import.tf files contain import blocks
	importFiles := filterFilesByName(files, "import.tf")
	require.NotEmpty(t, importFiles, "expected at least one import.tf file")

	for _, f := range importFiles {
		content, err := os.ReadFile(f.Path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "import {", "import.tf should contain import blocks")
	}

	// Verify resources.tf files contain resource blocks
	resourceFiles := filterFilesByName(files, "resources.tf")
	require.NotEmpty(t, resourceFiles, "expected at least one resources.tf file")

	for _, f := range resourceFiles {
		content, err := os.ReadFile(f.Path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "resource ", "resources.tf should contain resource blocks")
	}

	// Verify unsupported resources generate TODO comments
	allContent := ""
	for _, f := range files {
		content, _ := os.ReadFile(f.Path)
		allContent += string(content)
	}
	assert.Contains(t, allContent, "# TODO: manual import required", "unsupported resources should generate TODO comments")

	// Verify connectivity layer grouping (VNet and NSG should be in connectivity/)
	connectivityImport := findFileByPath(files, filepath.Join(targetDir, "connectivity", "import.tf"))
	assert.NotNil(t, connectivityImport, "expected connectivity/import.tf for network resources")

	// ----------------------------------------------------------------
	// Step 7: Verify the full flow is coherent
	// ----------------------------------------------------------------
	t.Log("Brownfield workflow: audit → import → validate PASSED")
	t.Logf("  Audit score: %d/100", report.Score.Overall)
	t.Logf("  Findings: %d total (%d critical, %d high)",
		report.Summary.TotalFindings, report.Summary.Critical, report.Summary.High)
	t.Logf("  Resources imported: %d (supported: %d, unsupported: %d)",
		len(allResources), countSupported(allResources), countUnsupported(allResources))
	t.Logf("  Files generated: %d", len(files))
}

func TestBrownfieldWorkflow_DiscoveryToImport(t *testing.T) {
	// Test the discovery → HCL generation path with mock data
	cli := &mockCLI{
		responses: map[string]any{
			"resource list --subscription sub-001": []any{
				map[string]any{
					"id":             "/subscriptions/sub-001/resourceGroups/rg-core",
					"type":           "Microsoft.Resources/resourceGroups",
					"name":           "rg-core",
					"subscriptionId": "sub-001",
					"resourceGroup":  "",
				},
				map[string]any{
					"id":             "/subscriptions/sub-001/resourceGroups/rg-core/providers/Microsoft.KeyVault/vaults/kv-core",
					"type":           "Microsoft.KeyVault/vaults",
					"name":           "kv-core",
					"subscriptionId": "sub-001",
					"resourceGroup":  "rg-core",
				},
			},
		},
	}

	d := importer.NewDiscovery(cli)
	resources, err := d.Discover(importer.DiscoveryOptions{Subscription: "sub-001"})
	require.NoError(t, err)
	require.Len(t, resources, 2)

	// Both types are in the MVP mapping
	for _, r := range resources {
		assert.True(t, r.Supported, "expected %s to be supported", r.AzureType)
	}

	// Generate import files
	tmpDir := t.TempDir()
	gen := importer.NewHCLGenerator()
	files := gen.GenerateAll(resources, filepath.Join(tmpDir, "imports"))
	require.NotEmpty(t, files)

	// Write and verify
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(f.Path, []byte(f.Content), 0o644))
	}

	// Verify KeyVault goes to general layer (not a CAF platform type)
	generalImport := findFileByPath(files, filepath.Join(tmpDir, "imports", "general", "import.tf"))
	assert.NotNil(t, generalImport, "expected general/import.tf for resource groups and key vaults")
}

// --- Helpers ---

func extractAutoFixableResources(report audit.AuditReport) []importer.ImportableResource {
	var resources []importer.ImportableResource
	for _, finding := range report.Findings {
		if !finding.AutoFixable {
			continue
		}
		for _, ref := range finding.Resources {
			tfType, supported := importer.MapTerraformType(ref.ResourceType)
			r := importer.ImportableResource{
				ID:            ref.ResourceID,
				AzureType:     strings.ToLower(ref.ResourceType),
				Name:          ref.Name,
				TerraformType: tfType,
				Supported:     supported,
			}
			if !supported {
				r.TerraformType = "unsupported"
				r.Note = "unsupported — manual import required"
			}
			resources = append(resources, r)
		}
	}
	return resources
}

func filterFilesByName(files []importer.GeneratedFile, name string) []importer.GeneratedFile {
	var result []importer.GeneratedFile
	for _, f := range files {
		if filepath.Base(f.Path) == name {
			result = append(result, f)
		}
	}
	return result
}

func findFileByPath(files []importer.GeneratedFile, path string) *importer.GeneratedFile {
	normalized := filepath.ToSlash(path)
	for _, f := range files {
		if filepath.ToSlash(f.Path) == normalized {
			return &f
		}
	}
	return nil
}

func countSupported(resources []importer.ImportableResource) int {
	count := 0
	for _, r := range resources {
		if r.Supported {
			count++
		}
	}
	return count
}

func countUnsupported(resources []importer.ImportableResource) int {
	count := 0
	for _, r := range resources {
		if !r.Supported {
			count++
		}
	}
	return count
}

// mockCLI implements azure.CLI for integration tests.
type mockCLI struct {
	responses map[string]any
}

func (m *mockCLI) RunJSON(args ...string) (any, error) {
	key := strings.Join(args, " ")
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return []any{}, nil
}
