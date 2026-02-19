package template

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	texttemplate "text/template"

	"github.com/kjourdan1/lzctl/internal/config"
	templatefs "github.com/kjourdan1/lzctl/templates"
)

// RenderedFile is a rendered output artifact.
type RenderedFile struct {
	Path    string
	Content string
}

// Engine renders config-driven templates into files.
type Engine struct {
	funcMap texttemplate.FuncMap
}

// NewEngine creates a new template engine with helper functions.
func NewEngine() (*Engine, error) {
	return &Engine{funcMap: HelperFuncMap()}, nil
}

// RenderAll renders core templates for sprint-2 and sprint-3 foundations.
func (e *Engine) RenderAll(cfg *config.LZConfig) ([]RenderedFile, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	templateToPath := []struct {
		TemplatePath string
		OutputPath   string
	}{
		{TemplatePath: "manifest/lzctl.yaml.tmpl", OutputPath: "lzctl.yaml"},
		{TemplatePath: "shared/backend.tf.tmpl", OutputPath: filepath.ToSlash(filepath.Join("platform", "shared", "backend.tf"))},
		{TemplatePath: "shared/backend.hcl.tmpl", OutputPath: filepath.ToSlash(filepath.Join("platform", "shared", "backend.hcl"))},
		{TemplatePath: "shared/providers.tf.tmpl", OutputPath: filepath.ToSlash(filepath.Join("platform", "shared", "providers.tf"))},
		{TemplatePath: "shared/gitignore.tmpl", OutputPath: ".gitignore"},
		{TemplatePath: "shared/readme.md.tmpl", OutputPath: "README.md"},
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Spec.CICD.Platform)) {
	case "azure-devops", "azuredevops":
		templateToPath = append(templateToPath,
			struct{ TemplatePath, OutputPath string }{TemplatePath: "pipelines/azuredevops/validate.yml.tmpl", OutputPath: ".azuredevops/pipelines/validate.yml"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "pipelines/azuredevops/deploy.yml.tmpl", OutputPath: ".azuredevops/pipelines/deploy.yml"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "pipelines/azuredevops/drift.yml.tmpl", OutputPath: ".azuredevops/pipelines/drift.yml"},
		)
	default:
		templateToPath = append(templateToPath,
			struct{ TemplatePath, OutputPath string }{TemplatePath: "pipelines/github/validate.yml.tmpl", OutputPath: ".github/workflows/validate.yml"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "pipelines/github/deploy.yml.tmpl", OutputPath: ".github/workflows/deploy.yml"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "pipelines/github/drift.yml.tmpl", OutputPath: ".github/workflows/drift.yml"},
		)
	}

	if strings.EqualFold(cfg.Spec.Platform.ManagementGroups.Model, "caf-lite") {
		templateToPath = append(templateToPath,
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management-groups/caf-lite/main.tf.tmpl", OutputPath: "platform/management-groups/main.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management-groups/caf-lite/variables.tf.tmpl", OutputPath: "platform/management-groups/variables.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management-groups/caf-lite/terraform.tfvars.tmpl", OutputPath: "platform/management-groups/terraform.tfvars"},
		)
	} else {
		templateToPath = append(templateToPath,
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management-groups/caf-standard/main.tf.tmpl", OutputPath: "platform/management-groups/main.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management-groups/caf-standard/variables.tf.tmpl", OutputPath: "platform/management-groups/variables.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management-groups/caf-standard/terraform.tfvars.tmpl", OutputPath: "platform/management-groups/terraform.tfvars"},
		)
	}

	templateToPath = append(templateToPath,
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management/main.tf.tmpl", OutputPath: "platform/management/main.tf"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management/variables.tf.tmpl", OutputPath: "platform/management/variables.tf"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/management/terraform.tfvars.tmpl", OutputPath: "platform/management/terraform.tfvars"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/governance/main.tf.tmpl", OutputPath: "platform/governance/main.tf"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/governance/variables.tf.tmpl", OutputPath: "platform/governance/variables.tf"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/governance/terraform.tfvars.tmpl", OutputPath: "platform/governance/terraform.tfvars"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/governance/policies/caf-default.tf.tmpl", OutputPath: "platform/governance/policies/caf-default.tf"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/identity/main.tf.tmpl", OutputPath: "platform/identity/main.tf"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/identity/variables.tf.tmpl", OutputPath: "platform/identity/variables.tf"},
		struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/identity/terraform.tfvars.tmpl", OutputPath: "platform/identity/terraform.tfvars"},
	)

	switch strings.ToLower(cfg.Spec.Platform.Connectivity.Type) {
	case "none":
		// No connectivity templates in no-connectivity mode.
	case "vwan":
		templateToPath = append(templateToPath,
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/vwan/main.tf.tmpl", OutputPath: "platform/connectivity/main.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/vwan/variables.tf.tmpl", OutputPath: "platform/connectivity/variables.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/vwan/terraform.tfvars.tmpl", OutputPath: "platform/connectivity/terraform.tfvars"},
		)
	default:
		useFW := cfg.Spec.Platform.Connectivity.Hub != nil && cfg.Spec.Platform.Connectivity.Hub.Firewall.Enabled
		if useFW {
			templateToPath = append(templateToPath,
				struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/hub-spoke-fw/main.tf.tmpl", OutputPath: "platform/connectivity/main.tf"},
				struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/hub-spoke-fw/variables.tf.tmpl", OutputPath: "platform/connectivity/variables.tf"},
				struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/hub-spoke-fw/terraform.tfvars.tmpl", OutputPath: "platform/connectivity/terraform.tfvars"},
			)
		} else {
			templateToPath = append(templateToPath,
				struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/hub-spoke-nva/main.tf.tmpl", OutputPath: "platform/connectivity/main.tf"},
				struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/hub-spoke-nva/variables.tf.tmpl", OutputPath: "platform/connectivity/variables.tf"},
				struct{ TemplatePath, OutputPath string }{TemplatePath: "platform/connectivity/hub-spoke-nva/terraform.tfvars.tmpl", OutputPath: "platform/connectivity/terraform.tfvars"},
			)
		}
	}

	files := make([]RenderedFile, 0, len(templateToPath)+(len(cfg.Spec.LandingZones)*4))

	for _, zone := range cfg.Spec.LandingZones {
		archetype := strings.ToLower(strings.TrimSpace(zone.Archetype))
		if archetype == "" {
			archetype = "corp"
		}
		baseOut := filepath.ToSlash(filepath.Join("landing-zones", Slugify(zone.Name)))
		baseTpl := filepath.ToSlash(filepath.Join("landing-zones", archetype))

		templateToPath = append(templateToPath,
			struct{ TemplatePath, OutputPath string }{TemplatePath: baseTpl + "/main.tf.tmpl", OutputPath: baseOut + "/main.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: baseTpl + "/variables.tf.tmpl", OutputPath: baseOut + "/variables.tf"},
			struct{ TemplatePath, OutputPath string }{TemplatePath: baseTpl + "/terraform.tfvars.tmpl", OutputPath: baseOut + "/terraform.tfvars"},
		)

		if zone.Blueprint != nil {
			blueprintFiles, bpErr := e.RenderBlueprint(zone.Name, zone.Blueprint, cfg)
			if bpErr != nil {
				return nil, bpErr
			}
			files = append(files, blueprintFiles...)
		}
	}

	ctx := map[string]interface{}{
		"Config":  cfg,
		"Version": "v0.1.0-dev",
	}

	for _, item := range templateToPath {
		var sb strings.Builder
		t, err := texttemplate.New(path.Base(item.TemplatePath)).Funcs(e.funcMap).ParseFS(templatefs.FS, item.TemplatePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", item.TemplatePath, err)
		}
		renderCtx := ctx
		if strings.HasPrefix(item.TemplatePath, "landing-zones/") {
			zoneName := filepath.Base(filepath.Dir(item.OutputPath))
			for _, zone := range cfg.Spec.LandingZones {
				if Slugify(zone.Name) == zoneName {
					renderCtx = map[string]interface{}{
						"Config":  cfg,
						"Version": "v0.1.0-dev",
						"Zone":    zone,
					}
					break
				}
			}
		}

		if err := t.ExecuteTemplate(&sb, path.Base(item.TemplatePath), renderCtx); err != nil {
			return nil, fmt.Errorf("rendering %s: %w", item.TemplatePath, err)
		}
		files = append(files, RenderedFile{
			Path:    item.OutputPath,
			Content: sb.String(),
		})
	}

	return files, nil
}

