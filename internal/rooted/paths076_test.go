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
	for p, want := range map[string]bool{"d/": true, "d/.": true, "d/..": true, ".": true, "d": false, "": false, "a.b": false, "d/a.b": false} {
		if got := SpelledAsDirectory(p); got != want {
			t.Errorf("SpelledAsDirectory(%q) = %v, want %v", p, got, want)
		}
	}
	if _, err := Resolve(root, "a.txt"+sep+"."); !errors.Is(err, ErrNotADirectory) {
		t.Errorf("Resolve(a.txt%s.) = %v, want ErrNotADirectory", sep, err)
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

// ADR-076 T2, ADR-081. Win32 opens these names as devices, in any case, with
// trailing spaces before an extension; Go's own rule (internal/filepathlite
// isReservedName) picks them, and on Windows every one is refused by name,
// whatever this build's GetFullPathName says. Only the last component is a
// candidate: a device name in the middle of a path cannot be created as a
// directory, so it fails loudly on its own.
func TestADeviceNameIsACandidateByGosRule(t *testing.T) {
	for in, want := range map[string]string{
		"NUL": "NUL", "nul.txt": "nul.txt", "con": "con", "Aux.go": "Aux.go", "PRN": "PRN",
		"COM1": "COM1", "lpt9.log": "lpt9.log", "COM\u00b9": "COM\u00b9", "LPT\u00b3.x": "LPT\u00b3.x",
		"CONIN$": "CONIN$", "conout$.txt": "conout$.txt", "NUL .txt": "NUL .txt", "nul:": "nul:",
		`sub\con`: "con", "dir/prn.md": "prn.md", "con/a.go": "con", `nul.txt\x.txt`: "nul.txt",
	} {
		if got := win32Device(in); got != want {
			t.Errorf("win32Device(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"console", "nullable.go", "COM10", "COM0", "LPT", "auxiliary.txt", "a/b/c.go", "", ".", "..", "CONIN", "xnul"} {
		if got := win32Device(in); got != "" {
			t.Errorf("win32Device(%q) = %q, which Windows opens as a file", in, got)
		}
	}
}

// Codex review of #237. The candidate check folded case with strings.ToUpper,
// whose result can be shorter than its input, and sliced it by the input's
// length: "ıı.txt" panicked. Every name is judged without a panic, and names
// that only fold to a device name in Unicode are not devices.
func TestADeviceCandidateNeverPanicsOnUnicode(t *testing.T) {
	for _, in := range []string{"\u0131\u0131.txt", "\u017f\u017f", "\u0131\u0131\u0131\u0131", "c\u00f6n", "N\u00dcL.txt", "\u212aOM1"} {
		if got := win32Device(in); got != "" {
			t.Errorf("win32Device(%q) = %q, which Windows does not open as a device", in, got)
		}
	}
	if got := win32Device("NUL/."); got != "NUL" {
		t.Errorf("win32Device(NUL/.) = %q: every component counts (ADR-081)", got)
	}
	if got := win32Device("NUL"); got != "NUL" {
		t.Errorf("win32Device(NUL) = %q", got)
	}
}
