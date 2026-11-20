//go:build !windows

package apply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-210 final pass. The directory syncs open what staging recorded, and refuse
// what they find if it is not that directory: a directory swapped after the
// renames for a FIFO (which would block the open under the write lock), a regular
// file (which would be synced as if it were the directory) or another directory
// is a landed write with a sync error naming it, never a sync of the wrong object.
func TestADirectorySwappedAfterTheRenamesIsNotSynced(t *testing.T) {
	swaps := map[string]func(t *testing.T, dir string){
		"fifo": func(t *testing.T, dir string) {
			if err := syscall.Mkfifo(dir, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"regular_file": func(t *testing.T, dir string) {
			if err := os.WriteFile(dir, []byte("not a directory"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"another_directory": func(t *testing.T, dir string) {
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, swap := range swaps {
		t.Run(name, func(t *testing.T) {
			root := evalOr(t.TempDir())
			rs, rd, rb := SyncsDirs, SyncDirFn, beforeDirSync
			t.Cleanup(func() { SyncsDirs, SyncDirFn, beforeDirSync = rs, rd, rb })
			SyncsDirs = true
			synced := 0
			SyncDirFn = func(f *os.File) error { synced++; return rd(f) }
			beforeDirSync = func() {
				moved := filepath.Join(root, "n.moved")
				if err := os.Rename(filepath.Join(root, "n"), moved); err != nil {
					t.Fatal(err)
				}
				swap(t, filepath.Join(root, "n"))
			}
			type out struct {
				res Result
				err error
			}
			done := make(chan out, 1)
			go func() {
				res, err := Apply(root, []Input{{Path: "n/k.txt", Op: "create", Body: []string{"made"}, Lines: -1}}, Options{Seen: map[string]Seen{}})
				done <- out{res, err}
			}()
			select {
			case o := <-done:
				var se *SyncError
				if !o.res.Applied || !errors.As(o.err, &se) || se.Path != "n" {
					t.Fatalf("applied=%v err=%v; want a landed write with a sync error naming n", o.res.Applied, o.err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("the directory sync blocked on the swapped path")
			}
		})
	}
}

// ADR-210 final pass. A rename through a symlinked directory syncs the directory
// it resolved to when the write was staged: a link retargeted after the renames
// does not send the sync to another directory.
func TestARetargetedLinkDoesNotMoveTheDirectorySync(t *testing.T) {
	root := evalOr(t.TempDir())
	for _, d := range []string{"s1", "s2", "t"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "s1", "x.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("s1", filepath.Join(root, "lnk")); err != nil {
		t.Fatal(err)
	}
	rs, rd, rb := SyncsDirs, SyncDirFn, beforeDirSync
	t.Cleanup(func() { SyncsDirs, SyncDirFn, beforeDirSync = rs, rd, rb })
	SyncsDirs = true
	var dirs []string
	SyncDirFn = func(f *os.File) error {
		rel, _ := filepath.Rel(root, f.Name())
		dirs = append(dirs, filepath.ToSlash(rel))
		return rd(f)
	}
	beforeDirSync = func() {
		if err := os.Remove(filepath.Join(root, "lnk")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("s2", filepath.Join(root, "lnk")); err != nil {
			t.Fatal(err)
		}
	}
	ledger := map[string]Seen{"lnk/x.txt": {SHA: seen.SHA([]byte("x\n"))}}
	res, err := Apply(root, []Input{{Path: "lnk/x.txt", Op: "rename", Body: []string{"t/x.txt"}, Lines: -1}}, Options{Seen: ledger})
	if err != nil || !res.Applied {
		t.Fatalf("applied=%v err=%v", res.Applied, err)
	}
	if got := strings.Join(dirs, " "); got != "s1 t" {
		t.Errorf("synced %q, want the directories the write changed, s1 and t", got)
	}
}
