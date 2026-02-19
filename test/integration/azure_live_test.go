//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func requireLiveAzure(t *testing.T) {
	t.Helper()
	if os.Getenv("AZURE_TENANT_ID") == "" || os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("live Azure integration test skipped: missing AZURE_TENANT_ID or AZURE_SUBSCRIPTION_ID")
	}
}

func TestLiveDoctor_AzureSession(t *testing.T) {
	requireLiveAzure(t)

	cmd := exec.Command("az", "account", "show", "--output", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("az account show failed: %v, output: %s", err, string(out))
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("invalid account json: %v", err)
	}
	if payload["tenantId"] == nil || payload["id"] == nil {
		t.Fatalf("missing tenantId/id in az account show output: %s", string(out))
	}
}

func TestLiveDrift_NoChanges(t *testing.T) {
	requireLiveAzure(t)

	cmd := exec.Command("az", "resource", "list", "--top", "1", "--output", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("az resource list failed: %v, output: %s", err, string(out))
	}

	var resources []map[string]any
	if err := json.Unmarshal(out, &resources); err != nil {
		t.Fatalf("invalid resource list json: %v", err)
	}
}
