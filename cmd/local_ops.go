package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kjourdan1/lzctl/internal/config"
)

var localLayerOrder = []string{
	"management-groups",
	"identity",
	"management",
	"governance",
	"connectivity",
}

var terraformPlanSummaryRegex = regexp.MustCompile(`Plan:\s+(\d+) to add,\s+(\d+) to change,\s+(\d+) to destroy`)

func localConfigPath() string {
	if strings.TrimSpace(cfgFile) != "" {
		return cfgFile
	}
	return filepath.Join(repoRoot, "lzctl.yaml")
}

func resolveLocalLayers(root, selected string) ([]string, error) {
	if strings.TrimSpace(selected) != "" {
		dir := filepath.Join(root, "platform", selected)
		if !dirExists(dir) {
			return nil, fmt.Errorf("layer directory not found: %s", dir)
		}
		return []string{selected}, nil
	}

	layers := make([]string, 0, len(localLayerOrder))
	for _, layer := range localLayerOrder {
		dir := filepath.Join(root, "platform", layer)
		if dirExists(dir) {
			layers = append(layers, layer)
		}
	}
	if len(layers) == 0 {
		return nil, fmt.Errorf("no platform layers found under %s", filepath.Join(root, "platform"))
	}
	return layers, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExistsLocal(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func ensureTerraformInstalled() error {
	if _, err := exec.LookPath("terraform"); err != nil {
		return fmt.Errorf("terraform not found in PATH (install with: winget install Hashicorp.Terraform)")
	}
	return nil
}

func runTerraformCmd(ctx context.Context, dir string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = dir
	// Inherit the full environment and append TF_INPUT=false so that the
	// process sees the current PATH (needed in tests that override PATH via
	// t.Setenv to point to a fake terraform binary).
	cmd.Env = append(os.Environ(), "TF_INPUT=false")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func parsePlanSummary(output string) (int, int, int) {
	matches := terraformPlanSummaryRegex.FindStringSubmatch(output)
	if len(matches) != 4 {
		return 0, 0, 0
	}
	add := parseInt(matches[1])
	change := parseInt(matches[2])
	destroy := parseInt(matches[3])
	return add, change, destroy
}

func parseInt(v string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}

var (
	cfgCache    *config.LZConfig
	cfgCacheErr error
	cfgCacheSet bool
)

// configCache returns the config, loading and caching it on first call.
// Commands that need config call this instead of config.Load() directly.
func configCache() (*config.LZConfig, error) {
	if !cfgCacheSet {
		cfgCache, cfgCacheErr = config.Load(localConfigPath())
		cfgCacheSet = true
	}
	return cfgCache, cfgCacheErr
}

// invalidateConfigCache forces a reload on the next configCache() call.
// Called by init after writing the config file for the first time.
func invalidateConfigCache() {
	cfgCacheSet = false
	cfgCache = nil
	cfgCacheErr = nil
}
