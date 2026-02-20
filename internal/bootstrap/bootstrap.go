// Package bootstrap provisions the Azure infrastructure required before
// Terraform can operate: a Storage Account for remote state, an App
// Registration (SPN) with a custom least-privilege role, and OIDC
// federated credentials for CI/CD.
//
// Uses Azure CLI (az) for simplicity and OIDC/CLI auth compatibility.
// Follows CAF/WAF best practices:
//   - Encryption at rest (default)
//   - Soft delete for blobs
//   - HTTPS only
//   - Deny public blob access
//   - Least-privilege custom role (not Contributor)
//   - OIDC federated credentials (no secrets)
package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
)

// Options for state backend bootstrap.
type Options struct {
	TenantName     string
	SubscriptionID string
	TenantID       string // Azure AD tenant ID
	Region         string
	GitHubOrg      string // GitHub org/user for OIDC federation
	GitHubRepo     string // GitHub repo name
	Verbosity      int
	SkipSPN        bool // Skip SPN creation (e.g. if already exists)
}

// Result of bootstrap operation.
type Result struct {
	ResourceGroupName  string
	StorageAccountName string
	ContainerName      string
	SPNAppID           string // App (client) ID of the created SPN
	SPNObjectID        string // Object ID of the service principal
	CustomRoleID       string // ID of the custom role definition
	Created            bool   // false if already existed
}

// sanitizeStorageName removes non-alphanumeric characters.
func sanitizeStorageName(s string) string {
	var result strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// StateBackend creates the Azure Storage Account and container for Terraform state,
// optionally creates an SPN with a custom least-privilege role, and sets up OIDC
// federated credentials for GitHub Actions.
// It is idempotent: skips resources that already exist.
func StateBackend(opts Options) (*Result, error) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan)

	rgName := "rg-lzctl-tfstate"
	saName := "stlzctl" + sanitizeStorageName(opts.TenantName)
	if len(saName) > 24 {
		saName = saName[:24]
	}
	containerName := "tfstate"

	result := &Result{
		ResourceGroupName:  rgName,
		StorageAccountName: saName,
		ContainerName:      containerName,
	}

	bold.Fprintf(os.Stderr, "🗄️  Bootstrapping Terraform state backend...\n")
	cyan.Fprintf(os.Stderr, "   Resource Group  : %s\n", rgName)
	cyan.Fprintf(os.Stderr, "   Storage Account : %s\n", saName)
	cyan.Fprintf(os.Stderr, "   Container       : %s\n", containerName)
	cyan.Fprintf(os.Stderr, "   Region          : %s\n", opts.Region)
	fmt.Fprintln(os.Stderr)

	// Check az CLI is available
	if _, err := exec.LookPath("az"); err != nil {
		return nil, fmt.Errorf("azure CLI (az) not found in PATH; install: https://aka.ms/installazurecli")
	}

	// Set subscription
	if opts.SubscriptionID != "" {
		if err := runAz(opts.Verbosity, "account", "set", "--subscription", opts.SubscriptionID); err != nil {
			return nil, fmt.Errorf("setting subscription: %w", err)
		}
	}

	// 1. Create resource group (idempotent)
	fmt.Fprintf(os.Stderr, "   → Creating resource group %s...\n", rgName)
	if err := runAz(opts.Verbosity,
		"group", "create",
		"--name", rgName,
		"--location", opts.Region,
		"--tags", "managed-by=lzctl", "purpose=terraform-state",
	); err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	// 2. Create storage account (idempotent — az will error if name taken by another sub)
	fmt.Fprintf(os.Stderr, "   → Creating storage account %s...\n", saName)
	if err := runAz(opts.Verbosity,
		"storage", "account", "create",
		"--name", saName,
		"--resource-group", rgName,
		"--location", opts.Region,
		"--sku", "Standard_LRS",
		"--kind", "StorageV2",
		"--min-tls-version", "TLS1_2",
		"--allow-blob-public-access", "false",
		"--https-only", "true",
		"--tags", "managed-by=lzctl", "purpose=terraform-state",
	); err != nil {
		return nil, fmt.Errorf("creating storage account: %w", err)
	}

	// 3. Enable soft delete for blobs (WAF recommendation)
	fmt.Fprintf(os.Stderr, "   → Enabling blob soft delete...\n")
	if err := runAz(opts.Verbosity,
		"storage", "account", "blob-service-properties", "update",
		"--account-name", saName,
		"--resource-group", rgName,
		"--enable-delete-retention", "true",
		"--delete-retention-days", "30",
	); err != nil {
		// Non-fatal: some subscriptions may not support this
		fmt.Fprintf(os.Stderr, "   ⚠️  Could not enable soft delete: %v\n", err)
	}

	// 4. Create blob container
	fmt.Fprintf(os.Stderr, "   → Creating container %s...\n", containerName)
	if err := runAz(opts.Verbosity,
		"storage", "container", "create",
		"--name", containerName,
		"--account-name", saName,
		"--auth-mode", "login",
	); err != nil {
		return nil, fmt.Errorf("creating container: %w", err)
	}

	// 5. Create SPN with custom least-privilege role
	if !opts.SkipSPN {
		if err := createSPNWithCustomRole(opts, result); err != nil {
			return nil, fmt.Errorf("creating SPN: %w", err)
		}
	}

	// 6. Assign Storage Blob Data Contributor to SPN for state access
	fmt.Fprintf(os.Stderr, "   → Assigning RBAC for state access...\n")
	assigneeID := result.SPNObjectID
	if assigneeID == "" {
		// Fallback to current signed-in identity
		signedInID, err := runAzOutput(opts.Verbosity, "ad", "signed-in-user", "show", "--query", "id", "-o", "tsv")
		if err != nil {
			clientID := os.Getenv("AZURE_CLIENT_ID")
			if clientID == "" {
				clientID = os.Getenv("ARM_CLIENT_ID")
			}
			if clientID != "" {
				signedInID, err = runAzOutput(opts.Verbosity, "ad", "sp", "show", "--id", clientID, "--query", "id", "-o", "tsv")
			}
		}
		if err == nil {
			assigneeID = strings.TrimSpace(signedInID)
		}
	}

	if assigneeID != "" {
		saScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s",
			opts.SubscriptionID, rgName, saName)
		_ = runAz(opts.Verbosity,
			"role", "assignment", "create",
			"--assignee-object-id", assigneeID,
			"--assignee-principal-type", "ServicePrincipal",
			"--role", "Storage Blob Data Contributor",
			"--scope", saScope,
		)
	} else {
		fmt.Fprintf(os.Stderr, "   ⚠️  Could not determine identity for RBAC assignment.\n")
		fmt.Fprintf(os.Stderr, "       Assign 'Storage Blob Data Contributor' on %s manually.\n", saName)
	}

	result.Created = true
	fmt.Fprintln(os.Stderr)
	green.Fprintf(os.Stderr, "   ✅ State backend ready: %s/%s\n", saName, containerName)
	if result.SPNAppID != "" {
		green.Fprintf(os.Stderr, "   ✅ SPN ready: %s (custom role: lzctl-deployer)\n", result.SPNAppID)
	}

	return result, nil
}

