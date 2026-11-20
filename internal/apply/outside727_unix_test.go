//go:build !windows

package apply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// linked727 makes a root holding d/f.txt, and a second directory with f.txt that
// d does NOT point at; link names a link to it, "out".
func linked727(t *testing.T) (root string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(root, "d", "f.txt"), filepath.Join(outside, "f.txt")} {
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Fatal(err)
	}
	return root
}

// #727. identityRefusal opened the file by name to learn why its identity could
// not be read: through a link out of the root it opened a file outside it and
// said the plan could be sent again. Through the held root it is refused.
func TestIdentityRefusalDoesNotOpenAFileOutsideTheRoot(t *testing.T) {
	root := linked727(t)
	if got := identityRefusal(root, "d/f.txt", filepath.Join(root, "d", "f.txt")); !strings.Contains(got, "send the plan again") {
		t.Errorf("the pair: a file inside the root: %q", got)
	}
	if got := identityRefusal(root, "out/f.txt", filepath.Join(root, "out", "f.txt")); strings.Contains(got, "send the plan again") {
		t.Errorf("a file outside the root was opened to explain itself: %q", got)
	}
}

// #727. changedSince asked a file why its identity could not be read by opening
// the name; through a link out of the root that was a file outside it. Through
// the tree it is refused, so the cause is the refusal and not "unread".
func TestChangedSinceDoesNotOpenAFileOutsideTheRoot(t *testing.T) {
	root := linked727(t)
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	prev := sameFileFn
	sameFileFn = func(a, b os.FileInfo) bool { return false }
	t.Cleanup(func() { sameFileFn = prev })
	was, err := os.Stat(filepath.Join(root, "d", "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	_, cause := changedSince(tr, filepath.Join(root, "out", "f.txt"), was)
	if errors.Is(cause, errIdentityUnread) {
		t.Errorf("a file outside the root was opened and found readable: %v", cause)
	}
	_, cause = changedSince(tr, filepath.Join(root, "d", "f.txt"), was)
	if !errors.Is(cause, errIdentityUnread) {
		t.Errorf("the pair: a file inside the root, cause %v, want its identity unread", cause)
	}
}

// #727. statHeld described a FIFO or an unreadable file by the name; through a
// link out of the root that was the type and size of a file outside it.
func TestStatHeldDoesNotDescribeAFileOutsideTheRoot(t *testing.T) {
	root := linked727(t)
	outside, err := os.Readlink(filepath.Join(root, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(outside, "p"), 0o644); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "d", "p"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fi, err := statHeld(root, filepath.Join(root, "d", "p")); err != nil || fi.Mode().IsRegular() {
		t.Errorf("the pair: a FIFO inside the root: %v %v", fi, err)
	}
	if fi, err := statHeld(root, filepath.Join(root, "out", "p")); err == nil {
		t.Errorf("a FIFO outside the root was described: %v", fi.Mode())
	}
}