// RenderBlueprint renders Terraform files for a landing-zone blueprint.
// The layer argument corresponds to the landing-zone name.
func (e *Engine) RenderBlueprint(layer string, blueprint *config.Blueprint, cfg *config.LZConfig) ([]RenderedFile, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if blueprint == nil {
		return nil, fmt.Errorf("blueprint cannot be nil")
	}

	zoneName := strings.TrimSpace(layer)
	if zoneName == "" {
		return nil, fmt.Errorf("blueprint target landing zone is required")
	}

	blueprintType := strings.ToLower(strings.TrimSpace(blueprint.Type))
	if blueprintType == "" {
		return nil, fmt.Errorf("blueprint type is required")
	}

	baseDir := filepath.ToSlash(filepath.Join("landing-zones", Slugify(zoneName), "blueprint"))

	switch blueprintType {
	case "paas-secure":
		mainTF := renderPaasSecureBlueprintMainTF(cfg, zoneName)
		variablesTF := renderPaasSecureBlueprintVariablesTF()
		tfvars, err := renderPaasSecureBlueprintTFVars(blueprint.Overrides)
		if err != nil {
			return nil, err
		}
		backendHCL := renderBlueprintBackendHCL(cfg, zoneName)

		return []RenderedFile{
			{Path: filepath.ToSlash(filepath.Join(baseDir, "main.tf")), Content: mainTF},
			{Path: filepath.ToSlash(filepath.Join(baseDir, "variables.tf")), Content: variablesTF},
			{Path: filepath.ToSlash(filepath.Join(baseDir, "blueprint.auto.tfvars")), Content: tfvars},
			{Path: filepath.ToSlash(filepath.Join(baseDir, "backend.hcl")), Content: backendHCL},
		}, nil

	case "aks-platform":
		return renderAKSPlatformBlueprint(baseDir, zoneName, blueprint, cfg)

	case "aca-platform", "avd-secure":
		return nil, fmt.Errorf("blueprint type %q is not implemented yet", blueprintType)
	default:
		return nil, fmt.Errorf("unsupported blueprint type %q", blueprintType)
	}
}

