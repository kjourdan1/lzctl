package oidcsetup

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockCommands sets up a command runner that returns canned responses.
func mockCommands(t *testing.T, responses map[string]struct {
	output string
	err    error
}) func() {
	t.Helper()
	old := GetCommandRunner()
	SetCommandRunner(func(name string, args ...string) ([]byte, error) {
		// Build a key from the command
		key := name + " " + strings.Join(args, " ")
		for pattern, resp := range responses {
			if strings.Contains(key, pattern) {
				return []byte(resp.output), resp.err
			}
		}
		// Default: return empty success
		return []byte("{}"), nil
	})
	return func() { SetCommandRunner(old) }
}

func TestParseGitHubRepo_HTTPS(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/kjourdan1/lzctl.git", "kjourdan1/lzctl"},
		{"https://github.com/kjourdan1/lzctl", "kjourdan1/lzctl"},
		{"https://github.com/org/my-repo.git", "org/my-repo"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := parseGitHubRepo(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseGitHubRepo_SSH(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:kjourdan1/lzctl.git", "kjourdan1/lzctl"},
		{"git@github.com:org/my-repo.git", "org/my-repo"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := parseGitHubRepo(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseGitHubRepo_Invalid(t *testing.T) {
	tests := []string{
		"https://gitlab.com/user/repo.git",
		"ftp://somewhere/repo",
		"not-a-url",
		"",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			_, err := parseGitHubRepo(tt)
			assert.Error(t, err)
		})
	}
}

func TestSetup_DryRun(t *testing.T) {
	result, err := Setup(Options{
		TenantID:       "test-tenant-id",
		TenantName:     "test",
		SubscriptionID: "test-sub-id",
		GitHubRepo:     "owner/repo",
		DryRun:         true,
	})
	assert.NoError(t, err)
	assert.Equal(t, "(dry-run)", result.AppID)
	assert.Contains(t, result.GitHubSecrets, "AZURE_CLIENT_ID")
	assert.Contains(t, result.GitHubSecrets, "AZURE_TENANT_ID")
	assert.Contains(t, result.GitHubSecrets, "AZURE_SUBSCRIPTION_ID")
	assert.Len(t, result.FederatedCredentials, 5)
}

func TestSetup_DetectsGitHubRepo(t *testing.T) {
	old := GetCommandRunner()
	defer SetCommandRunner(old)

	SetCommandRunner(func(name string, args ...string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")
		if strings.Contains(key, "git remote get-url") {
			return []byte("https://github.com/kjourdan1/lzctl.git\n"), nil
		}
		return []byte("{}"), nil
	})

	// Dry run to just test detection
	result, err := Setup(Options{
		TenantID:       "test-tenant-id",
		TenantName:     "test",
		SubscriptionID: "test-sub-id",
		DryRun:         true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetup_FullFlow(t *testing.T) {
	old := GetCommandRunner()
	defer SetCommandRunner(old)

	appJSON, _ := json.Marshal(appInfo{AppID: "app-123", ID: "obj-456"})

	SetCommandRunner(func(name string, args ...string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")

		switch {
		case strings.Contains(key, "az ad app list"):
			// No existing app
			return []byte("null"), nil

		case strings.Contains(key, "az ad app create"):
			return appJSON, nil

		case strings.Contains(key, "az ad sp show"):
			return []byte(""), fmt.Errorf("not found")

		case strings.Contains(key, "az ad sp create"):
			return []byte("sp-789\n"), nil

		case strings.Contains(key, "federated-credential create"):
			return []byte("{}"), nil

		case strings.Contains(key, "az role assignment create"):
			return []byte("{}"), nil

		case strings.Contains(key, "gh secret set"):
			return []byte(""), nil

		default:
			return []byte("{}"), nil
		}
	})

	result, err := Setup(Options{
		TenantID:       "tenant-id-123",
		TenantName:     "mytest",
		SubscriptionID: "sub-id-456",
		GitHubRepo:     "owner/repo",
		Verbose:        true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "app-123", result.AppID)
	assert.Equal(t, "obj-456", result.ObjectID)
	assert.Equal(t, "sp-789", result.ServicePrincipalID)
	assert.Len(t, result.FederatedCredentials, 5)
	assert.NotEmpty(t, result.RoleAssignments)
	assert.NotEmpty(t, result.GitHubSecrets)
}

func TestSetup_ReusesExistingApp(t *testing.T) {
	old := GetCommandRunner()
	defer SetCommandRunner(old)

	appJSON, _ := json.Marshal(appInfo{AppID: "existing-app", ID: "existing-obj"})

	SetCommandRunner(func(name string, args ...string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")

		switch {
		case strings.Contains(key, "az ad app list"):
			return appJSON, nil

		case strings.Contains(key, "az ad sp show"):
			return []byte("existing-sp\n"), nil

		case strings.Contains(key, "federated-credential create"):
			return []byte("{}"), nil

		case strings.Contains(key, "az role assignment create"):
			return []byte("{}"), nil

		case strings.Contains(key, "gh secret set"):
			return []byte(""), nil

		default:
			return []byte("{}"), nil
		}
	})

	result, err := Setup(Options{
		TenantID:       "tid",
		TenantName:     "test",
		SubscriptionID: "sid",
		GitHubRepo:     "o/r",
		Verbose:        true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "existing-app", result.AppID)
	assert.Equal(t, "existing-sp", result.ServicePrincipalID)
}

func TestSetup_FederatedCredentialAlreadyExists(t *testing.T) {
	old := GetCommandRunner()
	defer SetCommandRunner(old)

	appJSON, _ := json.Marshal(appInfo{AppID: "app-1", ID: "obj-1"})

	SetCommandRunner(func(name string, args ...string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")

		switch {
		case strings.Contains(key, "az ad app list"):
			return appJSON, nil

		case strings.Contains(key, "az ad sp show"):
			return []byte("sp-1\n"), nil

		case strings.Contains(key, "federated-credential create"):
			return []byte("FederatedIdentityCredentialAlreadyExists"), fmt.Errorf("already exists")

		case strings.Contains(key, "az role assignment create"):
			return []byte("{}"), nil

		case strings.Contains(key, "gh secret set"):
			return []byte(""), nil

		default:
			return []byte("{}"), nil
		}
	})

	result, err := Setup(Options{
		TenantID:       "tid",
		TenantName:     "test",
		SubscriptionID: "sid",
		GitHubRepo:     "o/r",
	})

	assert.NoError(t, err)
	// All 5 federated creds should show as existing
	for _, fc := range result.FederatedCredentials {
		assert.Contains(t, fc, "(existing)")
	}
}

func TestSetup_NoGitHubRepo_Fails(t *testing.T) {
	old := GetCommandRunner()
	defer SetCommandRunner(old)

	SetCommandRunner(func(name string, args ...string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")
		if strings.Contains(key, "git remote") {
			return []byte(""), fmt.Errorf("no remote")
		}
		return []byte("{}"), nil
	})

	_, err := Setup(Options{
		TenantID:   "tid",
		TenantName: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot detect GitHub repo")
}
