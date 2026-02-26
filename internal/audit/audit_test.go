package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEvent_InfersFieldsFromArgs(t *testing.T) {
	event := BuildEvent([]string{"lzctl", "rollback", "--tenant", "contoso", "--ring", "wave1", "--repo-root", "C:/repo"}, "failure", 7, 1500*time.Millisecond)

	assert.Equal(t, "rollback", event.Operation)
	assert.Equal(t, "contoso", event.Tenant)
	assert.Equal(t, "wave1", event.Ring)
	assert.Equal(t, 7, event.ExitCode)
	assert.Equal(t, int64(1500), event.DurationMs)
	assert.Equal(t, "C:/repo", event.MetadataValue("repoRoot"))
}

func TestSanitize(t *testing.T) {
	assert.Equal(t, "operation", sanitize(""))
	assert.Equal(t, "policy-deploy", sanitize("policy/deploy"))
	assert.Equal(t, "apply-ring-wave1", sanitize("apply:ring wave1"))
}

func TestWriteTenantAudit_FileIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	event := BuildEvent([]string{"lzctl", "apply", "--tenant", "contoso", "--repo-root", dir}, "success", 0, time.Second)

	err := writeTenantAudit(dir, event)
	require.NoError(t, err)

	logDir := filepath.Join(dir, "tenants", "contoso", "logs")
	entries, err := os.ReadDir(logDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	info, err := entries[0].Info()
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o400), info.Mode().Perm(), "tenant audit file must be read-only (0o400)")
}
