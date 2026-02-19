package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/audit"
	"github.com/kjourdan1/lzctl/internal/azure"
	"github.com/kjourdan1/lzctl/internal/importer"
	"github.com/kjourdan1/lzctl/internal/wizard"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Generate Terraform import blocks for existing Azure resources",
	Long: `Discovers existing Azure resources, generates Terraform import blocks and
HCL resource stubs to enable progressive IaC adoption.

Sources:
  --from <audit-report.json>    Import from a previous audit report
  --subscription <id>           Discover resources in a subscription
  --resource-group <name>       Discover resources in a specific resource group

Without flags, an interactive wizard guides resource selection.

Generated files are placed in imports/ by default (override with --layer).`,
	RunE: runImport,
}

var (
	importFrom          string
	importSubscription  string
	importResourceGroup string
	importInclude       string
	importExclude       string
	importLayer         string
)

func init() {
	importCmd.Flags().StringVar(&importFrom, "from", "", "path to audit report JSON")
	importCmd.Flags().StringVar(&importSubscription, "subscription", "", "subscription ID to discover resources from")
	importCmd.Flags().StringVar(&importResourceGroup, "resource-group", "", "resource group to discover resources from")
	importCmd.Flags().StringVar(&importInclude, "include", "", "comma-separated Azure resource types to include")
	importCmd.Flags().StringVar(&importExclude, "exclude", "", "comma-separated Azure resource types to exclude")
	importCmd.Flags().StringVar(&importLayer, "layer", "", "target layer for generated files (e.g., connectivity)")

	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	root, _ := filepath.Abs(repoRoot)

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)

	// Determine import mode: from audit report, CLI flags, or interactive wizard.
	mode := resolveImportMode()

	var resources []importer.ImportableResource
	var targetDir string
	var err error

	switch mode {
	case "audit-report":
		bold.Fprintln(os.Stderr, "📦 Importing from audit report...")
		resources, err = importFromAuditReport(importFrom)
		if err != nil {
			return err
		}
		targetDir = resolveTargetDir(root)

	case "flags":
		bold.Fprintln(os.Stderr, "🔍 Discovering Azure resources...")
		resources, err = discoverResources()
		if err != nil {
			return err
		}
		targetDir = resolveTargetDir(root)

	default: // interactive
		bold.Fprintln(os.Stderr, "🧙 Starting import wizard...")
		wiz := wizard.NewImportWizard(nil)
		cfg, wizErr := wiz.Run()
		if wizErr != nil {
			return wizErr
		}

		if cfg.Source == "audit-report" {
			resources, err = importFromAuditReport(cfg.AuditReportPath)
		} else {
			opts := importer.DiscoveryOptions{
				Subscription:  cfg.Subscription,
				ResourceGroup: cfg.ResourceGroup,
			}
			d := importer.NewDiscovery(nil)
			resources, err = d.Discover(opts)
		}
		if err != nil {
			return err
		}

		targetDir = filepath.Join(root, cfg.TargetDir)

		// Interactive resource selection
		if len(resources) > 0 {
			resources, err = wiz.SelectResources(resources)
			if err != nil {
				return err
			}
		}
	}

	if len(resources) == 0 {
		yellow.Fprintln(os.Stderr, "⚠️  No importable resources found")
		return nil
	}

	// Apply include/exclude filters (non-interactive mode)
	if mode != "interactive" {
		resources = applyTypeFilters(resources, importInclude, importExclude)
	}

	// Report summary
	supported := 0
	unsupported := 0
	for _, r := range resources {
		if r.Supported {
			supported++
		} else {
			unsupported++
		}
	}

	fmt.Fprintf(os.Stderr, "\n📊 Resources: %d total, %d supported, %d unsupported\n",
		len(resources), supported, unsupported)

	// JSON output mode
	if jsonOutput {
		return outputImportJSON(resources)
	}

	// Generate HCL files
	gen := importer.NewHCLGenerator()
	files := gen.GenerateAll(resources, targetDir)

	if dryRun {
		bold.Fprintln(os.Stderr, "\n📝 Dry-run — files that would be generated:")
		for _, f := range files {
			fmt.Fprintf(os.Stderr, "  📄 %s\n", f.Path)
		}
		return nil
	}

	// Write files to disk
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			return fmt.Errorf("creating directory %s: %w", dir, mkErr)
		}
		if writeErr := os.WriteFile(f.Path, []byte(f.Content), 0o644); writeErr != nil {
			return fmt.Errorf("writing file %s: %w", f.Path, writeErr)
		}
		fmt.Fprintf(os.Stderr, "  ✅ %s\n", f.Path)
	}

	// Check for conflicts with existing Terraform state
	checkExistingConflicts(root, resources, yellow)

	green.Fprintln(os.Stderr, "\n✅ Import files generated successfully")
	fmt.Fprintln(os.Stderr, "\n📌 Next step: run `terraform plan` to verify zero-diff")

	return nil
}

