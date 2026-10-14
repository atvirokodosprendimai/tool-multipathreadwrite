package ingest

import (
	"strings"
	"testing"
)

// The Codex review of #371, second pass. A CR the parser would strip is refused
// in every terminator mode, not only when the content is LF: the last line of
// CRLF content that ends in a bare CR, and a CR before a CRLF.
func TestACRTheParserWouldStripIsRefusedInEveryMode(t *testing.T) {
	for _, c := range []string{"a\nb\r", "a\r\nb\r", "a\r\r\n", "a\r\nb\r\nc\r"} {
		if _, err := CompileCreate("m.txt", []byte(c)); err == nil || !strings.Contains(err.Error(), "bare CR") {
			t.Errorf("%q: %v, want a refusal naming the bare CR", c, err)
		}
	}
	for _, c := range []string{"a\nb\n", "a\r\nb\r\n", "a\rb\rc\r", "a"} {
		if _, err := CompileCreate("ok.txt", []byte(c)); err != nil {
			t.Errorf("%q refused: %v", c, err)
		}
	}
}

// A line the plan parser's bounded scanner cannot read is refused while
// compiling, so it is not counted as a plan that failed to parse.
func TestALineBeyondThePlanScannerIsRefusedWhileCompiling(t *testing.T) {
	long := strings.Repeat("x", maxPlanLine)
	if _, err := CompileCreate("big.txt", []byte(long)); err == nil || !strings.Contains(err.Error(), "longer than") {
		t.Errorf("a %d-byte line: %v, want a refusal", len(long), err)
	}
	if out, err := CompileCreate("fits.txt", []byte(strings.Repeat("x", 1<<20))); err != nil {
		t.Errorf("a 1 MiB line refused: %v", err)
	} else if h := parseOne(t, out); len(h.Body) != 1 || len(h.Body[0]) != 1<<20 {
		t.Errorf("a 1 MiB line did not round-trip: %d lines", len(h.Body))
	}
}
