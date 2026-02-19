package policy

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityCriticalAssignmentsSince(t *testing.T) {
	tmpDir := t.TempDir()
	policiesDir := filepath.Join(tmpDir, "policies")
	require.NoError(t, os.MkdirAll(policiesDir, 0o755))

	workflow := `apiVersion: lzctl/v1
kind: PolicyWorkflow
metadata:
  name: policy-workflow
spec:
  definitions: []
  initiatives: []
  assignments:
    - name: deny-public-ip
      scope: /providers/Microsoft.Management/managementGroups/root
      state: deploy
      enforcementMode: Default
      securityCritical: true
      incidentTicket: INC-1234
      lastUpdated: 2026-02-18T12:00:00Z
      compliance:
        evaluated: 0
        compliant: 0
        nonCompliant: 0
        exempt: 0
        lastScan: ""
      remediationTasks: []
    - name: audit-storage-https
      scope: /providers/Microsoft.Management/managementGroups/root
      state: deploy
      enforcementMode: DoNotEnforce
      securityCritical: true
      lastUpdated: 2026-02-18T12:01:00Z
      compliance:
        evaluated: 0
        compliant: 0
        nonCompliant: 0
        exempt: 0
        lastScan: ""
      remediationTasks: []
  exemptions: []
`
	require.NoError(t, os.WriteFile(filepath.Join(policiesDir, "workflow.yaml"), []byte(workflow), 0o644))

	since := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	items, err := SecurityCriticalAssignmentsSince(tmpDir, since)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "deny-public-ip", items[0].Name)
	assert.Equal(t, "INC-1234", items[0].IncidentTicket)
}
