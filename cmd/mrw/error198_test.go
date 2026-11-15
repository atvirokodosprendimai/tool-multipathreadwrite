package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-198 Decision 7. A write that stopped at its commit returns an error that
// begins with the file as the platform spells it; a CLI --json refusal prints
// that error in its document and again on stderr, and both begin as the
// receipt's row does. The receipt separator is set to `\` and the plan names a
// file with a backslash in it (q\n.txt) beside a path under it, so the commit
// meets `q\n.txt` already made; the same file is q/n.txt on Windows, where the
// backslash is the separator and q a directory.
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
	const stops = "@@ q\\n.txt 0 create\nx\n@@ q\\n.txt/x 0 create\ny\n"

	t.Run("under --json the document's error and the stderr line begin as the row does", func(t *testing.T) {
		root := t.TempDir()
		err, out := runMrw(t, root, "write", "--no-check", "--json", planFile(t, stops))
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
		const want = "q/n.txt appeared before commit"
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

	t.Run("without --json the exit-2 line is what it was", func(t *testing.T) {
		root := t.TempDir()
		err, _ := runMrw(t, root, "write", "--no-check", planFile(t, stops))
		if err == nil {
			t.Fatal("the plan did not stop")
		}
		if got := err.Error(); !strings.HasPrefix(got, "q/n.txt") || strings.Contains(got, `q\n`) {
			t.Errorf("the plain exit-2 line %q does not begin with q/n.txt", got)
		}
	})

	t.Run("an error that begins with no path of the result is printed as it is", func(t *testing.T) {
		root := t.TempDir()
		if e := os.MkdirAll(filepath.Join(root, `x\d`), 0o755); e != nil {
			t.Fatal(e)
		}
		err, out := runMrw(t, root, "write", "--no-check", "--json", planFile(t, "@@ x\\d 1 replace\nz\n"))
		if err == nil {
			t.Fatalf("a directory named in a plan was written: %s", out)
		}
		if !strings.HasPrefix(err.Error(), `x\d`) {
			t.Errorf("the stderr line %q, want it to begin with x\\d as it is", err.Error())
		}
		var doc struct {
			Error string `json:"error"`
		}
		if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil || !strings.HasPrefix(doc.Error, `x\d`) {
			t.Errorf("the document's error %q, want it to begin with x\\d as it is", doc.Error)
		}
	})
}
