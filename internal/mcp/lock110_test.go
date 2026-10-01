package mcp

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-110 T3, the Codex review of #307. At the smallest ceiling a write is
// allowed under, a write refused before any hunk had a verdict — the write
// lock held past its wait — overflowed the receipt, and the terminal sentence
// said only "0 of 0 hunk(s) failed": the lock, the holder and the remedy were
// gone. The refusal's own words now survive the ceiling.
func TestAWriteLockRefusalSurvivesTheSmallestCeiling(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	release, err := seen.LockWrites(root)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	t.Setenv("MRW_WRITE_LOCK_TIMEOUT", "0")
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	MaxResultChars = minWriteCeiling()

	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nb\n"})
	text := served0(t, res)
	if isErr, _ := res["isError"].(bool); !isErr || !strings.Contains(text, "write lock") || !strings.Contains(text, strconv.Itoa(os.Getpid())) {
		t.Errorf("a write-lock refusal at a %d-character ceiling lost its words (isError %v):\n%s", MaxResultChars, res["isError"], text)
	}
}
