package mcp

import (
	"strings"
	"testing"
)

// ADR-108 T7. A read printed a file's raw bytes and JSON replaced the ones that
// are not valid UTF-8, while the checkpoint licensed the original bytes: a
// line copied back from what the host showed could corrupt the file. Such a
// file is served without a checkpoint and with a line saying why, and nothing
// is held; a file holding a literal U+FFFD is valid UTF-8 and serves as before.
func TestNonUTF8ContentIsServedUnlicensedOverMCP(t *testing.T) {
	root, _ := checkout(t, "latin.txt", "caf\xe9\n")
	text := served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"latin.txt"}}))
	if strings.Contains(text, "-- ck ") || !strings.Contains(text, "not valid UTF-8") {
		t.Errorf("a Latin-1 file was served with a checkpoint, or without saying why:\n%s", text)
	}
	if store, err := loadPending(root); err != nil || len(store) != 0 {
		t.Errorf("a non-UTF-8 serve held %d checkpoint(s) (%v)", len(store), err)
	}

	root, _ = checkout(t, "fffd.txt", "a�b\n")
	if text := served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"fffd.txt"}})); !strings.Contains(text, "-- ck ") {
		t.Errorf("a file holding a literal U+FFFD lost its checkpoint:\n%s", text)
	}
}
