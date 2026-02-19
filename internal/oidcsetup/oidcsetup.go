// Package oidcsetup automates the creation of an Azure AD App Registration
// with GitHub Actions OIDC federation, RBAC role assignments, and GitHub
// repository secrets — so that PR validation workflows can authenticate
// to Azure without storing long-lived credentials.
//
// It shells out to `az` (Azure CLI) and `gh` (GitHub CLI), both of which
// must be installed and authenticated before calling Setup().
package oidcsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// Options configures the OIDC setup.
type Options struct {
	TenantID       string // Azure AD tenant ID
	TenantName     string // lzctl tenant name (used in app display name)
	SubscriptionID string // Azure subscription ID for RBAC + GitHub secret
	GitHubRepo     string // owner/repo — auto-detected from git remote if empty
	AppDisplayName string // display name for the App Registration (auto-generated if empty)
	Verbose        bool
	DryRun         bool
}

// Result captures what was created.
type Result struct {
	AppID                string   `json:"appId"`
	ObjectID             string   `json:"objectId"`
	ServicePrincipalID   string   `json:"servicePrincipalId"`
	FederatedCredentials []string `json:"federatedCredentials"`
	RoleAssignments      []string `json:"roleAssignments"`
	GitHubSecrets        []string `json:"githubSecrets"`
}

// commandRunner abstracts exec.Command for testing.
var commandRunner = func(name string, args ...string) ([]byte, error) {
	return exec.CommandContext(context.Background(), name, args...).CombinedOutput()
}

// SetCommandRunner replaces the command runner (for testing).
func SetCommandRunner(fn func(string, ...string) ([]byte, error)) {
	commandRunner = fn
}

// GetCommandRunner returns the current command runner (for test save/restore).
func GetCommandRunner() func(string, ...string) ([]byte, error) {
	return commandRunner
}

// Setup creates or reuses an Azure AD App Registration, configures OIDC
// federation for GitHub Actions, assigns RBAC roles, and stores the
// credentials as GitHub repository secrets.
func Setup(opts Options) (*Result, error) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen, color.Bold)

	// ── Resolve GitHub repo ────────────────────────────────────
	if opts.GitHubRepo == "" {
		repo, err := detectGitHubRepo()
		if err != nil {
			return nil, fmt.Errorf("cannot detect GitHub repo: %w; pass --github-repo owner/repo explicitly", err)
		}
		opts.GitHubRepo = repo
	}

	if opts.AppDisplayName == "" {
		opts.AppDisplayName = fmt.Sprintf("lzctl-%s-github-actions", opts.TenantName)
	}

	bold.Fprintf(os.Stderr, "\n🔐 Setting up OIDC for GitHub Actions\n")
	fmt.Fprintf(os.Stderr, "   Tenant:      %s\n", opts.TenantName)
	fmt.Fprintf(os.Stderr, "   GitHub repo: %s\n", opts.GitHubRepo)
	fmt.Fprintf(os.Stderr, "   App name:    %s\n", opts.AppDisplayName)
	fmt.Fprintln(os.Stderr)

	if opts.DryRun {
		return dryRunResult(opts), nil
	}

	result := &Result{}

	// ── Step 1: Create or reuse App Registration ───────────────
	cyan.Fprintln(os.Stderr, "   1/5 Creating App Registration...")
	appID, objectID, err := createOrGetAppRegistration(opts)
	if err != nil {
		return nil, fmt.Errorf("app registration: %w", err)
	}
	result.AppID = appID
	result.ObjectID = objectID
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "        App ID:    %s\n", appID)
		fmt.Fprintf(os.Stderr, "        Object ID: %s\n", objectID)
	}

	// ── Step 2: Create Service Principal ───────────────────────
	cyan.Fprintln(os.Stderr, "   2/5 Creating Service Principal...")
	spID, err := createOrGetServicePrincipal(appID, opts)
	if err != nil {
		return nil, fmt.Errorf("service principal: %w", err)
	}
	result.ServicePrincipalID = spID
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "        SP ID: %s\n", spID)
	}

	// ── Step 3: Add Federated Credentials (OIDC) ──────────────
	cyan.Fprintln(os.Stderr, "   3/5 Configuring OIDC federated credentials...")
	fedCreds, err := configureFederatedCredentials(objectID, opts)
	if err != nil {
		return nil, fmt.Errorf("federated credentials: %w", err)
	}
	result.FederatedCredentials = fedCreds

	// ── Step 4: Assign RBAC Roles ─────────────────────────────
	cyan.Fprintln(os.Stderr, "   4/5 Assigning RBAC roles...")
	roles, err := assignRBACRoles(spID, opts)
	if err != nil {
		return nil, fmt.Errorf("RBAC roles: %w", err)
	}
	result.RoleAssignments = roles

	// ── Step 5: Store GitHub Secrets ──────────────────────────
	cyan.Fprintln(os.Stderr, "   5/5 Storing GitHub repository secrets...")
	secrets, err := storeGitHubSecrets(appID, opts)
	if err != nil {
		return nil, fmt.Errorf("GitHub secrets: %w", err)
	}
	result.GitHubSecrets = secrets

	fmt.Fprintln(os.Stderr)
	green.Fprintln(os.Stderr, "   ✅ OIDC setup complete!")
	fmt.Fprintf(os.Stderr, "   App Registration: %s (%s)\n", opts.AppDisplayName, appID)
	fmt.Fprintf(os.Stderr, "   GitHub secrets:   AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID\n")
	fmt.Fprintf(os.Stderr, "   PR workflows can now authenticate to Azure via OIDC.\n")

	return result, nil
}

