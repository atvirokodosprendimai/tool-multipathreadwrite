package apply

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func hookLink(t *testing.T, f func(from, to string)) {
	t.Helper()
	real := afterLink
	t.Cleanup(func() { afterLink = real })
	afterLink = f
}

func mrwLeftovers(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	for _, n := range names(t, dir) {
		if strings.HasPrefix(n, ".mrw") {
			out = append(out, n)
		}
	}
	return out
}

// #744 review, P1-1. A destination swapped for another file after the link is a
// failure of the step, not a change at the source: the move was not made, the
// staged temp is taken back with the plan, and the foreign file stays.
func TestACreateTargetReplacedAfterTheLinkFailsAndLeavesNoTemp(t *testing.T) {
	root := t.TempDir()
	hookLink(t, func(_, to string) {
		if filepath.Base(to) == "n.txt" {
			if err := os.Remove(to); err != nil {
				t.Fatal(err)
			}
			write(t, root, "n.txt", "FOREIGN\n")
		}
	})
	res, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a create whose target was replaced after the link reported success: %v %+v", err, res.Files)
	}
	if got := read(t, root, "n.txt"); got != "FOREIGN\n" {
		t.Errorf("n.txt = %q, want the file that replaced it", got)
	}
	if left := mrwLeftovers(t, root); len(left) != 0 || len(res.LeftBehind) != 0 {
		t.Errorf("the staged temp leaked: %v, left_behind %v", left, res.LeftBehind)
	}
}

func TestARenameDestinationReplacedAfterTheLinkFailsAndKeepsTheSource(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	hookLink(t, func(_, to string) {
		if filepath.Base(to) == "b.txt" {
			if err := os.Remove(to); err != nil {
				t.Fatal(err)
			}
			write(t, root, "b.txt", "FOREIGN\n")
		}
	})
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied || !strings.Contains(err.Error(), "appeared before commit") {
		t.Fatalf("a rename whose destination was replaced after the link: %v applied %v", err, res.Applied)
	}
	if read(t, root, "a.txt") != "A\n" || read(t, root, "b.txt") != "FOREIGN\n" {
		t.Errorf("a.txt %q b.txt %q: the source must stay and the foreign file too", read(t, root, "a.txt"), read(t, root, "b.txt"))
	}
}

// #744 review, P1-2. After a successful link a destination that cannot be looked
// at leaves two names: the receipt names the one mrw made.
func TestADestinationThatCannotBeLookedAtAfterTheLinkIsNamedAsLeftBehind(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory whose search permission can be taken away from the caller")
	}
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	d := filepath.Join(root, "d")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	hookLink(t, func(_, to string) {
		if filepath.Base(to) == "b.txt" {
			if err := os.Chmod(d, 0); err != nil {
				t.Fatal(err)
			}
		}
	})
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"d/b.txt"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a rename whose destination could not be looked at reported success: %v", err)
	}
	if !slices.Contains(res.LeftBehind, "d/b.txt") {
		t.Errorf("left_behind = %v, want the name mrw made, d/b.txt", res.LeftBehind)
	}
	if err := os.Chmod(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if read(t, root, "a.txt") != "A\n" || read(t, root, "d/b.txt") != "A\n" {
		t.Error("both names of the file should exist")
	}
}

// #744 review, P1-3. Only an error that says the entry cannot be hard-linked here
// enters check-then-rename; any other link error is the move's, and a fallback
// lookup that fails other than not-exist refuses the move.
func TestTheFallbackEntersOnlyWhereHardLinksAreUnavailableAndOnlyOnAFreeName(t *testing.T) {
	root := t.TempDir()
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	realLink, realLstat := linkFn, fallbackLstatFn
	t.Cleanup(func() { linkFn, fallbackLstatFn = realLink, realLstat })
	p := func(n string) string { return filepath.Join(root, n) }
	linkErr := func(e error) { linkFn = func(*tree, string, string) error { return &os.LinkError{Op: "link", Err: e} } }

	linkErr(syscall.EXDEV)
	write(t, root, "a.txt", "A\n")
	write(t, root, "taken.txt", "T\n")
	if err := tr.renameNew(p("a.txt"), p("taken.txt")); !errors.Is(err, fs.ErrExist) || read(t, root, "taken.txt") != "T\n" || read(t, root, "a.txt") != "A\n" {
		t.Errorf("no hard links, destination taken: %v, want it refused and both files kept", err)
	}
	if err := tr.renameNew(p("a.txt"), p("free.txt")); err != nil || read(t, root, "free.txt") != "A\n" {
		t.Errorf("no hard links, destination free: %v, want the rename", err)
	}
	write(t, root, "c.txt", "C\n")
	linkErr(syscall.EIO)
	if err := tr.renameNew(p("c.txt"), p("d.txt")); !errors.Is(err, syscall.EIO) {
		t.Errorf("a link error that is not 'no hard links': %v, want it returned", err)
	}
	if _, err := os.Lstat(p("d.txt")); err == nil || read(t, root, "c.txt") != "C\n" {
		t.Error("a link error that is not 'no hard links' still renamed")
	}
	linkErr(syscall.EPERM)
	fallbackLstatFn = func(*tree, string) (fs.FileInfo, error) { return nil, syscall.EACCES }
	if err := tr.renameNew(p("c.txt"), p("e.txt")); !errors.Is(err, syscall.EACCES) {
		t.Errorf("a destination that cannot be looked at: %v, want it refused", err)
	}
	if _, err := os.Lstat(p("e.txt")); err == nil || read(t, root, "c.txt") != "C\n" {
		t.Error("a destination that cannot be looked at was renamed onto")
	}
}

