package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const shownNote = "(reads here record no showing: MRW_SHOWN=off)"

// writeShown writes plan to a file and runs `mrw write --no-check [extra] FILE`
// in root with MRW_SHOWN set to value, returning everything printed and the
// exit code.
func writeShown(t *testing.T, root, value, plan string, extra ...string) (string, int) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(file, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := append(append([]string{"write", "--no-check"}, extra...), file)
	out, errOut, code := runShown(t, root, value, argv...)
	return out + errOut, code
}

// bytesAre asserts the bytes of root/name.
func bytesAre(t *testing.T, label, root, name, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(root, name))
	if err != nil || string(got) != want {
		t.Errorf("%s: %s holds %q (%v), want %q", label, name, got, err, want)
	}
}

// refusedWith asserts a refused write: exit 1, every wanted phrase, no unwanted one.
func refusedWith(t *testing.T, label, out string, code int, want, unwanted []string) {
	t.Helper()
	if code != 1 {
		t.Errorf("%s: exit %d, want 1:\n%s", label, code, out)
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("%s: the refusal does not hold %q:\n%s", label, w, out)
		}
	}
	for _, u := range unwanted {
		if strings.Contains(out, u) {
			t.Errorf("%s: the refusal holds %q, which cannot help under the switch:\n%s", label, u, out)
		}
	}
}

