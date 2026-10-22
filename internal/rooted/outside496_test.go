package rooted

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Issue #496. A path that leaves the root through ../ or a link was refused in
// words only, so the read could not tell the caller where the file really is;
// the refusal is now a typed *OutsideError carrying that, with the same text.
// The pair: a path inside the root is not one, and a leaf that is not there
// says so (Exists), since the directory above it is all that was resolved.
func TestAnEscapeFromTheRootIsTypedAndKeepsItsWords(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "root")
	other := filepath.Join(parent, "other")
	for _, d := range []string{root, other} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(other, "hosts"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	absRoot, err := Abs(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Resolve(root, filepath.Join("..", "other", "hosts"))
	var oe *OutsideError
	if !errors.As(err, &oe) {
		t.Fatalf("a path out of the root by ../ is not an *OutsideError: %v", err)
	}
	want := filepath.Join("..", "other", "hosts") + " resolves to " + filepath.Join(other, "hosts") + ", which is outside the root " + absRoot
	if err.Error() != want {
		t.Errorf("the words changed:\n got %q\nwant %q", err.Error(), want)
	}
	if oe.Resolved != filepath.Join(other, "hosts") || oe.Root != absRoot || !oe.Exists {
		t.Errorf("OutsideError = %+v; want the real path of the file, the root, and Exists", oe)
	}

	_, err = Resolve(root, filepath.Join("..", "other", "missing"))
	if !errors.As(err, &oe) || oe.Exists {
		t.Errorf("a leaf that is not there: %v / %+v; want an OutsideError with Exists false", err, oe)
	}

	if _, err := Resolve(root, "inside.txt"); errors.As(err, &oe) {
		t.Errorf("a path inside the root is an OutsideError: %v", err)
	}
}