func renderPaasSecureBlueprintMainTF(cfg *config.LZConfig, zoneName string) string {
	connectivityState := ConnectivityRemoteState(cfg)
	managementState := fmt.Sprintf(`data "terraform_remote_state" "management" {
  backend = "azurerm"
  config = {
    resource_group_name  = %q
    storage_account_name = %q
    container_name       = %q
    key                  = "platform-management.tfstate"
    subscription_id      = %q
    use_azuread_auth     = true
  }
}
`, cfg.Spec.StateBackend.ResourceGroup, cfg.Spec.StateBackend.StorageAccount, cfg.Spec.StateBackend.Container, cfg.Spec.StateBackend.Subscription)

	return fmt.Sprintf(`# Generated by lzctl blueprint catalog (paas-secure)
terraform {
  required_version = ">= 1.5.0"
}

%s
%s

data "azurerm_private_dns_zone" "azurewebsites" {
  name                = %q
  resource_group_name = data.terraform_remote_state.connectivity.outputs.private_dns_resource_group_name
}

data "azurerm_private_dns_zone" "vault" {
  name                = %q
  resource_group_name = data.terraform_remote_state.connectivity.outputs.private_dns_resource_group_name
}

module "key_vault" {
  source  = "Azure/avm-res-keyvault-vault/azurerm"
  version = "0.9.0"

  name                              = "kv-%s"
  location                          = var.location
  resource_group_name               = var.workload_resource_group_name
  public_network_access_enabled     = false
  soft_delete_retention_days        = var.keyvault_soft_delete_retention_days
}

module "function_app" {
  source  = "Azure/avm-ptn-function-app-storage-private-endpoints/azurerm"
  version = "0.2.0"

  location            = var.location
  resource_group_name = var.workload_resource_group_name
  app_service_plan_sku = var.appservice_sku
  runtime_stack       = var.appservice_runtime_stack
  public_network_access_enabled = false
}

module "apim" {
  count   = var.apim_enabled ? 1 : 0
  source  = "Azure/avm-res-apimanagement-service/azurerm"
  version = "0.4.0"

  name                          = "apim-%s"
  location                      = var.location
  resource_group_name           = var.workload_resource_group_name
  sku_name                      = var.apim_sku
  public_network_access_enabled = false
}

resource "azurerm_private_endpoint" "function_app" {
  name                = "pe-func-%s"
  location            = var.location
  resource_group_name = var.workload_resource_group_name
  subnet_id           = data.terraform_remote_state.connectivity.outputs.workload_private_endpoint_subnet_id

  private_service_connection {
    name                           = "func-private-link"
    private_connection_resource_id = module.function_app.function_app_id
    subresource_names              = ["sites"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "default"
    private_dns_zone_ids = [data.azurerm_private_dns_zone.azurewebsites.id]
  }
}

resource "azurerm_private_endpoint" "key_vault" {
  name                = "pe-kv-%s"
  location            = var.location
  resource_group_name = var.workload_resource_group_name
  subnet_id           = data.terraform_remote_state.connectivity.outputs.workload_private_endpoint_subnet_id

  private_service_connection {
    name                           = "kv-private-link"
    private_connection_resource_id = module.key_vault.resource_id
    subresource_names              = ["vault"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "default"
    private_dns_zone_ids = [data.azurerm_private_dns_zone.vault.id]
  }
}

output "workload_resource_group_id" {
  value = data.terraform_remote_state.connectivity.outputs.workload_resource_group_id
}

output "vnet_id" {
  value = data.terraform_remote_state.connectivity.outputs.vnet_id
}

output "key_vault_id" {
  value = module.key_vault.resource_id
}
`, connectivityState, managementState, DNSZoneRef("appservice"), DNSZoneRef("keyvault"), Slugify(zoneName), Slugify(zoneName), Slugify(zoneName), Slugify(zoneName))
}

