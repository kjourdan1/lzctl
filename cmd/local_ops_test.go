package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlanSummary(t *testing.T) {
	add, change, destroy := parsePlanSummary("Plan: 3 to add, 2 to change, 1 to destroy")
	assert.Equal(t, 3, add)
	assert.Equal(t, 2, change)
	assert.Equal(t, 1, destroy)
}

func TestResolveLocalLayers_All(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "management-groups"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "identity"), 0o755))

	layers, err := resolveLocalLayers(root, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"management-groups", "identity"}, layers)
}

func TestResolveLocalLayers_Target(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "governance"), 0o755))

	layers, err := resolveLocalLayers(root, "governance")
	require.NoError(t, err)
	assert.Equal(t, []string{"governance"}, layers)
}

func TestResolveLocalLayers_AllCAFOrder(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "connectivity"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "management"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "identity"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "governance"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform", "management-groups"), 0o755))

	layers, err := resolveLocalLayers(root, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"management-groups", "identity", "management", "governance", "connectivity"}, layers)
}

func TestResolveLocalLayers_NoPlatformLayers_ReturnsError(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "platform"), 0o755))

	layers, err := resolveLocalLayers(root, "")
	assert.Nil(t, layers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no platform layers found")
}
