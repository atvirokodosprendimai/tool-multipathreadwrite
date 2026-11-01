package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// receipt188Tree is a root with two small files and no declared check.
func receipt188Tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("one\ntwo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const receipt188Plan = "@@ a.txt 1 replace\nONE\n@@ b.txt 2 replace\nTWO\n"

// ADR-188. MRW_RECEIPT selects the receipt mrw write prints when no flag
// does: quiet as --quiet, terse as --terse, full or unset as before; a flag on
// the call wins; any other value is ignored with one note on stderr; a failed
// hunk prints whatever it says. In the native-versus-MRW runs MRW's write
// receipts were a quarter of what its agent read back, and no agent ever
// passed --quiet or --terse itself (docs/bench/observability/NATIVE-RESULTS.md).
func TestTheReceiptSwitchSetsTheWriteReceiptUnlessAFlagDoes(t *testing.T) {
	for _, tc := range []struct {
		name, env string
		flags     []string
		okRows    int    // per-hunk "ok   " rows
		terse     bool   // one "ok   PATH  N hunk(s)" line per file
		note      string // on stderr
	}{
		{"unset", "", nil, 2, false, ""},
		{"full", "full", nil, 2, false, ""},
		{"quiet", "quiet", nil, 0, false, ""},
		{"terse", "terse", nil, 0, true, ""},
		{"--terse wins over quiet", "quiet", []string{"--terse"}, 0, true, ""},
		{"--quiet wins over terse", "terse", []string{"--quiet"}, 0, false, ""},
		{"a bad value is ignored", "loud", nil, 2, false, "MRW_RECEIPT=loud"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MRW_RECEIPT", tc.env)
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			root := receipt188Tree(t)
			if _, err := readIn(t, root, "a.txt", "b.txt"); err != nil {
				t.Fatal(err)
			}
			out, errOut, code := project178Run(t, root, "", append(append([]string{"write"}, tc.flags...), planFile(t, receipt188Plan))...)
			if code != 0 {
				t.Fatalf("exit %d:\n%s%s", code, out, errOut)
			}
			rows := 0
			for _, l := range strings.Split(out, "\n") {
				if strings.HasPrefix(l, "ok   ") && strings.Contains(l, " replace ") {
					rows++
				}
			}
			if rows != tc.okRows {
				t.Errorf("%d ok rows, want %d:\n%s", rows, tc.okRows, out)
			}
			if got := strings.Contains(out, "1 hunk(s)  -1 +1"); got != tc.terse {
				t.Errorf("terse lines %v, want %v:\n%s", got, tc.terse, out)
			}
			if !strings.Contains(out, "2 hunk(s), 2 file(s), 0 failed") {
				t.Errorf("no summary line:\n%s", out)
			}
			if tc.note == "" && strings.Contains(errOut, "MRW_RECEIPT") || tc.note != "" && !strings.Contains(errOut, tc.note) {
				t.Errorf("stderr %q, want note %q", errOut, tc.note)
			}
		})
	}
	t.Run("a failure prints under quiet", func(t *testing.T) {
		t.Setenv("MRW_RECEIPT", "quiet")
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := receipt188Tree(t)
		if _, err := readIn(t, root, "a.txt"); err != nil {
			t.Fatal(err)
		}
		out, _, code := project178Run(t, root, "", "write", planFile(t, "@@ a.txt 9 replace\nX\n"))
		if code != 1 || !strings.Contains(out, "FAIL a.txt 9 replace") {
			t.Errorf("exit %d, want 1 with the FAIL row:\n%s", code, out)
		}
	})
}
