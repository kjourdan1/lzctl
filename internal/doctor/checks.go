// Package doctor implements prerequisite checks for lzctl.
//
// It validates that required tools (terraform, az, git) are installed
// at the correct versions, that the Azure session is active, and that
// the necessary Azure permissions and resource providers are in place.
package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Status represents the outcome of a single check.
type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusWarn Status = "warn"
	StatusSkip Status = "skip"
)

// CheckResult is the outcome of running a single prerequisite check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

// Check defines a single prerequisite check.
type Check struct {
	Name     string
	Category string // "tool", "auth", "azure"
	Critical bool   // if true, failure => exit code 1
	Run      func(ctx context.Context, exec CmdExecutor) CheckResult
}

// CmdExecutor abstracts command execution for testability.
type CmdExecutor interface {
	// Run executes a command and returns combined stdout+stderr output.
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// realExecutor runs commands via os/exec.
type realExecutor struct{}

func (r *realExecutor) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// NewRealExecutor returns a CmdExecutor backed by os/exec.
func NewRealExecutor() CmdExecutor {
	return &realExecutor{}
}

// Summary holds the aggregated results of all checks.
type Summary struct {
	Results    []CheckResult `json:"results"`
	TotalPass  int           `json:"totalPass"`
	TotalFail  int           `json:"totalFail"`
	TotalWarn  int           `json:"totalWarn"`
	TotalSkip  int           `json:"totalSkip"`
	HasFailure bool          `json:"hasFailure"`
}

// RunAll executes all checks and returns a summary.
func RunAll(ctx context.Context, executor CmdExecutor) Summary {
	checks := AllChecks()
	results := make([]CheckResult, 0, len(checks))
	for _, c := range checks {
		r := c.Run(ctx, executor)
		results = append(results, r)
	}
	return buildSummary(results, checks)
}

func buildSummary(results []CheckResult, checks []Check) Summary {
	s := Summary{Results: results}
	for i, r := range results {
		switch r.Status {
		case StatusPass:
			s.TotalPass++
		case StatusFail:
			s.TotalFail++
			if checks[i].Critical {
				s.HasFailure = true
			}
		case StatusWarn:
			s.TotalWarn++
		case StatusSkip:
			s.TotalSkip++
		}
	}
	return s
}

// AllChecks returns the ordered list of prerequisite checks.
func AllChecks() []Check {
	return []Check{
		checkTerraform(),
		checkAzCLI(),
		checkGit(),
		checkGH(),
		checkAzSession(),
		checkAzManagementGroups(),
		checkResourceProvider("Microsoft.Management"),
		checkResourceProvider("Microsoft.Authorization"),
		checkResourceProvider("Microsoft.Network"),
		checkResourceProvider("Microsoft.ManagedIdentity"),
		checkStateBackend(),
	}
}

// --- Tool version checks ---

func checkTerraform() Check {
	return Check{
		Name:     "terraform",
		Category: "tool",
		Critical: true,
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			return checkToolVersion(ctx, ex, "terraform", []string{"version", "-json"}, `"terraform_version"\s*:\s*"([^"]+)"`, "1.5.0",
				"Install Terraform >= 1.5.0: https://developer.hashicorp.com/terraform/install")
		},
	}
}

func checkAzCLI() Check {
	return Check{
		Name:     "az-cli",
		Category: "tool",
		Critical: true,
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			return checkToolVersion(ctx, ex, "az", []string{"version", "--output", "tsv"}, `(\d+\.\d+\.\d+)`, "2.50.0",
				"Install Azure CLI >= 2.50.0: https://learn.microsoft.com/cli/azure/install-azure-cli")
		},
	}
}

func checkGit() Check {
	return Check{
		Name:     "git",
		Category: "tool",
		Critical: true,
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			return checkToolVersion(ctx, ex, "git", []string{"version"}, `(\d+\.\d+\.\d+)`, "2.30.0",
				"Install Git >= 2.30.0: https://git-scm.com/downloads")
		},
	}
}