// #744 review, P1-4. A source replaced after its bytes were read, before its
// identity was taken, was judged by the old sha and the new identity.
func TestASourceReplacedBetweenItsReadAndItsIdentityIsRefused(t *testing.T) {
	sum := sha256.Sum256([]byte("A\n"))
	for _, op := range []string{"unlink", "rename"} {
		t.Run(op, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "a.txt", "A\n")
			real := loadFn
			t.Cleanup(func() { loadFn = real })
			loadFn = func(r, p string) (text, bool, error) {
				o, e, err := real(r, p)
				if filepath.Base(p) == "a.txt" {
					fi, err := os.Stat(p)
					if err != nil {
						t.Fatal(err)
					}
					// Same size and time: only the identity says it is another file.
					replaceWith(t, root, "a.txt", "B\n")
					if err := os.Chtimes(p, fi.ModTime(), fi.ModTime()); err != nil {
						t.Fatal(err)
					}
				}
				return o, e, err
			}
			in := Input{Path: "a.txt", Op: op, SHA: hex.EncodeToString(sum[:]), Lines: -1, Index: 0}
			if op == "rename" {
				in.Body = []string{"b.txt"}
			}
			res, err := Apply(root, []Input{in}, Options{})
			if err != nil || res.Applied || res.Failed != 1 {
				t.Fatalf("a source replaced after its read: %v %+v", err, res)
			}
			if h := hunkFor(t, res, "a.txt"); !strings.Contains(h.Reason, "changed while mrw read it") {
				t.Errorf("reason %q", h.Reason)
			}
			if read(t, root, "a.txt") != "B\n" {
				t.Error("the replacement was removed or moved")
			}
			if _, err := os.Lstat(filepath.Join(root, "b.txt")); err == nil {
				t.Error("the replacement was moved to b.txt")
			}
		})
	}
}

// #744 review, P2-6. The aside's name is fresh, so nothing there can be replaced:
// the move to it is a plain rename, which vacates the old name at once.
func TestAnUnlinkMovesTheFileToItsAsideWithAPlainRename(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	var kept []bool
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	commitRenameFn = func(tr *tree, from, to string) error {
		if strings.HasPrefix(filepath.Base(to), ".mrw-aside-") {
			kept = append(kept, tr.keep)
		}
		return real(tr, from, to)
	}
	if _, err := Apply(root, []Input{{Path: "a.txt", Op: "unlink", Lines: -1, Index: 0}}, Options{}); err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 || kept[0] {
		t.Errorf("the aside move used a no-replace tree: %v", kept)
	}
}

// #744 review, P2-7. The probe's descriptor stays open until its file is
// removed: on ext4 the freed inode number goes to the file swapped in, and only
// an open descriptor keeps it. Where inode numbers are not reused the order is
// what is seen: nothing closes the probe before it is removed.
func TestTheProbeFileIsNotClosedBeforeItIsRemoved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows opens without delete sharing, so the probe is closed first there")
	}
	root := t.TempDir()
	var events []string
	realClose, realRemove := closeProbeFn, removeFn
	t.Cleanup(func() { closeProbeFn, removeFn = realClose, realRemove })
	closeProbeFn = func(f *os.File) error { events = append(events, "close"); return realClose(f) }
	removeFn = func(tr *tree, p string) error {
		if filepath.Base(p) == "n.txt" {
			events = append(events, "remove")
		}
		return realRemove(tr, p)
	}
	if _, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(events, []string{"remove", "close"}) {
		t.Errorf("events %v, want the probe removed while its descriptor is still open", events)
	}
}

// ... and an empty file that carries the probe's modification time, swapped in at
// the moment of the lookup, is not the probe: nothing about it differs but its
// identity.
func TestAnEmptyFileWithTheProbesTimeThatReplacedItIsNotRemoved(t *testing.T) {
	root := t.TempDir()
	done := false
	realLstat := lstatFn
	t.Cleanup(func() { lstatFn = realLstat })
	lstatFn = func(tr *tree, p string) (fs.FileInfo, error) {
		if !done && filepath.Base(p) == "n.txt" {
			done = true
			fi, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			write(t, root, "n.txt", "")
			if err := os.Chtimes(p, time.Now(), fi.ModTime()); err != nil {
				t.Fatal(err)
			}
		}
		return realLstat(tr, p)
	}
	res, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{})
	if err == nil && res.Applied {
		t.Fatalf("a create landed over the file that replaced its probe: %+v", res.Files)
	}
	if _, err := os.Lstat(filepath.Join(root, "n.txt")); err != nil {
		t.Errorf("the empty file that replaced the probe was removed: %v", err)
	}
}
