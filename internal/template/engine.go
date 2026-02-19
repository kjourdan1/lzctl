package template

import (
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
	}

	ctx := map[string]interface{}{
		"Config":  cfg,
		"Version": "v0.1.0-dev",
	}

	files := make([]RenderedFile, 0, len(templateToPath))
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
