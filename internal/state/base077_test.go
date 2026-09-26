package state

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-077. The boundary asks on every resolve whether a path is inside mrw's
// state, so naming the base must not make it; Dir still makes its own
// directory under the same base.
func TestBaseDoesNotCreateTheStateDirectory(t *testing.T) {
	home := filepath.Join(t.TempDir(), "x")
	t.Setenv("XDG_STATE_HOME", home)
	base, err := Base()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "mrw"); base != want {
		t.Fatalf("Base() = %q, want %q", base, want)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("Base() made %s (stat: %v)", base, err)
	}
	dir, err := Dir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(dir) != base {
		t.Errorf("Dir = %s, which is not directly under Base %s", dir, base)
	}
}
