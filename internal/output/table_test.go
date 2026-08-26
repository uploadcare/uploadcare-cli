package output

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateMiddle(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"under", "photo.jpg", 20, "photo.jpg"},
		{"exact", "photo.jpg", 9, "photo.jpg"},
		{"keeps extension", "vacation-photos-iceland-2026-final-v3.jpg", 20, "vacation-...l-v3.jpg"},
		{"multibyte", "日本語のファイル名です.png", 10, "日本...png"},
		{"below ellipsis", "photo.jpg", 2, "ph"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateMiddle(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("TruncateMiddle(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
			if n := displayWidth(got); n > tt.max {
				t.Errorf("result is %d columns, exceeds max %d", n, tt.max)
			}
			if !utf8.ValidString(got) {
				t.Errorf("result %q is not valid UTF-8", got)
			}
		})
	}
}

func TestTruncateEnd(t *testing.T) {
	if got := TruncateEnd("photo.jpg", 20); got != "photo.jpg" {
		t.Errorf("under limit should be unchanged, got %q", got)
	}
	got := TruncateEnd("日本語のファイル名です", 8)
	if want := "日本..."; got != want {
		t.Errorf("TruncateEnd = %q, want %q", got, want)
	}
	if !utf8.ValidString(got) {
		t.Errorf("result %q is not valid UTF-8", got)
	}
}

func TestSanitizeCell(t *testing.T) {
	got := SanitizeCell("bad\tname\nwith\rcontrol\x00chars\x7f")
	if want := "bad name with control chars "; got != want {
		t.Errorf("SanitizeCell = %q, want %q", got, want)
	}
	if got := SanitizeCell("日本語 ok"); got != "日本語 ok" {
		t.Errorf("printable runes should survive, got %q", got)
	}
	// C1 controls: \u009b is CSI, which a terminal may act on.
	if got := SanitizeCell("csi\u009b31mred\u0085next"); got != "csi 31mred next" {
		t.Errorf("C1 controls should be replaced, got %q", got)
	}
}

// longName is longer than any terminal, like the filenames in issue #4.
var longName = strings.Repeat("a-very-long-file-name-", 15) + "final.jpg"

func fileListTable() *TableData {
	td := NewTableData("UUID", "SIZE", "FILENAME", "STORED", "UPLOADED").Flexible(2)
	td.AddRow("a1b2c3d4-e5f6-7890-abcd-ef1234567890", "1258000", longName, "true", "2026-03-01T00:00:00Z")
	td.AddRow("b2c3d4e5-f6a7-8901-bcde-f12345678901", "348160", "doc.pdf", "false", "2026-03-02T00:00:00Z")
	return td
}

func lines(t *testing.T, out string) []string {
	t.Helper()
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

func TestTableFormatter_FitsWidth(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{Width: 120}

	if err := f.Format(&buf, fileListTable()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := lines(t, buf.String())
	if len(got) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d:\n%s", len(got), buf.String())
	}
	for i, line := range got {
		if n := displayWidth(line); n > 120 {
			t.Errorf("line %d is %d columns, exceeds width 120: %q", i, n, line)
		}
	}
	if strings.Contains(buf.String(), longName) {
		t.Error("long filename should have been shortened")
	}
	if !strings.Contains(got[1], "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("UUID must stay intact, got:\n%s", got[1])
	}
	if !strings.Contains(got[1], "2026-03-01T00:00:00Z") {
		t.Errorf("timestamp must stay intact, got:\n%s", got[1])
	}
	if !strings.Contains(got[1], "...") {
		t.Errorf("shortened cell should be marked with an ellipsis, got:\n%s", got[1])
	}
}

func TestTableFormatter_NotATerminalKeepsFullValues(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{} // Width 0: a bytes.Buffer is never a terminal.

	if err := f.Format(&buf, fileListTable()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), longName) {
		t.Error("piped output must not be truncated")
	}
}

func TestTableFormatter_NarrowWidthProtectsFixedColumns(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{Width: 40}

	if err := f.Format(&buf, fileListTable()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 40 columns cannot hold a UUID plus a timestamp, so the row still
	// overflows — but only the flexible column was shortened, and it stopped
	// at minFlexWidth rather than eating into the UUID.
	out := buf.String()
	if !strings.Contains(out, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("UUID must not be shortened to make room, got:\n%s", out)
	}
	row := lines(t, out)[1]
	fields := strings.Fields(row)
	if n := displayWidth(fields[2]); n > minFlexWidth {
		t.Errorf("flexible column is %d columns, want at most %d: %q", n, minFlexWidth, fields[2])
	}
}

func TestTableFormatter_UnmarkedColumnsAreNeverShortened(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{Width: 40}

	// A detail table marks nothing flexible: the value is what the user copies.
	td := NewTableData()
	td.AddRow("URL:", "https://ucarecdn.com/a1b2c3d4-e5f6-7890-abcd-ef1234567890/"+longName)

	if err := f.Format(&buf, td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), longName) {
		t.Error("unmarked column must be printed in full")
	}
}

