// Package output provides styled terminal output utilities for lzctl.
//
// It wraps charmbracelet/log for structured logging and charmbracelet/lipgloss
// for styled output. All user-facing output should go through this package
// rather than using fmt.Println directly.
//
// Features:
//   - Styled logging with emoji prefixes (Info, Warn, Error, Debug)
//   - JSON output mode for CI/scripting (--json flag)
//   - NO_COLOR environment variable support
//   - Verbose/debug mode via -v flag
package output