func resolveImportMode() string {
	if importFrom != "" {
		return "audit-report"
	}
	if importSubscription != "" || importResourceGroup != "" {
		return "flags"
	}
	return "interactive"
}

func resolveTargetDir(root string) string {
	if importLayer != "" {
		return filepath.Join(root, importLayer)
	}
	return filepath.Join(root, "imports")
}

func importFromAuditReport(path string) ([]importer.ImportableResource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading audit report: %w", err)
	}

	var report audit.AuditReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parsing audit report: %w", err)
	}

	resources := make([]importer.ImportableResource, 0)
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

	return resources, nil
}

func discoverResources() ([]importer.ImportableResource, error) {
	cli := azure.NewAzCLI()
	d := importer.NewDiscovery(cli)

	opts := importer.DiscoveryOptions{
		Subscription:  importSubscription,
		ResourceGroup: importResourceGroup,
	}

	return d.Discover(opts)
}

func applyTypeFilters(resources []importer.ImportableResource, include, exclude string) []importer.ImportableResource {
	includeSet := parseCSV(include)
	excludeSet := parseCSV(exclude)

	if len(includeSet) == 0 && len(excludeSet) == 0 {
		return resources
	}

	filtered := make([]importer.ImportableResource, 0, len(resources))
	for _, r := range resources {
		normalized := strings.ToLower(r.AzureType)
		if len(includeSet) > 0 && !includeSet[normalized] {
			continue
		}
		if excludeSet[normalized] {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func parseCSV(value string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.ToLower(strings.TrimSpace(item))
		if trimmed != "" {
			result[trimmed] = true
		}
	}
	return result
}

func checkExistingConflicts(root string, resources []importer.ImportableResource, yellow *color.Color) {
	// Walk platform/ and landing-zones/ looking for .tf files that might
	// already manage the same resources.
	dirs := []string{
		filepath.Join(root, "platform"),
		filepath.Join(root, "landing-zones"),
	}

	existingResourceIDs := make(map[string]bool)
	for _, dir := range dirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".tf") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			// Simple heuristic: look for Azure resource IDs in existing TF files
			for _, r := range resources {
				if r.ID != "" && strings.Contains(string(content), r.ID) {
					existingResourceIDs[r.ID] = true
				}
			}
			return nil
		})
	}

	if len(existingResourceIDs) > 0 {
		yellow.Fprintf(os.Stderr, "\n⚠️  %d resource(s) may conflict with existing Terraform-managed resources:\n", len(existingResourceIDs))
		for id := range existingResourceIDs {
			fmt.Fprintf(os.Stderr, "  - %s\n", id)
		}
	}
}

type importJSONOutput struct {
	Status    string                       `json:"status"`
	Total     int                          `json:"total"`
	Supported int                          `json:"supported"`
	Resources []importer.ImportableResource `json:"resources"`
}

func outputImportJSON(resources []importer.ImportableResource) error {
	supported := 0
	for _, r := range resources {
		if r.Supported {
			supported++
		}
	}

	output := importJSONOutput{
		Status:    "ok",
		Total:     len(resources),
		Supported: supported,
		Resources: resources,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling import JSON: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
