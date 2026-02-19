package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRollbackTimestamp(t *testing.T) {
	ts, err := parseRollbackTimestamp("20260218-153045")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 2, 18, 15, 30, 45, 0, time.UTC), ts)

	ts, err = parseRollbackTimestamp("2026-02-18T15:30:45Z")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 2, 18, 15, 30, 45, 0, time.UTC), ts)

	_, err = parseRollbackTimestamp("bad")
	require.Error(t, err)
}
