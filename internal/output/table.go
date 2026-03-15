package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
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

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	if len(td.Headers) > 0 {
		fmt.Fprintln(tw, strings.Join(td.Headers, "\t"))
	}

	// Rows
	for _, row := range td.Rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	return tw.Flush()
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
