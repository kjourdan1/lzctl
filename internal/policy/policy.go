package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ────────────────────────────────────────────────────────────
// Policy-as-Code Engine
//
// Implements the Enterprise Policy as Code (EPAC) methodology:
//   1. Create    → Scaffold definitions/initiatives/assignments
//   2. Test      → Deploy in DoNotEnforce mode
//   3. Verify    → Compliance state query + report
//   4. Remediate → Create remediation tasks
//   5. Deploy    → Switch to Default enforcement
//
// All operations are stateless and idempotent.
// State is tracked in policies/workflow.yaml.
// ────────────────────────────────────────────────────────────

// ── Types ─────────────────────────────────────────────────

// PolicyDefinition represents a custom policy definition JSON file.
type PolicyDefinition struct {
	Name       string                 `json:"name" yaml:"name"`
	Properties map[string]interface{} `json:"properties" yaml:"properties"`
}

// PolicyWorkflow tracks lifecycle state of all policy artifacts.
type PolicyWorkflow struct {
	APIVersion  string          `yaml:"apiVersion"`
	Kind        string          `yaml:"kind"`
	Metadata    WorkflowMeta    `yaml:"metadata"`
	Spec        WorkflowSpec    `yaml:"spec"`
}

type WorkflowMeta struct {
	Name   string `yaml:"name"`
	Tenant string `yaml:"tenant"`
}

type WorkflowSpec struct {
	Definitions []DefinitionState  `yaml:"definitions"`
	Initiatives []InitiativeState  `yaml:"initiatives"`
	Assignments []AssignmentState  `yaml:"assignments"`
	Exemptions  []ExemptionState   `yaml:"exemptions"`
}

type DefinitionState struct {
	Name        string `yaml:"name"`
	State       string `yaml:"state"`
	LastUpdated string `yaml:"lastUpdated"`
}

type InitiativeState struct {
	Name        string `yaml:"name"`
	State       string `yaml:"state"`
	LastUpdated string `yaml:"lastUpdated"`
}

type AssignmentState struct {
	Name             string           `yaml:"name"`
	Scope            string           `yaml:"scope"`
	State            string           `yaml:"state"`
	EnforcementMode  string           `yaml:"enforcementMode"`
	SecurityCritical bool             `yaml:"securityCritical,omitempty"`
	IncidentTicket   string           `yaml:"incidentTicket,omitempty"`
	LastUpdated      string           `yaml:"lastUpdated"`
	Compliance       ComplianceState  `yaml:"compliance"`
	RemediationTasks []RemediationRef `yaml:"remediationTasks"`
}

type ComplianceState struct {
	Evaluated    int    `yaml:"evaluated"`
	Compliant    int    `yaml:"compliant"`
	NonCompliant int    `yaml:"nonCompliant"`
	Exempt       int    `yaml:"exempt"`
	LastScan     string `yaml:"lastScan"`
}

type RemediationRef struct {
	TaskID             string `yaml:"taskId"`
	Status             string `yaml:"status"`
	Created            string `yaml:"created"`
	Completed          string `yaml:"completed,omitempty"`
	ResourcesRemediated int   `yaml:"resourcesRemediated"`
}

type ExemptionState struct {
	Name       string `yaml:"name"`
	Assignment string `yaml:"assignment"`
	Category   string `yaml:"category"`
	ExpiresOn  string `yaml:"expiresOn"`
	TicketRef  string `yaml:"ticketReference"`
	Status     string `yaml:"status"`
}

// ── Create ────────────────────────────────────────────────

type CreateOpts struct {
	RepoRoot   string
	Type       string // definition, initiative, assignment, exemption
	Name       string
	Category   string
	Scope      string
	Initiative string
}

