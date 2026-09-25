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

// ADR-073. Unsplittable names what makes bytes impossible to edit by line: a
// UTF-32 byte-order mark (checked first, because FF FE is its prefix), a UTF-16
// one, or a NUL in the first 8 KiB. UTF-8, with or without its own BOM, splits.
func TestUnsplittableNamesEveryEncodingItKnows(t *testing.T) {
	for in, want := range map[string]string{
		"\xff\xfe\x00\x00a\x00\x00\x00":  "is UTF-32 (BOM FF FE 00 00)",
		"\x00\x00\xfe\xff\x00\x00\x00a":  "is UTF-32 (BOM 00 00 FE FF)",
		"\xff\xfea\x00":                  "is UTF-16 (BOM FF FE)",
		"\xfe\xff\x00a":                  "is UTF-16 (BOM FE FF)",
		"ab\x00cd":                       "holds a NUL byte at offset 2",
		"plain text\n":                   "",
		"\xef\xbb\xbfutf-8 with a BOM\n": "",
		"":                               "",
	} {
		if got := Unsplittable([]byte(in)); got != want {
			t.Errorf("Unsplittable(%q) = %q, want %q", in, got, want)
		}
	}
}
