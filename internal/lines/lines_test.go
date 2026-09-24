package lines

import "testing"

// ADR-065 T1. An empty file has no lines on every surface: read's `0L`, a
// write's empty text and MCP paging all start from this.
func TestSplitOfAnEmptyFileHasNoLines(t *testing.T) {
	ls, eol, final := Split("")
	if len(ls) != 0 || eol != "\n" || final {
		t.Fatalf("Split(\"\") = %q, %q, %v; want no lines, LF, not final", ls, eol, final)
	}
}
