package adversarial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// --root/-C is documented as "resolve every path relative to DIR". A caller
// pointing mrw at a checkout is naming the thing it may change, so a hunk that
// climbs out with ../ is editing a file the caller never scoped — and nothing
// in the plan format marks it, while the receipt prints the relative path, so
// the escape is invisible in the output too.
func TestAPlanCannotWriteOutsideTheRoot(t *testing.T) {
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

	if res.Failed != 1 {
		t.Errorf("a hunk addressing %q was accepted: failed=%d, applied=%v",
			"../escaped.txt", res.Failed, res.Applied)
	}
	if _, statErr := os.Stat(filepath.Join(outer, "escaped.txt")); statErr == nil {
		t.Error("mrw wrote outside the root it was given")
	}
	if res.Failed == 1 && !strings.Contains(res.Hunks[0].Reason, "outside the root") {
		t.Errorf("the refusal does not say why: %s", res.Hunks[0].Reason)
	}
}

// A path that stays inside the root after cleaning is fine, including one that
// climbs and comes back. Refusing that would be a false alarm.
func TestAPathThatClimbsAndReturnsIsStillInsideTheRoot(t *testing.T) {
	root := tree(t, map[string]string{"sub/a.go": goFile})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "sub/../sub/a.go", Start: 1, End: 1, Op: "replace",
		Body: []string{"package q"}, Lines: unset,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("a path inside the root was refused: %s", res.Hunks[0].Reason)
	}
	if got := readFile(t, root, "sub/a.go"); !strings.HasPrefix(got, "package q") {
		t.Errorf("the edit did not land:\n%s", got)
	}
}

// A CRLF file must come back using CRLF. Every untouched line keeps whatever it
// had, and a body line supplied by the plan takes the file's own convention —
// otherwise one edit leaves two conventions in the file, and the diff is bigger
// and stranger than the change asked for.
func TestACRLFFileKeepsItsLineEndings(t *testing.T) {
	root := tree(t, map[string]string{"crlf.txt": "alpha\r\nbeta\r\ngamma\r\n"})

	if _, err := apply.Apply(root, []apply.Input{{
		Path: "crlf.txt", Start: 2, End: 2, Op: "replace",
		Body: []string{"BETA"}, Lines: unset,
	}}, apply.Options{}); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, root, "crlf.txt")
	if want := "alpha\r\nBETA\r\ngamma\r\n"; got != want {
		t.Errorf("CRLF was not preserved:\n got %q\nwant %q", got, want)
	}
}

// An LF file must stay one: the line-ending fix must not start writing CRLF
// into files that never had it.
func TestAnLFFileKeepsItsLineEndings(t *testing.T) {
	root := tree(t, map[string]string{"lf.txt": "alpha\nbeta\ngamma\n"})

	if _, err := apply.Apply(root, []apply.Input{{
		Path: "lf.txt", Start: 2, End: 2, Op: "replace",
		Body: []string{"BETA"}, Lines: unset,
	}}, apply.Options{}); err != nil {
		t.Fatal(err)
	}

	if got, want := readFile(t, root, "lf.txt"), "alpha\nBETA\ngamma\n"; got != want {
		t.Errorf("LF file changed convention:\n got %q\nwant %q", got, want)
	}
}

// A file that mixes conventions is not converted either way. mrw is an editor,
// not a formatter: lines it did not touch survive byte for byte, which is also
// what keeps its sha equal to the one internal/seen records.
func TestAMixedEndingFileIsNotNormalised(t *testing.T) {
	root := tree(t, map[string]string{"mixed.txt": "alpha\r\nbeta\ngamma\r\n"})

	if _, err := apply.Apply(root, []apply.Input{{
		Path: "mixed.txt", Start: 2, End: 2, Op: "replace",
		Body: []string{"BETA"}, Lines: unset,
	}}, apply.Options{}); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, root, "mixed.txt")
	if !strings.HasPrefix(got, "alpha\r\n") || !strings.HasSuffix(got, "gamma\r\n") {
		t.Errorf("untouched lines lost their endings: %q", got)
	}
}

// Editing through a symlink must edit the file the link points at. Writing by
// rename is what makes a crash mid-write safe, but renaming over the link
// replaces it with a regular file: the edit lands somewhere new, the real file
// is untouched, and the receipt says nothing about the tree's shape changing.
func TestEditingThroughASymlinkKeepsTheSymlink(t *testing.T) {
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
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("link.go was a symlink and is now a regular file")
	}
	if got := readFile(t, root, "real.go"); !strings.HasPrefix(got, "package q") {
		t.Errorf("the edit did not reach the file the link pointed at:\n%s", got)
	}
}

// And a symlink is not a way around the boundary: following one must not write
// where the caller did not scope.
func TestASymlinkCannotCarryAWriteOutsideTheRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outer, "outside.go")
	if err := os.WriteFile(target, []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.go")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	res, err := apply.Apply(root, []apply.Input{{
		Path: "link.go", Start: 1, End: 1, Op: "replace",
		Body: []string{"package q"}, Lines: unset,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if res.Failed != 1 {
		t.Errorf("a symlink pointing out of the root was followed: failed=%d", res.Failed)
	}
	if b, _ := os.ReadFile(target); string(b) != goFile {
		t.Error("the file outside the root was rewritten through a symlink")
	}
}

// A file terminated with lone "\r" — old-Mac text, and what a generator
// emitting bare carriage returns leaves behind — has lines like any other file,
// and its interior must be addressable rather than the whole file counting as
// one unsplittable line.
func TestACRTerminatedFileHasAddressableLines(t *testing.T) {
	root := tree(t, map[string]string{"cr.txt": "one\rtwo\rthree\r"})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "cr.txt", Start: 2, End: 2, Op: "replace",
		Body: []string{"TWO"}, Lines: unset,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("line 2 of a three-line CR-terminated file was unaddressable: %s", res.Hunks[0].Reason)
	}
	if got, want := readFile(t, root, "cr.txt"), "one\rTWO\rthree\r"; got != want {
		t.Errorf("CR termination was not preserved:\n got %q\nwant %q", got, want)
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
