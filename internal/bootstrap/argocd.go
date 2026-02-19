package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/kjourdan1/lzctl/internal/config"
)

// ArgoCDFederatedOptions holds parameters for creating the ArgoCD source-controller
// Workload Identity Federation (WIF) federated credential (E9-S3).
type ArgoCDFederatedOptions struct {
	// IdentityName is the name of the user-assigned managed identity (from
	// config.Spec.Platform.Identity).
	IdentityName string
	// IdentityResourceGroup is the resource group that hosts the managed identity.
	// Defaults to the state backend resource group when empty.
	IdentityResourceGroup string
	// SubscriptionID is the subscription that owns the managed identity.
	SubscriptionID string
	// AKSClusterName is the name of the AKS cluster whose OIDC issuer URL is queried.
	AKSClusterName string
	// AKSResourceGroup is the resource group that hosts the AKS cluster.
	AKSResourceGroup string
	// Verbosity controls az CLI output verbosity (0 = silent, 1 = verbose).
	Verbosity int
}

// ArgoCDFederatedResult contains the details of the created federated credential.
type ArgoCDFederatedResult struct {
	// FederatedCredentialName is the name of the created federated credential.
	FederatedCredentialName string
	// OIDCIssuerURL is the OIDC issuer URL of the AKS cluster.
	OIDCIssuerURL string
	// Subject is the Kubernetes service account subject used in the credential.
	Subject string
}

// ArgoCDFederatedCredentialName is the canonical name for the ArgoCD
// source-controller federated credential.
const ArgoCDFederatedCredentialName = "argocd-source-controller"

// ArgoCDSourceControllerSubject is the Kubernetes subject for the ArgoCD
// source-controller service account (namespace: argocd).
const ArgoCDSourceControllerSubject = "system:serviceaccount:argocd:source-controller"

// CreateArgoCDFederatedCredential creates a Workload Identity Federation (WIF)
// federated credential on the platform managed identity so that the ArgoCD
// source-controller can authenticate to Azure AD (and by extension to GitHub /
// ADO) without storing secrets in the cluster.
//
// Flow (E9-S3):
//  1. Query the AKS cluster OIDC issuer URL via `az aks show`.
//  2. Create the federated credential on the managed identity via
//     `az identity federated-credential create`.
//
// Prerequisites:
//   - lzctl bootstrap must already have run (managed identity exists).
//   - aks-platform blueprint must have been applied (AKS cluster exists + OIDC enabled).
//   - Caller must have "Managed Identity Contributor" on the identity resource group.
func CreateArgoCDFederatedCredential(opts ArgoCDFederatedOptions) (*ArgoCDFederatedResult, error) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen, color.Bold)

	bold.Fprintf(os.Stderr, "🔐 Creating ArgoCD source-controller federated credential (E9-S3)...\n")
	cyan.Fprintf(os.Stderr, "   Managed Identity : %s\n", opts.IdentityName)
	cyan.Fprintf(os.Stderr, "   AKS Cluster      : %s\n", opts.AKSClusterName)
	cyan.Fprintf(os.Stderr, "   Subject          : %s\n", ArgoCDSourceControllerSubject)
	fmt.Fprintln(os.Stderr)

	// Step 1: Retrieve OIDC issuer URL from the AKS cluster.
	issuerURL, err := getAKSOIDCIssuerURL(opts)
	if err != nil {
		return nil, fmt.Errorf("retrieving AKS OIDC issuer URL: %w", err)
	}
	issuerURL = strings.TrimSpace(issuerURL)
	if issuerURL == "" {
		return nil, fmt.Errorf("AKS cluster %q does not have OIDC issuer enabled; set oidc_issuer_enabled = true in the blueprint", opts.AKSClusterName)
	}

	fmt.Fprintf(os.Stderr, "   → OIDC issuer: %s\n", issuerURL)

	// Step 2: Create the federated credential definition.
	credDef := map[string]interface{}{
		"name":      ArgoCDFederatedCredentialName,
		"issuer":    issuerURL,
		"subject":   ArgoCDSourceControllerSubject,
		"audiences": []string{"api://AzureADTokenExchange"},
		"description": fmt.Sprintf(
			"ArgoCD source-controller WIF credential — created by lzctl for cluster %s",
			opts.AKSClusterName,
		),
	}

	credJSON, err := json.Marshal(credDef)
	if err != nil {
		return nil, fmt.Errorf("marshalling federated credential: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "lzctl-argocd-fed-*.json")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.Write(credJSON); writeErr != nil {
		return nil, fmt.Errorf("writing federated credential JSON: %w", writeErr)
	}
	tmpFile.Close()

	identityRG := opts.IdentityResourceGroup
	if identityRG == "" {
		identityRG = "rg-lzctl-tfstate"
	}

	// Step 3: Create the federated credential (idempotent: ignore "already exists").
	fmt.Fprintf(os.Stderr, "   → Creating federated credential %q on identity %q...\n",
		ArgoCDFederatedCredentialName, opts.IdentityName)

	err = runAz(opts.Verbosity,
		"identity", "federated-credential", "create",
		"--name", ArgoCDFederatedCredentialName,
		"--identity-name", opts.IdentityName,
		"--resource-group", identityRG,
		"--issuer", issuerURL,
		"--subject", ArgoCDSourceControllerSubject,
		"--audiences", "api://AzureADTokenExchange",
	)
	if err != nil {
		// Check if it already exists (idempotent).
		existingOut, listErr := runAzOutput(opts.Verbosity,
			"identity", "federated-credential", "show",
			"--name", ArgoCDFederatedCredentialName,
			"--identity-name", opts.IdentityName,
			"--resource-group", identityRG,
			"--query", "name", "-o", "tsv",
		)
		if listErr != nil || strings.TrimSpace(existingOut) != ArgoCDFederatedCredentialName {
			return nil, fmt.Errorf("creating ArgoCD federated credential: %w", err)
		}
		fmt.Fprintf(os.Stderr, "   → Federated credential already exists — skipped.\n")
	}

	green.Fprintf(os.Stderr, "\n   ✅ ArgoCD source-controller federated credential ready.\n")
	fmt.Fprintf(os.Stderr, "\n📌 Next step: annotate the service account in the cluster:\n")
	fmt.Fprintf(os.Stderr, "     kubectl annotate sa source-controller -n argocd \\\n")
	fmt.Fprintf(os.Stderr, "       azure.workload.identity/client-id=<identity-client-id>\n")

	return &ArgoCDFederatedResult{
		FederatedCredentialName: ArgoCDFederatedCredentialName,
		OIDCIssuerURL:           issuerURL,
		Subject:                 ArgoCDSourceControllerSubject,
	}, nil
}

