package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
	lztemplate "github.com/kjourdan1/lzctl/internal/template"
)

var (
	addBlueprintZone      string
	addBlueprintType      string
	addBlueprintOverrides []string
	addBlueprintOverwrite bool
)

var addBlueprintCmd = &cobra.Command{
	Use:   "add-blueprint",
	Short: "Attach a secure blueprint to an existing landing zone",
	Long: `Adds a blueprint definition under spec.landingZones[].blueprint and generates
Terraform blueprint files under landing-zones/<name>/blueprint.

Blueprint types:
  - paas-secure
  - aks-platform
  - aca-platform
  - avd-secure

In CI/headless mode, provide --landing-zone and --type.
The global --config flag can be used to point to a non-default lzctl.yaml path.`,
	RunE: runAddBlueprint,
}

func init() {
	addBlueprintCmd.Flags().StringVar(&addBlueprintZone, "landing-zone", "", "target landing zone name")
	addBlueprintCmd.Flags().StringVar(&addBlueprintType, "type", "", "blueprint type (paas-secure|aks-platform|aca-platform|avd-secure)")
	addBlueprintCmd.Flags().StringSliceVar(&addBlueprintOverrides, "set", nil, "blueprint override in path=value format (repeatable), e.g. apim.enabled=false")
	addBlueprintCmd.Flags().BoolVar(&addBlueprintOverwrite, "overwrite", false, "overwrite an existing blueprint on the landing zone")

	rootCmd.AddCommand(addBlueprintCmd)
}

func runAddBlueprint(cmd *cobra.Command, args []string) error {
	_ = args
	cfg, err := configCache()
	if err != nil {
		return fmt.Errorf("load config: %w (run lzctl init first)", err)
	}

	if len(cfg.Spec.LandingZones) == 0 {
		return fmt.Errorf("no landing zones found in lzctl.yaml")
	}

	zoneName := strings.TrimSpace(addBlueprintZone)
	blueprintType := strings.TrimSpace(addBlueprintType)

	if !effectiveCIMode() {
		zoneName, blueprintType, err = completeBlueprintInputsInteractive(zoneName, blueprintType, cfg)
		if err != nil {
			return err
		}
	}

	if zoneName == "" {
		return fmt.Errorf("--landing-zone is required")
	}
	if blueprintType == "" {
		return fmt.Errorf("--type is required")
	}
	if err := validateBlueprintType(blueprintType); err != nil {
		return err
	}

	zoneIndex := -1
	for i, z := range cfg.Spec.LandingZones {
		if strings.EqualFold(strings.TrimSpace(z.Name), zoneName) {
			zoneIndex = i
			break
		}
	}
	if zoneIndex == -1 {
		return fmt.Errorf("landing zone %q not found in lzctl.yaml", zoneName)
	}

	if cfg.Spec.LandingZones[zoneIndex].Blueprint != nil && !addBlueprintOverwrite {
		return fmt.Errorf("landing zone %q already has a blueprint (use --overwrite to replace)", cfg.Spec.LandingZones[zoneIndex].Name)
	}

	overrides, err := parseBlueprintOverrides(addBlueprintOverrides)
	if err != nil {
		return err
	}

	cfg.Spec.LandingZones[zoneIndex].Blueprint = &config.Blueprint{
		Type:      strings.ToLower(strings.TrimSpace(blueprintType)),
		Overrides: overrides,
	}

	engine, err := lztemplate.NewEngine()
	if err != nil {
		return fmt.Errorf("create template engine: %w", err)
	}

	blueprintFiles, err := engine.RenderBlueprint(cfg.Spec.LandingZones[zoneIndex].Name, cfg.Spec.LandingZones[zoneIndex].Blueprint, cfg)
	if err != nil {
		return err
	}

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	if dryRun {
		writer := lztemplate.Writer{DryRun: true}
		paths, writeErr := writer.WriteAll(blueprintFiles, absRoot)
		if writeErr != nil {
			return writeErr
		}
		color.Yellow("⚡ [DRY-RUN] Blueprint would be added to landing zone '%s'", cfg.Spec.LandingZones[zoneIndex].Name)
		for _, p := range paths {
			fmt.Printf("  - %s\n", p)
		}
		return nil
	}

	if err := config.Save(cfg, localConfigPath()); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	writer := lztemplate.Writer{DryRun: false}
	if _, err := writer.WriteAll(blueprintFiles, absRoot); err != nil {
		return err
	}

	if _, err := lztemplate.WriteLandingZoneMatrix(cfg, absRoot); err != nil {
		return fmt.Errorf("update landing-zone matrix: %w", err)
	}

	updater, err := lztemplate.NewPipelineUpdater()
	if err != nil {
		return fmt.Errorf("create pipeline updater: %w", err)
	}
	if _, err := updater.UpdatePipelines(cfg, absRoot); err != nil {
		return fmt.Errorf("update pipelines: %w", err)
	}

	if err := runValidate(cmd, nil); err != nil {
		return err
	}

	color.Green("✓ Blueprint '%s' added to landing zone '%s'", cfg.Spec.LandingZones[zoneIndex].Blueprint.Type, cfg.Spec.LandingZones[zoneIndex].Name)
	return nil
}

