package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-065 T1. MCP paging counts the lines read serves. A CR-only file used to
// count as ONE line (bufio.Scanner splits on \n), so paging arithmetic and the
// served lines disagreed. The oracle is the fixture: the numbered lines served
// across every page are exactly 1..n with the fixture's content, none twice;
// next_read only advances the loop, bounded so a stuck cursor fails. A page
// that was served but not acknowledged licenses nothing; after ack, a line on
// it is writable.
func TestPagingACROnlyFileToExhaustionCoversEveryLine(t *testing.T) {
	const n = 12000
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	want := make([]string, n)
	var b strings.Builder
	for i := range want {
		want[i] = fmt.Sprintf("line %05d padding padding padding padding padding", i+1)
		b.WriteString(want[i])
		b.WriteString("\r")
	}
	if err := os.WriteFile(filepath.Join(root, "cr.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	var got []string
	var firstPage map[string]any
	spec := "cr.txt"
	for page := 0; ; page++ {
		if page > n {
			t.Fatalf("still paging after %d pages — the continuation is not advancing", page)
		}
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}})
		if firstPage == nil {
			firstPage = res
		}
		got = append(got, numberedLines(t, served0(t, res))...)
		next := nextOf(t, res)
		if next == "" {
			break
		}
		if next == spec {
			t.Fatalf("page %d hands back the spec it was given (%q)", page, next)
		}
		spec = next
	}
	if len(got) != n {
		t.Fatalf("paging served %d lines of a %d-line CR-only file", len(got), n)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("served line %d = %q, want %q", i+1, got[i], want[i])
		}
	}

	call(t, root, "mrw_write", map[string]any{"plan": "@@ cr.txt 1 replace\nX\n"})
	if body, err := os.ReadFile(filepath.Join(root, "cr.txt")); err != nil || !strings.HasPrefix(string(body), want[0]+"\r") {
		t.Fatalf("a write to an unacknowledged line changed the file (err %v)", err)
	}
	w := call(t, root, "mrw_write", map[string]any{
		"plan": "@@ cr.txt 1 replace\nX\n", "ack": checkpointsIn(served0(t, firstPage))})
	if w["isError"] == true {
		t.Fatalf("a write to an acknowledged line was refused: %v", w["content"])
	}
	body, err := os.ReadFile(filepath.Join(root, "cr.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), "X\rline 00002") {
		t.Fatalf("the acknowledged write did not replace exactly line 1: %q", string(body[:40]))
	}
}
