package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Final review of #740. A diagnostic that prints a path printed a name holding a
// line feed raw, so the name continued on the next line and the line after it read
// as another file's. Every way a read names a file it served nothing of is run
// with a name holding a line feed, and the name never splits a line.
func TestNoReadDiagnosticSplitsAFileNameAcrossLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a line feed is not a legal Windows file name")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "draft\nnotes.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Skipf("this filesystem keeps no such name: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "j\nson.json"), []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	const name = "draft\nnotes.txt"
	for _, argv := range [][]string{
		{"read", "--max-chars", "0", name},
		{"read", "--max-chars", "0", name + ":1-2"},
		{"read", "--max-chars", "0", name + ":/one/"},
		{"read", "--max-chars", "0", "--grep", "one", "."},
		{"read", "--max-chars", "0", "--outline", name},
		{"read", "--max-chars", "0", "--json-path", "$.a", name},
		{"read", "--max-chars", "40", "--json-path", "$.a,$.a,$.a", "j\nson.json"}, // the header fits, the values do not
		{"read", "--stat", name + ":1"},
		{"read", "--count", "--grep", "zzz", name},
		{"read", "--grep", "zzz", name},
		{"read", name + ":9"},
	} {
		var out string
		var code int
		errText := stderrOf(t, func() { out, code = runIn(t, root, argv...) })
		for _, text := range []string{out, errText} {
			if strings.Contains(text, "draft\nnotes.txt") || strings.Contains(text, "j\nson.json") {
				t.Errorf("%q (exit %d) printed the name with a raw line feed:\n%s", argv, code, text)
			}
		}
	}
}
