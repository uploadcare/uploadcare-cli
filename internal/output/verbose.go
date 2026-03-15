package output

import (
	"fmt"
	"io"
	"time"
)

// VerboseLogger writes HTTP request/response details to an io.Writer (typically stderr).
// It is safe to call methods on a nil *VerboseLogger — they are no-ops.
type VerboseLogger struct {
	w io.Writer
}

// NewVerboseLogger creates a VerboseLogger that writes to w.
// Returns nil if verbose mode is disabled.
func NewVerboseLogger(enabled bool, w io.Writer) *VerboseLogger {
	if !enabled {
		return nil
	}
	return &VerboseLogger{w: w}
}

// Enabled reports whether verbose logging is active.
func (v *VerboseLogger) Enabled() bool {
	return v != nil
}

// Request logs an outgoing HTTP request line: --> METHOD URL
func (v *VerboseLogger) Request(method, url string) {
	if v == nil {
		return
	}
	fmt.Fprintf(v.w, "--> %s %s\n", method, url)
}

// Response logs an incoming HTTP response line: <-- STATUS (duration)
func (v *VerboseLogger) Response(status string, duration time.Duration) {
	if v == nil {
		return
	}
	fmt.Fprintf(v.w, "<-- %s (%dms)\n", status, duration.Milliseconds())
}

// AuthSource logs which credential source was used.
func (v *VerboseLogger) AuthSource(source string) {
	if v == nil {
		return
	}
	fmt.Fprintf(v.w, "  auth: %s\n", source)
}

// Retry logs a retry attempt.
func (v *VerboseLogger) Retry(attempt int, reason string) {
	if v == nil {
		return
	}
	fmt.Fprintf(v.w, "  retry #%d: %s\n", attempt, reason)
}
