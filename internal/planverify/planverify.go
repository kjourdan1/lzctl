// Package planverify implements tfplan integrity verification for lzctl.
//
// Threat model:
//   - A compromised CI step could modify the tfplan between plan and apply.
//   - A forged tfplan (protobuf-reconstructed) could target out-of-scope resources.
//
// Verification flow:
//  1. After `terraform plan -out=tfplan`, call Sign(planFile, keyFile) to produce tfplan.sha256
//  2. Before `terraform apply tfplan`, call Verify(planFile, sigFile) which:
//     a. Recomputes SHA256 of tfplan
//     b. Compares with stored value in tfplan.sha256
//     c. (Optional) Parses tfplan JSON to validate subscription scope
package planverify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// SigFile returns the path to the SHA256 signature file for the given plan file.
func SigFile(planFile string) string {
	return planFile + ".sha256"
}

// Sign computes the SHA256 of planFile and writes it to planFile.sha256.
// Call this immediately after `terraform plan -out=<planFile>`.
func Sign(planFile string) (string, error) {
	hash, err := sha256File(planFile)
	if err != nil {
		return "", fmt.Errorf("planverify: computing SHA256 for %s: %w", planFile, err)
	}

	sigFile := SigFile(planFile)
	if err := os.WriteFile(sigFile, []byte(hash), 0o600); err != nil {
		return "", fmt.Errorf("planverify: writing sig file %s: %w", sigFile, err)
	}

	return hash, nil
}

// VerifyResult holds the result of a plan verification.
type VerifyResult struct {
	PlanFile     string
	ExpectedHash string
	ActualHash   string
	Valid        bool
}

// Verify reads planFile.sha256 and compares it to the actual SHA256 of planFile.
// Returns an error (and halts apply) if the hashes do not match.
func Verify(planFile string) (*VerifyResult, error) {
	sigFile := SigFile(planFile)

	expectedBytes, err := os.ReadFile(sigFile)
	if err != nil {
		return nil, fmt.Errorf("planverify: cannot read sig file %s: %w (was Sign() called after plan?)", sigFile, err)
	}
	expected := strings.TrimSpace(string(expectedBytes))

	actual, err := sha256File(planFile)
	if err != nil {
		return nil, fmt.Errorf("planverify: computing SHA256 for %s: %w", planFile, err)
	}

	result := &VerifyResult{
		PlanFile:     planFile,
		ExpectedHash: expected,
		ActualHash:   actual,
		Valid:        expected == actual,
	}

	if !result.Valid {
		return result, fmt.Errorf( //nolint:revive,staticcheck // intentional structured security error
			"plan integrity failed: tfplan SHA256 mismatch (file: %s, expected: %s, got: %s)",
			planFile, expected, actual,
		)
	}

	return result, nil
}

// ScopeViolation is returned when a plan targets resources outside the declared tenant scope.
type ScopeViolation struct {
	SubscriptionID string
	ResourceAddr   string
}

// ValidateScope parses the tfplan JSON (via `terraform show -json`) and checks that
// all planned subscription IDs belong to the declared set.
//
// allowedSubscriptions is the list of subscription IDs from tenant.Spec.Environments[*].Subscriptions.
// planDir is the directory containing the tfplan file (terraform must be run from there).
func ValidateScope(ctx context.Context, planFile string, allowedSubscriptions []string, planDir string) ([]ScopeViolation, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Build a set for O(1) lookup
	allowed := make(map[string]bool, len(allowedSubscriptions))
	for _, sub := range allowedSubscriptions {
		allowed[strings.ToLower(sub)] = true
	}

	// Run `terraform show -json <planFile>` to get the plan in JSON format
	cmd := exec.CommandContext(ctx, "terraform", "show", "-json", filepath.Base(planFile)) //nolint:gosec // input is a local file path, not user-controlled
	cmd.Dir = planDir
	cmd.Env = append(os.Environ(), "TF_INPUT=false")

	out, err := cmd.Output()
	if err != nil {
		// Non-fatal for scope validation — log a warning but do not block
		return nil, fmt.Errorf("planverify: running terraform show -json: %w (scope validation skipped)", err)
	}

	return parsePlanScopeViolations(out, allowed)
}

