package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// #741, the paged path of #716, and the continuation that fits. A file named with
// a line feed has its header split in two by read.Run; every page of it, first,
// middle and the last that fits (which takes the ordinary path, through
// markServed), is served unmarked with the header on one line and the note, and
// holds nothing. The header LINE is asserted, not a name somewhere in the text
// (the footer names it too), and next_read is followed to the end.
func TestEveryPageOfAFileNamedWithALineFeedIsServedUnmarkedWithOneLineHeader(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a line feed is not a legal Windows file name")
	}
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	var body strings.Builder
	for i := 1; i <= 400; i++ {
		fmt.Fprintf(&body, "line %05d of the fixture\n", i)
	}
	name := "a\nb.txt"
	if err := os.WriteFile(filepath.Join(root, name), []byte(body.String()), 0o644); err != nil {
		t.Skipf("this filesystem keeps no such name: %v", err)
	}
	withCeiling(t, 4000)
	spec := name
	pages, fitted := 0, false
	for ; spec != "" && pages < 30; pages++ {
		res := call(t, root, "mrw_read", map[string]any{"specs": []string{spec}})
		text := firstText(t, res)
		if res["isError"] == true {
			t.Fatalf("page %d (%q) was an error:\n%.400s", pages, spec, text)
		}
		var header bool
		for _, l := range strings.Split(text, "\n") {
			if strings.HasPrefix(l, `==> "a\nb.txt"  `) {
				header = true
			}
			if strings.HasPrefix(l, "b.txt  ") {
				t.Errorf("page %d: the header was split, its second half is the line %q", pages, l)
			}
		}
		if !header {
			t.Errorf("page %d has no one-line header for the escaped name:\n%.500s", pages, text)
		}
		if strings.Contains(text, "-- ck ") {
			t.Errorf("page %d was bracketed with checkpoints", pages)
		}
		if !strings.Contains(text, "name holds a control byte") {
			t.Errorf("page %d does not say the file cannot be licensed:\n%.500s", pages, text)
		}
		next, _ := receipt(t, res)["next_read"].(string)
		if next == "" {
			fitted = !strings.Contains(text, "-- PARTIAL:")
		}
		spec = next
	}
	if pages < 2 || !fitted {
		t.Fatalf("followed %d page(s), last page fitted: %v; the fixture must page and end on a page that fits", pages, fitted)
	}
	if l, err := seen.Load(root); err != nil || len(l) != 0 {
		t.Errorf("the ledger holds %v (%v)", l, err)
	}
}
