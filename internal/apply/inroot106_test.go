package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// isLink reports whether p is still a link (a symlink, or on Windows a
// junction, which Go reports as irregular since 1.23).
func isLink(t *testing.T, p string) bool {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0
}

// ADR-106 T2. os.Root follows a relative link that stays inside and refuses an
// absolute one, so the write path hands it resolved paths. An in-root link —
// relative or absolute — and on Windows an in-root junction must stay writable:
// an edit, an unlink and a rename beneath the link apply, change the real
// files, and keep the link itself.
func TestInRootLinksStayWritableThroughTheRoot(t *testing.T) {
	root := t.TempDir()
	write(t, root, filepath.Join("real", "a.txt"), abcde)
	write(t, root, filepath.Join("real", "c.txt"), "c\n")
	write(t, root, filepath.Join("real", "d.txt"), "d\n")
	write(t, root, "f.txt", abcde)
	links := map[string]string{"rel": "real"}
	if err := os.Symlink("real", filepath.Join(root, "rel")); err != nil {
		t.Skipf("symlinks cannot be made here: %v", err)
	}
	abs, err := filepath.Abs(filepath.Join(root, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(abs, filepath.Join(root, "abs.txt")); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "junc"), filepath.Join(root, "real")).CombinedOutput(); err != nil {
			t.Fatalf("mklink /J: %v\n%s", err, out)
		}
		links["junc"] = "real"
	}
	for name := range links {
		t.Run(name, func(t *testing.T) {
			res, err := Apply(root, []Input{
				{Path: name + "/a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"via " + name}, Lines: -1, Index: 0},
			}, Options{Force: true})
			if err != nil || !res.Applied {
				t.Fatalf("an edit through the in-root %s link was refused: %v %+v", name, err, res)
			}
			if got := read(t, root, filepath.Join("real", "a.txt")); got != "via "+name+"\nb\nc\nd\ne\n" {
				t.Errorf("real/a.txt holds %q", got)
			}
			if !isLink(t, filepath.Join(root, name)) {
				t.Errorf("%s is no longer a link", name)
			}
		})
	}
	res, err := Apply(root, []Input{
		{Path: "abs.txt", Start: 1, End: 1, Op: "replace", Body: []string{"via abs"}, Lines: -1, Index: 0},
		{Path: "rel/c.txt", Op: "unlink", Lines: -1, Index: 1},
		{Path: "rel/d.txt", Op: "rename", Body: []string{"rel/e.txt"}, Lines: -1, Index: 2},
	}, Options{Force: true})
	if err != nil || !res.Applied {
		t.Fatalf("an edit through an absolute in-root link, and an unlink and a rename beneath a relative one, were refused: %v %+v", err, res)
	}
	if got := read(t, root, "f.txt"); got != "via abs\nb\nc\nd\ne\n" {
		t.Errorf("f.txt holds %q", got)
	}
	if !isLink(t, filepath.Join(root, "abs.txt")) {
		t.Error("abs.txt is no longer a link")
	}
	if _, err := os.Lstat(filepath.Join(root, "real", "c.txt")); !os.IsNotExist(err) {
		t.Errorf("real/c.txt was not removed: %v", err)
	}
	if got := read(t, root, filepath.Join("real", "e.txt")); got != "d\n" {
		t.Errorf("real/e.txt holds %q", got)
	}
}

// ADR-106 T2, and the boundary the record draws for ADR-105's left_behind:
// a staged temp whose directory another process renames away before a failed
// commit is at a path mrw never made. It is not claimed as mrw's leftover,
// nothing outside the root is touched, and the moved temp stays where the
// other process put it.
func TestATempMovedByAnotherProcessIsNotClaimed(t *testing.T) {
	root := t.TempDir()
	write(t, root, filepath.Join("sub", "a.txt"), abcde)
	write(t, root, "y.txt", abcde)
	real := stageFileFn
	t.Cleanup(func() { stageFileFn = real })
	stageFileFn = func(tr *tree, path string, tx text) (staged, error) {
		sf, err := real(tr, path, tx)
		if filepath.Base(path) == "a.txt" {
			if rerr := os.Rename(filepath.Join(root, "sub"), filepath.Join(root, "sub.saved")); rerr != nil {
				t.Fatal(rerr)
			}
			if merr := os.Mkdir(filepath.Join(root, "sub"), 0o755); merr != nil {
				t.Fatal(merr)
			}
		}
		return sf, err
	}
	res, err := Apply(root, []Input{
		{Path: "sub/a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "y.txt", Start: 1, End: 1, Op: "replace", Body: []string{"Y"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	// ADR-132: the target's directory is gone since mrw read it — the
	// target's doing, so a refused hunk with no error.
	if err != nil || res.Applied || res.Failed != 1 {
		t.Fatalf("a commit whose temp was moved away was not refused as the target's: %v %+v", err, res)
	}
	if len(res.LeftBehind) != 0 {
		t.Errorf("LeftBehind = %q, want nothing: the moved temp is at a path mrw never made", res.LeftBehind)
	}
	ents, rerr := os.ReadDir(filepath.Join(root, "sub.saved"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	moved := 0
	for _, e := range ents {
		if isTemp(e.Name()) {
			moved++
		}
	}
	if moved != 1 {
		t.Errorf("sub.saved holds %d temp file(s), want the one the other process moved there", moved)
	}
}
