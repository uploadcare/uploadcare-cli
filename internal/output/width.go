package output

import (
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
	"golang.org/x/term"
)

const (
	// defaultWidth is assumed when stdout is a terminal of unreported size.
	defaultWidth = 80

	// minFlexWidth is the narrowest a shrinkable column may become. Below this
	// a truncated value carries no information worth the line it occupies.
	minFlexWidth = 12

	ellipsis = "..."
)

// terminalWidth reports the usable width of w, or 0 when w is not a terminal
// (piped, redirected, or a test buffer) and output must not be truncated.
func terminalWidth(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return 0
	}
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return 0
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return defaultWidth
	}
	return width
}

// displayWidth reports how many terminal columns s occupies. Counting runes is
// not enough: CJK characters and most emoji draw two columns wide, while
// combining marks and joiners draw none.
func displayWidth(s string) int {
	return uniseg.StringWidth(s)
}

// cluster is one grapheme cluster and the columns it draws. Cutting anywhere
// other than a cluster boundary splits a character: a flag emoji becomes a lone
// regional indicator, an accent detaches from its letter.
type cluster struct {
	text  string
	width int
}

func clusters(s string) []cluster {
	out := make([]cluster, 0, len(s))
	state := -1
	for len(s) > 0 {
		var c string
		var w int
		c, s, w, state = uniseg.FirstGraphemeClusterInString(s, state)
		out = append(out, cluster{c, w})
	}
	return out
}

// prefixLen returns how many leading clusters of cs fit in max columns.
func prefixLen(cs []cluster, max int) int {
	w, n := 0, 0
	for _, c := range cs {
		if w+c.width > max {
			break
		}
		w += c.width
		n++
	}
	return n
}

// suffixLen returns how many trailing clusters of cs fit in max columns.
func suffixLen(cs []cluster, max int) int {
	w, n := 0, 0
	for i := len(cs) - 1; i >= 0; i-- {
		if w+cs[i].width > max {
			break
		}
		w += cs[i].width
		n++
	}
	return n
}

func join(cs []cluster) string {
	var b strings.Builder
	for _, c := range cs {
		b.WriteString(c.text)
	}
	return b.String()
}

// TruncateMiddle shortens s to at most max terminal columns, keeping both head
// and tail so a file extension stays visible:
// "vacation-photos-ice...-final-v3.jpg".
func TruncateMiddle(s string, max int) string {
	if displayWidth(s) <= max {
		return s
	}
	cs := clusters(s)
	if max <= len(ellipsis) {
		return join(cs[:prefixLen(cs, max)])
	}
	keep := max - len(ellipsis)
	head := (keep + 1) / 2

	h := prefixLen(cs, head)
	// Scan the tail only over what the head left behind, so a zero-width
	// cluster cannot be emitted twice.
	t := suffixLen(cs[h:], keep-head)
	return join(cs[:h]) + ellipsis + join(cs[len(cs)-t:])
}

// TruncateEnd shortens s to at most max terminal columns, marking the cut with
// a trailing "...".
func TruncateEnd(s string, max int) string {
	if displayWidth(s) <= max {
		return s
	}
	cs := clusters(s)
	if max <= len(ellipsis) {
		return join(cs[:prefixLen(cs, max)])
	}
	return join(cs[:prefixLen(cs, max-len(ellipsis))]) + ellipsis
}

// SanitizeCell replaces control characters with spaces so that one record
// always renders as one line. Filenames and metadata come back from the API
// unfiltered, and a literal tab or newline in a cell splits the row. This
// covers C1 as well as C0, so a filename cannot smuggle U+009B — the one-byte
// form of CSI — past the formatter and into the terminal.
func SanitizeCell(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
