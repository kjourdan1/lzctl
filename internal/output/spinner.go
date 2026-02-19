package output

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Spinner provides a simple terminal spinner for long-running operations.
// It is safe to use concurrently and handles Ctrl+C gracefully.
type Spinner struct {
	message string
	done    chan struct{}
	once    sync.Once
	active  bool
	mu      sync.Mutex
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation in a goroutine.
// It is safe to call Start multiple times; only the first call starts the spinner.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	if JSONMode {
		return // no spinner in JSON mode
	}

	go s.run()
}

func (s *Spinner) run() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	if NoColor() {
		frames = []string{"|", "/", "-", "\\"}
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-s.done:
			fmt.Fprintf(os.Stderr, "\r\033[K") // clear line
			return
		case <-ticker.C:
			fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], s.message)
			i++
		}
	}
}

// Stop stops the spinner. It is safe to call Stop multiple times.
func (s *Spinner) Stop() {
	s.once.Do(func() {
		s.mu.Lock()
		s.active = false
		s.mu.Unlock()
		close(s.done)
	})
}

// StopWithSuccess stops the spinner and prints a success message.
func (s *Spinner) StopWithSuccess(msg string) {
	s.Stop()
	Success(msg)
}

// StopWithError stops the spinner and prints an error message.
func (s *Spinner) StopWithError(msg string) {
	s.Stop()
	Fail(msg)
}

// StopWithWarning stops the spinner and prints a warning message.
func (s *Spinner) StopWithWarning(msg string) {
	s.Stop()
	Warn(msg)
}

// WithSpinner runs a function with a spinner, stopping it when done.
// Returns the error from the function, if any.
func WithSpinner(message string, fn func() error) error {
	sp := NewSpinner(message)
	sp.Start()
	err := fn()
	if err != nil {
		sp.StopWithError(message + " — failed")
	} else {
		sp.StopWithSuccess(message)
	}
	return err
}
