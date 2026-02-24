package output

import (
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/log"
)

// Logger is the global styled logger for lzctl.
// All user-facing output should go through this logger.
var (
	logger   *log.Logger
	loggerMu sync.Mutex
	logLevel = log.InfoLevel

	// JSONMode controls whether output should be JSON-formatted.
	JSONMode bool

	// Verbose controls debug-level output.
	Verbose bool
)

// Init initializes the global logger with the given settings.
// Call this once at startup (typically from root command PersistentPreRun).
func Init(verbose bool, jsonMode bool) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	Verbose = verbose
	JSONMode = jsonMode
	if verbose {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}
	logger = newLogger(os.Stderr)
}

func newLogger(w io.Writer) *log.Logger {
	l := log.NewWithOptions(w, log.Options{
		ReportTimestamp: false,
		Level:           logLevel,
	})
	if NoColor() {
		l.SetStyles(plainStyles())
	}
	return l
}

// getLogger returns the global logger, initializing with defaults if needed.
func getLogger() *log.Logger {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	if logger == nil {
		logger = newLogger(os.Stderr)
	}
	return logger
}

// Info prints an informational message.
func Info(msg string, keyvals ...interface{}) {
	if JSONMode {
		return // JSON mode suppresses text output; use JSON() instead
	}
	getLogger().Info(msg, keyvals...)
}

// Warn prints a warning message.
func Warn(msg string, keyvals ...interface{}) {
	if JSONMode {
		return
	}
	getLogger().Warn(msg, keyvals...)
}

// Error prints an error message.
func Error(msg string, keyvals ...interface{}) {
	if JSONMode {
		return
	}
	getLogger().Error(msg, keyvals...)
}

// Debug prints a debug message (only visible with -v flag).
func Debug(msg string, keyvals ...interface{}) {
	if JSONMode {
		return
	}
	getLogger().Debug(msg, keyvals...)
}

// Fatal prints an error message and exits with code 1.
func Fatal(msg string, keyvals ...interface{}) {
	getLogger().Fatal(msg, keyvals...)
}

// Success prints a success message with a checkmark prefix.
func Success(msg string) {
	if JSONMode {
		return
	}
	if NoColor() {
		getLogger().Info("[OK] " + msg)
	} else {
		getLogger().Info("✅ " + msg)
	}
}

// Fail prints a failure message with an X prefix.
func Fail(msg string) {
	if JSONMode {
		return
	}
	if NoColor() {
		getLogger().Error("[FAIL] " + msg)
	} else {
		getLogger().Error("❌ " + msg)
	}
}

// Step prints a step progress message.
func Step(msg string) {
	if JSONMode {
		return
	}
	if NoColor() {
		getLogger().Info(">> " + msg)
	} else {
		getLogger().Info("▸ " + msg)
	}
}