func TestTableFormatter_ControlCharsKeepOneRowPerRecord(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}

	td := NewTableData("UUID", "FILENAME", "STORED").Flexible(1)
	td.AddRow("a1b2c3d4-e5f6-7890-abcd-ef1234567890", "two\nlines\tand\ttabs.jpg", "true")

	if err := f.Format(&buf, td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := lines(t, buf.String()); len(got) != 2 {
		t.Errorf("expected 2 lines (header + 1 row), got %d:\n%s", len(got), buf.String())
	}
}

func TestTableFormatter_DoesNotMutateInput(t *testing.T) {
	td := fileListTable()
	var buf bytes.Buffer

	if err := (&TableFormatter{Width: 60}).Format(&buf, td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if td.Rows[0][2] != longName {
		t.Errorf("formatting must not modify the caller's data, got %q", td.Rows[0][2])
	}
}

// A CJK filename draws two terminal columns per character. Measured in runes it
// slips under the budget and then wraps, which is the very thing the fitter
// exists to prevent.
func TestTableFormatter_WideCharsFitTheBudget(t *testing.T) {
	for _, width := range []int{72, 80, 100, 140} {
		var buf bytes.Buffer
		td := NewTableData("UUID", "SIZE", "FILENAME", "STORED").Flexible(2)
		td.AddRow("a1b2c3d4-e5f6-7890-abcd-ef1234567890", "1258000", "日本語の写真ファイル名です-最終版.jpg", "true")
		td.AddRow("b2c3d4e5-f6a7-8901-bcde-f12345678901", "348160", "ascii-document-name-here.pdf", "true")

		if err := (&TableFormatter{Width: width}).Format(&buf, td); err != nil {
			t.Fatalf("width %d: unexpected error: %v", width, err)
		}

		got := lines(t, buf.String())
		if len(got) != 3 {
			t.Errorf("width %d: expected 3 lines, got %d:\n%s", width, len(got), buf.String())
		}
		for i, line := range got {
			if n := displayWidth(line); n > width {
				t.Errorf("width %d: line %d draws %d columns:\n%s", width, i, n, line)
			}
		}
	}
}

// Columns must line up in drawn position, not in rune position.
func TestTableFormatter_WideCharsStayAligned(t *testing.T) {
	var buf bytes.Buffer
	td := NewTableData("FILENAME", "STORED").Flexible(0)
	td.AddRow("日本語のファイル.jpg", "true")
	td.AddRow("ascii.jpg", "false")
	td.AddRow("mixed-日本-name.png", "true")

	if err := (&TableFormatter{Width: 100}).Format(&buf, td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var want int
	for i, line := range lines(t, buf.String()) {
		col := displayWidth(line[:strings.LastIndex(line, "  ")+2])
		if i == 0 {
			want = col
			continue
		}
		if col != want {
			t.Errorf("line %d starts its last column at %d, want %d:\n%s", i, col, want, line)
		}
	}
}

// Cutting between the two regional indicators of a flag, or between a letter
// and its combining accent, produces a mangled character.
func TestTruncateMiddle_KeepsGraphemeClustersWhole(t *testing.T) {
	const flag = "\U0001F1EF\U0001F1F5" // 🇯🇵, two runes drawn as one glyph
	combining := "cafe\u0301-photo-archive-final.jpg"

	for _, max := range []int{4, 8, 12, 13, 14, 15, 16, 20} {
		got := TruncateMiddle("trip-"+flag+"-photos-final.jpg", max)
		if strings.ContainsRune(got, 0x1F1EF) != strings.ContainsRune(got, 0x1F1F5) {
			t.Errorf("max %d: split the flag emoji: %q", max, got)
		}
		if n := displayWidth(got); n > max {
			t.Errorf("max %d: result draws %d columns: %q", max, n, got)
		}

		got = TruncateMiddle(combining, max)
		if strings.ContainsRune(got, 0x0301) && !strings.Contains(got, "e\u0301") {
			t.Errorf("max %d: detached the combining accent: %q", max, got)
		}
	}
}
