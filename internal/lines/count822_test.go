package lines

import "testing"

// TestCountAgreesWithSplit holds #822: Count is how a header-only read numbers a
// file without building its lines, so it must give the number Split gives for
// every terminator Split knows, mixed and unterminated text included.
func TestCountAgreesWithSplit(t *testing.T) {
	for _, s := range []string{"", "x", "\n", "\r", "\r\n", "\r\r\n", "\r\n\n", "a\rb\r", "a\rb", "\n\r", "a\nb\r\nc", "\x00\n", "\xff\r", "\xef\xbb\xbfone\n", "a\r\nb\r\n", "a\r\nb"} {
		want, _, _ := Split(s)
		if got := Count([]byte(s)); got != len(want) {
			t.Errorf("Count(%q) = %d, Split returned %d lines", s, got, len(want))
		}
	}
}