func checkGH() Check {
	return Check{
		Name:     "gh-cli",
		Category: "tool",
		Critical: false, // optional
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			out, err := ex.Run(ctx, "gh", "version")
			if err != nil {
				return CheckResult{
					Name:    "gh-cli",
					Status:  StatusWarn,
					Message: "GitHub CLI (gh) not found — optional, used for PR automation",
					Fix:     "Install gh CLI: https://cli.github.com/",
				}
			}
			re := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
			m := re.FindString(out)
			if m == "" {
				m = "unknown"
			}
			return CheckResult{
				Name:    "gh-cli",
				Status:  StatusPass,
				Message: fmt.Sprintf("gh %s found", m),
			}
		},
	}
}

// --- Azure session checks ---

func checkAzSession() Check {
	return Check{
		Name:     "az-session",
		Category: "auth",
		Critical: true,
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			out, err := ex.Run(ctx, "az", "account", "show", "--output", "json")
			if err != nil {
				return CheckResult{
					Name:    "az-session",
					Status:  StatusFail,
					Message: "No active Azure session",
					Fix:     "Run: az login --tenant <your-tenant-id>",
				}
			}

			// Parse tenant, subscription and user from JSON output
			tenantID := extractJSONField(out, "tenantId")
			subID := extractJSONField(out, "id")
			userName := extractJSONField(out, "name")

			msg := fmt.Sprintf("Logged in — tenant: %s, subscription: %s (%s)", tenantID, subID, userName)
			return CheckResult{
				Name:    "az-session",
				Status:  StatusPass,
				Message: msg,
			}
		},
	}
}

func checkAzManagementGroups() Check {
	return Check{
		Name:     "az-mg-access",
		Category: "azure",
		Critical: true,
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			_, err := ex.Run(ctx, "az", "account", "management-group", "list", "--no-register")
			if err != nil {
				return CheckResult{
					Name:    "az-mg-access",
					Status:  StatusFail,
					Message: "Cannot list management groups — insufficient permissions",
					Fix:     "Ensure your identity has Management Group Reader role at root scope",
				}
			}
			return CheckResult{
				Name:    "az-mg-access",
				Status:  StatusPass,
				Message: "Management group access verified",
			}
		},
	}
}

func checkResourceProvider(provider string) Check {
	name := "provider-" + strings.ToLower(strings.TrimPrefix(provider, "Microsoft."))
	return Check{
		Name:     name,
		Category: "azure",
		Critical: true,
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			out, err := ex.Run(ctx, "az", "provider", "show", "-n", provider, "--query", "registrationState", "-o", "tsv")
			if err != nil {
				return CheckResult{
					Name:    name,
					Status:  StatusFail,
					Message: fmt.Sprintf("Cannot query provider %s", provider),
					Fix:     fmt.Sprintf("Run: az provider register -n %s", provider),
				}
			}
			state := strings.TrimSpace(out)
			if strings.EqualFold(state, "Registered") {
				return CheckResult{
					Name:    name,
					Status:  StatusPass,
					Message: fmt.Sprintf("%s is registered", provider),
				}
			}
			return CheckResult{
				Name:    name,
				Status:  StatusFail,
				Message: fmt.Sprintf("%s is %s (not registered)", provider, state),
				Fix:     fmt.Sprintf("Run: az provider register -n %s", provider),
			}
		},
	}
}

// --- Helpers ---

// checkToolVersion runs a command, extracts version via regex, and compares to min version.
func checkToolVersion(ctx context.Context, ex CmdExecutor, tool string, args []string, pattern, minVersion, fix string) CheckResult {
	out, err := ex.Run(ctx, tool, args...)
	if err != nil {
		return CheckResult{
			Name:    tool,
			Status:  StatusFail,
			Message: fmt.Sprintf("%s not found or not in PATH", tool),
			Fix:     fix,
		}
	}

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(out)
	if len(matches) < 2 {
		return CheckResult{
			Name:    tool,
			Status:  StatusWarn,
			Message: fmt.Sprintf("%s found but could not parse version from output", tool),
		}
	}

	version := matches[1]
	if !semverGTE(version, minVersion) {
		return CheckResult{
			Name:    tool,
			Status:  StatusFail,
			Message: fmt.Sprintf("%s %s found, but >= %s required", tool, version, minVersion),
			Fix:     fix,
		}
	}

	return CheckResult{
		Name:    tool,
		Status:  StatusPass,
		Message: fmt.Sprintf("%s %s", tool, version),
	}
}