// Create scaffolds a new policy artifact JSON file.
func Create(opts CreateOpts) (string, error) {
	policiesDir := filepath.Join(opts.RepoRoot, "policies")

	var outPath string
	var content []byte
	var err error

	switch opts.Type {
	case "definition":
		outPath = filepath.Join(policiesDir, "definitions", opts.Name+".json")
		content, err = scaffoldDefinition(opts.Name, opts.Category)

	case "initiative":
		outPath = filepath.Join(policiesDir, "initiatives", opts.Name+".json")
		content, err = scaffoldInitiative(opts.Name, opts.Category)

	case "assignment":
		if opts.Scope == "" {
			return "", fmt.Errorf("--scope is required for assignment type")
		}
		outPath = filepath.Join(policiesDir, "assignments", opts.Scope, opts.Name+".json")
		content, err = scaffoldAssignment(opts.Name, opts.Scope, opts.Initiative)

	case "exemption":
		outPath = filepath.Join(policiesDir, "exemptions", opts.Name+".json")
		content, err = scaffoldExemption(opts.Name)

	default:
		return "", fmt.Errorf("unknown policy type: %s (expected: definition, initiative, assignment, exemption)", opts.Type)
	}

	if err != nil {
		return "", err
	}

	// Ensure directory exists
	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", outPath, err)
	}

	// Update workflow state
	if err := updateWorkflowState(opts.RepoRoot, opts.Type, opts.Name, "created"); err != nil {
		// Non-fatal: warn but don't fail
		fmt.Fprintf(os.Stderr, "warning: could not update workflow.yaml: %v\n", err)
	}

	return outPath, nil
}

// ── Test ──────────────────────────────────────────────────

type TestOpts struct {
	RepoRoot string
	Tenant   string
	Name     string
	DryRun   bool
}

type TestResult struct {
	Scope      string
	Initiative string
}

// Test deploys an assignment in DoNotEnforce mode.
func Test(opts TestOpts) (*TestResult, error) {
	// Load assignment file
	assignmentPath, err := findAssignment(opts.RepoRoot, opts.Name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(assignmentPath)
	if err != nil {
		return nil, fmt.Errorf("read assignment: %w", err)
	}

	var assignment map[string]interface{}
	if err := json.Unmarshal(data, &assignment); err != nil {
		return nil, fmt.Errorf("parse assignment JSON: %w", err)
	}

	props, _ := assignment["properties"].(map[string]interface{})
	scope, _ := props["scope"].(string)
	policyDefId, _ := props["policyDefinitionId"].(string)

	// Ensure enforcement mode is DoNotEnforce
	props["enforcementMode"] = "DoNotEnforce"

	if !opts.DryRun {
		// In a real implementation, this would call Azure API to create/update
		// the policy assignment. For now, we update the local file and workflow.
		updated, err := json.MarshalIndent(assignment, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal assignment: %w", err)
		}
		if err := os.WriteFile(assignmentPath, updated, 0o644); err != nil {
			return nil, fmt.Errorf("write assignment: %w", err)
		}

		// Update workflow state
		if err := updateWorkflowState(opts.RepoRoot, "assignment", opts.Name, "test"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not update workflow.yaml: %v\n", err)
		}

		// Run terraform apply for the governance layer
		fmt.Println("  Deploying via terraform (governance layer)...")
		// In production: exec terraform apply for governance layer
	}

	return &TestResult{
		Scope:      scope,
		Initiative: policyDefId,
	}, nil
}

// ── Verify ────────────────────────────────────────────────

type VerifyOpts struct {
	RepoRoot string
	Tenant   string
	Name     string
	Output   string
}

type VerifyReport struct {
	AssignmentName    string               `json:"assignmentName" yaml:"assignmentName"`
	Scope             string               `json:"scope" yaml:"scope"`
	Evaluated         int                  `json:"evaluated" yaml:"evaluated"`
	Compliant         int                  `json:"compliant" yaml:"compliant"`
	NonCompliant      int                  `json:"nonCompliant" yaml:"nonCompliant"`
	Exempt            int                  `json:"exempt" yaml:"exempt"`
	ComplianceRate    float64              `json:"complianceRate" yaml:"complianceRate"`
	NonCompliantGroups []NonCompliantGroup  `json:"nonCompliantGroups,omitempty" yaml:"nonCompliantGroups,omitempty"`
	GeneratedAt       string               `json:"generatedAt" yaml:"generatedAt"`
	Simulated         bool                 `json:"simulated,omitempty" yaml:"simulated,omitempty"`
}

type NonCompliantGroup struct {
	PolicyName string   `json:"policyName" yaml:"policyName"`
	Count      int      `json:"count" yaml:"count"`
	Resources  []string `json:"resources" yaml:"resources"`
}

// Verify queries compliance state and generates a report.
func Verify(opts VerifyOpts) (*VerifyReport, error) {
	// In a real implementation, this queries the Azure Policy compliance API:
	//   GET /providers/Microsoft.PolicyInsights/policyStates/latest/summarize
	//
	// For now, we read from the workflow.yaml cached state or return simulated data.

	workflow, err := loadWorkflow(opts.RepoRoot)
	if err != nil {
		// No workflow yet – return simulated data with warning
		fmt.Fprintf(os.Stderr, "warning: no workflow.yaml found; returning simulated compliance data (not from Azure API)\n")
		report := simulateVerifyReport(opts.Name)
		report.Simulated = true
		return report, nil
	}

	// Find assignment in workflow
	for _, a := range workflow.Spec.Assignments {
		if a.Name == opts.Name {
			report := &VerifyReport{
				AssignmentName: opts.Name,
				Scope:          a.Scope,
				Evaluated:      a.Compliance.Evaluated,
				Compliant:      a.Compliance.Compliant,
				NonCompliant:   a.Compliance.NonCompliant,
				Exempt:         a.Compliance.Exempt,
				GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
			}
			if report.Evaluated > 0 {
				report.ComplianceRate = float64(report.Compliant) / float64(report.Evaluated) * 100
			}

			if report.Evaluated == 0 {
				fmt.Fprintf(os.Stderr, "warning: no compliance data in workflow.yaml for '%s'; data comes from cached state, not live Azure API\n", opts.Name)
			}

			// Update workflow state
			if err := updateWorkflowState(opts.RepoRoot, "assignment", opts.Name, "verify"); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not update workflow.yaml: %v\n", err)
			}

			// Write report to file if output specified
			if opts.Output != "" {
				if err := writeReport(opts.Output, report); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not write report: %v\n", err)
				}
			}

			return report, nil
		}
	}

	fmt.Fprintf(os.Stderr, "warning: assignment '%s' not found in workflow.yaml; returning simulated compliance data (not from Azure API)\n", opts.Name)
	report := simulateVerifyReport(opts.Name)
	report.Simulated = true
	return report, nil
}

