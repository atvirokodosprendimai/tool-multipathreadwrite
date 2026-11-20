//go:build !windows

package rooted

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func realRoot727(t *testing.T) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func slurp727(t *testing.T, f *os.File) string {
	t.Helper()
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func notRoot727(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads a directory whatever its mode")
	}
}

// #727 review P2-1. os.Root opens each directory for reading, so a directory
// that may be searched but not read (chmod 111) refused a file that can be
// read. The open falls back to the name, with an identity check.
func TestAFileBelowASearchOnlyDirectoryIsStillRead(t *testing.T) {
	notRoot727(t)
	root := realRoot727(t)
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0o111); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })
	f, _, err := OpenRead(root, filepath.Join(sub, "a.txt"))
	if err != nil {
		t.Fatalf("a readable file below a search-only directory: %v", err)
	}
	if got := slurp727(t, f); got != "A\n" {
		t.Errorf("read %q", got)
	}
}

// A root at mode 0311 cannot be opened as an os.Root at all.
func TestAFileBelowARootThatCannotBeReadIsStillRead(t *testing.T) {
	notRoot727(t)
	root := realRoot727(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o311); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	f, _, err := OpenRead(root, filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatalf("a readable file in a root at 0311: %v", err)
	}
	if got := slurp727(t, f); got != "A\n" {
		t.Errorf("read %q", got)
	}
}

// A directory swapped for a link out DURING the fallback, after the by-name
// open and before the identity check, is refused, and so is one swapped for a
// link to another file of the root.
func TestASwapDuringTheByNameFallbackIsRefused(t *testing.T) {
	notRoot727(t)
	for name, target := range map[string]func(root, outside string) string{
		"outside":         func(root, outside string) string { return outside },
		"inside, another": func(root, outside string) string { return filepath.Join(root, "other") },
	} {
		root, outside := realRoot727(t), realRoot727(t)
		sub := filepath.Join(root, "sub")
		for _, d := range []string{sub, filepath.Join(root, "other"), outside} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d, "a.txt"), []byte("X\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Chmod(sub, 0o111); err != nil {
			t.Fatal(err)
		}
		aim := target(root, outside)
		prev := fallbackGapFn
		fallbackGapFn = func() {
			fallbackGapFn = prev
			_ = os.Chmod(sub, 0o755)
			if err := os.Rename(sub, sub+".old"); err != nil {
				t.Error(err)
				return
			}
			if err := os.Symlink(aim, sub); err != nil {
				t.Error(err)
			}
		}
		f, _, err := OpenRead(root, filepath.Join(sub, "a.txt"))
		fallbackGapFn = prev
		_ = os.Chmod(sub+".old", 0o755)
		if err == nil {
			_ = f.Close()
			t.Errorf("%s: a swap during the fallback was not refused", name)
		}
	}
}

// #727 review P2-2. A link at the end of the path that resolves to nothing is a
// file that does not exist, not "path escapes from parent", whether it points
// inside the root by an absolute or a relative name, or outside it.
func TestADanglingLinkIsAMissingFileNotAnEscape(t *testing.T) {
	root, outside := realRoot727(t), realRoot727(t)
	for name, target := range map[string]string{
		"relative": "nowhere.txt",
		"abs-in":   filepath.Join(root, "nowhere.txt"),
		"abs-out":  filepath.Join(outside, "nowhere.txt"),
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
		_, _, err := OpenRead(root, filepath.Join(root, name))
		if !errors.Is(err, fs.ErrNotExist) || (err != nil && strings.Contains(err.Error(), "escapes")) {
			t.Errorf("%s: %v, want a file that does not exist", name, err)
		}
	}
}

// #727 review P3-5. mrw's own state is refused after resolution too (ADR-077):
// the name was judged once and may have changed.
func TestOpenReadRefusesMrwsOwnState(t *testing.T) {
	root := realRoot727(t)
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	if err := os.MkdirAll(filepath.Join(root, ".st", "mrw", "k"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{".st/mrw/k/seen": "ledger\n", ".st/notes.txt": "notes\n"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if f, _, err := OpenRead(root, filepath.Join(root, ".st", "mrw", "k", "seen")); !errors.Is(err, ErrOwnState) {
		if f != nil {
			_ = f.Close()
		}
		t.Errorf("mrw's ledger: %v, want ErrOwnState", err)
	}
	f, _, err := OpenRead(root, filepath.Join(root, ".st", "notes.txt"))
	if err != nil {
		t.Fatalf("a sibling of mrw's state: %v", err)
	}
	_ = f.Close()
}

// #727 review P2-3. A walk reader holds the directories it reads from, up to a
// bound: files in more directories than that, read together, all open, and the
// handles are released by Close.
func TestAWalkReaderOpensFilesInMoreDirectoriesThanItKeepsOpen(t *testing.T) {
	root := realRoot727(t)
	const dirs = maxDirRoots + 20
	for i := 0; i < dirs; i++ {
		d := filepath.Join(root, "d"+string(rune('A'+i%26))+string(rune('a'+i/26)), "sub")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "f.txt"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rd, err := NewWalkReader(root)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error)
	for round := 0; round < 3; round++ {
		for i := 0; i < dirs; i++ {
			go func() {
				p := filepath.Join(root, "d"+string(rune('A'+i%26))+string(rune('a'+i/26)), "sub", "f.txt")
				f, _, err := rd.Open(p)
				if err == nil {
					_ = f.Close()
				}
				done <- err
			}()
		}
		for i := 0; i < dirs; i++ {
			if err := <-done; err != nil {
				t.Errorf("a read in round %d: %v", round, err)
			}
		}
	}
	rd.Close()
	if len(rd.dirs) != 0 {
		t.Errorf("%d directory handles are still held after Close", len(rd.dirs))
	}
}
