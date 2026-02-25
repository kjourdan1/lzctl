package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleConfig() *config.LZConfig {
	cfg := &config.LZConfig{
		APIVersion: "lzctl/v1",
		Kind:       "LandingZone",
		Metadata: config.Metadata{
			Name:          "contoso-alz",
			Tenant:        "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
			PrimaryRegion: "westeurope",
		},
		Spec: config.Spec{
			Platform: config.Platform{
				ManagementGroups: config.ManagementGroupsConfig{Model: "caf-standard"},
				Connectivity:     config.ConnectivityConfig{Type: "none"},
				Identity:         config.IdentityConfig{Type: "workload-identity-federation"},
				Management: config.ManagementConfig{
					LogAnalytics: config.LogAnalyticsConfig{RetentionDays: 90},
					Defender:     config.DefenderConfig{Enabled: true, Plans: []string{"VirtualMachines"}},
				},
			},
			Governance: config.Governance{Policies: config.PolicyConfig{Assignments: []string{"deploy-mdfc-config"}}},
			Naming:     config.Naming{Convention: "caf"},
			StateBackend: config.StateBackend{
				ResourceGroup:  "rg-lzctl-state",
				StorageAccount: "stlzctlstate",
				Container:      "tfstate",
				Subscription:   "00000000-0000-0000-0000-000000000000",
			},
			CICD: config.CICD{
				Platform: "github-actions",
				BranchPolicy: config.BranchPolicy{
					MainBranch: "main",
					RequirePR:  true,
				},
			},
		},
	}
	config.ApplyDefaults(cfg)
	return cfg
}

func TestRenderAll(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	files, err := engine.RenderAll(sampleConfig())
	require.NoError(t, err)
	assert.Len(t, files, 22)

	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
		assert.NotEmpty(t, f.Content)
	}

	assert.True(t, paths["lzctl.yaml"])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "shared", "backend.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "shared", "backend.hcl"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "shared", "providers.tf"))])
	assert.True(t, paths[".gitignore"])
	assert.True(t, paths["README.md"])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "management-groups", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "management", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "governance", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "governance", "policies", "caf-default.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "identity", "main.tf"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join(".github", "workflows", "validate.yml"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join(".github", "workflows", "deploy.yml"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join(".github", "workflows", "drift.yml"))])
}

func TestRenderAll_ConnectivityVariants(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	t.Run("vwan", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.Platform.Connectivity.Type = "vwan"
		cfg.Spec.Platform.Connectivity.Hub = nil

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		paths := map[string]bool{}
		for _, f := range files {
			paths[f.Path] = true
		}
		assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "connectivity", "main.tf"))])
	})

	t.Run("hub_spoke_fw", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.Platform.Connectivity.Type = "hub-spoke"
		cfg.Spec.Platform.Connectivity.Hub = &config.HubConfig{
			Region:       "westeurope",
			AddressSpace: "10.0.0.0/16",
			Firewall:     config.FirewallConfig{Enabled: true, SKU: "Standard"},
		}

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		found := false
		for _, f := range files {
			if f.Path == filepath.ToSlash(filepath.Join("platform", "connectivity", "main.tf")) {
				found = true
				assert.Contains(t, f.Content, "azure_firewall")
			}
		}
		assert.True(t, found)
	})

	t.Run("azure_devops_pipeline", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.CICD.Platform = "azure-devops"

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		paths := map[string]bool{}
		for _, f := range files {
			paths[f.Path] = true
		}

		assert.True(t, paths[filepath.ToSlash(filepath.Join(".azuredevops", "pipelines", "validate.yml"))])
		assert.True(t, paths[filepath.ToSlash(filepath.Join(".azuredevops", "pipelines", "deploy.yml"))])
		assert.True(t, paths[filepath.ToSlash(filepath.Join(".azuredevops", "pipelines", "drift.yml"))])
	})

	t.Run("landing_zone_archetypes", func(t *testing.T) {
		cfg := sampleConfig()
		cfg.Spec.LandingZones = []config.LandingZone{
			{Name: "corp-zone", Archetype: "corp", AddressSpace: "10.10.0.0/24", Connected: true, Tags: map[string]string{"env": "prod"}},
			{Name: "online-zone", Archetype: "online", AddressSpace: "10.20.0.0/24", Connected: true},
			{Name: "sandbox-zone", Archetype: "sandbox", AddressSpace: "10.30.0.0/24", Connected: false},
		}

		files, renderErr := engine.RenderAll(cfg)
		require.NoError(t, renderErr)

		paths := map[string]string{}
		for _, f := range files {
			paths[f.Path] = f.Content
		}

		assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "corp-zone", "main.tf")))
		assert.Contains(t, paths[filepath.ToSlash(filepath.Join("landing-zones", "corp-zone", "main.tf"))], "corp_vnet")

		assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "online-zone", "main.tf")))
		assert.Contains(t, paths[filepath.ToSlash(filepath.Join("landing-zones", "online-zone", "main.tf"))], "allow-https-inbound")

		assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "sandbox-zone", "main.tf")))
		assert.NotContains(t, paths[filepath.ToSlash(filepath.Join("landing-zones", "sandbox-zone", "main.tf"))], "virtual_network_peering")
	})
}

