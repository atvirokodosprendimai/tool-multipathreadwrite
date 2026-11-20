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

// #741, the paged path of #716. firstPage served the first page of a file named
// with a line feed with checkpoints woven through it and held them against the
// raw name, which no ledger record can hold (#702); the page said nothing of it.
// The page is served unmarked, the header on one line, with the note #716 gives
// the fitting path; nothing is held and the ledger records nothing.
func TestAPagedReadOfAFileNamedWithALineFeedIsServedUnmarked(t *testing.T) {
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
	res := call(t, root, "mrw_read", map[string]any{"specs": []string{name}})
	text := firstText(t, res)
	if res["isError"] == true || !strings.Contains(text, "-- PARTIAL:") {
		t.Fatalf("the read was not served as a page (isError %v):\n%.400s", res["isError"], text)
	}
	if strings.Contains(text, "-- ck ") {
		t.Errorf("a file whose name holds a line feed was bracketed with checkpoints:\n%.600s", text)
	}
	if !strings.Contains(text, `"a\nb.txt"`) || !strings.Contains(text, "name holds a control byte") {
		t.Errorf("the page does not say the file cannot be licensed:\n%.600s", text)
	}
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 0 {
		t.Errorf("the ledger holds %v", l)
	}
}
