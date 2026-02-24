package planverify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSign_And_Verify(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "tfplan")
	content := []byte("fake terraform plan binary content")
	if err := os.WriteFile(planFile, content, 0o600); err != nil {
		t.Fatal(err)
	}

	// Sign
	hash, err := Sign(planFile)
	if err != nil {
		t.Fatalf("Sign() error: %v", err)
	}

	// Verify expected hash
	h := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(h[:])
	if hash != expectedHash {
		t.Errorf("Sign() hash = %s, want %s", hash, expectedHash)
	}

	// Sig file should exist
	sigFile := SigFile(planFile)
	if _, err := os.Stat(sigFile); os.IsNotExist(err) {
		t.Fatal("sig file not created")
	}

	// Verify should pass
	result, err := Verify(planFile)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !result.Valid {
		t.Error("Verify() result.Valid = false, want true")
	}
	if result.ExpectedHash != result.ActualHash {
		t.Errorf("hash mismatch: expected=%s, actual=%s", result.ExpectedHash, result.ActualHash)
	}
}

func TestVerify_TamperedPlan(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "tfplan")
	if err := os.WriteFile(planFile, []byte("original content"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Sign(planFile); err != nil {
		t.Fatalf("Sign() error: %v", err)
	}

	// Tamper with the plan file
	if err := os.WriteFile(planFile, []byte("tampered content"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Verify(planFile)
	if err == nil {
		t.Fatal("Verify() should return error for tampered plan")
	}
	if result.Valid {
		t.Error("Verify() result.Valid = true for tampered plan, want false")
	}
}

func TestVerify_MissingSigFile(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "tfplan")
	if err := os.WriteFile(planFile, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Don't sign — no sig file
	_, err := Verify(planFile)
	if err == nil {
		t.Fatal("Verify() should error when sig file is missing")
	}
}

func TestSigFile(t *testing.T) {
	got := SigFile("/tmp/plan.tfplan")
	want := "/tmp/plan.tfplan.sha256"
	if got != want {
		t.Errorf("SigFile() = %s, want %s", got, want)
	}
}

func TestParsePlanScopeViolations_NoViolations(t *testing.T) {
	plan := makePlanJSON([]string{"sub-a"}, map[string]string{
		"azurerm_resource_group.rg": "sub-a",
	})
	allowed := map[string]bool{"sub-a": true}

	violations, err := parsePlanScopeViolations(plan, allowed)
	if err != nil {
		t.Fatalf("parsePlanScopeViolations() error: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestParsePlanScopeViolations_WithViolation(t *testing.T) {
	plan := makePlanJSON(nil, map[string]string{
		"azurerm_resource_group.rg": "rogue-sub-id",
	})
	allowed := map[string]bool{"sub-a": true}

	violations, err := parsePlanScopeViolations(plan, allowed)
	if err != nil {
		t.Fatalf("parsePlanScopeViolations() error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].SubscriptionID != "rogue-sub-id" {
		t.Errorf("violation sub = %s, want rogue-sub-id", violations[0].SubscriptionID)
	}
}

func TestParsePlanScopeViolations_ProviderViolation(t *testing.T) {
	plan := makePlanJSON([]string{"rogue-sub"}, nil)
	allowed := map[string]bool{"sub-a": true}

	violations, err := parsePlanScopeViolations(plan, allowed)
	if err != nil {
		t.Fatalf("parsePlanScopeViolations() error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 provider violation, got %d", len(violations))
	}
	if violations[0].ResourceAddr != "(provider config)" {
		t.Errorf("violation addr = %s, want (provider config)", violations[0].ResourceAddr)
	}
}

func TestParsePlanScopeViolations_ChildModules(t *testing.T) {
	planData := map[string]interface{}{
		"planned_values": map[string]interface{}{
			"root_module": map[string]interface{}{
				"resources": []interface{}{},
				"child_modules": []interface{}{
					map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"address": "module.child.azurerm_rg.rg",
								"values": map[string]interface{}{
									"subscription_id": "rogue-child-sub",
								},
							},
						},
					},
				},
			},
		},
		"configuration": map[string]interface{}{
			"provider_config": map[string]interface{}{},
		},
	}
	data, _ := json.Marshal(planData)
	allowed := map[string]bool{"sub-a": true}

	violations, err := parsePlanScopeViolations(data, allowed)
	if err != nil {
		t.Fatalf("parsePlanScopeViolations() error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 child module violation, got %d", len(violations))
	}
	if violations[0].SubscriptionID != "rogue-child-sub" {
		t.Errorf("violation sub = %s, want rogue-child-sub", violations[0].SubscriptionID)
	}
}

func TestCollectResources_Recursive(t *testing.T) {
	mod := planModule{
		Resources: []planResource{{Address: "root.res"}},
		ChildModules: []planModule{
			{
				Resources: []planResource{{Address: "child.res"}},
				ChildModules: []planModule{
					{Resources: []planResource{{Address: "grandchild.res"}}},
				},
			},
		},
	}

	resources := collectResources(mod)
	if len(resources) != 3 {
		t.Errorf("collectResources() = %d resources, want 3", len(resources))
	}
}

func TestParsePlanScopeViolations_InvalidJSON(t *testing.T) {
	_, err := parsePlanScopeViolations([]byte("not json"), map[string]bool{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// makePlanJSON builds a minimal terraform plan JSON for testing.
func makePlanJSON(providerSubs []string, resourceSubs map[string]string) []byte {
	providerConfig := map[string]interface{}{}
	for i, sub := range providerSubs {
		key := "azurerm"
		if i > 0 {
			key = "azurerm." + sub
		}
		providerConfig[key] = map[string]interface{}{
			"expressions": map[string]interface{}{
				"subscription_id": map[string]interface{}{
					"constant_value": sub,
				},
			},
		}
	}

	resources := []interface{}{}
	for addr, sub := range resourceSubs {
		resources = append(resources, map[string]interface{}{
			"address": addr,
			"values":  map[string]interface{}{"subscription_id": sub},
		})
	}

	plan := map[string]interface{}{
		"planned_values": map[string]interface{}{
			"root_module": map[string]interface{}{
				"resources": resources,
			},
		},
		"configuration": map[string]interface{}{
			"provider_config": providerConfig,
		},
	}
	data, _ := json.Marshal(plan)
	return data
}