func TestRenderAll_GitHubDeployWorkflow_OrderAndBackendConfig(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	files, err := engine.RenderAll(sampleConfig())
	require.NoError(t, err)

	contentByPath := map[string]string{}
	for _, file := range files {
		contentByPath[file.Path] = file.Content
	}

	deployPath := filepath.ToSlash(filepath.Join(".github", "workflows", "deploy.yml"))
	deploy := contentByPath[deployPath]
	require.NotEmpty(t, deploy)

	assert.Contains(t, deploy, "for d in platform/management-groups platform/identity platform/management platform/governance platform/connectivity")
	assert.Contains(t, deploy, "terraform -chdir=\"$d\" init -input=false -backend-config=../../backend.hcl")

}

func TestRenderAll_GitHubValidateWorkflow_Order(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	files, err := engine.RenderAll(sampleConfig())
	require.NoError(t, err)

	contentByPath := map[string]string{}
	for _, file := range files {
		contentByPath[file.Path] = file.Content
	}

	validatePath := filepath.ToSlash(filepath.Join(".github", "workflows", "validate.yml"))
	validate := contentByPath[validatePath]
	require.NotEmpty(t, validate)

	assert.Contains(t, validate, "for d in platform/management-groups platform/identity platform/management platform/governance platform/connectivity")
	assert.Contains(t, validate, "terraform -chdir=\"$d\" init -backend=false -input=false")
	assert.Contains(t, validate, "terraform -chdir=\"$d\" validate")
}

func TestRenderAll_SharedBackendConfigRouting(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	files, err := engine.RenderAll(sampleConfig())
	require.NoError(t, err)

	contentByPath := map[string]string{}
	for _, file := range files {
		contentByPath[file.Path] = file.Content
	}

	backendHCLPath := filepath.ToSlash(filepath.Join("platform", "shared", "backend.hcl"))
	backendTFPath := filepath.ToSlash(filepath.Join("platform", "shared", "backend.tf"))

	backendHCL := contentByPath[backendHCLPath]
	backendTF := contentByPath[backendTFPath]

	require.NotEmpty(t, backendHCL)
	require.NotEmpty(t, backendTF)

	assert.Contains(t, backendHCL, "resource_group_name  = \"rg-lzctl-state\"")
	assert.Contains(t, backendHCL, "storage_account_name = \"stlzctlstate\"")
	assert.Contains(t, backendHCL, "container_name       = \"tfstate\"")
	assert.Contains(t, backendHCL, "subscription_id      = \"00000000-0000-0000-0000-000000000000\"")

	assert.Contains(t, backendTF, "key                  = \"platform/shared/terraform.tfstate\"")
	assert.Contains(t, backendTF, "use_azuread_auth     = true")
}