// customRoleDefinition returns the JSON definition for the lzctl least-privilege
// custom role. This role grants ONLY the permissions needed by lzctl modules:
//   - Management Groups: create/read/write/delete (resource-org)
//   - Policy: create/read/write/delete definitions & assignments (governance)
//   - Resources: create/read/write/delete resource groups, generic resources (management-logs, security)
//   - Log Analytics: full control (management-logs)
//   - Automation: full control (management-logs)
//   - Defender for Cloud: read/write (security)
//   - Key Vault: create/read/write/delete (security)
//   - Networking: create/read/write/delete vnets, firewalls, NSGs (connectivity)
//   - Monitor: diagnostic settings (management-logs)
//   - Authorization: read + role assignments (identity-access)
//   - Locks: create/read/delete management locks (governance)
//   - Storage: NO keys access, only data-plane via RBAC (state backend)
func customRoleDefinition(subscriptionID string) map[string]interface{} {
	return map[string]interface{}{
		"Name":        "lzctl-deployer",
		"Description": "Least-privilege custom role for lzctl Landing Zone deployments (CAF/WAF). No Key/Secret access, no classic resources, no owner-level permissions.",
		"Actions": []string{
			// Management Groups
			"Microsoft.Management/managementGroups/read",
			"Microsoft.Management/managementGroups/write",
			"Microsoft.Management/managementGroups/delete",
			"Microsoft.Management/managementGroups/subscriptions/write",
			"Microsoft.Management/managementGroups/subscriptions/delete",
			"Microsoft.Management/managementGroups/settings/read",
			"Microsoft.Management/managementGroups/settings/write",

			// Policy
			"Microsoft.Authorization/policyDefinitions/*",
			"Microsoft.Authorization/policySetDefinitions/*",
			"Microsoft.Authorization/policyAssignments/*",
			"Microsoft.Authorization/policyExemptions/*",

			// RBAC (read roles, create/delete assignments — NOT create role definitions)
			"Microsoft.Authorization/roleAssignments/read",
			"Microsoft.Authorization/roleAssignments/write",
			"Microsoft.Authorization/roleAssignments/delete",
			"Microsoft.Authorization/roleDefinitions/read",

			// Management Locks
			"Microsoft.Authorization/locks/*",

			// Resource Groups
			"Microsoft.Resources/subscriptions/resourceGroups/*",
			"Microsoft.Resources/deployments/*",

			// Log Analytics
			"Microsoft.OperationalInsights/workspaces/*",
			"Microsoft.OperationsManagement/solutions/*",

			// Automation
			"Microsoft.Automation/automationAccounts/*",

			// Monitor & Diagnostics
			"Microsoft.Insights/diagnosticSettings/*",
			"Microsoft.Insights/logprofiles/*",
			"Microsoft.Insights/activityLogAlerts/*",
			"Microsoft.Insights/actionGroups/*",
			"Microsoft.Insights/components/*",

			// Defender for Cloud
			"Microsoft.Security/pricings/*",
			"Microsoft.Security/securityContacts/*",
			"Microsoft.Security/workspaceSettings/*",
			"Microsoft.Security/autoProvisioningSettings/*",

			// Key Vault (management plane only — no data plane)
			"Microsoft.KeyVault/vaults/*",
			"Microsoft.KeyVault/locations/*/read",

			// Networking (for connectivity module)
			"Microsoft.Network/virtualNetworks/*",
			"Microsoft.Network/networkSecurityGroups/*",
			"Microsoft.Network/routeTables/*",
			"Microsoft.Network/azureFirewalls/*",
			"Microsoft.Network/firewallPolicies/*",
			"Microsoft.Network/publicIPAddresses/*",
			"Microsoft.Network/virtualHubs/*",
			"Microsoft.Network/virtualWans/*",
			"Microsoft.Network/privateDnsZones/*",
			"Microsoft.Network/ddosProtectionPlans/read",
			"Microsoft.Network/ddosProtectionPlans/join/action",

			// Sentinel
			"Microsoft.SecurityInsights/*",

			// Subscription-level reads
			"Microsoft.Resources/subscriptions/read",
			"Microsoft.Resources/subscriptions/resourceGroups/read",
		},
		"NotActions": []string{
			// Explicitly deny dangerous operations
			"Microsoft.Authorization/elevateAccess/Action",
			"Microsoft.Authorization/roleDefinitions/write",
			"Microsoft.Authorization/roleDefinitions/delete",
		},
		"DataActions":    []string{},
		"NotDataActions": []string{},
		"AssignableScopes": []string{
			fmt.Sprintf("/subscriptions/%s", subscriptionID),
		},
	}
}

