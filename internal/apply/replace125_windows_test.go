//go:build windows

package apply

import (
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
	if err == nil || res.Applied {
		t.Fatalf("a held target did not stop the plan: err %v, applied %v", err, res.Applied)
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
	if q := hunkFor(t, res, "q?.txt"); q.Status != StatusFailed || !strings.Contains(q.Reason, "q?.txt") {
		t.Errorf("the invalid name was not refused on its hunk: %+v", q)
	}
	if read(t, root, "a.txt") != "a\n" {
		t.Error("a.txt was written beside a refused name")
	}
}