func renderPaasSecureBlueprintVariablesTF() string {
	return `variable "location" {
  type        = string
  description = "Azure region for workload resources"
}

variable "workload_resource_group_name" {
  type        = string
  description = "Workload resource group name"
}

variable "appservice_sku" {
  type        = string
  default     = "P1v3"
}

variable "appservice_runtime_stack" {
  type        = string
  default     = "DOTNET|8.0"
}

variable "apim_enabled" {
  type        = bool
  default     = true
}

variable "apim_sku" {
  type        = string
  default     = "Developer_1"
}

variable "keyvault_soft_delete_retention_days" {
  type        = number
  default     = 90
}
`
}

func renderPaasSecureBlueprintTFVars(overrides map[string]any) (string, error) {
	appServiceSKU := "P1v3"
	runtimeStack := "DOTNET|8.0"
	apimEnabled := true
	apimSKU := "Developer_1"
	keyvaultRetention := 90

	if appService := asStringMap(overrides, "appService"); appService != nil {
		if v, ok := appService["sku"].(string); ok && strings.TrimSpace(v) != "" {
			appServiceSKU = strings.TrimSpace(v)
		}
		if v, ok := appService["runtimeStack"].(string); ok && strings.TrimSpace(v) != "" {
			runtimeStack = strings.TrimSpace(v)
		}
	}

	if apim := asStringMap(overrides, "apim"); apim != nil {
		if v, ok := apim["enabled"].(bool); ok {
			apimEnabled = v
		}
		if v, ok := apim["sku"].(string); ok && strings.TrimSpace(v) != "" {
			apimSKU = strings.TrimSpace(v)
		}
	}

	if kv := asStringMap(overrides, "keyVault"); kv != nil {
		switch v := kv["softDeleteRetentionDays"].(type) {
		case int:
			keyvaultRetention = v
		case int64:
			keyvaultRetention = int(v)
		case float64:
			keyvaultRetention = int(v)
		}
	}

	return fmt.Sprintf(`appservice_sku = %q
appservice_runtime_stack = %q
apim_enabled = %t
apim_sku = %q
keyvault_soft_delete_retention_days = %d
`, appServiceSKU, runtimeStack, apimEnabled, apimSKU, keyvaultRetention), nil
}

