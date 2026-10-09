//go:build windows

package apply

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// ADR-129 amendment (the Windows chaos round of 2026-10-09). respelling asked
// the OS about every entry of the directory, and Win32 cannot open a name that
// ends in a dot or a space: the call failed, and the failure was read as "the
// destination is another entry", so one such sibling refused every case-only
// rename in its directory, "already exists". Such a name cannot be the
// source's hard link either, so it is passed over.
func TestAnUnopenableSiblingDoesNotRefuseACaseOnlyRename(t *testing.T) {
	root := t.TempDir()
	write(t, root, "plain.txt", "x\n")
	for _, name := range []string{"dot.", "sp ", "plain.txt."} {
		p := `\\?\` + filepath.Join(root, name)
		if err := os.WriteFile(p, []byte("y\n"), 0o644); err != nil {
			t.Skipf("this volume will not make %q: %v", name, err)
		}
		t.Cleanup(func() { _ = os.Remove(p) }) // runs before TempDir's RemoveAll, which cannot open the name
	}
	res, err := Apply(root, []Input{{Path: "plain.txt", Op: "rename", Body: []string{"PLAIN.TXT"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied {
		t.Fatalf("a case-only rename beside names ending in a dot or a space was refused: %s", failedReason(res))
	}
	if got := names(t, root); !slices.Contains(got, "PLAIN.TXT") || slices.Contains(got, "plain.txt") {
		t.Fatalf("the directory holds %v, want PLAIN.TXT and not plain.txt", got)
	}
}

// ADR-129 Decision 1 still holds beside such a name. A hard link to the source
// that is itself NAMED with a trailing dot can be made through an extended-length
// path, so an unopenable entry cannot simply be passed over: the case-only rename
// stays refused, as it is for a hard link under an ordinary name (the Codex
// review of #360).
func TestAnUnopenableHardLinkOfTheSourceStillRefusesACaseOnlyRename(t *testing.T) {
	root := t.TempDir()
	write(t, root, "plain.txt", "x\n")
	alias := `\\?\` + filepath.Join(root, "dot.")
	if err := os.Link(filepath.Join(root, "plain.txt"), alias); err != nil {
		t.Skipf("this volume will not link a name ending in a dot: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(alias) }) // runs before TempDir's RemoveAll, which cannot open the name
	res, err := Apply(root, []Input{{Path: "plain.txt", Op: "rename", Body: []string{"PLAIN.TXT"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatalf("a case-only rename of a file that has a second hard link, named with a trailing dot, was applied")
	}
	if got := names(t, root); !slices.Contains(got, "plain.txt") {
		t.Fatalf("the directory holds %v, want plain.txt untouched", got)
	}
}
