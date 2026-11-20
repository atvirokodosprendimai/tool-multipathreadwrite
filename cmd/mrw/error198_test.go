package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-198 Decision 7. A write that stopped returns an error that begins with
// the file as the platform spells it; a CLI --json refusal prints that error in
// its document and again on stderr, and both begin as the receipt's row does.
// The receipt separator is set to `\` and the plan names a directory with a
// backslash in its name (q\n.txt), whose load stops the write with its verdict
// on its hunk (#703); on Windows the backslash is the separator, and q\n.txt
// is n.txt under q. The stop once came from creates of q\n.txt and q\n.txt/x,
// which #701 refuses before anything is staged. An error that begins with no
// path of the result comes back as it was: lead198_test.go in internal/apply.
func TestAStoppedWritesErrorBeginsAsTheRowDoes(t *testing.T) {
	prev := receiptSep
	receiptSep = '\\'
	t.Cleanup(func() { receiptSep = prev })
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	planFile := func(t *testing.T, text string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "plan")
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	const stops = "@@ q\\n.txt 1 replace\nz\n"
	tree := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		if e := os.MkdirAll(filepath.Join(root, `q\n.txt`), 0o755); e != nil {
			t.Fatal(e)
		}
		return root
	}

	t.Run("under --json the document's error and the stderr line begin as the row does", func(t *testing.T) {
		err, out := runMrw(t, tree(t), "write", "--no-check", "--json", planFile(t, stops))
		if err == nil {
			t.Fatalf("the plan did not stop: %s", out)
		}
		var doc struct {
			Error string `json:"error"`
			Hunks []struct {
				Path string `json:"path"`
			} `json:"hunks"`
		}
		if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
			t.Fatalf("stdout is not one JSON document: %v\n%s", jerr, out)
		}
		const want = "q/n.txt: "
		if !strings.HasPrefix(doc.Error, want) {
			t.Errorf("the document's error %q, want it to begin %q", doc.Error, want)
		}
		if !strings.HasPrefix(err.Error(), want) {
			t.Errorf("the stderr line %q, want it to begin %q", err.Error(), want)
		}
		if len(doc.Hunks) == 0 || doc.Hunks[0].Path != "q/n.txt" {
			t.Errorf("the row's path is not q/n.txt: %+v", doc.Hunks)
		}
		if ec, ok := err.(interface{ ExitCode() int }); !ok || ec.ExitCode() != 2 {
			t.Errorf("exit code of %v, want 2", err)
		}
	})

	t.Run("without --json the exit-2 line begins as the row does", func(t *testing.T) {
		err, _ := runMrw(t, tree(t), "write", "--no-check", planFile(t, stops))
		if err == nil {
			t.Fatal("the plan did not stop")
		}
		if got := err.Error(); !strings.HasPrefix(got, "q/n.txt") || strings.Contains(got, `q\n`) {
			t.Errorf("the plain exit-2 line %q does not begin with q/n.txt", got)
		}
	})
}