func renderBlueprintBackendHCL(cfg *config.LZConfig, zoneName string) string {
	return fmt.Sprintf(`resource_group_name  = %q
storage_account_name = %q
container_name       = %q
key                  = %q
subscription_id      = %q
use_azuread_auth     = true
`, cfg.Spec.StateBackend.ResourceGroup, cfg.Spec.StateBackend.StorageAccount, cfg.Spec.StateBackend.Container, "landing-zones-"+Slugify(zoneName)+"-blueprint.tfstate", cfg.Spec.StateBackend.Subscription)
}

func asStringMap(overrides map[string]any, key string) map[string]any {
	if overrides == nil {
		return nil
	}
	v, ok := overrides[key]
	if !ok {
		return nil
	}
	if result, ok := v.(map[any]any); ok {
		normalized := make(map[string]any, len(result))
		for k, value := range result {
			normalized[fmt.Sprint(k)] = value
		}
		return normalized
	}
	if result, ok := v.(map[string]any); ok {
		return result
	}
	return nil
}

func blueprintOverridesJSON(overrides map[string]any) string {
	if overrides == nil {
		return "{}"
	}
	b, err := json.Marshal(overrides)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// RenderZone renders templates for a single landing zone only.
func (e *Engine) RenderZone(cfg *config.LZConfig, zone config.LandingZone) ([]RenderedFile, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	archetype := strings.ToLower(strings.TrimSpace(zone.Archetype))
	if archetype == "" {
		archetype = "corp"
	}

	baseOut := filepath.ToSlash(filepath.Join("landing-zones", Slugify(zone.Name)))
	baseTpl := filepath.ToSlash(filepath.Join("landing-zones", archetype))

	templateToPath := []struct {
		TemplatePath string
		OutputPath   string
	}{
		{TemplatePath: baseTpl + "/main.tf.tmpl", OutputPath: baseOut + "/main.tf"},
		{TemplatePath: baseTpl + "/variables.tf.tmpl", OutputPath: baseOut + "/variables.tf"},
		{TemplatePath: baseTpl + "/terraform.tfvars.tmpl", OutputPath: baseOut + "/terraform.tfvars"},
	}

	ctx := map[string]interface{}{
		"Config":  cfg,
		"Version": "v0.1.0-dev",
		"Zone":    zone,
	}

	files := make([]RenderedFile, 0, len(templateToPath))
	for _, item := range templateToPath {
		var sb strings.Builder
		t, err := texttemplate.New(path.Base(item.TemplatePath)).Funcs(e.funcMap).ParseFS(templatefs.FS, item.TemplatePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", item.TemplatePath, err)
		}
		if err := t.ExecuteTemplate(&sb, path.Base(item.TemplatePath), ctx); err != nil {
			return nil, fmt.Errorf("rendering %s: %w", item.TemplatePath, err)
		}
		files = append(files, RenderedFile{
			Path:    item.OutputPath,
			Content: sb.String(),
		})
	}

	return files, nil
}

// renderTemplate renders a single template with the given context.
func (e *Engine) renderTemplate(templatePath string, ctx map[string]interface{}) (string, error) {
	var sb strings.Builder
	t, err := texttemplate.New(path.Base(templatePath)).Funcs(e.funcMap).ParseFS(templatefs.FS, templatePath)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", templatePath, err)
	}
	if err := t.ExecuteTemplate(&sb, path.Base(templatePath), ctx); err != nil {
		return "", fmt.Errorf("rendering %s: %w", templatePath, err)
	}
	return sb.String(), nil
}

// ─── aks-platform blueprint rendering ────────────────────────────────────────

