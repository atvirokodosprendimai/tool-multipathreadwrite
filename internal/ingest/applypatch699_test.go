package ingest

import (
	"strings"
	"testing"
)

// TestAFileMarkerWithNoPathIsRefused: an Add File or Update File marker that
// names no path compiled to nothing, its body was dropped, and a sibling
// section landed on its own with no verdict for the lost one (#699). It is a
// compile refusal, and the same patch without it compiles.
func TestAFileMarkerWithNoPathIsRefused(t *testing.T) {
	root := writeTree(t, map[string]string{"demo.go": demo})
	valid := "*** Add File: ok.txt\n+ok\n"
	for _, c := range []struct{ name, section string }{
		{"Add File", "*** Add File:\n+lost\n"},
		{"Add File with spaces", "*** Add File:   \n+lost\n"},
		{"Update File", "*** Update File:\n@@\n-x\n+y\n"},
		{"Update File before a hunk line", "*** Update File: \n-x\n+y\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			doc := "*** Begin Patch\n" + valid + c.section + "*** End Patch\n"
			out, err := CompileApplyPatch(root, []byte(doc))
			if err == nil || !strings.Contains(err.Error(), "names no path") {
				t.Fatalf("a marker naming no path compiled: err=%v\n%s", err, out)
			}
		})
	}
	if _, err := CompileApplyPatch(root, []byte("*** Begin Patch\n"+valid+"*** End Patch\n")); err != nil {
		t.Fatalf("the valid section alone was refused: %v", err)
	}
}
