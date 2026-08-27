package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// colGap is the number of spaces between columns.
const colGap = 2

// TableFormatter writes data as a human-readable table.
type TableFormatter struct {
	// Width is the line width to fit rows into. Zero means detect it from the
	// writer, which yields 0 for anything that is not a terminal — piped output
	// is never truncated.
	Width int
}

// Format writes data as an aligned table.
// data must be a TableData value.
func (f *TableFormatter) Format(w io.Writer, data any) error {
	td, ok := data.(*TableData)
	if !ok {
		// Fall back to fmt for non-table data.
		_, err := fmt.Fprintf(w, "%v\n", data)
		return err
	}

	headers, rows := td.sanitized()

	width := f.Width
	if width == 0 {
		width = terminalWidth(w)
	}
	if width > 0 {
		fitColumns(headers, rows, td.flex, width)
	}

	// Pad in terminal columns rather than deferring to text/tabwriter, which
	// measures cells in runes and so misaligns — and overflows the width
	// budget on — CJK and emoji text.
	cols := columnWidths(headers, rows)

	var buf strings.Builder
	if len(headers) > 0 {
		bold := color.New(color.Bold)
		buf.WriteString(bold.Sprint(renderRow(headers, cols)))
		buf.WriteByte('\n')
	}
	for _, row := range rows {
		buf.WriteString(renderRow(row, cols))
		buf.WriteByte('\n')
	}

	_, err := io.WriteString(w, buf.String())
	return err
}

// columnWidths returns the width in terminal columns of each column's widest
// cell, header included.
func columnWidths(headers []string, rows [][]string) []int {
	n := len(headers)
	for _, row := range rows {
		n = max(n, len(row))
	}
	widths := make([]int, n)
	for i, h := range headers {
		widths[i] = max(widths[i], displayWidth(h))
	}
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], displayWidth(cell))
		}
	}
	return widths
}

// renderRow pads every cell but the last to its column width. The final cell
// is left unpadded so rows carry no trailing whitespace.
func renderRow(cells []string, cols []int) string {
	var b strings.Builder
	for i, cell := range cells {
		b.WriteString(cell)
		if i == len(cells)-1 {
			break
		}
		b.WriteString(strings.Repeat(" ", cols[i]-displayWidth(cell)+colGap))
	}
	return b.String()
}

// lineWidth is the width of the widest rendered row for the given columns.
func lineWidth(cols []int) int {
	if len(cols) == 0 {
		return 0
	}
	w := colGap * (len(cols) - 1)
	for _, c := range cols {
		w += c
	}
	return w
}

// fitColumns shortens the flexible columns in place until the widest line fits
// within width. They are shrunk widest-first and never below minFlexWidth; when
// that is not enough the line is left to wrap. Columns the caller did not mark
// flexible are never touched, so fixed-format values — UUIDs, timestamps — stay
// intact and copy-pasteable at any terminal width.
func fitColumns(headers []string, rows [][]string, flex []int, width int) {
	if len(flex) == 0 {
		return
	}

	cols := columnWidths(headers, rows)
	line := lineWidth(cols)
	if line <= width {
		return
	}

	candidates := make([]int, 0, len(flex))
	for _, c := range flex {
		if c >= 0 && c < len(cols) {
			candidates = append(candidates, c)
		}
	}
	sort.SliceStable(candidates, func(a, b int) bool {
		return cols[candidates[a]] > cols[candidates[b]]
	})

	for _, c := range candidates {
		if line <= width {
			return
		}
		floor := minFlexWidth
		if c < len(headers) {
			floor = max(floor, displayWidth(headers[c]))
		}
		cut := min(cols[c]-floor, line-width)
		if cut <= 0 {
			continue
		}
		target := cols[c] - cut
		for _, row := range rows {
			if c < len(row) {
				row[c] = TruncateMiddle(row[c], target)
			}
		}
		// A wide cluster may not divide evenly into the target, so re-measure
		// rather than assuming the column now sits exactly at it.
		cols = columnWidths(headers, rows)
		line = lineWidth(cols)
	}
}

// TableData is the structured input for the table formatter.
type TableData struct {
	Headers []string
	Rows    [][]string

	flex []int
}

// NewTableData creates a TableData with the given headers.
func NewTableData(headers ...string) *TableData {
	return &TableData{Headers: headers}
}

// AddRow appends a row of values to the table.
func (td *TableData) AddRow(values ...string) {
	td.Rows = append(td.Rows, values)
}

// Flexible marks the columns holding user-controlled text — filenames, URLs,
// paths — that may be shortened to fit the terminal. Columns left unmarked are
// printed in full. Detail tables deliberately mark nothing: their value column
// is what the user came to copy.
func (td *TableData) Flexible(cols ...int) *TableData {
	td.flex = cols
	return td
}

// sanitized returns copies of the headers and rows with control characters
// replaced, so callers' data is never mutated by fitting.
func (td *TableData) sanitized() ([]string, [][]string) {
	headers := make([]string, len(td.Headers))
	for i, h := range td.Headers {
		headers[i] = SanitizeCell(h)
	}
	rows := make([][]string, len(td.Rows))
	for i, row := range td.Rows {
		rows[i] = make([]string, len(row))
		for j, cell := range row {
			rows[i][j] = SanitizeCell(cell)
		}
	}
	return headers, rows
}