// renderAKSPlatformBlueprint generates all Terraform and supporting files for
// the aks-platform blueprint (E9-S2 + E9-S4 + E9-S5).
func renderAKSPlatformBlueprint(baseDir, zoneName string, blueprint *config.Blueprint, cfg *config.LZConfig) ([]RenderedFile, error) {
	aksCfg, err := config.ParseAKSBlueprintConfig(blueprint.Overrides)
	if err != nil {
		return nil, fmt.Errorf("aks-platform: parsing overrides: %w", err)
	}
	if err := config.ValidateArgoCDConfig(aksCfg.ArgoCD); err != nil {
		return nil, fmt.Errorf("aks-platform: argocd config: %w", err)
	}

	// Secure-by-default: defender is enabled unless overrides explicitly set it false.
	defenderDefault := true
	if defOvr, ok := blueprint.Overrides["defender"]; ok {
		if defMap, ok2 := defOvr.(map[string]any); ok2 {
			if v, ok3 := defMap["enabled"].(bool); ok3 {
				defenderDefault = v
			}
		}
	}

	mainTF := renderAKSPlatformMainTF(cfg, zoneName, aksCfg)
	variablesTF := renderAKSPlatformVariablesTF()
	tfvars := renderAKSPlatformTFVars(aksCfg, defenderDefault)
	backendHCL := renderBlueprintBackendHCL(cfg, zoneName)
	makefile := renderAKSPlatformMakefile(cfg, zoneName)

	files := []RenderedFile{
		{Path: filepath.ToSlash(filepath.Join(baseDir, "main.tf")), Content: mainTF},
		{Path: filepath.ToSlash(filepath.Join(baseDir, "variables.tf")), Content: variablesTF},
		{Path: filepath.ToSlash(filepath.Join(baseDir, "blueprint.auto.tfvars")), Content: tfvars},
		{Path: filepath.ToSlash(filepath.Join(baseDir, "backend.hcl")), Content: backendHCL},
		{Path: filepath.ToSlash(filepath.Join(baseDir, "Makefile")), Content: makefile},
	}

	// E9-S4: ApplicationSet manifest when ArgoCD is enabled.
	if aksCfg.ArgoCD.Enabled && aksCfg.ArgoCD.RepoURL != "" {
		appSet := renderArgoCDAppSet(zoneName, aksCfg.ArgoCD)
		files = append(files, RenderedFile{
			Path:    filepath.ToSlash(filepath.Join(baseDir, "argocd", "appset.yaml")),
			Content: appSet,
		})
	}

	return files, nil
}

func renderAKSPlatformMainTF(cfg *config.LZConfig, zoneName string, aksCfg config.AKSBlueprintConfig) string {
	slug := Slugify(zoneName)
	connectivityState := ConnectivityRemoteState(cfg)

	acrSKU := "Premium"
	if aksCfg.ACR.SKU != "" {
		acrSKU = aksCfg.ACR.SKU
	}

	argoBlock := ""
	if aksCfg.ArgoCD.Enabled {
		argoBlock = renderArgoCDTerraformBlock(slug, aksCfg.ArgoCD)
	}

	managementState := fmt.Sprintf(`data "terraform_remote_state" "management" {
  backend = "azurerm"
  config = {
    resource_group_name  = %q
    storage_account_name = %q
    container_name       = %q
    key                  = "platform-management.tfstate"
    subscription_id      = %q
    use_azuread_auth     = true
  }
}
`, cfg.Spec.StateBackend.ResourceGroup, cfg.Spec.StateBackend.StorageAccount, cfg.Spec.StateBackend.Container, cfg.Spec.StateBackend.Subscription)

	return fmt.Sprintf(`# Generated by lzctl blueprint catalog (aks-platform)
# secure-by-default: private cluster, Defender for Containers, Azure Policy add-on
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.80"
    }
  }
}

%s
%s

# ── Private DNS zones (centralized in platform/connectivity) ──────────────────
data "azurerm_private_dns_zone" "acr" {
  name                = %q
  resource_group_name = data.terraform_remote_state.connectivity.outputs.private_dns_resource_group_name
}

data "azurerm_private_dns_zone" "vault" {
  name                = %q
  resource_group_name = data.terraform_remote_state.connectivity.outputs.private_dns_resource_group_name
}

# ── AKS cluster (AVM — private, Defender, OIDC/WIF) ──────────────────────────
module "aks" {
  source  = "Azure/avm-ptn-aks-production/azurerm"
  version = "0.1.0"

  name                = "aks-%s"
  location            = var.location
  resource_group_name = var.workload_resource_group_name

  kubernetes_version             = var.aks_kubernetes_version
  private_cluster_enabled        = true   # secure-by-default
  azure_policy_enabled           = true   # governance guardrails
  oidc_issuer_enabled            = true   # WIF support
  workload_identity_enabled      = true
  microsoft_defender_enabled     = var.defender_enabled
}

# ── Container Registry (AVM — Premium + Private Endpoint) ────────────────────
module "acr" {
  source  = "Azure/avm-res-containerregistry-registry/azurerm"
  version = "0.2.0"

  name                          = "acr%s"
  location                      = var.location
  resource_group_name           = var.workload_resource_group_name
  sku                           = %q
  public_network_access_enabled = false
}

resource "azurerm_private_endpoint" "acr" {
  name                = "pe-acr-%s"
  location            = var.location
  resource_group_name = var.workload_resource_group_name
  subnet_id           = data.terraform_remote_state.connectivity.outputs.workload_private_endpoint_subnet_id

  private_service_connection {
    name                           = "acr-private-link"
    private_connection_resource_id = module.acr.resource_id
    subresource_names              = ["registry"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "default"
    private_dns_zone_ids = [data.azurerm_private_dns_zone.acr.id]
  }
}

# ── Key Vault (AVM — soft delete, no public access) ──────────────────────────
module "key_vault" {
  source  = "Azure/avm-res-keyvault-vault/azurerm"
  version = "0.9.0"

  name                          = "kv-%s"
  location                      = var.location
  resource_group_name           = var.workload_resource_group_name
  public_network_access_enabled = false
  soft_delete_retention_days    = 90
}

resource "azurerm_private_endpoint" "key_vault" {
  name                = "pe-kv-%s"
  location            = var.location
  resource_group_name = var.workload_resource_group_name
  subnet_id           = data.terraform_remote_state.connectivity.outputs.workload_private_endpoint_subnet_id

  private_service_connection {
    name                           = "kv-private-link"
    private_connection_resource_id = module.key_vault.resource_id
    subresource_names              = ["vault"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "default"
    private_dns_zone_ids = [data.azurerm_private_dns_zone.vault.id]
  }
}
%s
# ── Mandatory outputs ─────────────────────────────────────────────────────────
output "workload_resource_group_id" {
  value = data.terraform_remote_state.connectivity.outputs.workload_resource_group_id
}

output "aks_cluster_id" {
  value = module.aks.resource_id
}

output "aks_oidc_issuer_url" {
  description = "OIDC issuer URL — used by bootstrap to create federated credentials (E9-S3)"
  value       = module.aks.oidc_issuer_url
}

output "acr_login_server" {
  value = module.acr.login_server
}

output "key_vault_id" {
  value = module.key_vault.resource_id
}
`,
		connectivityState, managementState,
		DNSZoneRef("acr"), DNSZoneRef("keyvault"),
		slug, slug, acrSKU, slug, slug, slug,
		argoBlock,
	)
}