// ADR-218 Decisions 4 and 5. Under MRW_SHOWN=off a write refuses a line number
// alone on any line with no recorded showing, whole licence or partial, and its
// refusals name what works there and no read: a pattern, anchor=, a
// search_replace block. Without the switch a number alone on a partial licence
// applies as it did; a showing already recorded, --force, anchor=, a pattern
// and a block behave as they did.
func TestAWriteUnderShownOffWantsTheTextQuoted(t *testing.T) {
	const numberTwo = "@@ f.txt 2 replace\nTWO\n"
	const two = "one\nTWO\nthree\nfour\nfive\n"
	noRead := []string{"mrw read", "Read it whole", "the read to make"}

	t.Run("number_alone_whole_refused", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt")
		out, code := writeShown(t, root, "off", numberTwo)
		refusedWith(t, "whole", out, code, []string{"line 2 of f.txt has no recorded showing in this version", "ddress it by pattern, give anchor=, or send the edit as a search_replace block " + shownNote}, noRead)
		bytesAre(t, "whole", root, "f.txt", shownFixture)
	})
	t.Run("number_alone_ranged_refused", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt:2")
		out, code := writeShown(t, root, "off", numberTwo)
		refusedWith(t, "ranged", out, code, []string{"line 2 of f.txt has no recorded showing in this version", shownNote}, noRead)
		bytesAre(t, "ranged", root, "f.txt", shownFixture)
	})
	t.Run("ranged_quiet_without_switch_applies", func(t *testing.T) {
		// The pair of the case above, on one licence: a ranged --quiet read
		// records exactly what a switched ranged read records. Without the
		// switch a number alone applies on it (ADR-190); with the switch on
		// the write alone, it is refused.
		plain := shownRoot(t)
		runShown(t, plain, "", "read", "--quiet", "f.txt:2")
		wantShowing(t, "the quiet licence", plain, "f.txt", 2, false)
		if out, code := writeShown(t, plain, "", numberTwo); code != 0 {
			t.Errorf("without the switch a number alone on a partial licence exited %d:\n%s", code, out)
		}
		bytesAre(t, "without the switch", plain, "f.txt", two)

		switched := shownRoot(t)
		runShown(t, switched, "", "read", "--quiet", "f.txt:2")
		out, code := writeShown(t, switched, "off", numberTwo)
		refusedWith(t, "the same licence, the switch on the write", out, code, []string{"no recorded showing", shownNote}, noRead)
		bytesAre(t, "with the switch", switched, "f.txt", shownFixture)
	})
	t.Run("switched_read_plain_write_applies", func(t *testing.T) {
		// One variable against number_alone_ranged_refused: the same switched
		// ranged read, and a write made without the switch, which behaves as
		// on main for a partial licence (ADR-190).
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt:2")
		if out, code := writeShown(t, root, "", numberTwo); code != 0 {
			t.Errorf("a write without the switch exited %d:\n%s", code, out)
		}
		bytesAre(t, "switched read, plain write", root, "f.txt", two)
	})
	t.Run("anchor_applies", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt:2")
		if out, code := writeShown(t, root, "off", "@@ f.txt 2 replace anchor=\"two\"\nTWO\n"); code != 0 {
			t.Errorf("exit %d:\n%s", code, out)
		}
		bytesAre(t, "anchor", root, "f.txt", two)
	})
	t.Run("pattern_applies", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt:2")
		if out, code := writeShown(t, root, "off", "@@ f.txt /^two$/ replace\nTWO\n"); code != 0 {
			t.Errorf("exit %d:\n%s", code, out)
		}
		bytesAre(t, "pattern", root, "f.txt", two)
	})
	t.Run("multi_refused_names_the_block", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt")
		out, code := writeShown(t, root, "off", "@@ f.txt 2-3 replace anchor=\"two\"\nTWO\nTHREE\n")
		refusedWith(t, "multi", out, code, []string{"needs a served line after 3", "send the edit as a search_replace block (mrw write --format=search_replace) whose SEARCH and REPLACE end in the same line, which keeps that line and quotes it; to only remove lines, a delete with anchor= " + shownNote}, noRead)
		bytesAre(t, "multi", root, "f.txt", shownFixture)
	})
	t.Run("block_applies", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt")
		block := "f.txt\n<<<<<<< SEARCH\ntwo\nthree\nfour\n=======\nTWO\nTHREE\nfour\n>>>>>>> REPLACE\n"
		if out, code := writeShown(t, root, "off", block, "--format=search_replace"); code != 0 {
			t.Errorf("exit %d:\n%s", code, out)
		}
		bytesAre(t, "block", root, "f.txt", "one\nTWO\nTHREE\nfour\nfive\n")
	})
	t.Run("earlier_showing_applies", func(t *testing.T) {
		// Decision 1's boundary, on the write: a showing already in the ledger
		// for this version is believed.
		root := shownRoot(t)
		runShown(t, root, "", "read", "f.txt:2")
		runShown(t, root, "off", "read", "f.txt")
		if out, code := writeShown(t, root, "off", numberTwo); code != 0 {
			t.Errorf("exit %d:\n%s", code, out)
		}
		bytesAre(t, "earlier showing", root, "f.txt", two)
	})
	t.Run("force_applies", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt:2")
		if out, code := writeShown(t, root, "off", numberTwo, "--force"); code != 0 {
			t.Errorf("exit %d:\n%s", code, out)
		}
		bytesAre(t, "force", root, "f.txt", two)
	})
	t.Run("occurrence_reason", func(t *testing.T) {
		root := shownRoot(t)
		const dup = "x\ny\nx\ny\n"
		if err := os.WriteFile(filepath.Join(root, "g.txt"), []byte(dup), 0o644); err != nil {
			t.Fatal(err)
		}
		runShown(t, root, "off", "read", "g.txt")
		out, code := writeShown(t, root, "off", "@@ g.txt /^x$/ replace occurrence=2\nX\n")
		refusedWith(t, "occurrence", out, code, []string{"before occurrence 2", "name the line by number with anchor=, or pass --force " + shownNote}, []string{"read them", "mrw read"})
		bytesAre(t, "occurrence", root, "g.txt", dup)
		// The pair: with the earlier match shown before the switch, the same
		// plan lands (Decision 1: a showing already recorded is believed).
		shown := shownRoot(t)
		if err := os.WriteFile(filepath.Join(shown, "g.txt"), []byte(dup), 0o644); err != nil {
			t.Fatal(err)
		}
		runShown(t, shown, "", "read", "g.txt:1")
		runShown(t, shown, "off", "read", "g.txt")
		if out, code := writeShown(t, shown, "off", "@@ g.txt /^x$/ replace occurrence=2\nX\n"); code != 0 {
			t.Errorf("with the earlier match shown, occurrence=2 exited %d:\n%s", code, out)
		}
		bytesAre(t, "occurrence, shown", shown, "g.txt", "x\ny\nX\ny\n")
	})
	t.Run("from_reason", func(t *testing.T) {
		root := shownRoot(t)
		runShown(t, root, "off", "read", "f.txt")
		out, code := writeShown(t, root, "off", "@@ new.txt 0 create from=f.txt:1-2\n")
		refusedWith(t, "from", out, code, []string{"type the lines in the plan " + shownNote}, []string{"Read them first", "mrw read"})
		if _, err := os.Stat(filepath.Join(root, "new.txt")); err == nil {
			t.Error("the refused copy made new.txt")
		}
		// The pair: lines shown before the switch are copied under it.
		shown := shownRoot(t)
		runShown(t, shown, "", "read", "f.txt:1-2")
		runShown(t, shown, "off", "read", "f.txt")
		if out, code := writeShown(t, shown, "off", "@@ new.txt 0 create from=f.txt:1-2\n"); code != 0 {
			t.Errorf("with the source shown, the copy exited %d:\n%s", code, out)
		}
		bytesAre(t, "from, shown", shown, "new.txt", "one\ntwo\n")
	})
}
