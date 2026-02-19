package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
