package adversarial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// KNOWN GAP — for M, because it changes what a served plan may do.
//
// --root/-C is documented as "resolve every path relative to DIR", which reads
// as a boundary and is not one: apply joins the hunk's path onto root and a
// "../" climbs straight out, creating directories on the way. Nothing marks
// such a hunk in the plan, and the receipt prints the relative path, so the
// escape is invisible in the output too.
//
// It is not a privilege boundary — the caller could have written the path
// directly — but it is a scope one, and mrw is a tool an agent points at a
// checkout. This test pins today's behaviour; deciding to refuse it is a
// served-path change and belongs in an ADR.
func TestKnownGap_APlanCanWriteOutsideTheRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := apply.Apply(root, []apply.Input{{
		Path: "../escaped.txt", Start: 0, End: 0, Op: "create",
		Body: []string{"written outside the root"}, Lines: unset,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	// The BYTES, not merely the existence: a file created outside the root with
	// the wrong contents is a different bug, and would pass a stat.
	escaped := filepath.Join(outer, "escaped.txt")
	b, statErr := os.ReadFile(escaped)
	if statErr != nil {
		t.Fatalf("mrw now refuses a hunk that leaves the root (failed=%d) — good, but the README, "+
			"the CLI help for -C and this test all describe the old behaviour", res.Failed)
	}
	if string(b) != "written outside the root\n" {
		t.Errorf("wrote outside the root, and with unexpected contents: %q", b)
	}
}

// KNOWN GAP — a CRLF file comes back mixed.
//
// readLines splits on "\n" only, so every untouched line keeps its trailing
// "\r" while a body line from the plan has none. The edit lands correctly and
// the file now has two conventions in it, which shows up as a bigger diff than
// the change asked for.
//
// The fix is not a one-liner: shaOf must reproduce the file's ORIGINAL bytes,
// because internal/seen hashes raw bytes and the two hashes have to agree or
// every CRLF file reads as "changed since mrw last saw it". So a line-ending
// mode has to be threaded from readLines through shaOf and writeFile together.
func TestKnownGap_ACRLFFileComesBackMixed(t *testing.T) {
	root := tree(t, map[string]string{"crlf.txt": "alpha\r\nbeta\r\ngamma\r\n"})

	if _, err := apply.Apply(root, []apply.Input{{
		Path: "crlf.txt", Start: 2, End: 2, Op: "replace",
		Body: []string{"BETA"}, Lines: unset,
	}}, apply.Options{}); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, root, "crlf.txt")
	if got == "alpha\r\nBETA\r\ngamma\r\n" {
		t.Error("CRLF is now preserved — delete this test and add the positive one to internal/apply")
	}
	if want := "alpha\r\nBETA\ngamma\r\n"; got != want {
		t.Errorf("CRLF handling changed to something else again:\n got %q\nwant %q", got, want)
	}
}

// KNOWN GAP — editing through a symlink replaces the link.
//
// writeFile creates a temp file beside the target and renames over it, which is
// what makes a crash mid-write leave the original intact. The cost is that the
// rename replaces the LINK rather than following it: the edit lands in a new
// regular file, the file the link pointed at is untouched, and the receipt says
// nothing about the tree's shape having changed.
func TestKnownGap_EditingThroughASymlinkReplacesTheLink(t *testing.T) {
	root := tree(t, map[string]string{"real.go": goFile})
	link := filepath.Join(root, "link.go")
	if err := os.Symlink("real.go", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := apply.Apply(root, []apply.Input{{
		Path: "link.go", Start: 1, End: 1, Op: "replace",
		Body: []string{"package q"}, Lines: unset,
	}}, apply.Options{}); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("symlinks are now followed on write — good, but say so where writeFile documents the rename")
	}
	if got := readFile(t, root, "link.go"); !strings.HasPrefix(got, "package q") {
		t.Errorf("the link was replaced by a regular file that does not hold the edit:\n%s", got)
	}
	if got := readFile(t, root, "real.go"); got != goFile {
		t.Errorf("the target of the link changed, so the write is no longer link-replacing:\n%s", got)
	}
}

// KNOWN GAP — a file terminated with lone "\r" is one line to mrw.
//
// readLines splits on "\n", so old-Mac text, and any file where a generator
// emitted bare carriage returns, is a single unaddressable line. The failure is
// at least loud: the address is reported out of range rather than applied
// somewhere wrong.
func TestKnownGap_ACRTerminatedFileIsOneLine(t *testing.T) {
	root := tree(t, map[string]string{"cr.txt": "one\rtwo\rthree\r"})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "cr.txt", Start: 2, End: 2, Op: "replace",
		Body: []string{"TWO"}, Lines: unset,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Error("line 2 of a CR-terminated file is now addressable; CR handling changed")
	}
	if res.Applied {
		t.Error("the run refused a hunk and applied anyway")
	}
	if got := readFile(t, root, "cr.txt"); got != "one\rtwo\rthree\r" {
		t.Errorf("the refused run still rewrote the file: %q", got)
	}
	if !strings.Contains(res.Hunks[0].Reason, "file has 1 lines") {
		t.Errorf("the refusal no longer says the file is one line: %s", res.Hunks[0].Reason)
	}
}

// The failure that must never become quiet: a plan whose hunks are fine except
// one leaves NOTHING written, and says so per hunk.
func TestOneBadHunkLeavesEveryFileUntouched(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile, "b.go": goFile})

	res, err := apply.Apply(root, []apply.Input{
		{Path: "a.go", Start: 1, End: 1, Op: "replace", Body: []string{"package q"}, Lines: unset},
		{Path: "b.go", Start: 3, End: 3, Op: "replace", Body: []string{"nope"}, Anchor: "NOT HERE", Lines: unset},
	}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if res.Applied {
		t.Error("a plan with a failed hunk reported itself applied")
	}
	if got := readFile(t, root, "a.go"); got != goFile {
		t.Errorf("the good hunk's file was written anyway:\n%s", got)
	}
	if res.Hunks[0].Status != "skipped" {
		t.Errorf("the good hunk reported %q, want skipped", res.Hunks[0].Status)
	}
	if len(res.Files) != 2 {
		t.Errorf("the receipt names %d file(s); the plan addressed 2", len(res.Files))
	}
}
