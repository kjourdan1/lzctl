package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// Reset state
	Init(false, false)
	assert.False(t, Verbose)
	assert.False(t, JSONMode)

	Init(true, true)
	assert.True(t, Verbose)
	assert.True(t, JSONMode)

	// Clean up
	Init(false, false)
}

func TestNoColor(t *testing.T) {
	// Save original
	orig, hadOrig := os.LookupEnv("NO_COLOR")
	defer func() {
		if hadOrig {
			os.Setenv("NO_COLOR", orig)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	}()

	os.Unsetenv("NO_COLOR")
	assert.False(t, NoColor())

	os.Setenv("NO_COLOR", "1")
	assert.True(t, NoColor())

	os.Setenv("NO_COLOR", "")
	assert.True(t, NoColor()) // any value, even empty, means no color
}

func TestJSONResult(t *testing.T) {
	tests := []struct {
		name     string
		result   JSONResult
		wantKeys []string
	}{
		{
			name:     "ok with data",
			result:   JSONResult{Status: "ok", Data: map[string]string{"key": "value"}},
			wantKeys: []string{"status", "data"},
		},
		{
			name:     "error",
			result:   JSONResult{Status: "error", Error: "something failed"},
			wantKeys: []string{"status", "error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			err := enc.Encode(tt.result)
			require.NoError(t, err)

			var decoded map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &decoded)
			require.NoError(t, err)

			for _, key := range tt.wantKeys {
				assert.Contains(t, decoded, key)
			}
			assert.Equal(t, tt.result.Status, decoded["status"])
		})
	}
}

func TestCLIError(t *testing.T) {
	t.Run("simple error", func(t *testing.T) {
		err := NewError("something broke")
		assert.Equal(t, "something broke", err.Error())
		assert.Nil(t, err.Unwrap())
		assert.Empty(t, err.Fix)
	})

	t.Run("error with fix", func(t *testing.T) {
		err := NewErrorWithFix("terraform not found", "Install terraform: https://terraform.io")
		assert.Equal(t, "terraform not found", err.Error())
		assert.Equal(t, "Install terraform: https://terraform.io", err.Fix)
	})

	t.Run("wrapped error", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := WrapError(cause, "failed to connect to Azure")
		assert.Equal(t, "failed to connect to Azure: connection refused", err.Error())
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("wrapped error with fix", func(t *testing.T) {
		cause := errors.New("401 unauthorized")
		err := WrapErrorWithFix(cause, "Azure authentication failed", "Run: az login")
		assert.Equal(t, "Azure authentication failed: 401 unauthorized", err.Error())
		assert.Equal(t, "Run: az login", err.Fix)
		assert.ErrorIs(t, err, cause)
	})
}

func TestSpinner(t *testing.T) {
	t.Run("stop is idempotent", func(t *testing.T) {
		sp := NewSpinner("test")
		sp.Start()
		sp.Stop()
		sp.Stop() // should not panic
	})

	t.Run("start is idempotent", func(t *testing.T) {
		sp := NewSpinner("test")
		sp.Start()
		sp.Start() // should not panic
		sp.Stop()
	})

	t.Run("json mode suppresses spinner", func(t *testing.T) {
		origJSON := JSONMode
		JSONMode = true
		defer func() { JSONMode = origJSON }()

		sp := NewSpinner("test")
		sp.Start() // should be a no-op
		sp.Stop()
	})
}

func TestWithSpinner(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		Init(false, true) // JSON mode to suppress output
		defer Init(false, false)

		err := WithSpinner("testing", func() error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		Init(false, true)
		defer Init(false, false)

		expectedErr := errors.New("boom")
		err := WithSpinner("testing", func() error {
			return expectedErr
		})
		assert.Equal(t, expectedErr, err)
	})
}
