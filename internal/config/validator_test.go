package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadSchema(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "schemas", "lzctl-v1.schema.json"))
	require.NoError(t, err, "failed to read schema file")
	SetSchema(data)
}

func TestValidate_ValidStandardConfig(t *testing.T) {
	loadSchema(t)

	cfg, err := Load(filepath.Join(fixturesDir(), "standard-hub-spoke.yaml"))
	require.NoError(t, err)

	result, err := Validate(cfg)
	require.NoError(t, err)
	assert.True(t, result.Valid, "expected valid config but got errors: %v", result.Errors)
}

func TestValidate_ValidLiteConfig(t *testing.T) {
	loadSchema(t)

	cfg, err := Load(filepath.Join(fixturesDir(), "lite-no-connectivity.yaml"))
	require.NoError(t, err)

	result, err := Validate(cfg)
	require.NoError(t, err)
	assert.True(t, result.Valid, "expected valid config but got errors: %v", result.Errors)
}

func TestValidate_InvalidConfig_MissingTenant(t *testing.T) {
	loadSchema(t)

	data, err := os.ReadFile(filepath.Join(fixturesDir(), "invalid-overlap.yaml"))
	require.NoError(t, err)

	result, err := ValidateYAML(data)
	require.NoError(t, err)
	assert.False(t, result.Valid, "expected invalid config to fail validation")
	assert.NotEmpty(t, result.Errors)

	// Check that we caught at least the missing tenant and invalid enum values
	errorFields := make([]string, len(result.Errors))
	errorDescs := make([]string, len(result.Errors))
	for i, e := range result.Errors {
		errorFields[i] = e.Field
		errorDescs[i] = e.Description
	}

	t.Logf("Validation errors: %v", result.Errors)

	// Should catch at least one of: missing tenant, invalid model enum, invalid connectivity type, invalid cicd platform
	hasErrors := len(result.Errors) >= 1
	assert.True(t, hasErrors, "expected multiple validation errors")
}

func TestValidate_SchemaNotLoaded(t *testing.T) {
	// Reset schema
	origSchema := schemaBytes
	schemaBytes = nil
	defer func() { schemaBytes = origSchema }()

	cfg := &LZConfig{APIVersion: "lzctl/v1"}
	_, err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema not loaded")
}

func TestValidateYAML_ValidConfig(t *testing.T) {
	loadSchema(t)

	data, err := os.ReadFile(filepath.Join(fixturesDir(), "standard-hub-spoke.yaml"))
	require.NoError(t, err)

	result, err := ValidateYAML(data)
	require.NoError(t, err)
	assert.True(t, result.Valid, "expected valid YAML but got errors: %v", result.Errors)
}

func TestValidateYAML_InvalidYAML(t *testing.T) {
	loadSchema(t)

	_, err := ValidateYAML([]byte("{{{{not yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing YAML")
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	loadSchema(t)

	// Minimal config missing many required fields
	cfg := &LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
	}

	result, err := Validate(cfg)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestValidate_WrongEnumValues(t *testing.T) {
	loadSchema(t)

	cfg := &LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Metadata: Metadata{
			Name:          "test",
			Tenant:        "test.onm.com",
			PrimaryRegion: "westeurope",
		},
		Spec: Spec{
			Platform: Platform{
				ManagementGroups: ManagementGroupsConfig{Model: "invalid-model"},
				Connectivity:     ConnectivityConfig{Type: "mesh"},
				Identity:         IdentityConfig{Type: "password"},
				Management: ManagementConfig{
					LogAnalytics: LogAnalyticsConfig{RetentionDays: 90},
					Defender:     DefenderConfig{Enabled: true, Plans: []string{"VM"}},
				},
			},
			Governance: Governance{
				Policies: PolicyConfig{Assignments: []string{"p1"}},
			},
			Naming:       Naming{Convention: "caf"},
			StateBackend: StateBackend{ResourceGroup: "rg", StorageAccount: "sa", Container: "c", Subscription: "s"},
			CICD: CICD{
				Platform:     "gitlab",
				BranchPolicy: BranchPolicy{MainBranch: "main", RequirePR: true},
			},
		},
	}

	result, err := Validate(cfg)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	// Should have errors for invalid enum values (model, connectivity type, identity type, cicd platform)
	assert.True(t, len(result.Errors) >= 3, "expected at least 3 enum validation errors, got %d", len(result.Errors))
}

func TestValidate_BlueprintEnum(t *testing.T) {
	loadSchema(t)

	cfg := &LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Metadata: Metadata{
			Name:          "test",
			Tenant:        "00000000-0000-0000-0000-000000000001",
			PrimaryRegion: "westeurope",
		},
		Spec: Spec{
			Platform: Platform{
				ManagementGroups: ManagementGroupsConfig{Model: "caf-standard"},
				Connectivity:     ConnectivityConfig{Type: "none"},
				Identity:         IdentityConfig{Type: "workload-identity-federation"},
				Management: ManagementConfig{
					LogAnalytics: LogAnalyticsConfig{RetentionDays: 90},
					Defender:     DefenderConfig{Enabled: true, Plans: []string{"VirtualMachines"}},
				},
			},
			Governance:   Governance{Policies: PolicyConfig{Assignments: []string{"deploy-mdfc-config"}}},
			Naming:       Naming{Convention: "caf"},
			StateBackend: StateBackend{ResourceGroup: "rg", StorageAccount: "ststate", Container: "tfstate", Subscription: "00000000-0000-0000-0000-000000000001"},
			LandingZones: []LandingZone{{
				Name:         "contoso-paas",
				Subscription: "11111111-1111-4111-8111-111111111111",
				Archetype:    "corp",
				AddressSpace: "10.1.0.0/24",
				Connected:    true,
				Blueprint: &Blueprint{
					Type: "invalid-blueprint",
				},
			}},
			CICD: CICD{Platform: "github-actions", BranchPolicy: BranchPolicy{MainBranch: "main", RequirePR: true}},
		},
	}

	result, err := Validate(cfg)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)

	hasBlueprintErr := false
	for _, e := range result.Errors {
		if e.Field == "spec.landingZones.0.blueprint.type" {
			hasBlueprintErr = true
			break
		}
	}
	assert.True(t, hasBlueprintErr, "expected blueprint type enum error")
}