func TestRenderAll_DriftWorkflowIncludesLandingZones(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.LandingZones = []config.LandingZone{
		{Name: "corp zone", Archetype: "corp", AddressSpace: "10.10.0.0/24", Connected: true},
	}

	files, err := engine.RenderAll(cfg)
	require.NoError(t, err)

	contentByPath := map[string]string{}
	for _, file := range files {
		contentByPath[file.Path] = file.Content
	}

	driftPath := filepath.ToSlash(filepath.Join(".github", "workflows", "drift.yml"))
	drift := contentByPath[driftPath]
	require.NotEmpty(t, drift)

	assert.Contains(t, drift, "landing-zones/corp-zone")
}

func TestRenderAll_DriftWorkflow_IncludesBlueprintDirs(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.LandingZones = []config.LandingZone{
		{
			Name: "corp-zone", Archetype: "corp", AddressSpace: "10.10.0.0/24", Connected: true,
			Blueprint: &config.Blueprint{Type: "paas-secure"},
		},
		{Name: "sandbox-zone", Archetype: "sandbox", AddressSpace: "10.20.0.0/24"},
	}

	files, err := engine.RenderAll(cfg)
	require.NoError(t, err)

	contentByPath := map[string]string{}
	for _, f := range files {
		contentByPath[f.Path] = f.Content
	}

	driftPath := filepath.ToSlash(filepath.Join(".github", "workflows", "drift.yml"))
	drift := contentByPath[driftPath]
	require.NotEmpty(t, drift)

	assert.Contains(t, drift, "landing-zones/corp-zone")
	assert.Contains(t, drift, "landing-zones/corp-zone/blueprint")
	// sandbox has no blueprint — no blueprint dir in drift
	assert.NotContains(t, drift, "landing-zones/sandbox-zone/blueprint")
}

func TestRenderAll_DeployWorkflow_IncludesLandingZonesAndBlueprints(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.LandingZones = []config.LandingZone{
		{
			Name: "app-zone", Archetype: "corp", AddressSpace: "10.1.0.0/24",
			Blueprint: &config.Blueprint{Type: "paas-secure"},
		},
	}

	files, err := engine.RenderAll(cfg)
	require.NoError(t, err)

	contentByPath := map[string]string{}
	for _, f := range files {
		contentByPath[f.Path] = f.Content
	}

	deployPath := filepath.ToSlash(filepath.Join(".github", "workflows", "deploy.yml"))
	deploy := contentByPath[deployPath]
	require.NotEmpty(t, deploy)

	assert.Contains(t, deploy, "landing-zones/app-zone")
	assert.Contains(t, deploy, "landing-zones/app-zone/blueprint")
	// Blueprint-specific backend config flag
	assert.Contains(t, deploy, "*/blueprint")
}

func TestRenderBlueprint_PaasSecure(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	files, err := engine.RenderBlueprint("contoso-paas", &config.Blueprint{Type: "paas-secure"}, cfg)
	require.NoError(t, err)
	require.Len(t, files, 4)

	content := map[string]string{}
	for _, f := range files {
		content[f.Path] = f.Content
	}

	assert.Contains(t, content, filepath.ToSlash(filepath.Join("landing-zones", "contoso-paas", "blueprint", "main.tf")))
	assert.Contains(t, content, filepath.ToSlash(filepath.Join("landing-zones", "contoso-paas", "blueprint", "variables.tf")))
	assert.Contains(t, content, filepath.ToSlash(filepath.Join("landing-zones", "contoso-paas", "blueprint", "blueprint.auto.tfvars")))
	assert.Contains(t, content, filepath.ToSlash(filepath.Join("landing-zones", "contoso-paas", "blueprint", "backend.hcl")))

	assert.Contains(t, content[filepath.ToSlash(filepath.Join("landing-zones", "contoso-paas", "blueprint", "main.tf"))], "output \"workload_resource_group_id\"")
	assert.Contains(t, content[filepath.ToSlash(filepath.Join("landing-zones", "contoso-paas", "blueprint", "main.tf"))], "privatelink.azurewebsites.net")
}

