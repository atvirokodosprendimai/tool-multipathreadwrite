package mcp

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// plantLedger157 writes header as the root's ledger, whose first line it is.
func plantLedger157(t *testing.T, root, header string) {
	t.Helper()
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte(header+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ADR-157 Decisions 1 and 3 over MCP. Under a newer ledger mrw_read says, on a
// fitting read and on a page, that an ack licenses nothing, and an mrw_write
// that left the read's checkpoints unacked is refused led by the cause with no
// ack advice. The pair: under a current ledger neither line appears and the
// unacked write keeps its ack advice.
func TestAnMCPReadUnderANewerLedgerSaysNoAckLicenses(t *testing.T) {
	const line = "an ack here licenses nothing"
	for _, c := range []struct {
		header string
		newer  bool
	}{{"#mrw-seen v6 v9.9.9", true}, {"#mrw-seen v5 v1.66.0 (05ec3f8)", false}} {
		root, path := checkout(t, "a.txt", "one\ntwo\n")
		plantLedger157(t, root, c.header)
		txt := served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}}))
		if strings.Contains(txt, line) != c.newer {
			t.Errorf("%s: a fitting read says %q: %v, want %v\n%s", c.header, line, strings.Contains(txt, line), c.newer, txt)
		}
		text := firstText(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nONE\n"}))
		if c.newer {
			if !strings.Contains(text, "a.txt cannot be edited by this mrw: the read ledger was written by a newer mrw (v9.9.9)") {
				t.Errorf("the unacked write is not refused led by the cause: %q", text)
			}
			if strings.Contains(text, "never acknowledged") {
				t.Errorf("the unacked write under a newer ledger advises an ack: %q", text)
			}
		} else if !strings.Contains(text, "never acknowledged") {
			t.Errorf("the pair: the unacked write lost its ack advice: %q", text)
		}
	}

	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	var b strings.Builder
	for i := 1; i <= 600; i++ {
		b.WriteString("line " + strconv.Itoa(i) + "\n")
	}
	root, path := checkout(t, "big.txt", b.String())
	plantLedger157(t, root, "#mrw-seen v6 v9.9.9")
	MaxResultChars = 6000
	txt := served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}}))
	if !strings.Contains(txt, "-- PARTIAL:") || !strings.Contains(txt, line) {
		t.Errorf("a page under a newer ledger does not say %q:\n%s", line, txt)
	}
}
