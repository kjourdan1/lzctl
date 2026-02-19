package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func setupPolicyDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	for _, sub := range []string{"definitions", "initiatives", "assignments/platform", "exemptions"} {
		if err := os.MkdirAll(filepath.Join(policiesDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Create minimal workflow.yaml
	wf := PolicyWorkflow{
		APIVersion: "lzctl/v1",
		Kind:       "PolicyWorkflow",
		Metadata:   WorkflowMeta{Name: "test-workflow"},
	}
	data, _ := yaml.Marshal(wf)
	if err := os.WriteFile(filepath.Join(policiesDir, "workflow.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCreate_Definition(t *testing.T) {
	dir := setupPolicyDir(t)
	path, err := Create(CreateOpts{
		RepoRoot: dir,
		Type:     "definition",
		Name:     "test-def",
		Category: "Security",
	})
	if err != nil {
		t.Fatalf("Create definition: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
	// Verify valid JSON
	data, _ := os.ReadFile(path)
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["name"] != "test-def" {
		t.Errorf("expected name 'test-def', got %v", parsed["name"])
	}
}

func TestCreate_Initiative(t *testing.T) {
	dir := setupPolicyDir(t)
	path, err := Create(CreateOpts{
		RepoRoot: dir,
		Type:     "initiative",
		Name:     "test-init",
		Category: "Compliance",
	})
	if err != nil {
		t.Fatalf("Create initiative: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestCreate_Assignment(t *testing.T) {
	dir := setupPolicyDir(t)
	path, err := Create(CreateOpts{
		RepoRoot:   dir,
		Type:       "assignment",
		Name:       "test-assign",
		Scope:      "platform",
		Initiative: "security-baseline",
	})
	if err != nil {
		t.Fatalf("Create assignment: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestCreate_AssignmentMissingScope(t *testing.T) {
	dir := setupPolicyDir(t)
	_, err := Create(CreateOpts{
		RepoRoot: dir,
		Type:     "assignment",
		Name:     "test-assign",
	})
	if err == nil {
		t.Fatal("expected error for assignment without scope")
	}
}

func TestCreate_Exemption(t *testing.T) {
	dir := setupPolicyDir(t)
	path, err := Create(CreateOpts{
		RepoRoot: dir,
		Type:     "exemption",
		Name:     "test-exempt",
	})
	if err != nil {
		t.Fatalf("Create exemption: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestCreate_UnknownType(t *testing.T) {
	dir := setupPolicyDir(t)
	_, err := Create(CreateOpts{
		RepoRoot: dir,
		Type:     "bogus",
		Name:     "test",
	})
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestVerify_SimulatedData(t *testing.T) {
	dir := t.TempDir() // No workflow.yaml
	report, err := Verify(VerifyOpts{
		RepoRoot: dir,
		Name:     "test-assignment",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !report.Simulated {
		t.Error("expected Simulated=true when no workflow.yaml exists")
	}
	if report.Evaluated == 0 {
		t.Error("expected non-zero simulated data")
	}
}

func TestVerify_FromWorkflow(t *testing.T) {
	dir := setupPolicyDir(t)

	// Add an assignment to the workflow
	wf := &PolicyWorkflow{
		APIVersion: "lzctl/v1",
		Kind:       "PolicyWorkflow",
		Metadata:   WorkflowMeta{Name: "test"},
		Spec: WorkflowSpec{
			Assignments: []AssignmentState{
				{
					Name:  "test-assign",
					Scope: "/providers/Microsoft.Management/managementGroups/root",
					State: "test",
					Compliance: ComplianceState{
						Evaluated:    50,
						Compliant:    45,
						NonCompliant: 5,
					},
				},
			},
		},
	}
	saveWorkflow(dir, wf)

	report, err := Verify(VerifyOpts{
		RepoRoot: dir,
		Name:     "test-assign",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Simulated {
		t.Error("expected Simulated=false when data comes from workflow")
	}
	if report.Evaluated != 50 {
		t.Errorf("expected Evaluated=50, got %d", report.Evaluated)
	}
	if report.ComplianceRate != 90.0 {
		t.Errorf("expected ComplianceRate=90, got %f", report.ComplianceRate)
	}
}

func TestStatus(t *testing.T) {
	dir := setupPolicyDir(t)

	// Add data to workflow
	wf := &PolicyWorkflow{
		APIVersion: "lzctl/v1",
		Kind:       "PolicyWorkflow",
		Metadata:   WorkflowMeta{Name: "test"},
		Spec: WorkflowSpec{
			Definitions: []DefinitionState{
				{Name: "def1", State: "created"},
			},
		},
	}
	saveWorkflow(dir, wf)

	report, err := Status(StatusOpts{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(report.Definitions) != 1 {
		t.Errorf("expected 1 definition, got %d", len(report.Definitions))
	}
}

func TestDiff_ScanFiles(t *testing.T) {
	dir := setupPolicyDir(t)

	// Create sample definition files
	defsDir := filepath.Join(dir, "policies", "definitions")
	os.WriteFile(filepath.Join(defsDir, "policy-a.json"), []byte(`{}`), 0o644)
	os.WriteFile(filepath.Join(defsDir, "policy-b.json"), []byte(`{}`), 0o644)

	report, err := Diff(DiffOpts{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(report.Unchanged) < 2 {
		t.Errorf("expected at least 2 unchanged items, got %d", len(report.Unchanged))
	}
}

func TestDeploy_EnforcementModeChange(t *testing.T) {
	dir := setupPolicyDir(t)

	// Create assignment with Audit effect
	assignment := map[string]interface{}{
		"name": "test-deploy",
		"properties": map[string]interface{}{
			"enforcementMode": "DoNotEnforce",
			"parameters": map[string]interface{}{
				"effect": map[string]interface{}{
					"value": "Audit",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(assignment, "", "  ")
	assignPath := filepath.Join(dir, "policies", "assignments", "platform", "test-deploy.json")
	os.WriteFile(assignPath, data, 0o644)

	err := Deploy(DeployOpts{
		RepoRoot: dir,
		Name:     "test-deploy",
		Force:    true,
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	// Verify enforcementMode changed to Default
	updated, _ := os.ReadFile(assignPath)
	var result map[string]interface{}
	json.Unmarshal(updated, &result)
	props := result["properties"].(map[string]interface{})
	if props["enforcementMode"] != "Default" {
		t.Errorf("expected enforcementMode 'Default', got %v", props["enforcementMode"])
	}

	// Verify Audit effect is NOT auto-converted to Deny (M2 fix)
	params := props["parameters"].(map[string]interface{})
	effectParam := params["effect"].(map[string]interface{})
	if effectParam["value"] != "Audit" {
		t.Errorf("effect should remain 'Audit' (not auto-converted), got %v", effectParam["value"])
	}
}

func TestSimulateVerifyReport(t *testing.T) {
	report := simulateVerifyReport("test")
	if report.Evaluated == 0 {
		t.Error("simulated report should have non-zero Evaluated")
	}
	if report.AssignmentName != "test" {
		t.Errorf("expected assignment name 'test', got %q", report.AssignmentName)
	}
	if report.GeneratedAt == "" {
		t.Error("expected non-empty GeneratedAt")
	}
}
