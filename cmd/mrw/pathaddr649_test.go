package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// #649. A path op has no line address: its row prints `-`, whatever the plan wrote
// and whichever way the hunk ended. The FAIL and skip rows printed the unresolved
// bound, `0`, though the row of a path op that applies already printed `-`. The
// JSON receipt's "addr" keeps the value it has (ADR-111).
//
// The skip cases are the ones that reach the old `0`: a hunk refused by refuseFile,
// clashAt or sameFileAs is skipped with the Addr its input carried, where a hunk
// that merely sat beside a failing one was validated first and already said `-`.
func TestAPathOpRowPrintsTheAddressThePlanWrote(t *testing.T) {
	tree := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		t.Setenv("MRW_RECEIPT", "full") // ADR-188: the environment may choose a receipt; the flags below must be what picks one
		writeFiles(t, root, map[string]string{"a.txt": "a\n", "b.txt": "b\n", "g.txt": "g\n"})
		if err := os.Chmod(filepath.Join(root, "b.txt"), 0o444); err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" {
			if err := os.Symlink("a.txt", filepath.Join(root, "l.txt")); err != nil {
				t.Fatal(err)
			}
		}
		if out, code := runIn(t, root, "read", "a.txt", "b.txt", "l.txt"); code != 0 && runtime.GOOS != "windows" {
			t.Fatalf("read: exit %d:\n%s", code, out)
		}
		return root
	}
	cases := []struct {
		name, plan string
		flags      []string
		want       []string
		unixOnly   bool
	}{
		{"an unlink of a missing file", "@@ nope.txt - unlink\n", nil, []string{"FAIL nope.txt - unlink (plan line 1): nope.txt does not exist"}, false},
		{"a rename of a missing file", "@@ nope.txt - rename\nz.txt\n", nil, []string{"FAIL nope.txt - rename (plan line 1): nope.txt does not exist"}, false},
		{"a rename of a file never read", "@@ g.txt - rename\nz.txt\n", nil, []string{"FAIL g.txt - rename (plan line 1): g.txt has not been read"}, false},
		{"a placeholder of 0 is printed as -", "@@ nope.txt 0 unlink\n", nil, []string{"FAIL nope.txt - unlink"}, false},
		{"--quiet keeps the failing row", "@@ nope.txt - unlink\n", []string{"--quiet"}, []string{"FAIL nope.txt - unlink"}, false},
		{"--terse keeps the failing row", "@@ nope.txt - unlink\n", []string{"--terse"}, []string{"FAIL nope.txt - unlink"}, false},
		{"an unlink skipped beside a directory spelling", "@@ a.txt/ 1 replace\nx\n@@ a.txt - unlink\n", nil, []string{"FAIL a.txt 1 replace", "skip a.txt - unlink"}, false},
		{"a rename skipped beside a directory spelling", "@@ a.txt/ 1 replace\nx\n@@ a.txt - rename\nz.txt\n", nil, []string{"FAIL a.txt 1 replace", "skip a.txt - rename"}, false},
		{"a rename skipped beside a read-only file", "@@ b.txt 1 replace\nx\n@@ b.txt - rename\nz.txt\n", nil, []string{"skip b.txt - rename"}, true},
		{"a rename skipped through a link to a file edited under its other name", "@@ a.txt 1 replace\nx\n@@ l.txt 1 replace\ny\n@@ l.txt - rename\nz.txt\n", nil, []string{"skip l.txt - rename"}, true},
		{"a dry run of an unlink", "@@ a.txt - unlink\n", []string{"--dry-run"}, []string{"parsed: a.txt - unlink body=0", "ok   a.txt - unlink"}, false},
		{"a dry run of a rename", "@@ a.txt 0 rename\nz.txt\n", []string{"--dry-run"}, []string{"parsed: a.txt - rename body=1", "ok   a.txt - rename"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.unixOnly && runtime.GOOS == "windows" {
				t.Skip("a read-only mode bit and a symlink are unix fixtures")
			}
			root := tree(t)
			args := append([]string{"--no-check"}, c.flags...)
			out, _ := writeIn(t, root, append(args, planFile(t, c.plan))...)
			for _, want := range c.want {
				if !strings.Contains(out, want) {
					t.Errorf("want a row %q:\n%s", want, out)
				}
			}
			for _, row := range strings.Split(out, "\n") {
				if strings.Contains(row, " 0 unlink") || strings.Contains(row, " 0 rename") {
					t.Errorf("a row names address 0 the plan never wrote: %q", row)
				}
			}
		})
	}
}

// --quiet is documented to drop ok and skip rows and keep the failing one and the
// summary: the skip row that the full receipt prints with `-` is absent here.
func TestQuietKeepsTheFailRowAndDropsTheSkipRowOfAPathOp(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("MRW_RECEIPT", "full")
	writeFiles(t, root, map[string]string{"a.txt": "a\n"})
	if out, code := runIn(t, root, "read", "a.txt"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	plan := "@@ a.txt/ 1 replace\nx\n@@ a.txt - unlink\n"
	full, _ := writeIn(t, root, "--no-check", planFile(t, plan))
	if !strings.Contains(full, "skip a.txt - unlink") {
		t.Fatalf("the full receipt lacks the skip row the quiet one must drop:\n%s", full)
	}
	out, _ := writeIn(t, root, "--no-check", "--quiet", planFile(t, plan))
	if !strings.Contains(out, "FAIL a.txt 1 replace") || !strings.Contains(out, "NOTHING WRITTEN") {
		t.Errorf("--quiet lost the failing row or the summary:\n%s", out)
	}
	if strings.Contains(out, "skip ") {
		t.Errorf("--quiet printed a skip row:\n%s", out)
	}
}

// ADR-111: the human row says `-`, and the JSON receipt keeps the "addr" it has: "0"
// on a failed path op. A change that normalised it would be a changed value.
func TestTheJSONAddrOfAFailedPathOpIsStillZero(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("MRW_RECEIPT", "full")
	plan := planFile(t, "@@ nope.txt - unlink\n")
	human, _ := writeIn(t, root, "--no-check", plan)
	if !strings.Contains(human, "FAIL nope.txt - unlink") {
		t.Fatalf("the human row does not say -:\n%s", human)
	}
	out, _ := writeIn(t, root, "--no-check", "--json", plan)
	var rec struct {
		Hunks []struct{ Addr, Op, Status string } `json:"hunks"`
	}
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(rec.Hunks) != 1 || rec.Hunks[0].Addr != "0" || rec.Hunks[0].Op != "unlink" || rec.Hunks[0].Status != "failed" {
		t.Errorf(`want one failed unlink with addr "0", got %+v`, rec.Hunks)
	}
}