// ────────────────────────────────────────────────────────────
// Step 1: App Registration
// ────────────────────────────────────────────────────────────

// appInfo holds the JSON output from `az ad app`.
type appInfo struct {
	AppID string `json:"appId"`
	ID    string `json:"id"`
}

func createOrGetAppRegistration(opts Options) (appID, objectID string, err error) {
	// Check if app already exists
	out, err := commandRunner("az", "ad", "app", "list",
		"--display-name", opts.AppDisplayName,
		"--query", "[0].{appId:appId, id:id}",
		"-o", "json")
	if err == nil {
		var existing appInfo
		if json.Unmarshal(out, &existing) == nil && existing.AppID != "" {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "        (reusing existing app: %s)\n", existing.AppID)
			}
			return existing.AppID, existing.ID, nil
		}
	}

	// Create new app
	out, err = commandRunner("az", "ad", "app", "create",
		"--display-name", opts.AppDisplayName,
		"--sign-in-audience", "AzureADMyOrg",
		"--query", "{appId:appId, id:id}",
		"-o", "json")
	if err != nil {
		return "", "", fmt.Errorf("az ad app create failed: %s", string(out))
	}

	var app appInfo
	if err := json.Unmarshal(out, &app); err != nil {
		return "", "", fmt.Errorf("parsing app create output: %w", err)
	}
	return app.AppID, app.ID, nil
}

// ────────────────────────────────────────────────────────────
// Step 2: Service Principal
// ────────────────────────────────────────────────────────────

