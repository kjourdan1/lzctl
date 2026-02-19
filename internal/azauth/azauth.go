// Package azauth manages Azure authentication for lzctl.
//
// Authentication strategy (in order):
//  1. Environment variables (AZURE_CLIENT_ID + AZURE_CLIENT_SECRET + AZURE_TENANT_ID)
//  2. Azure CLI session (az login)
//  3. Interactive browser login (opens a popup)
//
// The package caches the credential for the duration of the CLI invocation.
package azauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/fatih/color"
)

// Credential holds a resolved Azure credential and the tenant it targets.
type Credential struct {
	TokenCredential azcore.TokenCredential
	TenantID        string
	Method          string // "environment", "cli", "browser"
}

// Options configures the authentication flow.
type Options struct {
	TenantID    string // Azure AD tenant ID — required
	Interactive bool   // allow browser popup if other methods fail (default true)
	Verbose     bool   // print auth debug info
}

// Login attempts to authenticate to Azure using multiple strategies.
// It returns a Credential on success, or a detailed error with setup instructions.
func Login(ctx context.Context, opts Options) (*Credential, error) {
	if opts.TenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for Azure authentication")
	}

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen, color.Bold)

	bold.Fprintf(os.Stderr, "🔐 Authenticating to Azure tenant: %s\n", opts.TenantID)

	// Strategy 1: Environment variables (SPN)
	if os.Getenv("AZURE_CLIENT_ID") != "" && os.Getenv("AZURE_TENANT_ID") != "" {
		if opts.Verbose {
			cyan.Fprintln(os.Stderr, "   Trying: environment variables (AZURE_CLIENT_ID)...")
		}
		cred, err := azidentity.NewEnvironmentCredential(&azidentity.EnvironmentCredentialOptions{})
		if err == nil {
			if err := testCredential(ctx, cred); err == nil {
				green.Fprintln(os.Stderr, "   ✅ Authenticated via environment variables (SPN)")
				return &Credential{TokenCredential: cred, TenantID: opts.TenantID, Method: "environment"}, nil
			}
		}
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "   ⚠️  Environment credential failed: %v\n", err)
		}
	}

	// Strategy 2: Azure CLI (az login)
	if opts.Verbose {
		cyan.Fprintln(os.Stderr, "   Trying: Azure CLI (az login)...")
	}
	cliCred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{
		TenantID: opts.TenantID,
	})
	if err == nil {
		if err := testCredential(ctx, cliCred); err == nil {
			green.Fprintln(os.Stderr, "   ✅ Authenticated via Azure CLI")
			return &Credential{TokenCredential: cliCred, TenantID: opts.TenantID, Method: "cli"}, nil
		} else if opts.Verbose {
			fmt.Fprintf(os.Stderr, "   ⚠️  Azure CLI credential failed: %v\n", err)
		}
	}

	// Strategy 3: Interactive browser login
	if opts.Interactive {
		fmt.Fprintln(os.Stderr)
		bold.Fprintln(os.Stderr, "🌐 Opening browser for Azure login...")
		fmt.Fprintf(os.Stderr, "   Tenant: %s\n", opts.TenantID)
		fmt.Fprintln(os.Stderr, "   A browser window will open. Sign in with an account that has")
		fmt.Fprintln(os.Stderr, "   Reader access to the target tenant.")
		fmt.Fprintln(os.Stderr)

		browserCred, err := azidentity.NewInteractiveBrowserCredential(&azidentity.InteractiveBrowserCredentialOptions{
			TenantID: opts.TenantID,
		})
		if err == nil {
			if err := testCredential(ctx, browserCred); err == nil {
				green.Fprintln(os.Stderr, "   ✅ Authenticated via browser login")
				return &Credential{TokenCredential: browserCred, TenantID: opts.TenantID, Method: "browser"}, nil
			} else {
				fmt.Fprintf(os.Stderr, "   ❌ Browser login failed: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "   ❌ Could not initiate browser login: %v\n", err)
		}
	}

	// All methods failed — print setup guide
	return nil, &AuthError{TenantID: opts.TenantID}
}

// testCredential verifies the credential can obtain a token.
func testCredential(ctx context.Context, cred azcore.TokenCredential) error {
	_, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	return err
}

// AuthError provides a detailed error with setup instructions.
type AuthError struct {
	TenantID string
}

// SubscriptionSummary holds basic info about an Azure subscription from the CLI.
type SubscriptionSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TenantID  string `json:"tenantId"`
	IsDefault bool   `json:"isDefault"`
}

