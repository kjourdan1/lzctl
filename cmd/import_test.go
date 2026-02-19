package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kjourdan1/lzctl/internal/audit"
	"github.com/kjourdan1/lzctl/internal/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportFromAuditReport(t *testing.T) {
	// Create a temporary audit report file
	report := audit.AuditReport{
		TenantID: "test-tenant",
		Findings: []audit.AuditFinding{
			{
				ID:          "GOV-001",
				Discipline:  "governance",
				Severity:    "high",
				Title:       "Missing NSG",
				AutoFixable: true,
				Resources: []audit.ResourceRef{
					{
						ResourceID:   "/subscriptions/sub-001/resourceGroups/rg-app/providers/Microsoft.Network/networkSecurityGroups/nsg-default",
						ResourceType: "Microsoft.Network/networkSecurityGroups",
						Name:         "nsg-default",
					},
				},
			},
			{
				ID:          "SEC-001",
				Discipline:  "security",
				Severity:    "critical",
				Title:       "Manual action needed",
				AutoFixable: false,
				Resources: []audit.ResourceRef{
					{
						ResourceID:   "/subscriptions/sub-001/resourceGroups/rg-app/providers/Microsoft.Compute/virtualMachines/vm01",
						ResourceType: "Microsoft.Compute/virtualMachines",
						Name:         "vm01",
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "audit-report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(reportPath, data, 0o644))

	resources, err := importFromAuditReport(reportPath)
	require.NoError(t, err)

	// Only auto-fixable findings should produce importable resources
	assert.Len(t, resources, 1)
	assert.Equal(t, "nsg-default", resources[0].Name)
	assert.Equal(t, "azurerm_network_security_group", resources[0].TerraformType)
	assert.True(t, resources[0].Supported)
}

func TestImportFromAuditReport_FileNotFound(t *testing.T) {
	_, err := importFromAuditReport("/nonexistent/report.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading audit report")
}

func TestImportFromAuditReport_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	_, err := importFromAuditReport(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing audit report")
}

func TestApplyTypeFilters(t *testing.T) {
	resources := testImportableResources()

	t.Run("no filters", func(t *testing.T) {
		result := applyTypeFilters(resources, "", "")
		assert.Len(t, result, len(resources))
	})

	t.Run("include filter", func(t *testing.T) {
		result := applyTypeFilters(resources, "microsoft.network/virtualnetworks", "")
		assert.Len(t, result, 1)
		assert.Equal(t, "vnet-hub", result[0].Name)
	})

	t.Run("exclude filter", func(t *testing.T) {
		result := applyTypeFilters(resources, "", "microsoft.storage/storageaccounts")
		assert.Len(t, result, 1)
		assert.Equal(t, "vnet-hub", result[0].Name)
	})

	t.Run("include and exclude", func(t *testing.T) {
		result := applyTypeFilters(resources, "microsoft.network/virtualnetworks,microsoft.storage/storageaccounts", "microsoft.storage/storageaccounts")
		assert.Len(t, result, 1)
		assert.Equal(t, "vnet-hub", result[0].Name)
	})
}

func TestParseCSV(t *testing.T) {
	result := parseCSV("  Type1 , type2 ,TYPE3 ,")
	assert.True(t, result["type1"])
	assert.True(t, result["type2"])
	assert.True(t, result["type3"])
	assert.False(t, result[""])
}

func TestResolveImportMode(t *testing.T) {
	origFrom := importFrom
	origSub := importSubscription
	origRG := importResourceGroup
	defer func() {
		importFrom = origFrom
		importSubscription = origSub
		importResourceGroup = origRG
	}()

	importFrom = "report.json"
	importSubscription = ""
	importResourceGroup = ""
	assert.Equal(t, "audit-report", resolveImportMode())

	importFrom = ""
	importSubscription = "sub-001"
	assert.Equal(t, "flags", resolveImportMode())

	importFrom = ""
	importSubscription = ""
	importResourceGroup = "rg-app"
	assert.Equal(t, "flags", resolveImportMode())

	importFrom = ""
	importSubscription = ""
	importResourceGroup = ""
	assert.Equal(t, "interactive", resolveImportMode())
}

func TestResolveTargetDir(t *testing.T) {
	origLayer := importLayer
	defer func() { importLayer = origLayer }()

	importLayer = ""
	dir := resolveTargetDir("/root")
	assert.Equal(t, filepath.Join("/root", "imports"), dir)

	importLayer = "platform/connectivity"
	dir = resolveTargetDir("/root")
	assert.Equal(t, filepath.Join("/root", "platform/connectivity"), dir)
}

func TestImportCmd_CIMode_RequiresSourceFlags(t *testing.T) {
	t.Setenv("CI", "true")

	_, _, err := executeCommand("import")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--ci mode requires import source flags")
}

func testImportableResources() []importer.ImportableResource {
	return []importer.ImportableResource{
		{
			ID:            "/subscriptions/sub-001/resourceGroups/rg-core/providers/Microsoft.Network/virtualNetworks/vnet-hub",
			AzureType:     "microsoft.network/virtualnetworks",
			Name:          "vnet-hub",
			TerraformType: "azurerm_virtual_network",
			Supported:     true,
		},
		{
			ID:            "/subscriptions/sub-001/resourceGroups/rg-core/providers/Microsoft.Storage/storageAccounts/stcore",
			AzureType:     "microsoft.storage/storageaccounts",
			Name:          "stcore",
			TerraformType: "azurerm_storage_account",
			Supported:     true,
		},
	}
}
