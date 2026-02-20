package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kjourdan1/lzctl/internal/config"
)

// PipelineUpdater regenerates pipeline files when landing zones change.
type PipelineUpdater struct {
	engine *Engine
}

// NewPipelineUpdater creates a new pipeline updater.
func NewPipelineUpdater() (*PipelineUpdater, error) {
	engine, err := NewEngine()
	if err != nil {
		return nil, err
	}
	return &PipelineUpdater{engine: engine}, nil
}

// UpdatePipelines re-renders pipeline files based on the current config,
// including landing zone deploy steps for each zone.
func (u *PipelineUpdater) UpdatePipelines(cfg *config.LZConfig, repoRoot string) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	files, err := u.engine.RenderPipelines(cfg)
	if err != nil {
		return nil, fmt.Errorf("rendering pipelines: %w", err)
	}

	writer := Writer{DryRun: false}
	return writer.WriteAll(files, repoRoot)
}

// UpdatePipelinesDryRun previews which files would be updated.
func (u *PipelineUpdater) UpdatePipelinesDryRun(cfg *config.LZConfig, repoRoot string) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	files, err := u.engine.RenderPipelines(cfg)
	if err != nil {
		return nil, fmt.Errorf("rendering pipelines: %w", err)
	}

	writer := Writer{DryRun: true}
	return writer.WriteAll(files, repoRoot)
}

// RenderPipelines renders only the CI/CD pipeline templates from the config.
func (e *Engine) RenderPipelines(cfg *config.LZConfig) ([]RenderedFile, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	type templateMapping struct {
		TemplatePath string
		OutputPath   string
	}

	var mappings []templateMapping

	switch strings.ToLower(strings.TrimSpace(cfg.Spec.CICD.Platform)) {
	case "azure-devops", "azuredevops":
		mappings = append(mappings,
			templateMapping{"pipelines/azuredevops/validate.yml.tmpl", ".azuredevops/pipelines/validate.yml"},
			templateMapping{"pipelines/azuredevops/deploy.yml.tmpl", ".azuredevops/pipelines/deploy.yml"},
			templateMapping{"pipelines/azuredevops/drift.yml.tmpl", ".azuredevops/pipelines/drift.yml"},
		)
	default:
		mappings = append(mappings,
			templateMapping{"pipelines/github/validate.yml.tmpl", ".github/workflows/validate.yml"},
			templateMapping{"pipelines/github/deploy.yml.tmpl", ".github/workflows/deploy.yml"},
			templateMapping{"pipelines/github/drift.yml.tmpl", ".github/workflows/drift.yml"},
		)
	}

	ctx := map[string]interface{}{
		"Config":  cfg,
		"Version": "v0.1.0-dev",
	}

	files := make([]RenderedFile, 0, len(mappings))
	for _, m := range mappings {
		content, err := e.renderTemplate(m.TemplatePath, ctx)
		if err != nil {
			return nil, err
		}
		files = append(files, RenderedFile{
			Path:    m.OutputPath,
			Content: content,
		})
	}

	return files, nil
}

// GenerateZoneMatrix creates a YAML matrix include snippet for landing zone
// pipeline steps. Blueprint layers are appended after their parent zone entry
// so that the CI/CD matrix respects the dependency order (LZ → blueprint).
func GenerateZoneMatrix(cfg *config.LZConfig) string {
	if cfg == nil || len(cfg.Spec.LandingZones) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[")
	first := true
	for _, zone := range cfg.Spec.LandingZones {
		if !first {
			sb.WriteString(", ")
		}
		slug := Slugify(zone.Name)
		sb.WriteString(fmt.Sprintf(`{"name": %q, "dir": "landing-zones/%s", "archetype": %q}`,
			zone.Name, slug, zone.Archetype))
		first = false
		// Blueprint layer always follows its parent landing zone.
		if zone.Blueprint != nil {
			sb.WriteString(", ")
			sb.WriteString(fmt.Sprintf(
				`{"name": %q, "dir": "landing-zones/%s/blueprint", "archetype": "blueprint", "blueprintType": %q}`,
				zone.Name+"-blueprint", slug, zone.Blueprint.Type))
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// LandingZoneDirs returns the ordered list of landing-zone and blueprint
// directories for use in pipeline shell loops. Each zone dir is followed
// immediately by its blueprint dir when a blueprint is configured.
func LandingZoneDirs(cfg *config.LZConfig) []string {
	dirs := make([]string, 0, len(cfg.Spec.LandingZones)*2)
	for _, zone := range cfg.Spec.LandingZones {
		slug := Slugify(zone.Name)
		dirs = append(dirs, "landing-zones/"+slug)
		if zone.Blueprint != nil {
			dirs = append(dirs, "landing-zones/"+slug+"/blueprint")
		}
	}
	return dirs
}

// WriteLandingZoneMatrix writes a zone-matrix.json file that CI/CD pipelines
// can consume for dynamic matrix generation.
func WriteLandingZoneMatrix(cfg *config.LZConfig, repoRoot string) (string, error) {
	matrix := GenerateZoneMatrix(cfg)
	outPath := filepath.Join(repoRoot, ".lzctl", "zone-matrix.json")

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", fmt.Errorf("creating .lzctl directory: %w", err)
	}

	if err := os.WriteFile(outPath, []byte(matrix+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("writing zone matrix: %w", err)
	}

	return outPath, nil
}