func createOrGetServicePrincipal(appID string, opts Options) (string, error) {
	// Check if SP already exists
	out, err := commandRunner("az", "ad", "sp", "show",
		"--id", appID,
		"--query", "id",
		"-o", "tsv")
	if err == nil {
		spID := strings.TrimSpace(string(out))
		if spID != "" {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "        (reusing existing SP)\n")
			}
			return spID, nil
		}
	}

	// Create SP
	out, err = commandRunner("az", "ad", "sp", "create",
		"--id", appID,
		"--query", "id",
		"-o", "tsv")
	if err != nil {
		return "", fmt.Errorf("az ad sp create failed: %s", string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// ────────────────────────────────────────────────────────────
// Step 3: Federated Credentials (OIDC)
// ────────────────────────────────────────────────────────────

type federatedCredentialSpec struct {
	Name      string   `json:"name"`
	Issuer    string   `json:"issuer"`
	Subject   string   `json:"subject"`
	Audiences []string `json:"audiences"`
}

func configureFederatedCredentials(objectID string, opts Options) ([]string, error) {
	// Define the OIDC subjects we need
	specs := []federatedCredentialSpec{
		{
			Name:      "github-pr",
			Issuer:    "https://token.actions.githubusercontent.com",
			Subject:   fmt.Sprintf("repo:%s:pull_request", opts.GitHubRepo),
			Audiences: []string{"api://AzureADTokenExchange"},
		},
		{
			Name:      "github-main",
			Issuer:    "https://token.actions.githubusercontent.com",
			Subject:   fmt.Sprintf("repo:%s:ref:refs/heads/main", opts.GitHubRepo),
			Audiences: []string{"api://AzureADTokenExchange"},
		},
		{
			Name:      "github-env-canary",
			Issuer:    "https://token.actions.githubusercontent.com",
			Subject:   fmt.Sprintf("repo:%s:environment:canary", opts.GitHubRepo),
			Audiences: []string{"api://AzureADTokenExchange"},
		},
		{
			Name:      "github-env-wave1",
			Issuer:    "https://token.actions.githubusercontent.com",
			Subject:   fmt.Sprintf("repo:%s:environment:wave1", opts.GitHubRepo),
			Audiences: []string{"api://AzureADTokenExchange"},
		},
		{
			Name:      "github-env-wave2",
			Issuer:    "https://token.actions.githubusercontent.com",
			Subject:   fmt.Sprintf("repo:%s:environment:wave2", opts.GitHubRepo),
			Audiences: []string{"api://AzureADTokenExchange"},
		},
	}

	var created []string
	for _, spec := range specs {
		if err := createFederatedCredential(objectID, spec, opts); err != nil {
			// If it already exists, skip silently
			if strings.Contains(err.Error(), "already exists") ||
				strings.Contains(err.Error(), "FederatedIdentityCredentialAlreadyExists") {
				if opts.Verbose {
					fmt.Fprintf(os.Stderr, "        (federated credential '%s' already exists — skipped)\n", spec.Name)
				}
				created = append(created, spec.Name+" (existing)")
				continue
			}
			return created, fmt.Errorf("creating federated credential '%s': %w", spec.Name, err)
		}
		created = append(created, spec.Name)
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "        ✓ %s → %s\n", spec.Name, spec.Subject)
		}
	}
	return created, nil
}

func createFederatedCredential(objectID string, spec federatedCredentialSpec, opts Options) error {
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	out, err := commandRunner("az", "ad", "app", "federated-credential", "create",
		"--id", objectID,
		"--parameters", string(specJSON))
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}

// ────────────────────────────────────────────────────────────
// Step 4: RBAC Role Assignments
// ────────────────────────────────────────────────────────────

func assignRBACRoles(spID string, opts Options) ([]string, error) {
	type roleScope struct {
		Role  string
		Scope string
		Desc  string
	}

	assignments := []roleScope{
		{
			Role:  "Reader",
			Scope: fmt.Sprintf("/providers/Microsoft.Management/managementGroups/%s", opts.TenantID),
			Desc:  "Reader on root Management Group",
		},
	}

	// If we have a subscription, add Reader on it too
	if opts.SubscriptionID != "" {
		assignments = append(assignments, roleScope{
			Role:  "Reader",
			Scope: fmt.Sprintf("/subscriptions/%s", opts.SubscriptionID),
			Desc:  "Reader on subscription",
		})
	}

	var created []string
	for _, a := range assignments {
		out, err := commandRunner("az", "role", "assignment", "create",
			"--assignee-object-id", spID,
			"--assignee-principal-type", "ServicePrincipal",
			"--role", a.Role,
			"--scope", a.Scope)
		if err != nil {
			outStr := string(out)
			// If already assigned, skip
			if strings.Contains(outStr, "already exists") ||
				strings.Contains(outStr, "RoleAssignmentExists") {
				if opts.Verbose {
					fmt.Fprintf(os.Stderr, "        (role '%s' already assigned — skipped)\n", a.Desc)
				}
				created = append(created, a.Desc+" (existing)")
				continue
			}
			return created, fmt.Errorf("assigning role '%s': %s", a.Desc, outStr)
		}
		created = append(created, a.Desc)
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "        ✓ %s\n", a.Desc)
		}
	}
	return created, nil
}

