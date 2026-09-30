package links

import (
	"io/fs"
	"os"
	"path/filepath"
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
