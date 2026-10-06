package apply

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// plan132 edits a.txt, then does last: a refusal of last must leave a.txt as
// it was and skip its hunk.
func plan132(last Input) []Input {
	last.Index = 1
	return []Input{{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0}, last}
}

func edit132(path string) Input {
	return Input{Path: path, Start: 1, End: 1, Op: "replace", Body: []string{"B"}, Lines: -1}
}

// ADR-132. A refusal found before the first rename whose cause is the target's
// — held, a permission, a name refused, changed or removed since mrw read it, a
// destination under a link to nothing — went through abortStage and returned
// an error: exit 2, while the same causes found at validation were exit 1. It
// is a hunk refusal now: no error, one failed hunk, the rest skipped, nothing
// written.
func TestATargetsStateRefusesItsHunkAndReturnsNoError(t *testing.T) {
	realStage, realProbe, realReplaceable := stageFileFn, probeNameFn, replaceableFn
	t.Cleanup(func() { stageFileFn, probeNameFn, replaceableFn = realStage, realProbe, realReplaceable })
	afterStaging := func(root string, do func(string) error) {
		stageFileFn = func(tr *tree, path string, tx text) (staged, error) {
			sf, err := realStage(tr, path, tx)
			if err == nil && filepath.Base(path) == "b.txt" {
				if derr := do(path); derr != nil {
					return sf, derr
				}
			}
			return sf, err
		}
	}
	for _, tc := range []struct {
		name string
		arm  func(t *testing.T, root string)
		last Input
	}{
		{"changed since read", func(t *testing.T, root string) {
			afterStaging(root, func(p string) error { return os.WriteFile(p, []byte("b\nmore\n"), 0o644) })
		}, edit132("b.txt")},
		{"removed since read", func(t *testing.T, root string) {
			afterStaging(root, os.Remove)
		}, edit132("b.txt")},
		{"a name the filesystem refuses", func(t *testing.T, root string) {
			probeNameFn = func(tr *tree, p string) error {
				if strings.HasSuffix(p, "new.txt") {
					return fmt.Errorf("the filesystem will not create this name: %w", &fs.PathError{Op: "open", Path: p, Err: syscall.EACCES})
				}
				return realProbe(tr, p)
			}
		}, Input{Path: "new.txt", Op: "create", Body: []string{"x"}, Lines: -1}},
		{"a target that cannot be replaced", func(t *testing.T, root string) {
			replaceableFn = func(tr *tree, p string) error {
				if filepath.Base(p) == "b.txt" {
					return &fs.PathError{Op: "open", Path: p, Err: fs.ErrPermission}
				}
				return nil
			}
		}, edit132("b.txt")},
		{"a destination under a link to nothing", func(t *testing.T, root string) {
			if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "dl")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
		}, Input{Path: "b.txt", Op: "rename", Body: []string{"dl/x.txt"}, Lines: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stageFileFn, probeNameFn, replaceableFn = realStage, realProbe, realReplaceable
			root := t.TempDir()
			write(t, root, "a.txt", "a\n")
			write(t, root, "b.txt", "b\n")
			tc.arm(t, root)
			res, err := Apply(root, plan132(tc.last), Options{Force: true})
			if err != nil {
				t.Fatalf("a target-caused refusal returned an error (exit 2): %v", err)
			}
			if res.Applied || res.Failed != 1 || res.Hunks[0].Status != StatusSkipped || res.Hunks[1].Status != StatusFailed {
				t.Fatalf("want one failed hunk and its sibling skipped: applied=%v failed=%d hunks=%+v", res.Applied, res.Failed, res.Hunks)
			}
			if got := read(t, root, "a.txt"); got != "a\n" {
				t.Fatalf("a.txt = %q: the sibling landed beside a refusal", got)
			}
		})
	}
}

// ADR-132 Decision 2. What the target cannot explain stays an error, exit 2:
// a full disk, an I/O error, a read-only filesystem, too many open files, an
// error nothing names, and a probe that could not be removed (ADR-105).
func TestAnEnvironmentFailureBeforeTheFirstRenameStaysAnError(t *testing.T) {
	realProbe, realReplaceable := probeNameFn, replaceableFn
	t.Cleanup(func() { probeNameFn, replaceableFn = realProbe, realReplaceable })
	errs := map[string]error{
		"ENOSPC":   syscall.ENOSPC,
		"EIO":      syscall.EIO,
		"EROFS":    syscall.EROFS,
		"EMFILE":   syscall.EMFILE,
		"unnamed":  errors.New("something nobody anticipated"),
		"leftover": &probeLeftError{err: &fs.PathError{Op: "remove", Path: "probe", Err: syscall.EACCES}},
	}
	for name, e := range errs {
		for _, site := range []string{"probe", "replaceable"} {
			t.Run(name+" from "+site, func(t *testing.T) {
				probeNameFn, replaceableFn = realProbe, realReplaceable
				last := edit132("b.txt")
				wrapped := fmt.Errorf("wrapped: %w", e)
				if site == "probe" {
					last = Input{Path: "new.txt", Op: "create", Body: []string{"x"}, Lines: -1}
					probeNameFn = func(*tree, string) error { return wrapped }
				} else {
					if name == "leftover" {
						t.Skip("a probe left behind comes only from the probe")
					}
					replaceableFn = func(_ *tree, p string) error {
						if filepath.Base(p) == "b.txt" {
							return wrapped
						}
						return nil
					}
				}
				root := t.TempDir()
				write(t, root, "a.txt", "a\n")
				write(t, root, "b.txt", "b\n")
				res, err := Apply(root, plan132(last), Options{Force: true})
				if err == nil {
					t.Fatalf("%s at staging was turned into a hunk refusal (exit 1): %+v", name, res.Hunks)
				}
				if res.Applied || read(t, root, "a.txt") != "a\n" {
					t.Fatal("something was written beside an environment failure")
				}
			})
		}
	}
}

// The Codex review of ADR-132. changedSince's stat can fail for the target's
// doing — the file removed, a permission withdrawn — or the system's; a check
// that took every failed stat for a detected change made an I/O error exit 1.
func TestAStatThatFailsAfterReadingIsClassifiedByItsCause(t *testing.T) {
	realStage, realStat := stageFileFn, statFn
	t.Cleanup(func() { stageFileFn, statFn = realStage, realStat })
	for _, tc := range []struct {
		name   string
		err    error
		target bool
	}{
		{"removed", fs.ErrNotExist, true},
		{"a permission withdrawn", syscall.EACCES, true},
		{"an I/O error", syscall.EIO, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "a.txt", "a\n")
			write(t, root, "b.txt", "b\n")
			statFn = func(p string) (os.FileInfo, error) {
				if filepath.Base(p) == "b.txt" {
					return nil, &fs.PathError{Op: "stat", Path: p, Err: tc.err}
				}
				return realStat(p)
			}
			res, err := Apply(root, plan132(edit132("b.txt")), Options{Force: true})
			if tc.target != (err == nil) {
				t.Fatalf("a failed stat (%s): err=%v, want an error %v", tc.name, err, !tc.target)
			}
			if res.Applied || res.Failed != 1 || read(t, root, "a.txt") != "a\n" {
				t.Fatalf("want one failed hunk and nothing written: %+v", res)
			}
		})
	}
}
