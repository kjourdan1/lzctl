package wizard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kjourdan1/lzctl/internal/importer"
)

// ImportConfig captures all inputs collected by the import wizard.
type ImportConfig struct {
	// Source determines how resources are discovered.
	// Values: "audit-report", "subscription", "resource-group"
	Source string

	// AuditReportPath is set when Source == "audit-report".
	AuditReportPath string

	// Subscription is the target subscription ID (for subscription/rg modes).
	Subscription string

	// ResourceGroup is the target resource group (optional filter).
	ResourceGroup string

	// IncludeTypes limits discovery to these Azure resource types.
	IncludeTypes []string

	// ExcludeTypes excludes these Azure resource types from discovery.
	ExcludeTypes []string

	// SelectedResources contains the resources the user chose to import.
	SelectedResources []importer.ImportableResource

	// TargetDir is where generated files are written.
	TargetDir string

	// Layer overrides the default target directory with a specific layer.
	Layer string

	// DryRun previews output without writing files.
	DryRun bool
}

// ImportWizard drives the interactive import flow.
type ImportWizard struct {
	prompter Prompter
}

// NewImportWizard returns an import wizard; if p is nil, survey is used.
func NewImportWizard(p Prompter) *ImportWizard {
	if p == nil {
		p = NewSurveyPrompter()
	}
	return &ImportWizard{prompter: p}
}

// Run collects import configuration interactively.
func (w *ImportWizard) Run() (*ImportConfig, error) {
	cfg := &ImportConfig{}
	var err error

	cfg.Source, err = w.prompter.Select(
		"Import source",
		[]string{"subscription", "resource-group", "audit-report"},
		"subscription",
	)
	if err != nil {
		return nil, handleImportPromptErr(err)
	}

	switch cfg.Source {
	case "audit-report":
		cfg.AuditReportPath, err = w.prompter.Input(
			"Path to audit report JSON",
			"audit-report.json",
			survey.ComposeValidators(ValidateNonEmpty),
		)
		if err != nil {
			return nil, handleImportPromptErr(err)
		}

	case "resource-group":
		cfg.Subscription, err = w.prompter.Input(
			"Subscription ID (UUID)",
			"",
			survey.ComposeValidators(ValidateTenantID), // UUID format validation
		)
		if err != nil {
			return nil, handleImportPromptErr(err)
		}

		cfg.ResourceGroup, err = w.prompter.Input(
			"Resource group name",
			"",
			survey.ComposeValidators(ValidateNonEmpty),
		)
		if err != nil {
			return nil, handleImportPromptErr(err)
		}

	default: // subscription
		cfg.Subscription, err = w.prompter.Input(
			"Subscription ID (UUID)",
			"",
			survey.ComposeValidators(ValidateTenantID),
		)
		if err != nil {
			return nil, handleImportPromptErr(err)
		}
	}

	cfg.TargetDir = "imports"
	targetDir, err := w.prompter.Input(
		"Target directory for generated files",
		"imports",
		nil,
	)
	if err != nil {
		return nil, handleImportPromptErr(err)
	}
	if strings.TrimSpace(targetDir) != "" {
		cfg.TargetDir = strings.TrimSpace(targetDir)
	}

	return cfg, nil
}

// SelectResources presents discovered resources and lets the user pick which to import.
func (w *ImportWizard) SelectResources(resources []importer.ImportableResource) ([]importer.ImportableResource, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	options := make([]string, len(resources))
	for i, r := range resources {
		status := "✅"
		if !r.Supported {
			status = "⚠️  unsupported"
		}
		options[i] = fmt.Sprintf("[%s] %s — %s (%s)", status, r.Name, r.AzureType, r.TerraformType)
	}

	// Default: select all supported resources
	defaults := make([]string, 0)
	for i, r := range resources {
		if r.Supported {
			defaults = append(defaults, options[i])
		}
	}

	selected, err := w.prompter.MultiSelect(
		"Select resources to import",
		options,
		defaults,
	)
	if err != nil {
		return nil, handleImportPromptErr(err)
	}

	selectedSet := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedSet[s] = true
	}

	result := make([]importer.ImportableResource, 0, len(selected))
	for i, opt := range options {
		if selectedSet[opt] {
			result = append(result, resources[i])
		}
	}

	return result, nil
}

func handleImportPromptErr(err error) error {
	if errors.Is(err, ErrCancelled) {
		return fmt.Errorf("import wizard cancelled: %w", ErrCancelled)
	}
	return err
}
