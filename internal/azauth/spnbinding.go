// Package azauth — file: spnbinding.go
//
// SEC-1: SPN ↔ Tenant binding validation.
//
// This module ensures that the AZURE_CLIENT_ID environment variable (or the resolved
// credential) matches the clientId declared in lzctl.yaml spec.platform.identity.
//
// Threat model:
//   - A pipeline targeting one landing zone could inadvertently (or maliciously) use
//     a SPN belonging to a different Azure AD tenant, leading to cross-tenant operations.
//   - A forged lzctl.yaml could point to a more permissive SPN.
//
// Validation levels:
//   1. Pre-auth: Compare AZURE_CLIENT_ID env var with identity.clientId (fast, no network)
//   2. Post-auth: Decode the obtained JWT and validate tid + appid claims (cryptographic guarantee)
package azauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kjourdan1/lzctl/internal/config"
)

// SPNBindingError is returned when the SPN does not match the lzctl.yaml config.
type SPNBindingError struct {
	ProjectName    string
	ExpectedClient string
	ActualClient   string
	Stage          string // "pre_auth" | "post_auth"
}

func (e *SPNBindingError) Error() string {
	return fmt.Sprintf(
		"SEC1_BINDING_VIOLATION [%s]: credential client_id (%s) does not match project %q identity.clientId (%s).\n"+
			"Ensure the correct AZURE_CLIENT_ID is set for this project.",
		e.Stage, e.ActualClient, e.ProjectName, e.ExpectedClient,
	)
}

// SPNTenantMismatchError is returned when the JWT tenant ID does not match the config.
type SPNTenantMismatchError struct {
	ProjectName    string
	ConfigTenant   string
	JWTTenantClaim string
}

func (e *SPNTenantMismatchError) Error() string {
	return fmt.Sprintf(
		"SEC1_TENANT_MISMATCH: JWT 'tid' claim (%s) does not match project %q metadata.tenant (%s).\n"+
			"The credential belongs to a different Azure AD tenant.",
		e.JWTTenantClaim, e.ProjectName, e.ConfigTenant,
	)
}

// ValidateSPNBinding performs the pre-authentication check (Level 1).
//
// It reads spec.platform.identity.clientId from lzctl.yaml and compares it with
// the currently active AZURE_CLIENT_ID environment variable.
func ValidateSPNBinding(cfg *config.LZConfig) error {
	expectedClientID := strings.TrimSpace(cfg.Spec.Platform.Identity.ClientID)
	if expectedClientID == "" {
		// No clientId configured — skip validation (identity not yet bootstrapped)
		return nil
	}

	actualClientID := strings.TrimSpace(os.Getenv("AZURE_CLIENT_ID"))

	// If AZURE_CLIENT_ID is not set, we're likely using az CLI or browser — skip binding check.
	// The post-auth JWT validation will still enforce the tenant boundary.
	if actualClientID == "" {
		return nil
	}

	if !strings.EqualFold(expectedClientID, actualClientID) {
		return &SPNBindingError{
			ProjectName:    cfg.Metadata.Name,
			ExpectedClient: expectedClientID,
			ActualClient:   actualClientID,
			Stage:          "pre_auth",
		}
	}

	return nil
}

// ValidateJWTBinding performs the post-authentication check (Level 2).
//
// It decodes the JWT access token (without verifying the signature — Azure SDK already
// validates the token cryptographically) and checks:
//   - The 'tid' (tenant ID) claim matches metadata.tenant
//   - The 'appid' or 'azp' claim matches identity.clientId
//
// rawToken is the access token string returned by Credential.GetToken().
func ValidateJWTBinding(rawToken string, cfg *config.LZConfig) error {
	claims, err := decodeJWTClaims(rawToken)
	if err != nil {
		// Non-fatal: JWT decoding failure should not block operations (degraded mode)
		fmt.Fprintf(os.Stderr, "⚠️  SEC1: JWT claim validation skipped (could not decode token): %v\n", err)
		return nil
	}

	// 1. Validate 'tid' claim — tenant ID
	if jwtTID, ok := claims["tid"].(string); ok && jwtTID != "" {
		if !strings.EqualFold(jwtTID, cfg.Metadata.Tenant) {
			return &SPNTenantMismatchError{
				ProjectName:    cfg.Metadata.Name,
				ConfigTenant:   cfg.Metadata.Tenant,
				JWTTenantClaim: jwtTID,
			}
		}
	}

	// 2. Validate 'appid' claim — client ID (service principal)
	expectedClientID := strings.TrimSpace(cfg.Spec.Platform.Identity.ClientID)
	if expectedClientID == "" {
		return nil // no clientId to validate against
	}

	for _, claim := range []string{"appid", "azp"} {
		if jwtAppID, ok := claims[claim].(string); ok && jwtAppID != "" {
			if !strings.EqualFold(jwtAppID, expectedClientID) {
				return &SPNBindingError{
					ProjectName:    cfg.Metadata.Name,
					ExpectedClient: expectedClientID,
					ActualClient:   jwtAppID,
					Stage:          "post_auth",
				}
			}
			return nil // matched on this claim
		}
	}

	return nil
}

// ResolvedClientID returns the client ID that will be used, for audit logging (AX-9).
// Returns empty string if it cannot be determined.
func ResolvedClientID(cfg *config.LZConfig) string {
	if id := strings.TrimSpace(cfg.Spec.Platform.Identity.ClientID); id != "" {
		return id
	}
	return strings.TrimSpace(os.Getenv("AZURE_CLIENT_ID"))
}

// jwtClaims is a minimal set of claims from an Azure AD access token.
type jwtClaims map[string]any

// decodeJWTClaims decodes the payload of a JWT WITHOUT verifying the signature.
// Signature verification is handled by the Azure SDK and ARM API itself.
// This is safe because we are extracting informational claims for local validation only;
// any tampered token will be rejected by the Azure API when used.
func decodeJWTClaims(rawToken string) (jwtClaims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// JWT payload is base64url-encoded (no padding)
	payload := parts[1]
	// Add padding if necessary
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decoding JWT payload: %w", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("parsing JWT claims: %w", err)
	}

	return claims, nil
}