func TestRenderBlueprint_PaasSecure_Overrides(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	files, err := engine.RenderBlueprint("contoso-paas", &config.Blueprint{
		Type: "paas-secure",
		Overrides: map[string]any{
			"apim":       map[string]any{"enabled": false, "sku": "Premium_1"},
			"appService": map[string]any{"sku": "P2v3", "runtimeStack": "NODE|20-lts"},
			"keyVault":   map[string]any{"softDeleteRetentionDays": 30},
		},
	}, cfg)
	require.NoError(t, err)

	var tfvars string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "blueprint.auto.tfvars") {
			tfvars = f.Content
			break
		}
	}
	require.NotEmpty(t, tfvars)
	assert.Contains(t, tfvars, `appservice_sku = "P2v3"`)
	assert.Contains(t, tfvars, `apim_enabled = false`)
	assert.Contains(t, tfvars, `keyvault_soft_delete_retention_days = 30`)
}

func TestRenderBlueprint_AKSPlatform(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	files, err := engine.RenderBlueprint("contoso-aks", &config.Blueprint{Type: "aks-platform"}, cfg)
	require.NoError(t, err)
	// No ArgoCD → 5 files: main.tf, variables.tf, blueprint.auto.tfvars, backend.hcl, Makefile
	require.Len(t, files, 5)

	content := map[string]string{}
	for _, f := range files {
		content[f.Path] = f.Content
	}

	mainPath := filepath.ToSlash(filepath.Join("landing-zones", "contoso-aks", "blueprint", "main.tf"))
	assert.Contains(t, content, mainPath)
	assert.Contains(t, content[mainPath], "output \"workload_resource_group_id\"")
	assert.Contains(t, content[mainPath], "private_cluster_enabled        = true")
	assert.Contains(t, content[mainPath], "azure_policy_enabled           = true")
	assert.Contains(t, content[mainPath], "oidc_issuer_enabled            = true")
	assert.Contains(t, content[mainPath], "output \"aks_oidc_issuer_url\"")
	// ACR + KV private endpoints
	assert.Contains(t, content[mainPath], "privatelink.azurecr.io")
	assert.Contains(t, content[mainPath], "privatelink.vaultcore.azure.net")

	makePath := filepath.ToSlash(filepath.Join("landing-zones", "contoso-aks", "blueprint", "Makefile"))
	assert.Contains(t, content, makePath)
	assert.Contains(t, content[makePath], "argocd-login")
	assert.Contains(t, content[makePath], "kubectl port-forward")

	// No ArgoCD appset file
	appsetPath := filepath.ToSlash(filepath.Join("landing-zones", "contoso-aks", "blueprint", "argocd", "appset.yaml"))
	assert.NotContains(t, content, appsetPath)
}

func TestRenderBlueprint_AKSPlatform_WithArgoCD(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	files, err := engine.RenderBlueprint("contoso-aks", &config.Blueprint{
		Type: "aks-platform",
		Overrides: map[string]any{
			"aks": map[string]any{"version": "1.30"},
			"acr": map[string]any{"sku": "Premium"},
			"argocd": map[string]any{
				"enabled":        true,
				"mode":           "extension",
				"repoUrl":        "https://github.com/org/app-repo",
				"targetRevision": "main",
				"appPath":        "apps/",
			},
		},
	}, cfg)
	require.NoError(t, err)
	// With ArgoCD and AppSet: 6 files
	require.Len(t, files, 6)

	content := map[string]string{}
	for _, f := range files {
		content[f.Path] = f.Content
	}

	// ApplicationSet generated (E9-S4)
	appsetPath := filepath.ToSlash(filepath.Join("landing-zones", "contoso-aks", "blueprint", "argocd", "appset.yaml"))
	require.Contains(t, content, appsetPath)
	assert.Contains(t, content[appsetPath], "kind: ApplicationSet")
	assert.Contains(t, content[appsetPath], "https://github.com/org/app-repo")
	assert.Contains(t, content[appsetPath], "selfHeal: true")

	// ArgoCD extension block in main.tf (E9-S2)
	mainPath := filepath.ToSlash(filepath.Join("landing-zones", "contoso-aks", "blueprint", "main.tf"))
	assert.Contains(t, content[mainPath], "azurerm_kubernetes_cluster_extension")
	assert.Contains(t, content[mainPath], "microsoft.flux")

	// tfvars reflects overrides
	tfvarsPath := filepath.ToSlash(filepath.Join("landing-zones", "contoso-aks", "blueprint", "blueprint.auto.tfvars"))
	assert.Contains(t, content[tfvarsPath], `aks_kubernetes_version = "1.30"`)
	assert.Contains(t, content[tfvarsPath], `argocd_enabled         = true`)
	assert.Contains(t, content[tfvarsPath], `argocd_mode            = "extension"`)
}

