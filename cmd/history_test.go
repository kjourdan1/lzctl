package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kjourdan1/lzctl/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryCmd_WithEvents_NoError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	auditDir := filepath.Join(home, ".lzctl")
	require.NoError(t, os.MkdirAll(auditDir, 0o755))

	event := audit.Event{
		Timestamp:  "2026-02-19T10:00:00Z",
		Operation:  "plan",
		Tenant:     "tenant-a",
		Args:       []string{"lzctl", "plan"},
		Result:     "success",
		ExitCode:   0,
		DurationMs: 120,
	}
	b, err := json.Marshal(event)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(auditDir, "audit.log"), append(b, '\n'), 0o644))

	_, stderr, runErr := executeCommandWithProcessIO(t, "history", "--limit", "1")
	assert.NoError(t, runErr)
	assert.Contains(t, stderr, "op=plan")
	assert.Contains(t, stderr, "tenant=tenant-a")
}

func TestHistoryCmd_TenantFilter_NoError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	auditDir := filepath.Join(home, ".lzctl")
	require.NoError(t, os.MkdirAll(auditDir, 0o755))

	events := []audit.Event{
		{
			Timestamp:  "2026-02-19T10:00:00Z",
			Operation:  "plan",
			Tenant:     "tenant-a",
			Args:       []string{"lzctl", "plan"},
			Result:     "success",
			ExitCode:   0,
			DurationMs: 120,
		},
		{
			Timestamp:  "2026-02-19T10:05:00Z",
			Operation:  "apply",
			Tenant:     "tenant-b",
			Args:       []string{"lzctl", "apply"},
			Result:     "failure",
			ExitCode:   1,
			DurationMs: 250,
		},
	}

	logPath := filepath.Join(auditDir, "audit.log")
	for _, event := range events {
		b, err := json.Marshal(event)
		require.NoError(t, err)
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		require.NoError(t, err)
		_, err = f.Write(append(b, '\n'))
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	_, stderr, runErr := executeCommandWithProcessIO(t, "history", "--tenant", "tenant-b", "--limit", "10")
	assert.NoError(t, runErr)
	assert.Contains(t, stderr, "tenant=tenant-b")
	assert.NotContains(t, stderr, "tenant=tenant-a")
}

func TestHistoryCmd_TenantFilter_NoMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	auditDir := filepath.Join(home, ".lzctl")
	require.NoError(t, os.MkdirAll(auditDir, 0o755))

	event := audit.Event{
		Timestamp:  "2026-02-19T10:00:00Z",
		Operation:  "plan",
		Tenant:     "tenant-a",
		Args:       []string{"lzctl", "plan"},
		Result:     "success",
		ExitCode:   0,
		DurationMs: 120,
	}
	b, err := json.Marshal(event)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(auditDir, "audit.log"), append(b, '\n'), 0o644))

	_, stderr, runErr := executeCommandWithProcessIO(t, "history", "--tenant", "tenant-z", "--limit", "10")
	assert.NoError(t, runErr)
	assert.Contains(t, stderr, "No matching audit events")
}
