package output

import (
	"errors"
	"io"
)

// ErrVerboseQuietConflict is returned when both --verbose and --quiet are specified.
var ErrVerboseQuietConflict = errors.New("--verbose and --quiet are mutually exclusive")

// Formatter is the interface for all output formatters.
// Commands call Format to write their results; the formatter
// decides how to render them (table, JSON, NDJSON, etc.).
type Formatter interface {
	// Format writes the given data to the output.
	// data can be a single object or a slice.
	Format(w io.Writer, data any) error
}

// FormatOptions holds options parsed from global flags that
// determine which formatter to use and how to configure it.
type FormatOptions struct {
	// JSON is true when --json is specified.
	JSON bool

	// Fields is the list of field names to include in JSON output.
	// Empty means include all fields.
	Fields []string

	// JQ is a jq expression to apply to JSON output.
	JQ string

	// Quiet suppresses all non-error output.
	Quiet bool

	// Verbose enables HTTP request/response logging to stderr.
	Verbose bool
}

// Validate checks for conflicting options.
func (o FormatOptions) Validate() error {
	if o.Verbose && o.Quiet {
		return ErrVerboseQuietConflict
	}
	return nil
}

// New creates the appropriate Formatter based on the given options.
func New(opts FormatOptions) Formatter {
	if opts.Quiet {
		return &quietFormatter{}
	}
	if opts.JSON || opts.JQ != "" {
		return &JSONFormatter{Fields: opts.Fields, JQ: opts.JQ}
	}
	return &TableFormatter{}
}

// quietFormatter discards all output.
type quietFormatter struct{}

func (q *quietFormatter) Format(w io.Writer, data any) error {
	return nil
}
