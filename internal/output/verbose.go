package output

import (
	"fmt"
	"io"
	"time"
)

// VerboseLogger writes diagnostic details to an io.Writer (typically stderr).
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

// Credential logs a resolved credential with its value partially masked.
// Only the first 4 characters are shown; the rest is replaced with asterisks.
func (v *VerboseLogger) Credential(name, value string) {
	if v == nil {
		return
	}
	fmt.Fprintf(v.w, "  %s: %s\n", name, MaskSecret(value))
}

// Info logs a general-purpose key-value diagnostic line.
func (v *VerboseLogger) Info(key, value string) {
	if v == nil {
		return
	}
	fmt.Fprintf(v.w, "  %s: %s\n", key, value)
}

// Infof logs a general-purpose formatted diagnostic line.
func (v *VerboseLogger) Infof(format string, args ...any) {
	if v == nil {
		return
	}
	fmt.Fprintf(v.w, "  "+format+"\n", args...)
}

// MaskSecret returns a masked version of a secret string.
// Shows the first 4 characters followed by asterisks.
func MaskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}
