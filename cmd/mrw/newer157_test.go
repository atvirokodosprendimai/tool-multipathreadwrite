package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// newerKey is how the sentence names the newer ledger's writer.
const newerKey = "written by a newer mrw (v9.9.9 (fffffff))"

// failLines is every FAIL line of a receipt.
func failLines(out string) []string {
	var fails []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "FAIL ") {
			fails = append(fails, l)
		}
	}
	return fails
}

// newerRoot is a checkout holding f.txt and g.txt under a ledger a newer mrw
// wrote, with the state home pointed at a temporary directory.
func newerRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, name := range []string{"f.txt", "g.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("one\ntwo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte("#mrw-seen v6 v9.9.9 (fffffff)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// writeUnderNewer runs plan text as mrw write --no-check in root, and returns the
// receipt, stderr and the exit code.
func writeUnderNewer(t *testing.T, root, planText string, args ...string) (string, string, int) {
	t.Helper()
	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte(planText), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := append(append([]string{"write", "--no-check"}, args...), plan)
	var out string
	var code int
	errOut := stderrOf(t, func() { out, code = runIn(t, root, argv...) })
	return out, errOut, code
}

// ADR-157 Decisions 1 and 2, through the command, under a newer ledger. A
// refused hunk's FAIL line is led by the cause and names no read and no
// --force; the sentence is said once per refused hunk and nowhere else; a write
// no refusal speaks for (a create, a dry run, a create over a file that exists)
// says it once on stderr; a read still says it once.
func TestANewerLedgersRefusalLeadsWithItsCause(t *testing.T) {
	root := newerRoot(t)

	out, errOut, code := writeUnderNewer(t, root, "@@ f.txt 1 replace\nONE\n")
	fails := failLines(out)
	if code != 1 || len(fails) != 1 || !strings.Contains(fails[0], "f.txt cannot be edited by this mrw: the read ledger was "+newerKey) {
		t.Fatalf("one refused hunk: exit %d, stdout %q", code, out)
	}
	if n := strings.Count(out+errOut, newerKey); n != 1 {
		t.Errorf("one refused hunk said the sentence %d times: stdout %q stderr %q", n, out, errOut)
	}
	if strings.Contains(out+errOut, "mrw read f.txt") || strings.Contains(out+errOut, "--force") {
		t.Errorf("the refusal still offers advice that cannot help: stdout %q stderr %q", out, errOut)
	}

	out, errOut, code = writeUnderNewer(t, root, "@@ f.txt 1 replace\nONE\n@@ g.txt 1 replace\nONE\n")
	fails = failLines(out)
	if code != 1 || len(fails) != 2 || strings.Count(out+errOut, newerKey) != 2 || strings.Contains(errOut, newerKey) {
		t.Errorf("two refused files: want two FAIL lines and the sentence in each, nowhere else; exit %d stdout %q stderr %q", code, out, errOut)
	}
	for _, f := range fails {
		if !strings.Contains(f, newerKey) {
			t.Errorf("a refused hunk's FAIL line does not carry the sentence: %q", f)
		}
	}

	for _, c := range []struct {
		name, plan string
		args       []string
		exit       int
	}{
		{"a create of a new file", "@@ new.txt 0 create\nx\n", nil, 0},
		{"a dry run", "@@ other.txt 0 create\nx\n", []string{"--dry-run"}, 0},
		{"a create over a file that exists", "@@ f.txt 0 create\nx\n", nil, 1},
		{"a plan that does not parse", "@@ f.txt 1 frobnicate\nx\n", nil, 2},
		{"a plan that does not parse, under --json", "@@ f.txt 1 frobnicate\nx\n", []string{"--json"}, 2},
	} {
		out, errOut, code := writeUnderNewer(t, root, c.plan, c.args...)
		if code != c.exit || strings.Count(errOut, newerKey) != 1 || strings.Contains(out, newerKey) {
			t.Errorf("%s: exit %d (want %d), want the sentence once on stderr; stdout %q stderr %q", c.name, code, c.exit, out, errOut)
		}
	}

	errOut = stderrOf(t, func() { _, code = runIn(t, root, "read", "f.txt") })
	if code != 0 || strings.Count(errOut, newerKey) != 1 {
		t.Errorf("a read: exit %d, stderr %q; want the sentence once", code, errOut)
	}
}