// renderArgoCDTerraformBlock generates the conditional ArgoCD extension or helm
// resource block (E9-S2).
func renderArgoCDTerraformBlock(slug string, argocd config.ArgoCDConfig) string {
	return fmt.Sprintf(`
# ── ArgoCD (opt-in — E9-S2) ───────────────────────────────────────────────────
# Mode "extension" uses the Microsoft Flux GitOps Arc extension (recommended).
# Mode "helm" installs the upstream Argo CD helm chart (more version control).
resource "azurerm_kubernetes_cluster_extension" "argocd" {
  count      = var.argocd_enabled && var.argocd_mode == "extension" ? 1 : 0
  name       = "argocd"
  cluster_id = module.aks.resource_id

  extension_type    = "microsoft.flux"
  release_train     = "stable"
  release_namespace = "argocd"
  configuration_settings = {
    "helm.versions" = "v3"
  }
}

resource "helm_release" "argocd" {
  count      = var.argocd_enabled && var.argocd_mode == "helm" ? 1 : 0
  name       = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = var.argocd_chart_version
  namespace  = "argocd"
  create_namespace = true

  # secure-by-default: no public LoadBalancer
  set { name = "server.service.type"; value = "ClusterIP" }
  set { name = "configs.params.server\\.insecure"; value = "false" }
}
`)
}

func renderAKSPlatformVariablesTF() string {
	return `variable "location" {
  type        = string
  description = "Azure region for workload resources"
}

variable "workload_resource_group_name" {
  type        = string
  description = "Workload resource group name"
}

variable "aks_kubernetes_version" {
  type    = string
  default = "1.30"
}

variable "acr_sku" {
  type    = string
  default = "Premium"
}

variable "defender_enabled" {
  type    = bool
  default = true
}

variable "argocd_enabled" {
  type    = bool
  default = false
}

variable "argocd_mode" {
  type    = string
  default = "extension" # "extension" | "helm"
}

variable "argocd_chart_version" {
  type    = string
  default = "6.7.3" # pinned — bump via lzctl upgrade (E9-S8)
}
`
}