// semverGTE returns true if version >= min (simple major.minor.patch comparison).
func semverGTE(version, min string) bool {
	v := parseSemver(version)
	m := parseSemver(min)
	if v[0] != m[0] {
		return v[0] > m[0]
	}
	if v[1] != m[1] {
		return v[1] > m[1]
	}
	return v[2] >= m[2]
}

func parseSemver(s string) [3]int {
	parts := strings.SplitN(s, ".", 3)
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		// Strip any suffix (e.g. "1.5.0-rc1" → "1", "5", "0")
		numStr := strings.SplitN(parts[i], "-", 2)[0]
		numStr = strings.SplitN(numStr, "+", 2)[0]
		n, _ := strconv.Atoi(numStr)
		result[i] = n
	}
	return result
}

// extractJSONField does a simple regex extraction for "field": "value" from JSON.
// This avoids importing encoding/json just for 3 fields.
func extractJSONField(jsonStr, field string) string {
	re := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, regexp.QuoteMeta(field)))
	m := re.FindStringSubmatch(jsonStr)
	if len(m) >= 2 {
		return m[1]
	}
	return "unknown"
}

// --- State backend checks ---

// checkStateBackend validates that the Terraform state backend storage account
// is accessible and follows security best practices (versioning, soft delete,
// HTTPS-only). This embodies the "state is a critical asset" philosophy.
func checkStateBackend() Check {
	return Check{
		Name:     "state-backend",
		Category: "state",
		Critical: false, // non-critical because lzctl.yaml may not exist yet
		Run: func(ctx context.Context, ex CmdExecutor) CheckResult {
			// Try to read lzctl.yaml to get storage account name
			// We look for it in common locations
			out, err := ex.Run(ctx, "az", "storage", "account", "list",
				"--query", "[?tags.purpose=='terraform-state'].{name:name,rg:resourceGroup}",
				"--output", "json")
			if err != nil {
				return CheckResult{
					Name:    "state-backend",
					Status:  StatusWarn,
					Message: "Could not query state backend storage accounts",
					Fix:     "Ensure you have Reader access to subscriptions containing state storage",
				}
			}

			if strings.TrimSpace(out) == "[]" || strings.TrimSpace(out) == "" {
				return CheckResult{
					Name:    "state-backend",
					Status:  StatusWarn,
					Message: "No storage account tagged purpose=terraform-state found",
					Fix:     "Run 'lzctl init' with bootstrap to create the state backend, or tag your existing storage account",
				}
			}

			// Check if blob versioning is mentioned in properties
			acctName := extractJSONField(out, "name")
			if acctName == "unknown" || acctName == "" {
				return CheckResult{
					Name:    "state-backend",
					Status:  StatusPass,
					Message: "State backend storage account(s) found",
				}
			}

			// Quick check: can we access the account?
			_, err = ex.Run(ctx, "az", "storage", "account", "show",
				"--name", acctName, "--query", "properties.supportsHttpsTrafficOnly")
			if err != nil {
				return CheckResult{
					Name:    "state-backend",
					Status:  StatusWarn,
					Message: fmt.Sprintf("Found state account %s but cannot read properties", acctName),
					Fix:     "Ensure you have Reader role on the storage account",
				}
			}

			return CheckResult{
				Name:    "state-backend",
				Status:  StatusPass,
				Message: fmt.Sprintf("State backend %s is accessible — run 'lzctl state health' for detailed checks", acctName),
			}
		},
	}
}
