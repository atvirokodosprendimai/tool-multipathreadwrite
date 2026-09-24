package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-065 T2. apply_patch compiles against the target's lines as the write
// engine numbers them. The document is LF-normalised (ADR-051 F-9), so on a
// CRLF target the old side "two" used to meet "two\r" and the edit was refused
// with "matched no lines".
func TestApplyPatchEditsACRLFFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "crlf.txt"), []byte("one\r\ntwo\r\nthree\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := "*** Begin Patch\n*** Update File: crlf.txt\n@@\n one\n-two\n+TWO\n three\n*** End Patch\n"
	plan, err := CompileApplyPatch(root, []byte(doc))
	if err != nil {
		t.Fatalf("apply_patch refused a CRLF target: %v", err)
	}
	if !strings.Contains(string(plan), "@@ crlf.txt 1-3 replace") || !strings.Contains(string(plan), "TWO") {
		t.Fatalf("the compiled plan does not replace the matched span 1-3:\n%s", plan)
	}
}

// ADR-065 T2. search_replace compiles against a CR-only target's lines, which
// used to be one line holding the whole file.
func TestSearchReplaceEditsACROnlyFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cr.txt"), []byte("one\rtwo\rthree\r"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := "cr.txt\n<<<<<<< SEARCH\ntwo\n=======\nTWO\n>>>>>>> REPLACE\n"
	plan, err := CompileSearchReplace(root, []byte(doc))
	if err != nil {
		t.Fatalf("search_replace refused a CR-only target: %v", err)
	}
	if !strings.Contains(string(plan), "@@ cr.txt 2 replace") {
		t.Fatalf("the compiled plan does not address line 2:\n%s", plan)
	}
}
