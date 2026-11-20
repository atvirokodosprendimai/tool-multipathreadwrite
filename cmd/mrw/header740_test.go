package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Review of #740. A file whose name holds a line feed printed a header that spanned
// two lines, which no reader could tell from a file named with the second line.
// Every header is one line: the name is printed quoted and escaped, as one of its
// kind, and nothing after it is mistaken for another file's header.
func TestAHeaderForANameWithALineFeedIsOneQuotedLine(t *testing.T) {
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
	out, code := runIn(t, root, "read", "a\nb", "c.txt", "missing\n 1| phantom")
	if code == 0 {
		t.Fatalf("a missing file read exit 0:\n%s", out)
	}
	var headers []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "==> ") {
			headers = append(headers, l)
		}
	}
	if len(headers) != 3 {
		t.Fatalf("%d header lines, want 3 (one per name):\n%s", len(headers), out)
	}
	if !strings.HasPrefix(headers[0], `==> "a\nb"  1L  4B  sha `) || !strings.HasPrefix(headers[1], "==> c.txt  1L  6B  sha ") {
		t.Errorf("headers %q, want the quoted odd name and the plain one", headers)
	}
	if !strings.HasPrefix(headers[2], `==> "missing\n 1| phantom"`) && !strings.HasPrefix(headers[2], `==> missing\n 1| phantom`) {
		t.Errorf("the unreadable name spans lines or is not escaped: %q", headers[2])
	}
	if strings.Contains(out, "\n 1| phantom") {
		t.Errorf("a name's second line reads as a served line:\n%s", out)
	}
}
