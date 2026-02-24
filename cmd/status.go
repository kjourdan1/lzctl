package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kjourdan1/lzctl/internal/config"
	"github.com/kjourdan1/lzctl/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status and configuration overview",
	Long: `Displays project metadata from lzctl.yaml, available platform layers,
and current Git repository state.

Examples:
  lzctl status
  lzctl status --json
  lzctl status --live`,
	RunE: runStatus,
}

var statusLive bool

func init() {
	statusCmd.Flags().BoolVar(&statusLive, "live", false, "query Azure for live resource counts")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	output.Init(verbosity > 0, jsonOutput)

	root, err := absRepoRoot()
	if err != nil {
		return err
	}

	// Load project metadata from lzctl.yaml.
	cfgPath := localConfigPath()
	cfg, loadErr := config.Load(cfgPath)

	// Get git info.
	gitInfo := getGitInfo(root)

	// Resolve platform layers.
	layers, _ := resolveLocalLayers(root, "")

	// JSON output mode.
	if jsonOutput {
		result := map[string]interface{}{
			"configFile": cfgPath,
			"layers":     layers,
		}
		if cfg != nil {
			result["project"] = map[string]interface{}{
				"name":          cfg.Metadata.Name,
				"tenant":        cfg.Metadata.Tenant,
				"primaryRegion": cfg.Metadata.PrimaryRegion,
				"connectivity":  cfg.Spec.Platform.Connectivity.Type,
				"landingZones":  len(cfg.Spec.LandingZones),
				"cicdPlatform":  cfg.Spec.CICD.Platform,
			}
		}
		if gitInfo != nil {
			result["git"] = gitInfo
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	bold := color.New(color.Bold)

	// Project metadata section.
	if cfg != nil {
		bold.Fprintf(os.Stderr, "📋 Project: %s\n", cfg.Metadata.Name)
		fmt.Fprintf(os.Stderr, "  Tenant:        %s\n", cfg.Metadata.Tenant)
		fmt.Fprintf(os.Stderr, "  Region:        %s\n", cfg.Metadata.PrimaryRegion)
		fmt.Fprintf(os.Stderr, "  Connectivity:  %s\n", cfg.Spec.Platform.Connectivity.Type)
		fmt.Fprintf(os.Stderr, "  Landing Zones: %d\n", len(cfg.Spec.LandingZones))
		fmt.Fprintf(os.Stderr, "  CI/CD:         %s\n", cfg.Spec.CICD.Platform)
		fmt.Fprintln(os.Stderr)
	} else {
		color.New(color.FgYellow).Fprintf(os.Stderr, "⚠️  No lzctl.yaml found (%v)\n", loadErr)
		fmt.Fprintf(os.Stderr, "   Run: lzctl init  to create a project\n\n")
	}

	// Platform layers section.
	bold.Fprintln(os.Stderr, "📂 Platform Layers")
	if len(layers) > 0 {
		for _, layer := range layers {
			color.New(color.FgGreen).Fprintf(os.Stderr, "  ✅ %s\n", layer)
		}
	} else {
		fmt.Fprintln(os.Stderr, "  (none generated yet)")
	}
	fmt.Fprintln(os.Stderr)

	// Git info section.
	if gitInfo != nil {
		bold.Fprintln(os.Stderr, "🔀 Git")
		fmt.Fprintf(os.Stderr, "  Branch: %s\n", gitInfo.Branch)
		if gitInfo.LastCommit != "" {
			fmt.Fprintf(os.Stderr, "  Last:   %s\n", gitInfo.LastCommit)
		}
		if gitInfo.Dirty {
			color.Yellow("  ⚠ Working tree has uncommitted changes")
		}
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

// gitInfoResult holds Git repository metadata.
type gitInfoResult struct {
	Branch     string `json:"branch"`
	LastCommit string `json:"lastCommit,omitempty"`
	Dirty      bool   `json:"dirty"`
}

// getGitInfo extracts current branch and last commit info.
func getGitInfo(repoDir string) *gitInfoResult {
	info := &gitInfoResult{}

	// Get current branch.
	branchOut, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return nil // Not a Git repo.
	}
	info.Branch = strings.TrimSpace(string(branchOut))

	// Get last commit.
	commitOut, err := exec.Command("git", "-C", repoDir, "log", "-1", "--format=%h %s (%cr)").Output()
	if err == nil {
		info.LastCommit = strings.TrimSpace(string(commitOut))
	}

	// Check dirty state.
	statusOut, err := exec.Command("git", "-C", repoDir, "status", "--porcelain").Output()
	if err == nil {
		info.Dirty = len(strings.TrimSpace(string(statusOut))) > 0
	}

	return info
}
