package doctor

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutor is a test double for CmdExecutor.
type mockExecutor struct {
	// responses maps "command arg1 arg2" → (output, error)
	responses map[string]mockResponse
}

type mockResponse struct {
	output string
	err    error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{responses: make(map[string]mockResponse)}
}

func (m *mockExecutor) Set(output string, err error, name string, args ...string) {
	key := buildKey(name, args...)
	m.responses[key] = mockResponse{output: output, err: err}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	key := buildKey(name, args...)
	r, ok := m.responses[key]
	if !ok {
		return "", fmt.Errorf("command not found: %s", key)
	}
	return r.output, r.err
}

func buildKey(name string, args ...string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + joinArgs(args)
}

func joinArgs(args []string) string {
	s := ""
	for i, a := range args {
		if i > 0 {
			s += " "
		}
		s += a
	}
	return s
}

// --- Semver tests ---

func TestSemverGTE(t *testing.T) {
	tests := []struct {
		version string
		min     string
		want    bool
	}{
		{"1.5.0", "1.5.0", true},
		{"1.6.0", "1.5.0", true},
		{"2.0.0", "1.5.0", true},
		{"1.4.9", "1.5.0", false},
		{"1.5.1", "1.5.0", true},
		{"0.9.0", "1.0.0", false},
		{"1.5.0-rc1", "1.5.0", true},
		{"2.50.0", "2.50.0", true},
		{"2.51.0", "2.50.0", true},
		{"2.49.0", "2.50.0", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s>=%s", tt.version, tt.min), func(t *testing.T) {
			assert.Equal(t, tt.want, semverGTE(tt.version, tt.min))
		})
	}
}

// --- Terraform check tests ---

func TestCheckTerraform_Pass(t *testing.T) {
	ex := newMockExecutor()
	ex.Set(`{"terraform_version": "1.9.2", "platform": "windows_amd64"}`, nil,
		"terraform", "version", "-json")

	check := checkTerraform()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusPass, r.Status)
	assert.Contains(t, r.Message, "1.9.2")
}

func TestCheckTerraform_TooOld(t *testing.T) {
	ex := newMockExecutor()
	ex.Set(`{"terraform_version": "1.3.0"}`, nil,
		"terraform", "version", "-json")

	check := checkTerraform()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
	assert.Contains(t, r.Message, "1.3.0")
	assert.Contains(t, r.Message, ">= 1.5.0")
	assert.NotEmpty(t, r.Fix)
}

func TestCheckTerraform_NotFound(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("", errors.New("exec: not found"),
		"terraform", "version", "-json")

	check := checkTerraform()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
	assert.Contains(t, r.Message, "not found")
}

// --- Azure CLI check tests ---

func TestCheckAzCLI_Pass(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("2.64.0\ncore\t2.64.0\ntelemetry\t1.0.0", nil,
		"az", "version", "--output", "tsv")

	check := checkAzCLI()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusPass, r.Status)
	assert.Contains(t, r.Message, "2.64.0")
}

func TestCheckAzCLI_TooOld(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("2.45.0\ncore\t2.45.0", nil,
		"az", "version", "--output", "tsv")

	check := checkAzCLI()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
	assert.Contains(t, r.Message, "2.45.0")
}

// --- Git check tests ---

func TestCheckGit_Pass(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("git version 2.43.0.windows.1", nil,
		"git", "version")

	check := checkGit()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusPass, r.Status)
	assert.Contains(t, r.Message, "2.43.0")
}

func TestCheckGit_TooOld(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("git version 2.25.1", nil,
		"git", "version")

	check := checkGit()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
}

// --- GH CLI check tests ---

func TestCheckGH_Present(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("gh version 2.45.0 (2024-03-01)", nil,
		"gh", "version")

	check := checkGH()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusPass, r.Status)
	assert.Contains(t, r.Message, "2.45.0")
}

func TestCheckGH_Missing(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("", errors.New("exec: not found"),
		"gh", "version")

	check := checkGH()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusWarn, r.Status)
	assert.Contains(t, r.Message, "optional")
}

// --- Azure session check tests ---

func TestCheckAzSession_LoggedIn(t *testing.T) {
	ex := newMockExecutor()
	accountJSON := `{
		"id": "00000000-1111-2222-3333-444444444444",
		"tenantId": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		"name": "My Subscription",
		"state": "Enabled"
	}`
	ex.Set(accountJSON, nil,
		"az", "account", "show", "--output", "json")

	check := checkAzSession()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusPass, r.Status)
	assert.Contains(t, r.Message, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	assert.Contains(t, r.Message, "00000000-1111-2222-3333-444444444444")
}

func TestCheckAzSession_NotLoggedIn(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("", errors.New("AADSTS error"),
		"az", "account", "show", "--output", "json")

	check := checkAzSession()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
	assert.Contains(t, r.Fix, "az login")
}

// --- Management group access tests ---

func TestCheckMGAccess_Pass(t *testing.T) {
	ex := newMockExecutor()
	ex.Set(`[{"name": "root-mg"}]`, nil,
		"az", "account", "management-group", "list", "--no-register")

	check := checkAzManagementGroups()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusPass, r.Status)
}

func TestCheckMGAccess_Fail(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("", errors.New("authorization failed"),
		"az", "account", "management-group", "list", "--no-register")

	check := checkAzManagementGroups()
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
	assert.Contains(t, r.Fix, "Management Group Reader")
}

