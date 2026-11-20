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
	// An Update File section with no hunk and no Move to was dropped the same
	// way, its sibling landing with no verdict for it.
	doc := "*** Begin Patch\n" + valid + "*** Update File: demo.go\n*** End Patch\n"
	if out, err := CompileApplyPatch(root, []byte(doc)); err == nil || !strings.Contains(err.Error(), "changes nothing") {
		t.Fatalf("an Update File section that changes nothing compiled: err=%v\n%s", err, out)
	}
	move := "*** Begin Patch\n*** Update File: demo.go\n*** Move to: moved.go\n*** End Patch\n"
	if _, err := CompileApplyPatch(root, []byte(move)); err != nil {
		t.Fatalf("an Update File section that only moves was refused: %v", err)
	}
	// The section is closed by whatever marker follows it, not only by End
	// Patch: each boundary refuses an empty Update File (Codex on #708).
	for name, next := range map[string]string{
		"Add File":    "*** Add File: ok.txt\n+ok\n",
		"Update File": "*** Update File: demo.go\n*** Move to: moved.go\n",
		"Delete File": "*** Delete File: demo.go\n",
	} {
		doc := "*** Begin Patch\n*** Update File: demo.go\n@@\n\n" + next + "*** End Patch\n"
		if out, err := CompileApplyPatch(root, []byte(doc)); err == nil || !strings.Contains(err.Error(), "changes nothing") {
			t.Errorf("an empty Update File closed by %s compiled: err=%v\n%s", name, err, out)
		}
	}
}

// TestAnEndPatchMarkerInsideABodyLineIsText: the envelope's terminator is a
// line of its own. A body line holding it as text was taken for the end, so the
// patch was refused for text after it, or with no real terminator compiled with
// its body cut (Codex on #708).
func TestAnEndPatchMarkerInsideABodyLineIsText(t *testing.T) {
	root := writeTree(t, map[string]string{"demo.go": demo})
	doc := "*** Begin Patch\n*** Add File: ok.txt\n+literal *** End Patch\n+after\n*** End Patch\n"
	out, err := CompileApplyPatch(root, []byte(doc))
	if err != nil {
		t.Fatalf("a body line holding the terminator as text was refused: %v", err)
	}
	if !strings.Contains(string(out), "literal *** End Patch\nafter\n") {
		t.Fatalf("the body was cut:\n%s", out)
	}
	cut := "*** Begin Patch\n*** Add File: ok.txt\n+literal *** End Patch\n+after\n"
	if out, err := CompileApplyPatch(root, []byte(cut)); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("a patch with no terminator line compiled: err=%v\n%s", err, out)
	}
	// Spaces or tabs after the terminator on its line were accepted before the
	// terminator had to be a whole line, and still are (Codex on #708).
	trailing := "*** Begin Patch\n*** Add File: ok.txt\n+ok\n*** End Patch \t"
	if _, err := CompileApplyPatch(root, []byte(trailing)); err != nil {
		t.Fatalf("a terminator line with trailing blanks was refused: %v", err)
	}
}
