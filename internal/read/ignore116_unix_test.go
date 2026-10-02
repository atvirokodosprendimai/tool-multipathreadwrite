//go:build unix

package read

import (
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"
)

// TestAnIgnoreFileThatIsNotRegularIsNotWaitedOn: a .gitignore that is a FIFO,
// or a link to a device, gives no rules and the walk answers at once.
func TestAnIgnoreFileThatIsNotRegularIsNotWaitedOn(t *testing.T) {
	for name, plant := range map[string]func(p string) error{
		"fifo":      func(p string) error { return syscall.Mkfifo(p, 0o644) },
		"/dev/zero": func(p string) error { return os.Symlink("/dev/zero", p) },
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			for _, d := range []string{".git/info", "d"} {
				if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(root, "d", "a.txt"), []byte("NEEDLE\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, p := range []string{filepath.Join(root, "d", ".gitignore"), filepath.Join(root, ".git", "info", "exclude")} {
				if err := plant(p); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan []Spec, 1)
			go func() {
				specs, _, _ := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("NEEDLE")})
				done <- specs
			}()
			select {
			case specs := <-done:
				if len(specs) != 1 {
					t.Errorf("the walk served %v, want d/a.txt alone", specs)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("the walk waited on an ignore file that is not a regular file")
			}
		})
	}
}

// TestALinkedGitignoreIsNotFollowed: a .gitignore that is a link is not read,
// as git does not read one. Followed, a link out of the root let a file mrw
// refuses to serve decide what it serves, and the count told what it held.
func TestALinkedGitignoreIsNotFollowed(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "rules")
	if err := os.WriteFile(outside, []byte("needle.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, d := range []string{".git", "sub"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "needle.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "sub", ".gitignore")); err != nil {
		t.Fatal(err)
	}
	var sk WalkSkipped
	specs, _, _ := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk})
	if len(specs) != 1 || specs[0].Path != "sub/needle.txt" || sk != (WalkSkipped{}) {
		t.Errorf("served %v skipped %+v, want sub/needle.txt and nothing skipped", specs, sk)
	}
}
