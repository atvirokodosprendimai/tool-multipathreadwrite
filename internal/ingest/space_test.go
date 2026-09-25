package ingest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// ADR-069 T4. Both foreign formats TrimSpaced the path after their marker, so
// a patch naming "x " compiled to a hunk on x. The grammar needs one separator
// space, not a trim; a marker followed only by whitespace still names no path.
func TestApplyPatchAndSearchReplaceKeepATrailingSpace(t *testing.T) {
	root := writeTree(t, map[string]string{"x ": "one\ntwo\n", "x": "one\ntwo\n", "prev.go": "a\n"})
	if _, err := os.Stat(filepath.Join(root, "x ")); err != nil {
		t.Skipf("this filesystem cannot hold a trailing-space name: %v", err)
	}
	if fi, _ := os.ReadDir(root); len(fi) != 3 {
		t.Skip("this filesystem folds a trailing space")
	}
	first := func(t *testing.T, text []byte, err error) plan.Hunk {
		t.Helper()
		if err != nil {
			t.Fatalf("compile: %v", err)
		}
		hs, err := plan.Parse(bytes.NewReader(text))
		if err != nil || len(hs) == 0 {
			t.Fatalf("parse: %v\n%s", err, text)
		}
		return hs[0]
	}
	for _, c := range []struct{ name, doc, path, dest string }{
		{"update", "*** Begin Patch\n*** Update File: x \n@@\n one\n-two\n+TWO\n*** End Patch\n", "x ", ""},
		{"add", "*** Begin Patch\n*** Add File: y \n+hi\n*** End Patch\n", "y ", ""},
		{"delete", "*** Begin Patch\n*** Delete File: x \n*** End Patch\n", "x ", ""},
		{"move", "*** Begin Patch\n*** Update File: x\n*** Move to: d \n*** End Patch\n", "x", "d "},
	} {
		t.Run(c.name, func(t *testing.T) {
			text, err := CompileApplyPatch(root, []byte(c.doc))
			h := first(t, text, err)
			if h.Path != c.path {
				t.Errorf("apply_patch %s compiled to %q, want %q:\n%s", c.name, h.Path, c.path, text)
			}
			if c.dest != "" && (len(h.Body) != 1 || h.Body[0] != c.dest) {
				t.Errorf("Move to compiled to %q, want %q", h.Body, c.dest)
			}
		})
	}
	if _, err := CompileApplyPatch(root, []byte("*** Begin Patch\n*** Delete File:   \n*** End Patch\n")); err == nil {
		t.Error("a Delete File with only spaces after the marker compiled")
	}

	sr := "<<<<<<< SEARCH x \none\n=======\nONE\n>>>>>>> REPLACE\n"
	text, err := CompileSearchReplace(root, []byte(sr))
	if h := first(t, text, err); h.Path != "x " {
		t.Errorf("search_replace on the SEARCH line compiled to %q, want %q", h.Path, "x ")
	}
	sr = "x \n<<<<<<< SEARCH\none\n=======\nONE\n>>>>>>> REPLACE\n"
	text, err = CompileSearchReplace(root, []byte(sr))
	if h := first(t, text, err); h.Path != "x " {
		t.Errorf("search_replace filename line compiled to %q, want %q", h.Path, "x ")
	}
	sr = "prev.go\n<<<<<<< SEARCH  \na\n=======\nb\n>>>>>>> REPLACE\n"
	text, err = CompileSearchReplace(root, []byte(sr))
	if h := first(t, text, err); h.Path != "prev.go" {
		t.Errorf("a SEARCH marker with trailing blanks did not fall back to the filename line: %q", h.Path)
	}
}