// planJSON is a minimal representation of the terraform plan JSON output.
// Full schema: https://developer.hashicorp.com/terraform/internals/json-format
type planJSON struct {
	PlannedValues struct {
		RootModule planModule `json:"root_module"`
	} `json:"planned_values"`
	Configuration struct {
		ProviderConfig map[string]struct {
			Expressions struct {
				SubscriptionID *struct {
					ConstantValue string `json:"constant_value"`
				} `json:"subscription_id"`
			} `json:"expressions"`
		} `json:"provider_config"`
	} `json:"configuration"`
}

type planModule struct {
	Resources    []planResource `json:"resources"`
	ChildModules []planModule   `json:"child_modules,omitempty"`
}

type planResource struct {
	Address string         `json:"address"`
	Values  map[string]any `json:"values"`
}

func parsePlanScopeViolations(planJSONBytes []byte, allowed map[string]bool) ([]ScopeViolation, error) {
	var plan planJSON
	if err := json.Unmarshal(planJSONBytes, &plan); err != nil {
		return nil, fmt.Errorf("planverify: parsing terraform plan JSON: %w", err)
	}

	var violations []ScopeViolation

	// Check provider-level subscription IDs
	for _, providerCfg := range plan.Configuration.ProviderConfig {
		if providerCfg.Expressions.SubscriptionID != nil {
			subID := strings.ToLower(providerCfg.Expressions.SubscriptionID.ConstantValue)
			if subID != "" && !allowed[subID] {
				violations = append(violations, ScopeViolation{
					SubscriptionID: subID,
					ResourceAddr:   "(provider config)",
				})
			}
		}
	}

	// Check resource-level subscription_id values (some resources embed it)
	// Recursively collect resources from root module and all child modules
	allResources := collectResources(plan.PlannedValues.RootModule)
	for _, res := range allResources {
		if subIDVal, ok := res.Values["subscription_id"]; ok {
			if subID, ok := subIDVal.(string); ok && subID != "" {
				if !allowed[strings.ToLower(subID)] {
					violations = append(violations, ScopeViolation{
						SubscriptionID: subID,
						ResourceAddr:   res.Address,
					})
				}
			}
		}
	}

	return violations, nil
}

// collectResources recursively gathers resources from a module and all child modules.
func collectResources(mod planModule) []planResource {
	resources := append([]planResource{}, mod.Resources...)
	for _, child := range mod.ChildModules {
		resources = append(resources, collectResources(child)...)
	}
	return resources
}

// ActionViolation represents a resource scheduled for a destructive change.
type ActionViolation struct {
	ResourceAddr string
	Action       string // "delete" | "replace"
}

var safeToDestroyPrefixes = []string{"null_resource", "random_", "azurerm_resource_group_template_deployment"}

func isSafeToDestroy(resourceType string) bool {
	for _, prefix := range safeToDestroyPrefixes {
		if resourceType == prefix || strings.HasPrefix(resourceType, prefix) {
			return true
		}
	}
	return false
}

// ValidateActions reads a tfplan JSON file (produced by `terraform show -json tfplan`)
// and returns all resources scheduled for deletion or replacement, excluding safe types.
func ValidateActions(planJSONFile string) ([]ActionViolation, error) {
	data, err := os.ReadFile(planJSONFile)
	if err != nil {
		return nil, fmt.Errorf("planverify: reading plan JSON %s: %w", planJSONFile, err)
	}
	var plan struct {
		ResourceChanges []struct {
			Address string `json:"address"`
			Type    string `json:"type"`
			Change  struct {
				Actions []string `json:"actions"`
			} `json:"change"`
		} `json:"resource_changes"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("planverify: parsing plan JSON: %w", err)
	}
	var violations []ActionViolation
	for _, rc := range plan.ResourceChanges {
		if isSafeToDestroy(rc.Type) {
			continue
		}
		hasDelete := slices.Contains(rc.Change.Actions, "delete")
		hasCreate := slices.Contains(rc.Change.Actions, "create")
		switch {
		case hasDelete && !hasCreate:
			violations = append(violations, ActionViolation{ResourceAddr: rc.Address, Action: "delete"})
		case hasDelete && hasCreate:
			violations = append(violations, ActionViolation{ResourceAddr: rc.Address, Action: "replace"})
		}
	}
	return violations, nil
}

// sha256File computes the hex-encoded SHA256 hash of a file.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
