package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	shownVar     = "MRW_SHOWN"
	shownLine    = "mrw: recorded as licensed, not shown (MRW_SHOWN=off)"
	codeModeVar  = "CLAUDE_CODE_EXPERIMENTAL_CODE_MODE"
	shownFixture = "one\ntwo\nthree\nfour\nfive\n"
)

// shownRoot is a checkout holding f.txt of five lines, with a state home of its
// own and neither variable set.
func shownRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv(shownVar, "")
	t.Setenv(codeModeVar, "")
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(shownFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// runShown runs mrw in root with MRW_SHOWN set to value, and returns what it
// printed on standard output (with the command's own writers), on standard
// error, and its exit code.
func runShown(t *testing.T, root, value string, argv ...string) (out, errOut string, code int) {
	t.Helper()
	t.Setenv(shownVar, value)
	errOut = stderrOf(t, func() { out, code = runIn(t, root, argv...) })
	return out, errOut, code
}

// wantShowing asserts that the ledger licenses line of path and whether it
// records that line as shown.
func wantShowing(t *testing.T, label, root, path string, line int, shown bool) {
	t.Helper()
	o, ok := servedOf(t, root, path)
	if !ok || !o.Covers(line, line) {
		t.Fatalf("%s: %s line %d is not licensed (held %v, ledger: %v)", label, path, line, ok, o.Served())
	}
	if got := o.FirstUnshown(line, line) == 0; got != shown {
		t.Errorf("%s: %s line %d shown = %v, want %v (ledger: %v)", label, path, line, got, shown, o.Served())
	}
}

// ADR-218 Decisions 1, 2 and 6. Under MRW_SHOWN=off a content read prints what
// it prints without it and records its lines as licensed and not shown; on,
// unset, an unknown value and Claude Code's Code Mode variable leave the read as
// it was; a showing already recorded for the version is kept, and one forgotten
// is not brought back.
func TestAReadUnderShownOffLicensesAndShowsNothing(t *testing.T) {
	t.Run("read_bytes_equal", func(t *testing.T) {
		plain, _, pc := runShown(t, shownRoot(t), "", "read", "f.txt")
		off, _, oc := runShown(t, shownRoot(t), "off", "read", "f.txt")
		if pc != 0 || oc != 0 || plain != off || !strings.Contains(off, "| three") {
			t.Errorf("exit %d and %d; the switched read printed\n%s\nwant the plain read's\n%s", pc, oc, off, plain)
		}
	})
	t.Run("ledger_no_showing", func(t *testing.T) {
		root := shownRoot(t)
		if out, _, code := runShown(t, root, "off", "read", "f.txt:2"); code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		wantShowing(t, "ranged", root, "f.txt", 2, false)
		wantShowing(t, "the neighbour it served", root, "f.txt", 3, false)
		whole := shownRoot(t)
		if out, _, code := runShown(t, whole, "off", "read", "f.txt"); code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		wantShowing(t, "whole", whole, "f.txt", 5, false)
	})
	t.Run("stderr_line", func(t *testing.T) {
		_, errOut, _ := runShown(t, shownRoot(t), "off", "read", "f.txt:2")
		if strings.Count(errOut, shownLine) != 1 || !strings.Contains(errOut, "search_replace") {
			t.Errorf("stderr %q, want the one line naming the switch and the block form", errOut)
		}
	})
	for name, value := range map[string]string{"on_is_plain": "on", "unset_is_plain": ""} {
		t.Run(name, func(t *testing.T) {
			root := shownRoot(t)
			_, errOut, code := runShown(t, root, value, "read", "f.txt:2")
			if code != 0 || strings.Contains(errOut, "MRW_SHOWN") {
				t.Errorf("exit %d, stderr %q; want a plain read that says nothing of the switch", code, errOut)
			}
			wantShowing(t, name, root, "f.txt", 2, true)
		})
	}
	t.Run("unknown_value_noted", func(t *testing.T) {
		root := shownRoot(t)
		_, errOut, code := runShown(t, root, "maybe", "read", "f.txt:2")
		if code != 0 || !strings.Contains(errOut, "mrw: MRW_SHOWN=maybe is not off or on; ignored") || strings.Contains(errOut, shownLine) {
			t.Errorf("exit %d, stderr %q; want the value named and ignored", code, errOut)
		}
		wantShowing(t, "unknown", root, "f.txt", 2, true)
	})
	t.Run("earlier_showing_kept", func(t *testing.T) {
		// Decision 1's boundary: the ledger is the checkout's, and a showing
		// already in it for this version outlives a switched read.
		root := shownRoot(t)
		runShown(t, root, "", "read", "f.txt:2")
		runShown(t, root, "off", "read", "f.txt:2-4")
		wantShowing(t, "shown before the switch", root, "f.txt", 2, true)
		wantShowing(t, "read only under the switch", root, "f.txt", 4, false)
	})
	t.Run("forget_then_none", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "", "read", "f.txt:2")
		if out, _, code := runShown(t, root, "off", "seen", "--forget"); code != 0 {
			t.Fatalf("seen --forget exited %d: %s", code, out)
		}
		runShown(t, root, "off", "read", "f.txt:2")
		wantShowing(t, "after the forget", root, "f.txt", 2, false)
	})
	t.Run("stat_says_nothing", func(t *testing.T) {
		// The note is for a read that served a line: --stat serves none.
		_, errOut, code := runShown(t, shownRoot(t), "off", "read", "--stat", "f.txt")
		if code != 0 || strings.Contains(errOut, "licensed, not shown") {
			t.Errorf("exit %d, stderr %q; want no note for a read that served no line", code, errOut)
		}
	})
	t.Run("if_changed_text_shows_nothing", func(t *testing.T) {
		// --if-changed answers by reference when the sha matches; with another
		// sha it serves the text, and that text is a switched read's.
		root := shownRoot(t)
		out, _, code := runShown(t, root, "off", "read", "--if-changed", "f.txt=00000000", "f.txt:2")
		if code != 0 || !strings.Contains(out, "| two") {
			t.Fatalf("exit %d, want the text served:\n%s", code, out)
		}
		wantShowing(t, "served by --if-changed", root, "f.txt", 2, false)
	})
	t.Run("code_mode_variable_not_read", func(t *testing.T) {
		// Decision 6: nothing is detected.
		root := shownRoot(t)
		t.Setenv(codeModeVar, "1")
		_, errOut, code := runShown(t, root, "", "read", "f.txt:2")
		if code != 0 || strings.Contains(errOut, "licensed, not shown") {
			t.Errorf("exit %d, stderr %q; want a plain read", code, errOut)
		}
		wantShowing(t, "Code Mode's variable alone", root, "f.txt", 2, true)
	})
}