func TestRenderBlueprint_AKSPlatform_InvalidArgoCD(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	_, err = engine.RenderBlueprint("contoso-aks", &config.Blueprint{
		Type: "aks-platform",
		Overrides: map[string]any{
			"argocd": map[string]any{
				"enabled": true,
				"mode":    "invalid-mode",
				"repoUrl": "https://github.com/org/app-repo",
			},
		},
	}, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "argocd.mode")
}

func TestRenderBlueprint_AKSPlatform_ArgoCD_MissingRepoURL(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	_, err = engine.RenderBlueprint("contoso-aks", &config.Blueprint{
		Type: "aks-platform",
		Overrides: map[string]any{
			"argocd": map[string]any{
				"enabled": true,
				"mode":    "extension",
				// repoUrl intentionally missing
			},
		},
	}, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "argocd.repoUrl")
}

func TestWriteAll_DryRun(t *testing.T) {
	writer := Writer{DryRun: true}
	dir := t.TempDir()
	files := []RenderedFile{{Path: "a/b.txt", Content: "hello"}}

	paths, err := writer.WriteAll(files, dir)
	require.NoError(t, err)
	assert.Len(t, paths, 1)

	_, statErr := os.Stat(filepath.Join(dir, "a", "b.txt"))
	assert.Error(t, statErr)
}

