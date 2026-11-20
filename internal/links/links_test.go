package links

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

type info struct{ mode os.FileMode }

func (i info) Name() string       { return "" }
func (i info) Size() int64        { return 0 }
func (i info) Mode() os.FileMode  { return i.mode }
func (i info) ModTime() time.Time { return time.Time{} }
func (i info) IsDir() bool        { return i.mode.IsDir() }
func (i info) Sys() any           { return nil }

// ADR-103 T2. The state key resolved a root with EvalSymlinks alone, which
// stops at a Windows junction, while the boundary walked through it: one
// checkout, two state directories, two writer locks. Real is now the one
// canonicalisation both use; with the walk on it follows a junction (a Go 1.23+
// junction is ModeIrregular, which EvalSymlinks does not follow), and with it
// off it leaves the spelling to EvalSymlinks.
func TestRealFollowsAJunctionWhenTheWalkIsOn(t *testing.T) {
	junction, target := filepath.FromSlash("/r/j"), filepath.FromSlash("/t")
	lfs := FS{
		Lstat: func(p string) (os.FileInfo, error) {
			switch filepath.Clean(p) {
			case filepath.FromSlash("/r"), target:
				return info{os.ModeDir}, nil
			case junction:
				return info{os.ModeIrregular}, nil
			}
			return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
		},
		Readlink: func(p string) (string, error) {
			if filepath.Clean(p) == junction {
				return target, nil
			}
			return "", &fs.PathError{Op: "readlink", Path: p, Err: fs.ErrInvalid}
		},
	}
	in := filepath.Join(junction, "repo")
	if got, want := realVia(in, lfs, true), filepath.Join(target, "repo"); got != want {
		t.Errorf("realVia(%s) with the walk on = %s, want %s", in, got, want)
	}
	if got := realVia(in, lfs, false); got != in {
		t.Errorf("realVia(%s) with the walk off = %s, want it left as spelled", in, got)
	}
}

// lstatAnswering is a filesystem whose root is a directory and whose
// /r/x answers Lstat with err; nothing else exists.
func lstatAnswering(err error) FS {
	return FS{
		Lstat: func(p string) (os.FileInfo, error) {
			switch filepath.Clean(p) {
			case filepath.FromSlash("/r"):
				return info{os.ModeDir}, nil
			case filepath.FromSlash("/r/x"):
				return nil, &fs.PathError{Op: "lstat", Path: p, Err: err}
			}
			return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
		},
		Readlink: func(p string) (string, error) {
			return "", &fs.PathError{Op: "readlink", Path: p, Err: fs.ErrInvalid}
		},
	}
}

// Issue #711 (ADR-071). A component whose Lstat fails for a reason that says
// nothing about whether it exists, an I/O error say, may be a junction: ending
// the walk there judged the path by its spelling while os.Open still followed
// it. Only a missing component or a name that cannot exist ends the walk; any
// other failure is refused.
func TestTheLinkWalkRefusesAnLstatFailureThatIsNeitherAbsenceNorAnInvalidName(t *testing.T) {
	in := filepath.FromSlash("/r/x/y")
	for name, err := range map[string]error{"EIO": syscall.EIO, "EINVAL": syscall.EINVAL, "ErrInvalid": fs.ErrInvalid} {
		got, gerr := Through(in, lstatAnswering(err))
		if gerr == nil {
			t.Errorf("%s: Through(%s) = %s with no error, want a refusal", name, in, got)
		}
	}
}

// The walk still ends where the component is simply not there, or cannot be:
// a create names a file that does not exist yet.
func TestTheLinkWalkEndsAtAnAbsentComponentOrAnInvalidName(t *testing.T) {
	in := filepath.FromSlash("/r/x/y")
	for name, err := range map[string]error{
		"not exist":    fs.ErrNotExist,
		"invalid name": invalidName(),
	} {
		got, gerr := Through(in, lstatAnswering(err))
		if gerr != nil || got != in {
			t.Errorf("%s: Through(%s) = %s, %v, want the path kept and no error", name, in, got, gerr)
		}
	}
}

// #711 review, P3-5. A failure beyond a junction must not drop the junction:
// the fallbacks (Real, and rooted's RealAsFarAsItExists) judged the fully
// spelled path, which put an escape through /r/j back inside the root.
func TestARefusedWalkReturnsNoPathSoTheFallbackKeepsTheSpelling(t *testing.T) {
	junction, out, x := filepath.FromSlash("/r/j"), filepath.FromSlash("/out"), filepath.FromSlash("/out/x")
	lfs := FS{
		Lstat: func(p string) (os.FileInfo, error) {
			switch filepath.Clean(p) {
			case filepath.FromSlash("/r"), out:
				return info{os.ModeDir}, nil
			case junction:
				return info{os.ModeIrregular}, nil
			case x:
				return nil, &fs.PathError{Op: "lstat", Path: p, Err: syscall.EIO}
			}
			return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
		},
		Readlink: func(p string) (string, error) {
			if filepath.Clean(p) == junction {
				return out, nil
			}
			return "", &fs.PathError{Op: "readlink", Path: p, Err: fs.ErrInvalid}
		},
	}
	in := filepath.FromSlash("/r/j/x/y")
	if got, err := Through(in, lfs); err == nil || got != "" {
		t.Errorf("Through(%s) = %q, %v, want no path and the refusal", in, got, err)
	}
	if got := realVia(in, lfs, true); got != in {
		t.Errorf("realVia(%s) = %s, want the path as spelled, which a later Resolve refuses", in, got)
	}
	// A junction that leads to itself: a partial result dropped the junction
	// and named /r/note.txt, a different file a later Resolve could approve.
	loop := filepath.FromSlash("/r/loop")
	self := FS{
		Lstat: func(p string) (os.FileInfo, error) {
			switch filepath.Clean(p) {
			case filepath.FromSlash("/r"):
				return info{os.ModeDir}, nil
			case loop:
				return info{os.ModeIrregular}, nil
			}
			return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
		},
		Readlink: func(p string) (string, error) { return loop, nil },
	}
	note := filepath.FromSlash("/r/loop/note.txt")
	if got, err := Through(note, self); err == nil || got != "" {
		t.Errorf("Through(%s) through a looping junction = %q, %v, want no path and the refusal", note, got, err)
	}
	if got := realVia(note, self, true); got != note {
		t.Errorf("realVia(%s) = %s, want the path as spelled", note, got)
	}
}

// invalidName is an Lstat error for a name the system refuses, as this
// platform words it: Win32 has its own errnos, and the Go-invented ones never
// come from a Windows Lstat.
func invalidName() error {
	if runtime.GOOS == "windows" {
		return syscall.Errno(123)
	}
	return syscall.ENAMETOOLONG
}

// P3-6. The Windows errno list is data every platform can test: each listed
// errno names an invalid name and ends the walk, and ERROR_NOT_READY, an
// I/O-class failure, does not.
func TestEveryWindowsInvalidNameErrnoEndsTheWalkAndNotReadyDoesNot(t *testing.T) {
	for _, e := range []syscall.Errno{123, 161, 206, 267} {
		if !anyErrno(&fs.PathError{Op: "lstat", Path: "x", Err: e}, windowsInvalidNames) {
			t.Errorf("errno %d is not in the list of invalid names", e)
		}
	}
	if anyErrno(&fs.PathError{Op: "lstat", Path: "x", Err: syscall.Errno(21)}, windowsInvalidNames) {
		t.Error("ERROR_NOT_READY (21) was taken for an invalid name")
	}
}