// --- Resource provider check tests ---

func TestCheckResourceProvider_Registered(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("Registered", nil,
		"az", "provider", "show", "-n", "Microsoft.Management", "--query", "registrationState", "-o", "tsv")

	check := checkResourceProvider("Microsoft.Management")
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusPass, r.Status)
	assert.Contains(t, r.Message, "registered")
}

func TestCheckResourceProvider_NotRegistered(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("NotRegistered", nil,
		"az", "provider", "show", "-n", "Microsoft.Network", "--query", "registrationState", "-o", "tsv")

	check := checkResourceProvider("Microsoft.Network")
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
	assert.Contains(t, r.Fix, "az provider register")
}

func TestCheckResourceProvider_Error(t *testing.T) {
	ex := newMockExecutor()
	ex.Set("", errors.New("not logged in"),
		"az", "provider", "show", "-n", "Microsoft.Authorization", "--query", "registrationState", "-o", "tsv")

	check := checkResourceProvider("Microsoft.Authorization")
	r := check.Run(context.Background(), ex)

	assert.Equal(t, StatusFail, r.Status)
}

// --- RunAll integration test ---

func TestRunAll_AllPass(t *testing.T) {
	ex := newMockExecutor()
	// Tools
	ex.Set(`{"terraform_version": "1.9.0"}`, nil, "terraform", "version", "-json")
	ex.Set("2.64.0\ncore\t2.64.0", nil, "az", "version", "--output", "tsv")
	ex.Set("git version 2.43.0", nil, "git", "version")
	ex.Set("gh version 2.45.0", nil, "gh", "version")
	// Auth
	ex.Set(`{"id": "sub-id", "tenantId": "tenant-id", "name": "MySub"}`, nil,
		"az", "account", "show", "--output", "json")
	// Azure
	ex.Set(`[{"name": "root"}]`, nil,
		"az", "account", "management-group", "list", "--no-register")
	ex.Set("Registered", nil, "az", "provider", "show", "-n", "Microsoft.Management", "--query", "registrationState", "-o", "tsv")
	ex.Set("Registered", nil, "az", "provider", "show", "-n", "Microsoft.Authorization", "--query", "registrationState", "-o", "tsv")
	ex.Set("Registered", nil, "az", "provider", "show", "-n", "Microsoft.Network", "--query", "registrationState", "-o", "tsv")
	ex.Set("Registered", nil, "az", "provider", "show", "-n", "Microsoft.ManagedIdentity", "--query", "registrationState", "-o", "tsv")

	summary := RunAll(context.Background(), ex)

	require.Len(t, summary.Results, 10)
	assert.Equal(t, 10, summary.TotalPass)
	assert.Equal(t, 0, summary.TotalFail)
	assert.False(t, summary.HasFailure)
}

func TestRunAll_SomeFailures(t *testing.T) {
	ex := newMockExecutor()
	// Terraform missing
	ex.Set("", errors.New("not found"), "terraform", "version", "-json")
	// Az CLI OK
	ex.Set("2.64.0\ncore\t2.64.0", nil, "az", "version", "--output", "tsv")
	// Git too old
	ex.Set("git version 2.20.0", nil, "git", "version")
	// GH missing (warning only)
	ex.Set("", errors.New("not found"), "gh", "version")
	// Not logged in
	ex.Set("", errors.New("AADSTS"), "az", "account", "show", "--output", "json")
	// MG access denied
	ex.Set("", errors.New("denied"), "az", "account", "management-group", "list", "--no-register")
	// Providers
	ex.Set("Registered", nil, "az", "provider", "show", "-n", "Microsoft.Management", "--query", "registrationState", "-o", "tsv")
	ex.Set("NotRegistered", nil, "az", "provider", "show", "-n", "Microsoft.Authorization", "--query", "registrationState", "-o", "tsv")
	ex.Set("Registered", nil, "az", "provider", "show", "-n", "Microsoft.Network", "--query", "registrationState", "-o", "tsv")
	ex.Set("Registered", nil, "az", "provider", "show", "-n", "Microsoft.ManagedIdentity", "--query", "registrationState", "-o", "tsv")

	summary := RunAll(context.Background(), ex)

	require.Len(t, summary.Results, 10)
	assert.True(t, summary.HasFailure)
	assert.Equal(t, 1, summary.TotalWarn)  // gh only
	assert.True(t, summary.TotalFail >= 4) // terraform, git, session, mg-access, authorization
}

// --- extractJSONField tests ---

func TestExtractJSONField(t *testing.T) {
	json := `{"tenantId": "abc-123", "id": "sub-456", "name": "MySub"}`
	assert.Equal(t, "abc-123", extractJSONField(json, "tenantId"))
	assert.Equal(t, "sub-456", extractJSONField(json, "id"))
	assert.Equal(t, "MySub", extractJSONField(json, "name"))
	assert.Equal(t, "unknown", extractJSONField(json, "missing"))
}

// --- StatusIcon tests ---

func TestStatusIcon(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	assert.Equal(t, "[PASS]", StatusIcon(StatusPass))
	assert.Equal(t, "[FAIL]", StatusIcon(StatusFail))
	assert.Equal(t, "[WARN]", StatusIcon(StatusWarn))
	assert.Equal(t, "[SKIP]", StatusIcon(StatusSkip))
}