func TestWriteAll_Write(t *testing.T) {
	writer := Writer{DryRun: false}
	dir := t.TempDir()
	files := []RenderedFile{{Path: "a/b.txt", Content: "hello"}}

	paths, err := writer.WriteAll(files, dir)
	require.NoError(t, err)
	assert.Len(t, paths, 1)

	data, readErr := os.ReadFile(filepath.Join(dir, "a", "b.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "hello", string(data))
}

func TestRenderTests_Disabled(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	// testing is nil by default in sampleConfig
	files, err := engine.RenderTests(cfg)
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestRenderTests_DisabledExplicit(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.Testing = &config.Testing{Enabled: false}
	files, err := engine.RenderTests(cfg)
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestRenderTests_Enabled_NoAssertions(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.Testing = &config.Testing{Enabled: true}

	files, err := engine.RenderTests(cfg)
	require.NoError(t, err)
	// 5 platform layers, no landing zones
	assert.Len(t, files, 5)

	paths := map[string]string{}
	for _, f := range files {
		paths[f.Path] = f.Content
	}
	assert.Contains(t, paths, filepath.ToSlash(filepath.Join("platform", "management-groups", "testing.tftest.hcl")))
	assert.Contains(t, paths, filepath.ToSlash(filepath.Join("platform", "connectivity", "testing.tftest.hcl")))
	assert.Contains(t, paths, filepath.ToSlash(filepath.Join("platform", "management", "testing.tftest.hcl")))

	// Every file must contain the smoke test
	for _, f := range files {
		assert.Contains(t, f.Content, `run "smoke_plan"`, "file %s missing smoke_plan run block", f.Path)
		assert.Contains(t, f.Content, "command = plan")
	}
}

func TestRenderTests_Enabled_WithAssertions(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.Testing = &config.Testing{
		Enabled: true,
		Assertions: []config.TestAssertion{
			{
				Name:         "log_retention_90_days",
				Layer:        "management",
				Condition:    "azurerm_log_analytics_workspace.this.retention_in_days == 90",
				ErrorMessage: "Log Analytics retention must be 90 days",
			},
			{
				Name:         "smoke_all",
				Layer:        "*",
				Condition:    "true",
				ErrorMessage: "all layers must plan",
			},
		},
	}

	files, err := engine.RenderTests(cfg)
	require.NoError(t, err)
	assert.Len(t, files, 5)

	paths := map[string]string{}
	for _, f := range files {
		paths[f.Path] = f.Content
	}

	mgmtPath := filepath.ToSlash(filepath.Join("platform", "management", "testing.tftest.hcl"))
	mgmtContent := paths[mgmtPath]
	// management layer gets its specific assertion + the wildcard one
	assert.Contains(t, mgmtContent, `run "log_retention_90_days"`)
	assert.Contains(t, mgmtContent, "retention_in_days == 90")
	assert.Contains(t, mgmtContent, `run "smoke_all"`)

	// management-groups gets only the wildcard assertion
	mgPath := filepath.ToSlash(filepath.Join("platform", "management-groups", "testing.tftest.hcl"))
	assert.NotContains(t, paths[mgPath], `run "log_retention_90_days"`)
	assert.Contains(t, paths[mgPath], `run "smoke_all"`)
}

func TestRenderTests_Enabled_WithLandingZones(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.LandingZones = []config.LandingZone{
		{Name: "lz-corp", Archetype: "corp", AddressSpace: "10.1.0.0/16"},
		{Name: "lz-online", Archetype: "online", AddressSpace: "10.2.0.0/16"},
	}
	cfg.Spec.Testing = &config.Testing{
		Enabled: true,
		Assertions: []config.TestAssertion{
			{Name: "universal", Layer: "*", Condition: "true", ErrorMessage: "should pass"},
		},
	}

	files, err := engine.RenderTests(cfg)
	require.NoError(t, err)
	// 5 platform layers + 2 landing zones
	assert.Len(t, files, 7)

	paths := map[string]string{}
	for _, f := range files {
		paths[f.Path] = f.Content
	}
	assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "lz-corp", "testing.tftest.hcl")))
	assert.Contains(t, paths, filepath.ToSlash(filepath.Join("landing-zones", "lz-online", "testing.tftest.hcl")))

	lzContent := paths[filepath.ToSlash(filepath.Join("landing-zones", "lz-corp", "testing.tftest.hcl"))]
	assert.Contains(t, lzContent, "lz-corp")
	assert.Contains(t, lzContent, `run "smoke_plan"`)
}

func TestRenderAll_WithTesting_AddsTestFiles(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	cfg := sampleConfig()
	cfg.Spec.Testing = &config.Testing{Enabled: true}

	files, err := engine.RenderAll(cfg)
	require.NoError(t, err)

	// Base 22 files + 5 test files (no connectivity layer in none mode, but RenderTests always generates 5)
	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "management", "testing.tftest.hcl"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "connectivity", "testing.tftest.hcl"))])
	assert.True(t, paths[filepath.ToSlash(filepath.Join("platform", "management-groups", "testing.tftest.hcl"))])
}

func TestFilterAssertions(t *testing.T) {
	assertions := []config.TestAssertion{
		{Name: "a", Layer: "management"},
		{Name: "b", Layer: "*"},
		{Name: "c", Layer: "connectivity"},
	}

	t.Run("specific_layer", func(t *testing.T) {
		result := filterAssertions(assertions, "management")
		assert.Len(t, result, 2) // "a" + "b"
		assert.Equal(t, "a", result[0].Name)
		assert.Equal(t, "b", result[1].Name)
	})

	t.Run("wildcard_only", func(t *testing.T) {
		result := filterAssertions(assertions, "identity")
		assert.Len(t, result, 1) // only "b"
		assert.Equal(t, "b", result[0].Name)
	})

	t.Run("empty_assertions", func(t *testing.T) {
		result := filterAssertions(nil, "management")
		assert.Nil(t, result)
	})
}
