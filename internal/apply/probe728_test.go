package apply

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// #728. A create's name probe makes the real path and removed it by name, so a
// file another process swapped in at that moment was removed and the create
// landed over nothing of its. The probe now removes only the file it made.
// The swap is injected at the first seam either version asks about the probe:
// the lookup that identifies the probe file (now), the removal (before).
func TestAProbeDoesNotRemoveAFileThatReplacedIt(t *testing.T) {
	root := t.TempDir()
	done := false
	swap := func(p string) {
		if !done && filepath.Base(p) == "n.txt" {
			done = true
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			write(t, root, "n.txt", "FOREIGN\n")
		}
	}
	realRemove, realLstat := removeFn, lstatFn
	t.Cleanup(func() { removeFn, lstatFn = realRemove, realLstat })
	removeFn = func(tr *tree, p string) error { swap(p); return realRemove(tr, p) }
	lstatFn = func(tr *tree, p string) (os.FileInfo, error) { swap(p); return realLstat(tr, p) }

	res, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{})
	if err == nil && res.Applied {
		t.Fatalf("a create landed over a file that replaced its probe: %+v", res.Files)
	}
	if got := read(t, root, "n.txt"); got != "FOREIGN\n" {
		t.Errorf("n.txt = %q: the file that replaced the probe was removed", got)
	}
}

// The pair: with nobody in the way the probe leaves nothing and the create lands.
func TestAProbeThatIsLeftAloneIsRemovedAndTheCreateLands(t *testing.T) {
	root := t.TempDir()
	res, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{})
	if err != nil || !res.Applied || read(t, root, "n.txt") != "PLAN\n" || len(res.LeftBehind) != 0 {
		t.Fatalf("a plain create: %v %+v", err, res)
	}
}

// inPlace runs change on root/n.txt the first time a seam asks about it: the
// same inode, changed, as when a freed inode number is reused for another file.
func inPlace(t *testing.T, change func(p string)) {
	t.Helper()
	done := false
	hit := func(tr *tree, p string) {
		if !done && filepath.Base(p) == "n.txt" {
			done = true
			change(filepath.Join(tr.abs, "n.txt"))
		}
	}
	realRemove, realLstat := removeFn, lstatFn
	t.Cleanup(func() { removeFn, lstatFn = realRemove, realLstat })
	removeFn = func(tr *tree, p string) error { hit(tr, p); return realRemove(tr, p) }
	lstatFn = func(tr *tree, p string) (os.FileInfo, error) { hit(tr, p); return realLstat(tr, p) }
}

// Review of #744. On Linux a freed inode number is reused at once, so identity
// alone read a swapped-in file as the probe: the probe also had to be empty, with
// the time it was made. A file with the probe's inode that is not empty is
// another's now.
func TestAProbeFileThatSomeoneWroteIntoIsNotRemoved(t *testing.T) {
	root := t.TempDir()
	inPlace(t, func(p string) {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("FOREIGN\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		// The time kept, so only the size says it is not the probe.
		if err := os.Chtimes(p, fi.ModTime(), fi.ModTime()); err != nil {
			t.Fatal(err)
		}
	})
	res, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{})
	if err == nil && res.Applied {
		t.Fatalf("a create landed over a file written into its probe: %+v", res.Files)
	}
	if got := read(t, root, "n.txt"); got != "FOREIGN\n" {
		t.Errorf("n.txt = %q: a file that is not empty was removed as the probe", got)
	}
}

func TestAProbeFileWhoseTimeMovedIsNotRemoved(t *testing.T) {
	root := t.TempDir()
	inPlace(t, func(p string) {
		later := time.Now().Add(time.Hour)
		if err := os.Chtimes(p, later, later); err != nil {
			t.Fatal(err)
		}
	})
	res, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{})
	if err == nil && res.Applied {
		t.Fatalf("a create landed over a file whose time moved: %+v", res.Files)
	}
	if _, err := os.Lstat(filepath.Join(root, "n.txt")); err != nil {
		t.Errorf("n.txt was removed though its time moved after the probe made it: %v", err)
	}
}
