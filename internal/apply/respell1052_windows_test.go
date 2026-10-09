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
// rename in its directory, "already exists". Such a name is asked through its
// extended-length path (lstatEntry), so it is compared with the source like any other.
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

// ADR-129 amendment. Every kind of absolute path has an extended spelling, and a
// relative one has none (the Codex review of #360: an ordinary UNC path was left
// on the plain Lstat, where a name ending in a dot is stripped).
func TestAnExtendedPathSpellsEveryKindOfAbsolutePath(t *testing.T) {
	for in, want := range map[string]string{
		`C:\x\dot.`:              `\\?\C:\x\dot.`,
		`\\server\share\x\dot.`:  `\\?\UNC\server\share\x\dot.`,
		`\\?\C:\x\dot.`:          `\\?\C:\x\dot.`,
		`\\?\UNC\srv\share\dot.`: `\\?\UNC\srv\share\dot.`,
		`\\.\C:\x\dot.`:          `\\?\C:\x\dot.`,
		`\\.\pipe\dot.`:          ``,
		`rel\dot.`:               ``,
		`dot.`:                   ``,
	} {
		if got := extendedPath(in); got != want {
			t.Errorf("extendedPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// A name ending in a dot beside the source, through an ordinary UNC root
// (\\localhost\C$\...), where the runner's administrative share is reachable;
// skipped where it is not. The hard-link refusal is pinned on a drive path by
// TestAnUnopenableHardLinkOfTheSourceStillRefusesACaseOnlyRename.
func TestAnOrdinaryUNCRootGetsTheSameAnswers(t *testing.T) {
	local := t.TempDir()
	vol := filepath.VolumeName(local)
	if len(vol) != 2 || vol[1] != ':' {
		t.Skip("the temp directory is not on a drive letter")
	}
	unc := `\\localhost\` + vol[:1] + `$` + local[2:]
	if _, err := os.Stat(unc); err != nil {
		t.Skipf("the administrative share is not reachable: %v", err)
	}
	write(t, unc, "plain.txt", "x\n")
	sib := `\\?\` + local + `\plain.txt.`
	if err := os.WriteFile(sib, []byte("y\n"), 0o644); err != nil {
		t.Skipf("this volume will not make a name ending in a dot: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(sib) })
	res, err := Apply(unc, []Input{{Path: "plain.txt", Op: "rename", Body: []string{"PLAIN.TXT"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied {
		t.Fatalf("through a UNC root, a sibling named plain.txt. refused a case-only rename: %s", failedReason(res))
	}
}