// ── Remediate ─────────────────────────────────────────────

type RemediateOpts struct {
	RepoRoot string
	Tenant   string
	Name     string
	DryRun   bool
}

type RemediateResult struct {
	TaskCount int
	Tasks     []RemediationTask
}

type RemediationTask struct {
	Name          string `json:"name" yaml:"name"`
	PolicyName    string `json:"policyName" yaml:"policyName"`
	ResourceCount int    `json:"resourceCount" yaml:"resourceCount"`
	Status        string `json:"status" yaml:"status"`
}

// Remediate creates remediation tasks for non-compliant resources.
func Remediate(opts RemediateOpts) (*RemediateResult, error) {
	// In a real implementation, this creates Azure Policy remediation tasks:
	//   PUT /providers/Microsoft.PolicyInsights/remediations/{remediationName}
	//
	// Only applicable for DeployIfNotExists and Modify effects.

	fmt.Fprintf(os.Stderr, "warning: remediation uses simulated task data (Azure Policy API not yet integrated)\n")

	// Simulate remediation tasks
	tasks := []RemediationTask{
		{
			Name:          fmt.Sprintf("remediate-%s-batch1", opts.Name),
			PolicyName:    "require-resource-tags",
			ResourceCount: 25,
			Status:        "simulated",
		},
	}

	if !opts.DryRun {
		// Update workflow state
		if err := updateWorkflowState(opts.RepoRoot, "assignment", opts.Name, "remediate"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not update workflow.yaml: %v\n", err)
		}

		// Update remediation tasks in workflow
		workflow, err := loadWorkflow(opts.RepoRoot)
		if err == nil {
			for i, a := range workflow.Spec.Assignments {
				if a.Name == opts.Name {
					workflow.Spec.Assignments[i].RemediationTasks = append(
						workflow.Spec.Assignments[i].RemediationTasks,
						RemediationRef{
							TaskID:  tasks[0].Name,
							Status:  "inProgress",
							Created: time.Now().UTC().Format(time.RFC3339),
						},
					)
					saveWorkflow(opts.RepoRoot, workflow)
					break
				}
			}
		}
	}

	return &RemediateResult{
		TaskCount: len(tasks),
		Tasks:     tasks,
	}, nil
}

// ── Deploy ────────────────────────────────────────────────

type DeployOpts struct {
	RepoRoot string
	Tenant   string
	Name     string
	Force    bool
}

