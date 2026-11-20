package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Issue #716. An mrw_read whose walk served a file named with a line feed failed
// with -32603 "holding checkpoints: no observation for a": the header, printed
// with the name raw, split in two before hold. The other files are served and
// keep their checkpoints; that one is named as unrecordable, as #702 does for the
// CLI.
func TestAReadServingAFileNamedWithALineFeedServesTheOthers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a line feed is not a legal Windows file name")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for name, body := range map[string]string{"a\nb": "odd\n", "c.txt": "plain\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Skipf("this filesystem keeps no such name: %v", err)
		}
	}
	text, l := lfServed(t, root, map[string]any{"grep": ".", "no_ignore": true})
	if !strings.Contains(text, "plain") || !strings.Contains(text, "-- ck ") {
		t.Errorf("the other file was not served with its checkpoints:\n%s", text)
	}
	if !strings.Contains(text, `"a\nb"`) || !strings.Contains(text, lfNote) {
		t.Errorf("the file named with a line feed is not named as unrecordable:\n%s", text)
	}
	// What the ack licensed is in the ledger: the plain file, and no record of the odd one.
	lfCovers(t, l, "c.txt", 1, 1)
	for k := range l {
		if strings.Contains(k, "\n") {
			t.Errorf("the ledger holds %q", k)
		}
	}
}
