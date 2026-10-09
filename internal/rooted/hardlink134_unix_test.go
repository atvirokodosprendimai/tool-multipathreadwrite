//go:build unix

package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Codex review of #361. A comparison that could not read the whole state
// base answered "not a state file" for the part it could not read, so a
// directory made unlistable removed a ledger from the comparison and left its
// alias served. A file with a second name is then refused saying so; a file
// with one name is judged as before.
func TestAnIncompleteScanOfTheStateRefusesAFileWithASecondName(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root lists any directory")
	}
	root, ledger := hardLinkRoot(t)
	locked := filepath.Join(filepath.Dir(ledger), "locked")
	if err := os.MkdirAll(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if err := os.WriteFile(filepath.Join(root, "single.txt"), []byte("one name\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(root)
	for who, resolve := range map[string]func(string) (string, error){
		"Resolve":          func(p string) (string, error) { return Resolve(root, p) },
		"Resolver.Resolve": r.Resolve,
	} {
		for _, p := range []string{"hl.txt", "ord2.txt"} {
			if _, err := resolve(p); err == nil || !strings.Contains(err.Error(), "cannot tell") {
				t.Errorf("%s(%s) with an unlistable state directory = %v, want it refused as not comparable", who, p, err)
			}
		}
		if _, err := resolve("single.txt"); err != nil {
			t.Errorf("%s(single.txt), a file with one name, was refused: %v", who, err)
		}
	}
}
