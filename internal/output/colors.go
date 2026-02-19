package output

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// NoColor returns true if colored output should be disabled.
// Respects the NO_COLOR environment variable (https://no-color.org/).
func NoColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

// Color definitions for consistent styling across the CLI.
var (
	ColorSuccess = lipgloss.Color("#2ECC71") // green
	ColorWarning = lipgloss.Color("#F39C12") // orange
	ColorError   = lipgloss.Color("#E74C3C") // red
	ColorInfo    = lipgloss.Color("#3498DB") // blue
	ColorMuted   = lipgloss.Color("#95A5A6") // gray
	ColorAccent  = lipgloss.Color("#9B59B6") // purple
)

// Style presets for common output patterns.
var (
	// StyleBold is for emphasis.
	StyleBold = lipgloss.NewStyle().Bold(true)

	// StyleTitle is for section headers.
	StyleTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)

	// StyleSuccess is for success indicators.
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)

	// StyleWarning is for warning indicators.
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)

	// StyleError is for error indicators.
	StyleError = lipgloss.NewStyle().Foreground(ColorError)

	// StyleMuted is for secondary/less important text.
	StyleMuted = lipgloss.NewStyle().Foreground(ColorMuted)
)

// plainStyles returns styles without color for NO_COLOR mode.
func plainStyles() *log.Styles {
	styles := log.DefaultStyles()
	// In NO_COLOR mode, the charmbracelet/log library already
	// handles stripping styles when rendering to a non-TTY.
	// We just return defaults here for the logger configuration.
	return styles
}
