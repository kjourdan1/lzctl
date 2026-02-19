package azauth

import (
	"context"
	"fmt"
	"testing"
)

func TestLogin_MissingTenantID(t *testing.T) {
	_, err := Login(context.Background(), Options{
		TenantID: "",
	})
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}

func TestAuthError_Format(t *testing.T) {
	err := &AuthError{
		TenantID: "test-tenant-id",
	}
	s := err.Error()
	if s == "" {
		t.Error("expected non-empty error string")
	}
	if len(s) < 50 {
		t.Error("expected detailed error message with setup instructions")
	}
}

func TestCredential_Struct(t *testing.T) {
	c := &Credential{
		TenantID: "test-tenant-id",
		Method:   "cli",
	}
	if c.TenantID != "test-tenant-id" {
		t.Errorf("unexpected tenant ID: %s", c.TenantID)
	}
	if c.Method != "cli" {
		t.Errorf("unexpected method: %s", c.Method)
	}
}

func TestDetectTenantID_Success(t *testing.T) {
	original := commandRunner
	defer func() { commandRunner = original }()

	commandRunner = func(name string, args ...string) ([]byte, error) {
		return []byte("72f988bf-86f1-41af-91ab-2d7cd011db47\n"), nil
	}

	tid, err := DetectTenantID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tid != "72f988bf-86f1-41af-91ab-2d7cd011db47" {
		t.Errorf("got %q, want 72f988bf-86f1-41af-91ab-2d7cd011db47", tid)
	}
}

func TestDetectTenantID_CLINotAvailable(t *testing.T) {
	original := commandRunner
	defer func() { commandRunner = original }()

	commandRunner = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("exec: az not found")
	}

	_, err := DetectTenantID()
	if err == nil {
		t.Fatal("expected error when az CLI not available")
	}
	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
	}
}

func TestDetectTenantID_EmptyOutput(t *testing.T) {
	original := commandRunner
	defer func() { commandRunner = original }()

	commandRunner = func(name string, args ...string) ([]byte, error) {
		return []byte("  \n"), nil
	}

	_, err := DetectTenantID()
	if err == nil {
		t.Fatal("expected error for empty tenant ID")
	}
}

func TestDetectSubscriptions_Success(t *testing.T) {
	original := commandRunner
	defer func() { commandRunner = original }()

	commandRunner = func(name string, args ...string) ([]byte, error) {
		return []byte(`[{"id":"sub-1","name":"My Sub","tenantId":"tid-1","isDefault":true}]`), nil
	}

	subs, err := DetectSubscriptions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("got %d subs, want 1", len(subs))
	}
	if subs[0].ID != "sub-1" {
		t.Errorf("got ID %q, want sub-1", subs[0].ID)
	}
	if subs[0].Name != "My Sub" {
		t.Errorf("got Name %q, want My Sub", subs[0].Name)
	}
	if !subs[0].IsDefault {
		t.Error("expected IsDefault=true")
	}
}

func TestDetectSubscriptions_CLINotAvailable(t *testing.T) {
	original := commandRunner
	defer func() { commandRunner = original }()

	commandRunner = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("exec: az not found")
	}

	_, err := DetectSubscriptions()
	if err == nil {
		t.Fatal("expected error when az CLI not available")
	}
}

func TestSubscriptionSummary_Fields(t *testing.T) {
	s := SubscriptionSummary{
		ID:        "sub-123",
		Name:      "Test",
		TenantID:  "tid-456",
		IsDefault: true,
	}
	if s.ID != "sub-123" {
		t.Errorf("unexpected ID: %s", s.ID)
	}
}

func TestOptions_Fields(t *testing.T) {
	opts := Options{
		TenantID:    "test",
		Interactive: true,
		Verbose:     false,
	}
	if opts.TenantID != "test" {
		t.Error("unexpected tenant ID")
	}
	if !opts.Interactive {
		t.Error("expected Interactive=true")
	}
}