// ADR-218 Decision 3. Under the switch `mrw hook read` records the lines a host
// Read returned as licensed and not shown, whatever the payload's tool_use_id;
// without it a payload records a showing, a repl_ id or not. The payload is the
// one ADR-191 captured: offset 4, limit 3 of a.txt.
func TestAHostReadUnderShownOffLicensesAndShowsNothing(t *testing.T) {
	for _, c := range []struct {
		name, value, id string
		shown           bool
	}{
		{"switched_no_showing", "off", "toolu_01ABC", false},
		{"switched_repl_id_no_showing", "off", "repl_a0fed6c2-63c9-4bde-a14b-489a8412c5b3", false},
		{"plain_shows", "", "toolu_01ABC", true},
		{"plain_repl_id_shows", "", "repl_a0fed6c2-63c9-4bde-a14b-489a8412c5b3", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := hookRoot(t)
			t.Setenv(shownVar, c.value)
			p := capturedPayloads(t, root)[1]
			p["tool_use_id"] = c.id
			out, errOut, code := runHook(t, root, jsonOf(t, p))
			quiet(t, c.name, out, errOut, code)
			wantShowing(t, c.name, root, "a.txt", 5, c.shown)
			if o, _ := servedOf(t, root, "a.txt"); o.Covers(9, 9) {
				t.Errorf("%s: line 9, which the Read did not return, is licensed (ledger: %v)", c.name, o.Served())
			}
		})
	}
}