func renderAKSPlatformTFVars(aksCfg config.AKSBlueprintConfig, defenderDefault bool) string {
	aksVer := "1.30"
	if aksCfg.AKS.Version != "" {
		aksVer = aksCfg.AKS.Version
	}
	acrSKU := "Premium"
	if aksCfg.ACR.SKU != "" {
		acrSKU = aksCfg.ACR.SKU
	}
	// Secure-by-default: defender enabled unless caller explicitly disabled it.
	defenderEnabled := defenderDefault

	argoEnabled := aksCfg.ArgoCD.Enabled
	argoMode := aksCfg.ArgoCD.ArgoCDMode()

	return fmt.Sprintf(`aks_kubernetes_version = %q
acr_sku                = %q
defender_enabled       = %t
argocd_enabled         = %t
argocd_mode            = %q
`, aksVer, acrSKU, defenderEnabled, argoEnabled, argoMode)
}

// renderArgoCDAppSet generates the ApplicationSet manifest for E9-S4.
func renderArgoCDAppSet(zoneName string, argocd config.ArgoCDConfig) string {
	slug := Slugify(zoneName)
	targetRevision := argocd.EffectiveTargetRevision()
	appPath := argocd.EffectiveAppPath()

	return fmt.Sprintf(`# Generated by lzctl blueprint catalog (aks-platform — ArgoCD ApplicationSet E9-S4)
# Safe to edit: lzctl will not overwrite this file unless --overwrite is passed.
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: %s-apps
  namespace: argocd
spec:
  generators:
    - git:
        repoURL: %s
        revision: %s
        directories:
          - path: %s*
  template:
    metadata:
      name: "{{path.basename}}"
    spec:
      project: default
      source:
        repoURL: %s
        targetRevision: %s
        path: "{{path}}"
      destination:
        server: https://kubernetes.default.svc
        namespace: "{{path.basename}}"
      syncPolicy:
        automated:
          prune: true
          selfHeal: true      # secure-by-default: drift auto-corrected
        syncOptions:
          - CreateNamespace=true
          - ServerSideApply=true
`, slug, argocd.RepoURL, targetRevision, appPath, argocd.RepoURL, targetRevision)
}

// renderAKSPlatformMakefile generates the helper Makefile for E9-S5 (ArgoCD UI
// access on a private cluster).
func renderAKSPlatformMakefile(cfg *config.LZConfig, zoneName string) string {
	slug := Slugify(zoneName)
	region := cfg.Metadata.PrimaryRegion

	return fmt.Sprintf(`# Generated by lzctl blueprint catalog (aks-platform — E9-S5)
# Provides helper targets for ArgoCD UI access on a private cluster.
# Safe to edit: lzctl will not overwrite this file unless --overwrite is passed.

AKS_NAME   ?= aks-%s
RG         ?= $(shell grep workload_resource_group_name blueprint.auto.tfvars | awk -F'"' '{print $$2}')
REGION     ?= %s
ARGOCD_NS  ?= argocd
ARGOCD_PORT ?= 8080

.PHONY: argocd-login argocd-sync-all argocd-status aks-credentials

## argocd-login: port-forward ArgoCD UI on localhost:$(ARGOCD_PORT)
argocd-login: aks-credentials
	kubectl port-forward svc/argocd-server -n $(ARGOCD_NS) $(ARGOCD_PORT):443 &
	@sleep 2
	argocd login localhost:$(ARGOCD_PORT) --username admin \
		--password $$(kubectl get secret argocd-initial-admin-secret \
		  -n $(ARGOCD_NS) -o jsonpath="{.data.password}" | base64 -d) \
		--insecure

## argocd-sync-all: force-sync all ArgoCD applications
argocd-sync-all: aks-credentials
	argocd app list -o name | xargs -I{} argocd app sync {}

## argocd-status: show sync + health status for all applications
argocd-status: aks-credentials
	argocd app list

## aks-credentials: fetch kubeconfig for the private AKS cluster
aks-credentials:
	az aks get-credentials \
		--name $(AKS_NAME) \
		--resource-group $(RG) \
		--overwrite-existing
`, slug, region)
}
