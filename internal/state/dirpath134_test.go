package state

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-134. DirPath names the directory Dir would make, and makes nothing.
func TestDirPathIsDirWithoutMakingIt(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	want, err := DirPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(want); err == nil {
		t.Fatalf("DirPath made %s", want)
	}
	got, err := Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Errorf("Dir = %s, DirPath = %s, want the same directory", got, want)
	}
}