func validateBlueprintType(value string) error {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "paas-secure", "aks-platform", "aca-platform", "avd-secure":
		return nil
	default:
		return fmt.Errorf("invalid blueprint type %q (allowed: paas-secure, aks-platform, aca-platform, avd-secure)", value)
	}
}

func completeBlueprintInputsInteractive(zoneName, blueprintType string, cfg *config.LZConfig) (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	resolvedZone := strings.TrimSpace(zoneName)
	if resolvedZone == "" {
		fmt.Fprintln(os.Stderr, "Available landing zones:")
		for _, z := range cfg.Spec.LandingZones {
			fmt.Fprintf(os.Stderr, "  - %s\n", z.Name)
		}
		fmt.Fprint(os.Stderr, "Select landing zone: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("reading landing zone input: %w", err)
		}
		resolvedZone = strings.TrimSpace(line)
	}

	resolvedType := strings.ToLower(strings.TrimSpace(blueprintType))
	if resolvedType == "" {
		fmt.Fprintln(os.Stderr, "Blueprint types: paas-secure, aks-platform, aca-platform, avd-secure")
		fmt.Fprint(os.Stderr, "Select blueprint type: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("reading blueprint type input: %w", err)
		}
		resolvedType = strings.ToLower(strings.TrimSpace(line))
	}

	return resolvedZone, resolvedType, nil
}

func parseBlueprintOverrides(entries []string) (map[string]any, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	result := map[string]any{}

	for _, entry := range entries {
		if strings.TrimSpace(entry) == "" || strings.TrimSpace(entry) == "[]" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid override %q (expected path=value)", entry)
		}
		path := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if path == "" {
			return nil, fmt.Errorf("invalid override %q (empty path)", entry)
		}

		segments := strings.Split(path, ".")
		for i := range segments {
			segments[i] = strings.TrimSpace(segments[i])
			if segments[i] == "" {
				return nil, fmt.Errorf("invalid override path %q", path)
			}
		}

		cursor := result
		for i := 0; i < len(segments)-1; i++ {
			key := segments[i]
			next, ok := cursor[key]
			if !ok {
				m := map[string]any{}
				cursor[key] = m
				cursor = m
				continue
			}
			cast, ok := next.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("override path conflict at %q", key)
			}
			cursor = cast
		}
		cursor[segments[len(segments)-1]] = parseScalarValue(value)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}

func parseScalarValue(value string) any {
	v := strings.TrimSpace(value)
	lower := strings.ToLower(v)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}
	if strings.Contains(v, ".") {
		var floatVal float64
		if _, err := fmt.Sscanf(v, "%f", &floatVal); err == nil {
			return floatVal
		}
	}
	var intVal int
	if _, err := fmt.Sscanf(v, "%d", &intVal); err == nil {
		return intVal
	}
	return v
}
