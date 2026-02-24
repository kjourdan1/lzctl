package exitcode

import (
	"errors"
	"strings"

	"github.com/kjourdan1/lzctl/internal/azauth"
)

const (
	OK            = 0
	Generic       = 1
	Validation    = 2
	Azure         = 3
	Terraform     = 4
	Drift         = 5
	Policy        = 6
	SecurityBlock = 7
)

type Error struct {
	Code  int
	Cause error
}

func (e *Error) Error() string {
	return e.Cause.Error()
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Cause: err}
}

func Of(err error) int {
	if err == nil {
		return OK
	}

	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}

	var spnErr *azauth.SPNBindingError
	if errors.As(err, &spnErr) {
		return SecurityBlock
	}

	var tenantErr *azauth.SPNTenantMismatchError
	if errors.As(err, &tenantErr) {
		return SecurityBlock
	}

	// Fallback: string-based classification for errors not yet wrapped with typed codes.
	// Each case here is a candidate for future replacement with a typed error.
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "plan_integrity_failed"),
		strings.Contains(msg, "plan_scope_violation"),
		strings.Contains(msg, "binding_violation"),
		strings.Contains(msg, "tenant_mismatch"),
		strings.Contains(msg, "state_integrity_failed"),
		strings.Contains(msg, "is locked by another deployment operation"):
		return SecurityBlock
	case strings.Contains(msg, "terraform") || strings.Contains(msg, "plan failed") || strings.Contains(msg, "apply failed"):
		return Terraform
	case strings.Contains(msg, "validation") || strings.Contains(msg, "invalid"):
		return Validation
	case strings.Contains(msg, "azure") || strings.Contains(msg, "arm"):
		return Azure
	default:
		return Generic
	}
}
