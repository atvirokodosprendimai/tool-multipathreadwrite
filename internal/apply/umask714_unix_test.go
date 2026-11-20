//go:build !windows

package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// #714. Staging made a new file 0644 whatever the umask, so under umask 077 a
// create was readable by everyone. A new file is 0666 &^ umask, the umask read
// when the process starts; a replacement keeps its target's mode. The child
// is this test binary started under `umask 077`, so the umask is the process's
// own and not a value the test hands in.
func TestANewFileHonoursTheUmaskAndAReplacementKeepsItsMode(t *testing.T) {
	if dir := os.Getenv("MRW_UMASK714_DIR"); dir != "" {
		res, err := Apply(dir, []Input{
			{Path: "new.txt", Op: "create", Body: []string{"n"}, Lines: -1, Index: 0},
			{Path: "old.txt", Op: "replace", Start: 1, End: 1, Body: []string{"o2"}, Lines: -1, Index: 1},
		}, Options{Force: true})
		if err != nil || !res.Applied {
			t.Fatalf("the child's write did not land: %v %+v", err, res.Hunks)
		}
		return
	}
	dir := t.TempDir()
	old := filepath.Join(dir, "old.txt")
	if err := os.WriteFile(old, []byte("o\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(old, 0o640); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", `umask 077; exec "$0" -test.run '^TestANewFileHonoursTheUmaskAndAReplacementKeepsItsMode$'`, os.Args[0])
	cmd.Env = append(os.Environ(), "MRW_UMASK714_DIR="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the child failed: %v\n%s", err, out)
	}
	for name, want := range map[string]os.FileMode{"new.txt": 0o600, "old.txt": 0o640} {
		fi, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if got := fi.Mode().Perm(); got != want {
			t.Errorf("%s is %04o, want %04o", name, got, want)
		}
	}
}
