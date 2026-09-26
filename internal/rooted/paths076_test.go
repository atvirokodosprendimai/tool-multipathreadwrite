package rooted

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-076 T1. A trailing separator names a directory — the OS refuses
// open("a.txt/") — and filepath.Join cleaned it away, so a read of a.txt/
// served a.txt. A directory keeps working with its separator, and a name that
// does not exist is left for the caller to report missing in its own words.
func TestAPathEndingInASeparatorMustNameADirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sep := string(filepath.Separator)
	if _, err := Resolve(root, "a.txt"+sep); !errors.Is(err, ErrNotADirectory) {
		t.Errorf("Resolve(%q) = %v, want ErrNotADirectory", "a.txt"+sep, err)
	}
	for _, p := range []string{"sub" + sep, "missing" + sep, "a.txt", "." + sep} {
		if _, err := Resolve(root, p); err != nil {
			t.Errorf("Resolve(%q) refused: %v", p, err)
		}
	}
	if !EndsInSeparator("d/") || EndsInSeparator("d") || EndsInSeparator("") {
		t.Error("EndsInSeparator does not read a trailing / as a separator")
	}
}

// ADR-076 T1. A root that is not there was judged lexically: every path under
// it was reported as resolving outside the root — to the root's parent — and a
// create under it made the root. It is named as missing instead, and a root
// that is a file is named as not a directory.
func TestARootThatDoesNotExistIsNamedAsMissing(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "nope")
	for name, call := range map[string]func() error{
		"Abs":     func() error { _, err := Abs(gone); return err },
		"Resolve": func() error { _, err := Resolve(gone, "a.txt"); return err },
	} {
		if err := call(); err == nil || !strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "outside") {
			t.Errorf("%s under a missing root: %v, want it named as not existing", name, err)
		}
	}
	file := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(file, []byte("f\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Abs(file); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("Abs of a root that is a file: %v, want it named as not a directory", err)
	}
}

// ADR-076 T2. Win32 opens these names as devices, in any case, with trailing
// spaces before an extension; Go's own rule (internal/filepathlite
// isReservedName) picks the candidates, and on Windows the OS is asked whether
// the name really opens a device, since Windows 11 reads nul.txt as a file.
// Only the last component is a candidate: a device name in the middle of a
// path cannot be created as a directory, so it fails loudly on its own.
func TestADeviceNameIsACandidateByGosRule(t *testing.T) {
	for in, want := range map[string]string{
		"NUL": "NUL", "nul.txt": "nul.txt", "con": "con", "Aux.go": "Aux.go", "PRN": "PRN",
		"COM1": "COM1", "lpt9.log": "lpt9.log", "COM\u00b9": "COM\u00b9", "LPT\u00b3.x": "LPT\u00b3.x",
		"CONIN$": "CONIN$", "conout$.txt": "conout$.txt", "NUL .txt": "NUL .txt", "nul:": "nul:",
		`sub\con`: "con", "dir/prn.md": "prn.md",
	} {
		if got := win32Device(in); got != want {
			t.Errorf("win32Device(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"console", "nullable.go", "COM10", "COM0", "LPT", "auxiliary.txt", "a/b/c.go", "", ".", "..", "CONIN", "xnul", "con/a.go", `nul\x.txt`} {
		if got := win32Device(in); got != "" {
			t.Errorf("win32Device(%q) = %q, which Windows opens as a file", in, got)
		}
	}
}
