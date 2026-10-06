//go:build windows

package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// hold opens name in root the way an editor or a language server does, with
// the sharing given, and keeps it open until the test ends.
func hold(t *testing.T, root, name string, share uint32) {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, share, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("holding %s: %v", name, err)
	}
	t.Cleanup(func() { _ = syscall.CloseHandle(h) })
}

// ADR-125 T2. A target another process holds without delete sharing, last in
// the plan, failed the commit's rename after the first file had landed:
// PARTIALLY APPLIED. Asked before the first rename, it writes nothing.
func TestAHeldTargetRefusesBeforeAnyRename(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	write(t, root, "b.txt", "b\n")
	hold(t, root, "b.txt", syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE)
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "b.txt", Start: 1, End: 1, Op: "replace", Body: []string{"B"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	// ADR-132: a file held by another process is the target's — a refused
	// hunk and no error, exit 1, as the same hold found at validation is.
	if err != nil || res.Applied {
		t.Fatalf("a held target did not refuse the plan as a hunk: err %v, applied %v", err, res.Applied)
	}
	if b := hunkFor(t, res, "b.txt"); b.Status != StatusFailed || !strings.Contains(b.Reason, "held open by another process") {
		t.Errorf("the held target's hunk does not say it is held: %+v", b)
	}
	if read(t, root, "a.txt") != "a\n" {
		t.Error("a.txt was written before the held target was refused")
	}
}

// ADR-125 T2. A holder that shares delete does not block the commit: Go's
// os.Root.Rename replaces with POSIX semantics on Windows. This pins that, so
// a toolchain that stops doing it fails here rather than for a user.
func TestAHolderThatSharesDeleteDoesNotBlockTheCommit(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	write(t, root, "b.txt", "b\n")
	hold(t, root, "b.txt", syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE)
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "b.txt", Start: 1, End: 1, Op: "replace", Body: []string{"B"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if err != nil || !res.Applied || read(t, root, "a.txt") != "A\n" || read(t, root, "b.txt") != "B\n" {
		t.Errorf("a holder sharing delete blocked the plan: err %v, %+v", err, res)
	}
}

// ADR-125 T1 on Windows. A name the system refuses printed one bare line at
// exit 2; it is refused on its hunk, naming why.
func TestAnInvalidNameGetsAReceipt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "q?.txt", Start: 0, Op: "create", Body: []string{"q"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if err != nil {
		t.Fatalf("an invalid name returned its error bare, with no receipt: %v", err)
	}
	if q := hunkFor(t, res, "q?.txt"); q.Status != StatusFailed || !strings.Contains(q.Reason, "q?.txt") || !strings.Contains(q.Reason, "not a valid name on this system") {
		t.Errorf("the invalid name was not refused on its hunk: %+v", q)
	}
	if read(t, root, "a.txt") != "a\n" {
		t.Error("a.txt was written beside a refused name")
	}
}

// The Codex review of #338. The probe opened an absolute path, so a parent
// swapped for a junction that leads out of the root after validation was
// followed, and the probe opened a file outside the root for DELETE. It opens
// the leaf relative to a parent handle taken through the root now: the swap
// is refused, and the file outside is never opened.
func TestTheProbeDoesNotFollowASwappedParentOutOfTheRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	write(t, root, "d/f.txt", "inside\n")
	write(t, outside, "f.txt", "outside\n")
	// Held without delete sharing: a probe that reached it would say so.
	hold(t, outside, "f.txt", syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE)
	real := replaceableFn
	t.Cleanup(func() { replaceableFn = real })
	replaceableFn = func(tr *tree, full string) error {
		if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "d"), outside).CombinedOutput(); err != nil {
			t.Fatalf("cannot make a junction here, so confinement would go unproved: %v %s", err, out)
		}
		return real(tr, full)
	}
	res, err := Apply(root, []Input{{Path: "d/f.txt", Op: "unlink", Lines: -1, Index: 0}}, Options{Force: true})
	// Whether the escape reads as the target's or not (ADR-132), the plan stops.
	if res.Applied || res.Failed == 0 {
		t.Fatalf("the probe through a swapped parent did not stop the plan: err %v, %+v", err, res)
	}
	if h := hunkFor(t, res, filepath.FromSlash("d/f.txt")); strings.Contains(h.Reason, "held open") {
		t.Errorf("the probe followed the junction out of the root and opened the file there: %+v", h)
	}
	if read(t, outside, "f.txt") != "outside\n" {
		t.Error("the file outside the root was changed")
	}
}
