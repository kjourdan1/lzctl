package azauth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kjourdan1/lzctl/internal/config"
)

func TestValidateSPNBinding_ClientIDMatch(t *testing.T) {
	t.Setenv("AZURE_CLIENT_ID", "11111111-1111-1111-1111-111111111111")

	cfg := &config.LZConfig{
		Metadata: config.Metadata{Name: "mbk-lz"},
		Spec: config.Spec{
			Platform: config.Platform{
				Identity: config.IdentityConfig{ClientID: "11111111-1111-1111-1111-111111111111"},
			},
		},
	}

	if err := ValidateSPNBinding(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateSPNBinding_Mismatch(t *testing.T) {
	t.Setenv("AZURE_CLIENT_ID", "22222222-2222-2222-2222-222222222222")

	cfg := &config.LZConfig{
		Metadata: config.Metadata{Name: "mbk-lz"},
		Spec: config.Spec{
			Platform: config.Platform{
				Identity: config.IdentityConfig{ClientID: "11111111-1111-1111-1111-111111111111"},
			},
		},
	}

	err := ValidateSPNBinding(cfg)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "SEC1_BINDING_VIOLATION") {
		t.Fatalf("expected binding violation error, got: %v", err)
	}
}

func TestValidateSPNBinding_NoClientID(t *testing.T) {
	cfg := &config.LZConfig{
		Metadata: config.Metadata{Name: "mbk-lz"},
	}

	if err := ValidateSPNBinding(cfg); err != nil {
		t.Fatalf("expected no error when clientId is empty, got: %v", err)
	}
}

func TestDecodeJWTClaims_ValidToken(t *testing.T) {
	raw := buildUnsignedJWT(map[string]any{
		"tid":   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"appid": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	})

	claims, err := decodeJWTClaims(raw)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if claims["tid"] != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected tid claim: %v", claims["tid"])
	}
}

func TestValidateJWTBinding_TenantMismatch(t *testing.T) {
	raw := buildUnsignedJWT(map[string]any{
		"tid":   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		"appid": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	})

	cfg := &config.LZConfig{
		Metadata: config.Metadata{
			Name:   "mbk-lz",
			Tenant: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
		Spec: config.Spec{
			Platform: config.Platform{
				Identity: config.IdentityConfig{ClientID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
			},
		},
	}

	err := ValidateJWTBinding(raw, cfg)
	if err == nil {
		t.Fatal("expected tenant mismatch error")
	}
	if !strings.Contains(err.Error(), "SEC1_TENANT_MISMATCH") {
		t.Fatalf("expected tenant mismatch error, got: %v", err)
	}
}

func buildUnsignedJWT(payload map[string]any) string {
	header := map[string]any{"alg": "none", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb) + "."
}