// getAKSOIDCIssuerURL retrieves the OIDC issuer URL from an AKS cluster.
func getAKSOIDCIssuerURL(opts ArgoCDFederatedOptions) (string, error) {
	if opts.SubscriptionID != "" {
		if err := runAz(opts.Verbosity, "account", "set", "--subscription", opts.SubscriptionID); err != nil {
			return "", fmt.Errorf("setting subscription: %w", err)
		}
	}

	issuerURL, err := runAzOutput(opts.Verbosity,
		"aks", "show",
		"--name", opts.AKSClusterName,
		"--resource-group", opts.AKSResourceGroup,
		"--query", "oidcIssuerProfile.issuerUrl",
		"-o", "tsv",
	)
	if err != nil {
		return "", fmt.Errorf("az aks show failed: %w", err)
	}
	return issuerURL, nil
}

// ArgoCDFederatedCredentialFromConfig is a convenience wrapper that derives
// ArgoCDFederatedOptions from an lzctl config + landing zone name.
// It computes the AKS cluster name using the lzctl naming convention.
func ArgoCDFederatedCredentialFromConfig(cfg *config.LZConfig, lzName string, verbosity int) (*ArgoCDFederatedResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Resolve the landing zone.
	var targetLZ *config.LandingZone
	for i := range cfg.Spec.LandingZones {
		if cfg.Spec.LandingZones[i].Name == lzName {
			targetLZ = &cfg.Spec.LandingZones[i]
			break
		}
	}
	if targetLZ == nil {
		return nil, fmt.Errorf("landing zone %q not found in config", lzName)
	}
	if targetLZ.Blueprint == nil || targetLZ.Blueprint.Type != "aks-platform" {
		return nil, fmt.Errorf("landing zone %q does not have an aks-platform blueprint; ArgoCD federated credential requires aks-platform", lzName)
	}

	slug := strings.ToLower(strings.ReplaceAll(lzName, " ", "-"))
	aksName := "aks-" + slug

	// Best-effort: derive AKS resource group from workload naming convention.
	// This is the resource group created by the blueprint (e.g. rg-<slug>-workload).
	aksRG := "rg-" + slug

	opts := ArgoCDFederatedOptions{
		IdentityName:          cfg.Spec.Platform.Identity.Name,
		IdentityResourceGroup: cfg.Spec.StateBackend.ResourceGroup,
		SubscriptionID:        targetLZ.Subscription,
		AKSClusterName:        aksName,
		AKSResourceGroup:      aksRG,
		Verbosity:             verbosity,
	}

	return CreateArgoCDFederatedCredential(opts)
}
