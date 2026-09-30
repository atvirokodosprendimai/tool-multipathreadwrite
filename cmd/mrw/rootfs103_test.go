package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// ADR-103 T1. With the filesystem root as the root, every path was refused as
// outside it. A file beneath it, named root-relative, is served.
func TestAReadUnderTheFilesystemRootIsServed(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"f.txt": "beneath the filesystem root\n"})
	real, err := filepath.EvalSymlinks(filepath.Join(dir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.VolumeName(real) + string(filepath.Separator)
	out, code := runIn(t, root, "read", strings.TrimPrefix(real, root))
	if code != 0 || !strings.Contains(out, "beneath the filesystem root") {
		t.Errorf("read under %s: exit %d:\n%s", root, code, out)
	}
}
