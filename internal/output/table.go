package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

// TableFormatter writes data as a human-readable table.
type TableFormatter struct{}

// Format writes data as a tab-aligned table.
// data must be a TableData value.
func (f *TableFormatter) Format(w io.Writer, data any) error {
	td, ok := data.(*TableData)
	if !ok {
		// Fall back to fmt for non-table data.
		_, err := fmt.Fprintf(w, "%v\n", data)
		return err
	}

	// Write through a buffer so tabwriter aligns plain text first,
	// then apply bold to the header line. Coloring before tabwriter
	// breaks alignment because ANSI escapes inflate byte counts.
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	// Header
	if len(td.Headers) > 0 {
		_, _ = fmt.Fprintln(tw, strings.Join(td.Headers, "\t"))
	}

	// Rows
	for _, row := range td.Rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	out := buf.String()

	// Bold the first line (header) after alignment is done.
	if len(td.Headers) > 0 {
		if i := strings.IndexByte(out, '\n'); i >= 0 {
			bold := color.New(color.Bold)
			out = bold.Sprint(out[:i]) + out[i:]
		}
	}

	_, err := io.WriteString(w, out)
	return err
}

// TableData is the structured input for the table formatter.
type TableData struct {
	Headers []string
	Rows    [][]string
}

// NewTableData creates a TableData with the given headers.
func NewTableData(headers ...string) *TableData {
	return &TableData{Headers: headers}
}

// AddRow appends a row of values to the table.
func (td *TableData) AddRow(values ...string) {
	td.Rows = append(td.Rows, values)
}