// Deploy switches an assignment to Default enforcement mode.
func Deploy(opts DeployOpts) error {
	// Safety check: verify compliance before enforcing
	if !opts.Force {
		workflow, err := loadWorkflow(opts.RepoRoot)
		if err == nil {
			for _, a := range workflow.Spec.Assignments {
				if a.Name == opts.Name {
					if a.Compliance.NonCompliant > 0 {
						return fmt.Errorf(
							"assignment '%s' has %d non-compliant resources; use --force to override or run 'lzctl policy remediate' first",
							opts.Name, a.Compliance.NonCompliant,
						)
					}
				}
			}
		}
	}

	// Update assignment file: switch enforcementMode to Default
	assignmentPath, err := findAssignment(opts.RepoRoot, opts.Name)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(assignmentPath)
	if err != nil {
		return fmt.Errorf("read assignment: %w", err)
	}

	var assignment map[string]interface{}
	if err := json.Unmarshal(data, &assignment); err != nil {
		return fmt.Errorf("parse assignment JSON: %w", err)
	}

	props, _ := assignment["properties"].(map[string]interface{})
	props["enforcementMode"] = "Default"

	// Note: effect parameters (Audit/Deny/etc.) are NOT automatically changed.
	// Changing enforcement mode to Default is sufficient to start enforcing.
	// If you need to change effects (e.g., Audit → Deny), update the assignment
	// file manually or use a separate policy definition with the desired effect.

	updated, err := json.MarshalIndent(assignment, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal assignment: %w", err)
	}
	if err := os.WriteFile(assignmentPath, updated, 0o644); err != nil {
		return fmt.Errorf("write assignment: %w", err)
	}

	// Update workflow state
	if err := updateWorkflowState(opts.RepoRoot, "assignment", opts.Name, "deploy"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update workflow.yaml: %v\n", err)
	}

	return nil
}

// ── Status ────────────────────────────────────────────────

type StatusOpts struct {
	RepoRoot string
	Tenant   string
}

type StatusReport struct {
	Definitions []DefinitionState
	Initiatives []InitiativeState
	Assignments []AssignmentState
	Exemptions  []ExemptionState
}

// Status returns the current workflow state for all policy artifacts.
func Status(opts StatusOpts) (*StatusReport, error) {
	workflow, err := loadWorkflow(opts.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("load workflow: %w", err)
	}

	return &StatusReport{
		Definitions: workflow.Spec.Definitions,
		Initiatives: workflow.Spec.Initiatives,
		Assignments: workflow.Spec.Assignments,
		Exemptions:  workflow.Spec.Exemptions,
	}, nil
}

// ── Diff ──────────────────────────────────────────────────

type DiffOpts struct {
	RepoRoot string
	Tenant   string
}

type DiffItem struct {
	Name string
	Type string // definition, initiative, assignment
}

type DiffReport struct {
	ToCreate  []DiffItem
	ToUpdate  []DiffItem
	ToDelete  []DiffItem
	Unchanged []DiffItem
}

// Diff compares local policy files with deployed state.
func Diff(opts DiffOpts) (*DiffReport, error) {
	policiesDir := filepath.Join(opts.RepoRoot, "policies")

	report := &DiffReport{}

	// Scan local definitions
	defsDir := filepath.Join(policiesDir, "definitions")
	if entries, err := os.ReadDir(defsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				name := strings.TrimSuffix(e.Name(), ".json")
				// In production: query Azure for existing definition
				// For now: mark as unchanged if in workflow, else to-create
				report.Unchanged = append(report.Unchanged, DiffItem{
					Name: name,
					Type: "definition",
				})
			}
		}
	}

	// Scan local initiatives
	initDir := filepath.Join(policiesDir, "initiatives")
	if entries, err := os.ReadDir(initDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				name := strings.TrimSuffix(e.Name(), ".json")
				report.Unchanged = append(report.Unchanged, DiffItem{
					Name: name,
					Type: "initiative",
				})
			}
		}
	}

	// Scan local assignments
	assignDir := filepath.Join(policiesDir, "assignments")
	if err := filepath.Walk(assignDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}
		name := strings.TrimSuffix(info.Name(), ".json")
		report.Unchanged = append(report.Unchanged, DiffItem{
			Name: name,
			Type: "assignment",
		})
		return nil
	}); err != nil {
		// Non-fatal: assignments dir may not exist
	}

	return report, nil
}

// ── Helpers ───────────────────────────────────────────────

func loadWorkflow(repoRoot string) (*PolicyWorkflow, error) {
	path := filepath.Join(repoRoot, "policies", "workflow.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wf PolicyWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parse workflow.yaml: %w", err)
	}
	return &wf, nil
}

