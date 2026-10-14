package apply

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-125 T1. A file mrw could not open — permission denied, held open by
// another process, a name the system refuses — returned its error bare from
// validation, before any hunk had a verdict, so the caller saw one `mrw: …`
// line at exit 2 and no receipt. It is refused on its hunk now, naming why,
// with every sibling skipped and nothing written.
func TestAnUnreadableTargetGetsAReceipt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	write(t, root, "b.txt", "b\n")
	real := loadFn
	t.Cleanup(func() { loadFn = real })
	loadFn = func(path string) (text, bool, error) {
		if filepath.Base(path) == "b.txt" {
			return text{eol: "\n"}, false, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
		}
		return real(path)
	}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "b.txt", Start: 1, End: 1, Op: "replace", Body: []string{"B"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if err != nil {
		t.Fatalf("an unopenable target returned its error bare, with no receipt: %v", err)
	}
	b := hunkFor(t, res, "b.txt")
	if res.Applied || b.Status != StatusFailed || !strings.Contains(b.Reason, "permission denied") {
		t.Errorf("the unopenable target was not refused on its hunk naming the cause: %+v", b)
	}
	if a := hunkFor(t, res, "a.txt"); a.Status != StatusSkipped || read(t, root, "a.txt") != "a\n" {
		t.Errorf("the sibling was not skipped and left alone: %+v", a)
	}
}

// ADR-125 T1. A file whose identity could not be read was refused with "send
// the plan again", which never helps when the cause is a permission: on Windows
// os.SameFile swallows the open's error. The reason names the error an open
// of the file gives, and keeps the old advice only when the open succeeds.
func TestTheIdentityRefusalNamesItsCause(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode bits do not deny reading on Windows; the Windows causes are T2's tests")
	}
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 file")
	}
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	write(t, root, "locked.txt", "l\n")
	full := filepath.Join(root, "locked.txt")
	if err := os.Chmod(full, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(full, 0o644) })
	real := sameFileFn
	t.Cleanup(func() { sameFileFn = real })
	sameFileFn = func(a, b fs.FileInfo) bool { return false }

	res, err := Apply(root, []Input{{Path: "locked.txt", Start: 1, End: 1, Op: "replace", Body: []string{"L"}, Lines: -1, Index: 0}}, Options{Force: true})
	h := hunkFor(t, res, "locked.txt")
	if err != nil || h.Status != StatusFailed || !strings.Contains(h.Reason, "permission denied") || strings.Contains(h.Reason, "send the plan again") {
		t.Errorf("an unreadable identity did not name its cause: err %v, %+v", err, h)
	}
	res, err = Apply(root, []Input{{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0}}, Options{Force: true})
	h = hunkFor(t, res, "a.txt")
	if err != nil || h.Status != StatusFailed || !strings.Contains(h.Reason, "send the plan again") {
		t.Errorf("with nothing to explain it, the refusal lost its advice: err %v, %+v", err, h)
	}
}

// ADR-125 T2. On Windows a target another process holds without delete
// sharing failed the commit's rename after the files before it had landed:
// PARTIALLY APPLIED, by plan order. Every existing target, unlink source and
// rename source is asked whether it can be replaced before the first rename,
// and a refusal writes nothing. The question is a seam, so the loop is proved
// here on every platform; the real probe is the Windows shards' test.
func TestEveryTargetIsAskedBeforeTheFirstRename(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	write(t, root, "u.txt", "u\n")
	write(t, root, "r.txt", "r\n")
	real := replaceableFn
	t.Cleanup(func() { replaceableFn = real })
	asked := map[string]bool{}
	replaceableFn = func(_ *tree, full string) error {
		asked[filepath.Base(full)] = true
		if filepath.Base(full) == "r.txt" {
			return &fs.PathError{Op: "open", Path: full, Err: fs.ErrPermission}
		}
		return nil
	}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "u.txt", Op: "unlink", Lines: -1, Index: 1},
		{Path: "r.txt", Op: "rename", Body: []string{"r2.txt"}, Lines: -1, Index: 2},
	}, Options{Force: true})
	for _, n := range []string{"a.txt", "u.txt", "r.txt"} {
		if !asked[n] {
			t.Errorf("%s was not asked whether it can be replaced", n)
		}
	}
	// ADR-132: a permission is the target's — a refused hunk, no error.
	if err != nil || res.Applied {
		t.Fatalf("a target that cannot be replaced did not refuse the plan: err %v, applied %v", err, res.Applied)
	}
	if r := hunkFor(t, res, "r.txt"); r.Status != StatusFailed || !strings.Contains(r.Reason, "permission denied") {
		t.Errorf("the refused target's hunk does not name the cause: %+v", r)
	}
	if read(t, root, "a.txt") != "a\n" || read(t, root, "u.txt") != "u\n" || read(t, root, "r.txt") != "r\n" {
		t.Error("something was written before the refused target")
	}
}

// The Codex review of #343. The force clause was cut from the formatted
// refusal, so a path holding the phrase lost it while the real advice stayed.
// It is cut from the format now, before the path is put in.
func TestTheForceClauseIsCutFromTheWordsNotTheCallersPath(t *testing.T) {
	root := t.TempDir()
	name := "a, or pass --force.txt"
	write(t, root, name, "a\n")
	res, _ := Apply(root, []Input{{Path: name, Start: 1, End: 1, Op: "replace", Body: []string{"b"}, Lines: -1}}, Options{Seen: map[string]Seen{}, NoForce: true})
	h := hunkFor(t, res, name)
	if h.Status != StatusFailed || !strings.Contains(h.Reason, name) || strings.Count(h.Reason, ", or pass --force") != strings.Count(h.Reason, name) {
		t.Errorf("the path lost the phrase, or the advice stayed: %q", h.Reason)
	}
}
