package rooted

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// noSys is a FileInfo whose identity cannot be read: Sys is nil on unix, and
// on Windows the path it is judged by does not exist.
type noSys struct{}

func (noSys) Name() string       { return "x" }
func (noSys) Size() int64        { return 0 }
func (noSys) Mode() os.FileMode  { return 0 }
func (noSys) ModTime() time.Time { return time.Time{} }
func (noSys) IsDir() bool        { return false }
func (noSys) Sys() any           { return nil }

// ADR-196 T7, from the Windows field test of v1.86.4: `write C:\$MFT` was
// refused as "inside mrw's own state directory, or a hard link to a file in
// it". The refusal is right (ADR-134 Decision 4: a comparison that cannot be
// completed refuses), but neither reason was true: Windows would not open $MFT
// to read its identity. The refusal names that, and the CLI prints the reason
// the check found rather than one sentence for all of them.
func TestARefusalForOwnStateSaysWhichReasonHeld(t *testing.T) {
	b := t.TempDir()
	bi, err := os.Stat(b)
	if err != nil {
		t.Fatal(err)
	}
	s := stateLinks{root: t.TempDir(), b: b, bi: bi}
	err = s.refusal(noSys{}, filepath.Join(t.TempDir(), "gone"), "X")
	if err == nil || !strings.Contains(err.Error(), "cannot read the file identity of X") || strings.Contains(err.Error(), "more than one name") || !errors.Is(err, ErrOwnState) {
		t.Errorf("an identity that cannot be read: %v, want it named as such", err)
	}

	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", home)
	inState := filepath.Join(home, "mrw", "x", "seen")
	if err := os.MkdirAll(filepath.Dir(inState), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inState, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := OwnStateRefusal(root, inState, inState); err == nil || !strings.Contains(err.Error(), "is inside mrw's own state directory") || strings.Contains(err.Error(), "hard link") {
		t.Errorf("a file in the state directory: %v, want it named as inside, not as a link", err)
	}
	plain := filepath.Join(root, "p.mrw")
	if err := os.WriteFile(plain, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := OwnStateRefusal(root, plain, plain); err != nil {
		t.Errorf("a plain file is refused: %v", err)
	}
}