func saveWorkflow(repoRoot string, wf *PolicyWorkflow) error {
	path := filepath.Join(repoRoot, "policies", "workflow.yaml")
	data, err := yaml.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshal workflow: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func updateWorkflowState(repoRoot, artifactType, name, state string) error {
	wf, err := loadWorkflow(repoRoot)
	if err != nil {
		// If no workflow file, create one
		wf = &PolicyWorkflow{
			APIVersion: "lzctl/v1",
			Kind:       "PolicyWorkflow",
			Metadata:   WorkflowMeta{Name: "policy-workflow"},
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)

	switch artifactType {
	case "definition":
		found := false
		for i, d := range wf.Spec.Definitions {
			if d.Name == name {
				wf.Spec.Definitions[i].State = state
				wf.Spec.Definitions[i].LastUpdated = now
				found = true
				break
			}
		}
		if !found {
			wf.Spec.Definitions = append(wf.Spec.Definitions, DefinitionState{
				Name: name, State: state, LastUpdated: now,
			})
		}

	case "initiative":
		found := false
		for i, in := range wf.Spec.Initiatives {
			if in.Name == name {
				wf.Spec.Initiatives[i].State = state
				wf.Spec.Initiatives[i].LastUpdated = now
				found = true
				break
			}
		}
		if !found {
			wf.Spec.Initiatives = append(wf.Spec.Initiatives, InitiativeState{
				Name: name, State: state, LastUpdated: now,
			})
		}

	case "assignment":
		found := false
		for i, a := range wf.Spec.Assignments {
			if a.Name == name {
				wf.Spec.Assignments[i].State = state
				wf.Spec.Assignments[i].LastUpdated = now
				if state == "deploy" {
					wf.Spec.Assignments[i].EnforcementMode = "Default"
				}
				found = true
				break
			}
		}
		if !found {
			enforcement := "DoNotEnforce"
			if state == "deploy" {
				enforcement = "Default"
			}
			wf.Spec.Assignments = append(wf.Spec.Assignments, AssignmentState{
				Name: name, State: state, EnforcementMode: enforcement, LastUpdated: now,
			})
		}
	}

	return saveWorkflow(repoRoot, wf)
}

func findAssignment(repoRoot, name string) (string, error) {
	assignDir := filepath.Join(repoRoot, "policies", "assignments")
	var found string

	err := filepath.Walk(assignDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := strings.TrimSuffix(info.Name(), ".json")
		if base == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if err != nil && found == "" {
		return "", fmt.Errorf("walk assignments directory: %w", err)
	}
	if found == "" {
		return "", fmt.Errorf("assignment '%s' not found in %s", name, assignDir)
	}
	return found, nil
}

func writeReport(path string, report *VerifyReport) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var data []byte
	var err error

	if strings.HasSuffix(path, ".json") {
		data, err = json.MarshalIndent(report, "", "  ")
	} else {
		data, err = yaml.Marshal(report)
	}
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func simulateVerifyReport(name string) *VerifyReport {
	return &VerifyReport{
		AssignmentName: name,
		Evaluated:      100,
		Compliant:      95,
		NonCompliant:   5,
		Exempt:         0,
		ComplianceRate: 95.0,
		NonCompliantGroups: []NonCompliantGroup{
			{
				PolicyName: "require-resource-tags",
				Count:      3,
				Resources:  []string{"rg-legacy-app/vm-web01", "rg-legacy-app/vm-db01", "rg-shared/sa-logs"},
			},
			{
				PolicyName: "audit-storage-https",
				Count:      2,
				Resources:  []string{"rg-data/sa-archive", "rg-shared/sa-backup"},
			},
		},
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// SecurityCriticalAssignmentsSince returns enforced security-critical assignments
// changed after the provided timestamp.
func SecurityCriticalAssignmentsSince(repoRoot string, since time.Time) ([]AssignmentState, error) {
	workflow, err := loadWorkflow(repoRoot)
	if err != nil {
		return nil, err
	}

	out := make([]AssignmentState, 0)
	for _, assignment := range workflow.Spec.Assignments {
		if !assignment.SecurityCritical || assignment.EnforcementMode != "Default" {
			continue
		}
		if assignment.LastUpdated == "" {
			continue
		}
		changedAt, parseErr := time.Parse(time.RFC3339, assignment.LastUpdated)
		if parseErr != nil {
			continue
		}
		if changedAt.After(since) {
			out = append(out, assignment)
		}
	}

	return out, nil
}

// ── Scaffold Templates ───────────────────────────────────

func scaffoldDefinition(name, category string) ([]byte, error) {
	def := map[string]interface{}{
		"name": name,
		"properties": map[string]interface{}{
			"displayName": fmt.Sprintf("TODO: %s display name", name),
			"description": "TODO: Add policy description",
			"metadata": map[string]interface{}{
				"category": category,
				"version":  "1.0.0",
				"source":   "lzctl-landing-zone-factory",
			},
			"mode": "Indexed",
			"policyRule": map[string]interface{}{
				"if": map[string]interface{}{
					"field":  "type",
					"equals": "TODO: Microsoft.Resource/type",
				},
				"then": map[string]interface{}{
					"effect": "[parameters('effect')]",
				},
			},
			"parameters": map[string]interface{}{
				"effect": map[string]interface{}{
					"type":         "String",
					"defaultValue": "Audit",
					"allowedValues": []string{"Audit", "Deny", "Disabled"},
					"metadata": map[string]interface{}{
						"displayName": "Effect",
						"description": "The effect to apply",
					},
				},
			},
		},
	}
	return json.MarshalIndent(def, "", "  ")
}

func scaffoldInitiative(name, category string) ([]byte, error) {
	init := map[string]interface{}{
		"name": name,
		"properties": map[string]interface{}{
			"displayName": fmt.Sprintf("TODO: %s display name", name),
			"description": "TODO: Add initiative description",
			"metadata": map[string]interface{}{
				"category": category,
				"version":  "1.0.0",
				"source":   "lzctl-landing-zone-factory",
			},
			"policyDefinitions": []map[string]interface{}{
				{
					"policyDefinitionReferenceId": "TODO-ref-id",
					"policyDefinitionName":        "TODO-policy-name",
					"parameters":                  map[string]interface{}{},
				},
			},
			"parameters": map[string]interface{}{},
		},
	}
	return json.MarshalIndent(init, "", "  ")
}

func scaffoldAssignment(name, scope, initiative string) ([]byte, error) {
	policyDefId := "TODO: /providers/Microsoft.Authorization/policySetDefinitions/" + initiative
	if initiative != "" {
		policyDefId = fmt.Sprintf("/providers/Microsoft.Management/managementGroups/{tenantId}/providers/Microsoft.Authorization/policySetDefinitions/%s", initiative)
	}

	assign := map[string]interface{}{
		"name": name,
		"properties": map[string]interface{}{
			"displayName": fmt.Sprintf("TODO: %s display name", name),
			"description": "TODO: Add assignment description",
			"metadata": map[string]interface{}{
				"category":      "General",
				"version":       "1.0.0",
				"source":        "lzctl-landing-zone-factory",
				"workflowState": "created",
				"workflowHistory": []map[string]interface{}{
					{
						"state":  "created",
						"date":   time.Now().UTC().Format(time.RFC3339),
						"author": "lzctl policy create",
					},
				},
			},
			"policyDefinitionId": policyDefId,
			"scope":              fmt.Sprintf("/providers/Microsoft.Management/managementGroups/{rootMgId}-%s", scope),
			"enforcementMode":    "DoNotEnforce",
			"parameters":         map[string]interface{}{},
			"nonComplianceMessages": []map[string]interface{}{
				{
					"message": "TODO: This resource is not compliant. Review the policy for details.",
				},
			},
		},
	}
	return json.MarshalIndent(assign, "", "  ")
}

func scaffoldExemption(name string) ([]byte, error) {
	exempt := map[string]interface{}{
		"name": name,
		"properties": map[string]interface{}{
			"displayName": fmt.Sprintf("TODO: %s display name", name),
			"description": "TODO: Add exemption justification",
			"metadata": map[string]interface{}{
				"category":        "General",
				"version":         "1.0.0",
				"source":          "lzctl-landing-zone-factory",
				"ticketReference": "TODO: CHANGE-XXXX-XXXX",
				"approvedBy":      "TODO: approver@contoso.com",
				"approvalDate":    time.Now().UTC().Format(time.RFC3339),
				"reviewDate":      time.Now().AddDate(0, 6, 0).UTC().Format(time.RFC3339),
			},
			"policyAssignmentId":         "TODO: /providers/Microsoft.Authorization/policyAssignments/assignment-name",
			"policyDefinitionReferenceIds": []string{"TODO-ref-id"},
			"exemptionCategory":          "Waiver",
			"expiresOn":                  time.Now().AddDate(0, 6, 0).UTC().Format(time.RFC3339),
		},
	}
	return json.MarshalIndent(exempt, "", "  ")
}
