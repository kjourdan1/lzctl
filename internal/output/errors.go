package output

import "fmt"

// CLIError represents a user-facing error with an optional suggested fix.
type CLIError struct {
	Message string // what went wrong
	Cause   error  // underlying error (optional)
	Fix     string // suggested fix (optional)
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *CLIError) Unwrap() error {
	return e.Cause
}

// NewError creates a new CLIError with just a message.
func NewError(message string) *CLIError {
	return &CLIError{Message: message}
}

// NewErrorWithFix creates a new CLIError with a message and suggested fix.
func NewErrorWithFix(message, fix string) *CLIError {
	return &CLIError{Message: message, Fix: fix}
}

// WrapError wraps an existing error with a message and optional fix.
func WrapError(err error, message string) *CLIError {
	return &CLIError{Message: message, Cause: err}
}

// WrapErrorWithFix wraps an existing error with a message and suggested fix.
func WrapErrorWithFix(err error, message, fix string) *CLIError {
	return &CLIError{Message: message, Cause: err, Fix: fix}
}

// PrintError prints a formatted error to stderr with optional fix suggestion.
// In JSON mode, it outputs a JSON error instead.
func PrintError(err error) {
	if JSONMode {
		JSONError(err)
		return
	}

	if cliErr, ok := err.(*CLIError); ok {
		Error(cliErr.Message)
		if cliErr.Cause != nil {
			Debug("cause", "error", cliErr.Cause)
		}
		if cliErr.Fix != "" {
			if NoColor() {
				Info("Fix: " + cliErr.Fix)
			} else {
				Info("💡 " + cliErr.Fix)
			}
		}
	} else {
		Error(err.Error())
	}
}
