package exitcode

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kjourdan1/lzctl/internal/azauth"
)

func TestOf_Nil(t *testing.T) {
	if code := Of(nil); code != OK {
		t.Errorf("Of(nil) = %d, want %d", code, OK)
	}
}

func TestOf_CodedError(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{"generic", Generic},
		{"validation", Validation},
		{"azure", Azure},
		{"terraform", Terraform},
		{"drift", Drift},
		{"policy", Policy},
		{"security_block", SecurityBlock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrap(tt.code, fmt.Errorf("some error"))
			if got := Of(err); got != tt.code {
				t.Errorf("Of(Wrap(%d, ...)) = %d, want %d", tt.code, got, tt.code)
			}
		})
	}
}

func TestOf_WrappedCodedError(t *testing.T) {
	inner := Wrap(Terraform, fmt.Errorf("plan failed"))
	wrapped := fmt.Errorf("outer: %w", inner)
	if got := Of(wrapped); got != Terraform {
		t.Errorf("Of(wrapped coded error) = %d, want %d", got, Terraform)
	}
}

func TestOf_SPNBindingError(t *testing.T) {
	err := &azauth.SPNBindingError{ProjectName: "test", ExpectedClient: "a", ActualClient: "b", Stage: "pre_auth"}
	if got := Of(err); got != SecurityBlock {
		t.Errorf("Of(SPNBindingError) = %d, want %d", got, SecurityBlock)
	}
}

func TestOf_SPNTenantMismatchError(t *testing.T) {
	err := &azauth.SPNTenantMismatchError{ProjectName: "test", ConfigTenant: "a", JWTTenantClaim: "b"}
	if got := Of(err); got != SecurityBlock {
		t.Errorf("Of(SPNTenantMismatchError) = %d, want %d", got, SecurityBlock)
	}
}

func TestOf_StringFallback(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want int
	}{
		{"plan_integrity", "plan_integrity_failed: hash mismatch", SecurityBlock},
		{"plan_scope_violation", "plan_scope_violation: wrong sub", SecurityBlock},
		{"binding_violation", "binding_violation detected", SecurityBlock},
		{"state_locked", "is locked by another deployment operation", SecurityBlock},
		{"terraform_keyword", "terraform init failed", Terraform},
		{"plan_failed", "plan failed for layer X", Terraform},
		{"apply_failed", "apply failed for layer X", Terraform},
		{"validation_keyword", "validation error in config", Validation},
		{"invalid_keyword", "invalid CIDR format", Validation},
		{"azure_keyword", "azure API returned 429", Azure},
		{"arm_keyword", "arm deployment timed out", Azure},
		{"generic_fallback", "something went wrong", Generic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.msg)
			if got := Of(err); got != tt.want {
				t.Errorf("Of(%q) = %d, want %d", tt.msg, got, tt.want)
			}
		})
	}
}

func TestWrap_NilError(t *testing.T) {
	if got := Wrap(Terraform, nil); got != nil {
		t.Errorf("Wrap(code, nil) = %v, want nil", got)
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := Wrap(Terraform, cause)

	var coded *Error
	if !errors.As(err, &coded) {
		t.Fatal("errors.As should match *Error")
	}
	if coded.Code != Terraform {
		t.Errorf("Code = %d, want %d", coded.Code, Terraform)
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the root cause through Unwrap")
	}
}

func TestError_ErrorMessage(t *testing.T) {
	err := Wrap(Validation, fmt.Errorf("bad input"))
	if err.Error() != "bad input" {
		t.Errorf("Error() = %q, want %q", err.Error(), "bad input")
	}
}