// ────────────────────────────────────────────────────────────
// Step 5: GitHub Secrets
// ────────────────────────────────────────────────────────────

func storeGitHubSecrets(appID string, opts Options) ([]string, error) {
	secrets := map[string]string{
		"AZURE_CLIENT_ID":       appID,
		"AZURE_TENANT_ID":       opts.TenantID,
		"AZURE_SUBSCRIPTION_ID": opts.SubscriptionID,
	}

	var stored []string
	for name, value := range secrets {
		if value == "" {
			continue
		}
		out, err := commandRunner("gh", "secret", "set", name,
			"--repo", opts.GitHubRepo,
			"--body", value)
		if err != nil {
			return stored, fmt.Errorf("setting secret %s: %s", name, string(out))
		}
		stored = append(stored, name)
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "        ✓ %s\n", name)
		}
	}
	return stored, nil
}

// ────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────

// detectGitHubRepo extracts owner/repo from the current git remote.
func detectGitHubRepo() (string, error) {
	out, err := commandRunner("git", "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("no git remote 'origin' found")
	}
	return parseGitHubRepo(strings.TrimSpace(string(out)))
}

// parseGitHubRepo extracts owner/repo from a GitHub URL.
// Supports HTTPS (https://github.com/owner/repo.git) and SSH (git@github.com:owner/repo.git).
func parseGitHubRepo(remoteURL string) (string, error) {
	// HTTPS: https://github.com/owner/repo.git
	httpsRe := regexp.MustCompile(`github\.com[/:]([^/]+)/([^/.]+?)(?:\.git)?$`)
	if matches := httpsRe.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return matches[1] + "/" + matches[2], nil
	}
	return "", fmt.Errorf("cannot parse GitHub repo from remote URL: %s", remoteURL)
}

func dryRunResult(opts Options) *Result {
	bold := color.New(color.Bold)
	bold.Fprintln(os.Stderr, "   [DRY RUN] Would perform:")
	fmt.Fprintf(os.Stderr, "   • Create App Registration: %s\n", opts.AppDisplayName)
	fmt.Fprintf(os.Stderr, "   • Create Service Principal\n")
	fmt.Fprintf(os.Stderr, "   • Add federated credentials for: pull_request, main, canary, wave1, wave2\n")
	fmt.Fprintf(os.Stderr, "   • Assign Reader role on root MG (%s)\n", opts.TenantID)
	if opts.SubscriptionID != "" {
		fmt.Fprintf(os.Stderr, "   • Assign Reader role on subscription (%s)\n", opts.SubscriptionID)
	}
	fmt.Fprintf(os.Stderr, "   • Set GitHub secrets: AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID\n")
	fmt.Fprintf(os.Stderr, "   • GitHub repo: %s\n", opts.GitHubRepo)
	return &Result{
		AppID:                "(dry-run)",
		FederatedCredentials: []string{"github-pr", "github-main", "github-env-canary", "github-env-wave1", "github-env-wave2"},
		GitHubSecrets:        []string{"AZURE_CLIENT_ID", "AZURE_TENANT_ID", "AZURE_SUBSCRIPTION_ID"},
	}
}