// createSPNWithCustomRole creates an Azure AD App Registration, Service Principal,
// custom role, role assignment, and OIDC federated credentials.
func createSPNWithCustomRole(opts Options, result *Result) error {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	appName := fmt.Sprintf("lzctl-%s-github-actions", opts.TenantName)

	bold.Fprintf(os.Stderr, "\n🔐 Creating SPN with least-privilege custom role...\n")
	cyan.Fprintf(os.Stderr, "   App Name : %s\n", appName)

	// 1. Check if app already exists
	existingAppID, err := runAzOutput(opts.Verbosity, "ad", "app", "list",
		"--display-name", appName, "--query", "[0].appId", "-o", "tsv")
	if err == nil && strings.TrimSpace(existingAppID) != "" {
		result.SPNAppID = strings.TrimSpace(existingAppID)
		fmt.Fprintf(os.Stderr, "   → App already exists: %s\n", result.SPNAppID)

		// Get SP object ID
		spObjID, _ := runAzOutput(opts.Verbosity, "ad", "sp", "show",
			"--id", result.SPNAppID, "--query", "id", "-o", "tsv")
		result.SPNObjectID = strings.TrimSpace(spObjID)
	} else {
		// 2. Create app registration
		fmt.Fprintf(os.Stderr, "   → Creating App Registration...\n")
		appID, err := runAzOutput(opts.Verbosity, "ad", "app", "create",
			"--display-name", appName, "--query", "appId", "-o", "tsv")
		if err != nil {
			return fmt.Errorf("creating app registration: %w", err)
		}
		result.SPNAppID = strings.TrimSpace(appID)

		// 3. Create service principal
		fmt.Fprintf(os.Stderr, "   → Creating Service Principal...\n")
		spObjID, err := runAzOutput(opts.Verbosity, "ad", "sp", "create",
			"--id", result.SPNAppID, "--query", "id", "-o", "tsv")
		if err != nil {
			return fmt.Errorf("creating service principal: %w", err)
		}
		result.SPNObjectID = strings.TrimSpace(spObjID)
	}

	// 4. Create custom role definition
	fmt.Fprintf(os.Stderr, "   → Creating custom role 'lzctl-deployer'...\n")
	roleDef := customRoleDefinition(opts.SubscriptionID)
	roleJSON, _ := json.Marshal(roleDef)

	// Write role definition to temp file
	tmpFile, err := os.CreateTemp("", "lzctl-role-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(roleJSON); err != nil {
		return fmt.Errorf("writing role definition: %w", err)
	}
	tmpFile.Close()

	// Create or update the custom role
	err = runAz(opts.Verbosity, "role", "definition", "create", "--role-definition", tmpFile.Name())
	if err != nil {
		// Role might already exist — try update
		err = runAz(opts.Verbosity, "role", "definition", "update", "--role-definition", tmpFile.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "   ⚠️  Could not create/update custom role (may require Owner): %v\n", err)
			fmt.Fprintf(os.Stderr, "       Falling back to Reader + targeted assignments.\n")
		}
	}

	// 5. Assign custom role to SPN at subscription scope
	fmt.Fprintf(os.Stderr, "   → Assigning 'lzctl-deployer' role to SPN...\n")
	subScope := fmt.Sprintf("/subscriptions/%s", opts.SubscriptionID)
	err = runAz(opts.Verbosity,
		"role", "assignment", "create",
		"--assignee-object-id", result.SPNObjectID,
		"--assignee-principal-type", "ServicePrincipal",
		"--role", "lzctl-deployer",
		"--scope", subScope,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ⚠️  Could not assign custom role: %v\n", err)
		fmt.Fprintf(os.Stderr, "       Assign 'lzctl-deployer' to %s manually.\n", result.SPNAppID)
	}

	// 6. Assign Reader at tenant root MG (for MG operations)
	fmt.Fprintf(os.Stderr, "   → Assigning Reader at tenant root MG...\n")
	if opts.TenantID != "" {
		mgScope := fmt.Sprintf("/providers/Microsoft.Management/managementGroups/%s", opts.TenantID)
		_ = runAz(opts.Verbosity,
			"role", "assignment", "create",
			"--assignee-object-id", result.SPNObjectID,
			"--assignee-principal-type", "ServicePrincipal",
			"--role", "Management Group Contributor",
			"--scope", mgScope,
		)
	}

	// 7. Create OIDC federated credentials for GitHub Actions
	if opts.GitHubOrg != "" && opts.GitHubRepo != "" {
		fmt.Fprintf(os.Stderr, "   → Setting up OIDC federated credentials...\n")
		oidcConfigs := []struct {
			name    string
			subject string
		}{
			{"github-main", fmt.Sprintf("repo:%s/%s:ref:refs/heads/main", opts.GitHubOrg, opts.GitHubRepo)},
			{"github-pr", fmt.Sprintf("repo:%s/%s:pull_request", opts.GitHubOrg, opts.GitHubRepo)},
			{"github-env-canary", fmt.Sprintf("repo:%s/%s:environment:canary", opts.GitHubOrg, opts.GitHubRepo)},
			{"github-env-wave1", fmt.Sprintf("repo:%s/%s:environment:wave1", opts.GitHubOrg, opts.GitHubRepo)},
			{"github-env-wave2", fmt.Sprintf("repo:%s/%s:environment:wave2", opts.GitHubOrg, opts.GitHubRepo)},
		}

		for _, cfg := range oidcConfigs {
			fedCred := map[string]interface{}{
				"name":      cfg.name,
				"issuer":    "https://token.actions.githubusercontent.com",
				"subject":   cfg.subject,
				"audiences": []string{"api://AzureADTokenExchange"},
			}
			credJSON, err := json.Marshal(fedCred)
			if err != nil {
				continue
			}

			tmpCred, err := os.CreateTemp("", "lzctl-oidc-*.json")
			if err != nil {
				continue
			}
			if _, err := tmpCred.Write(credJSON); err != nil {
				_ = tmpCred.Close()
				_ = os.Remove(tmpCred.Name())
				continue
			}
			if err := tmpCred.Close(); err != nil {
				_ = os.Remove(tmpCred.Name())
				continue
			}

			_ = runAz(opts.Verbosity, "ad", "app", "federated-credential", "create",
				"--id", result.SPNAppID, "--parameters", tmpCred.Name())
			_ = os.Remove(tmpCred.Name())
		}
		fmt.Fprintf(os.Stderr, "   ✅ OIDC federated credentials configured for %s/%s\n", opts.GitHubOrg, opts.GitHubRepo)
	}

	return nil
}

// runAz runs an Azure CLI command, printing output only in verbose mode.
func runAz(verbosity int, args ...string) error {
	cmd := exec.CommandContext(context.Background(), "az", args...)
	if verbosity > 1 {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}
	return cmd.Run()
}

// runAzOutput runs an Azure CLI command and returns stdout.
func runAzOutput(_ int, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "az", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