// commandRunner abstracts exec.Command for testing.
var commandRunner = func(name string, args ...string) ([]byte, error) {
	return exec.CommandContext(context.Background(), name, args...).Output()
}

// SetCommandRunner replaces the command runner (for testing).
func SetCommandRunner(fn func(string, ...string) ([]byte, error)) {
	commandRunner = fn
}

// GetCommandRunner returns the current command runner (for test save/restore).
func GetCommandRunner() func(string, ...string) ([]byte, error) {
	return commandRunner
}

// DetectTenantID attempts to read the tenant ID from the active Azure CLI session.
// Returns the tenant ID or an error with instructions to run `az login`.
func DetectTenantID() (string, error) {
	out, err := commandRunner("az", "account", "show", "--query", "tenantId", "-o", "tsv")
	if err != nil {
		return "", fmt.Errorf("could not detect tenant ID from Azure CLI; run 'az login' first or pass --tenant-id explicitly")
	}
	tid := strings.TrimSpace(string(out))
	if tid == "" {
		return "", fmt.Errorf("Azure CLI returned empty tenant ID; run 'az login' first")
	}
	return tid, nil
}

// DetectSubscriptionID returns the default subscription ID from the active Azure CLI session.
func DetectSubscriptionID() (string, error) {
	out, err := commandRunner("az", "account", "show", "--query", "id", "-o", "tsv")
	if err != nil {
		return "", fmt.Errorf("could not detect subscription ID from Azure CLI; run 'az login' first or pass --subscription-id explicitly")
	}
	sid := strings.TrimSpace(string(out))
	if sid == "" {
		return "", fmt.Errorf("Azure CLI returned empty subscription ID; run 'az login' first")
	}
	return sid, nil
}

// DetectSubscriptions returns all subscriptions visible to the current Azure CLI session.
func DetectSubscriptions() ([]SubscriptionSummary, error) {
	out, err := commandRunner("az", "account", "list", "--query", "[].{id:id, name:name, tenantId:tenantId, isDefault:isDefault}", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("could not list subscriptions from Azure CLI; run 'az login' first")
	}
	var subs []SubscriptionSummary
	if err := json.Unmarshal(out, &subs); err != nil {
		return nil, fmt.Errorf("parsing Azure CLI subscription list: %w", err)
	}
	return subs, nil
}

func (e *AuthError) Error() string {
	var sb strings.Builder
	sb.WriteString("❌ Azure authentication failed. No valid credential found.\n\n")
	sb.WriteString("To connect lzctl to your Azure tenant, use ONE of these methods:\n\n")

	sb.WriteString("━━━ Method 1: Azure CLI (easiest for local dev) ━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("  az login --tenant %s\n", e.TenantID))
	sb.WriteString("  lzctl assess --tenant <name>\n\n")

	sb.WriteString("━━━ Method 2: Service Principal (CI/CD & automation) ━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("  # 1. Create an App Registration in tenant %s\n", e.TenantID))
	sb.WriteString("  az ad app create --display-name \"lzctl-assess\"\n\n")
	sb.WriteString("  # 2. Create a Service Principal\n")
	sb.WriteString("  az ad sp create --id <app-id>\n\n")
	sb.WriteString("  # 3. Assign Reader role on the root management group\n")
	sb.WriteString("  az role assignment create \\\n")
	sb.WriteString("    --assignee <app-id> \\\n")
	sb.WriteString("    --role \"Reader\" \\\n")
	sb.WriteString(fmt.Sprintf("    --scope \"/providers/Microsoft.Management/managementGroups/%s\"\n\n", e.TenantID))
	sb.WriteString("  # 4. Set environment variables\n")
	sb.WriteString(fmt.Sprintf("  $env:AZURE_TENANT_ID = \"%s\"\n", e.TenantID))
	sb.WriteString("  $env:AZURE_CLIENT_ID = \"<app-id>\"\n")
	sb.WriteString("  $env:AZURE_CLIENT_SECRET = \"<secret>\"  # or use federated credentials\n\n")

	sb.WriteString("━━━ Method 3: Interactive browser ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  lzctl assess --tenant <name>  (will open a browser popup)\n\n")

	sb.WriteString("━━━ Required permissions for assess ━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  • Reader on root Management Group (for MG hierarchy + subscriptions)\n")
	sb.WriteString("  • Reader on subscriptions (for VNets, diagnostics)\n")
	sb.WriteString("  • Policy Reader (for policy assignments & compliance)\n")
	sb.WriteString("  • (Optional) Security Reader for Defender status\n")

	return sb.String()
}
