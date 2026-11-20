package mcp

import (
	"strings"
	"testing"
)

// Final review of #740. read.Run prints a name holding a control byte quoted and
// escaped, and a literal file named with those quote characters used to print the
// same text, so one file's observation answered for the other's header and the
// line-feed file's checkpoints were issued and never held.
func TestAnEscapedHeaderNeverEqualsALiteralFileName(t *testing.T) {
	root := lfFiles(t, map[string]string{"draft\nnotes.txt": "alpha\n", `"draft\nnotes.txt"`: "beta\n"})
	text, l := lfServed(t, root, map[string]any{"specs": []any{"draft\nnotes.txt", `"draft\nnotes.txt"`}})
	if n := strings.Count(text, lfNote); n != 1 {
		t.Errorf("%d notes, want 1 (the file with the line feed):\n%s", n, text)
	}
	if n := strings.Count(text, " open lines "); n != 1 {
		t.Errorf("%d checkpointed files, want 1 (the literal one):\n%s", n, text)
	}
	lfCovers(t, l, `"draft\nnotes.txt"`, 1, 1)
	if len(l) != 1 {
		t.Errorf("the ledger holds %+v, want only the literal file", l)
	}
}

// A trailing carriage return is part of a name the ledger keeps (ADR-068), so a
// file named with one is checkpointed and licensed like any other.
func TestANameWithATrailingCarriageReturnIsCheckpointed(t *testing.T) {
	root := lfFiles(t, map[string]string{"notes\r": "one\n"})
	text, l := lfServed(t, root, map[string]any{"specs": []any{"notes\r"}})
	if strings.Contains(text, lfNote) {
		t.Errorf("a name the ledger can hold was called unrecordable:\n%s", text)
	}
	lfCovers(t, l, "notes\r", 1, 1)
}
