package apply

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// tree701 lists every entry under root, so a refused plan can be shown to have
// left the root as it was, made directories included.
func tree701(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func link701(t *testing.T, root, target, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("a symlink needs privilege on Windows")
	}
	if err := os.Symlink(filepath.FromSlash(target), filepath.Join(root, filepath.FromSlash(name))); err != nil {
		t.Fatal(err)
	}
}

// refusedAtStaging runs the plan for real. Where a path goes through a link
// is the filesystem's to say, so validation (and a dry run) may pass; staging
// asks again, and the plan stops with nothing written, one refusal and no
// error, siblings skipped (#701 review, ADR-132).
func refusedAtStaging(t *testing.T, root string, in []Input, want string) {
	t.Helper()
	before := tree701(t, root)
	res, err := Apply(root, in, Options{Force: true})
	if err != nil {
		t.Fatalf("the plan stopped with an error, want a refusal: %v", err)
	}
	if res.Applied || res.Failed != 1 {
		t.Fatalf("applied %v failed %d, want one refusal: %+v", res.Applied, res.Failed, res.Hunks)
	}
	var named bool
	for _, h := range res.Hunks {
		switch h.Status {
		case StatusFailed:
			named = strings.Contains(h.Reason, want) && strings.Contains(h.Reason, "nothing was written")
		case StatusOK:
			t.Errorf("hunk %s reports ok beside a refusal", h.Path)
		}
	}
	if !named {
		t.Errorf("no failed hunk says %q: %+v", want, res.Hunks)
	}
	if after := tree701(t, root); strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Errorf("the refused plan changed the root:\nbefore %v\nafter  %v", before, after)
	}
}

// Codex P1 on #724: ln -s d l with d absent. d/y makes d, l/x is then d/x, and
// d/x/z makes d/x a directory. Staging's own directory is what gives it away.
func TestADanglingLinkThatStagingTurnsIntoAPathUnderAnotherCreateIsRefused(t *testing.T) {
	root := t.TempDir()
	link701(t, root, "d", "l")
	refusedAtStaging(t, root, []Input{create701("d/y"), create701("l/x"), create701("d/x/z")}, "exists now")
}

// Codex's second round: a dangling target holding `..`. a/jump -> ../b/deep,
// link -> a/jump/../new, so link/x is b/new/x once b/new is made; a key worked
// out beforehand cleaned `jump/..` away and put it under a/new.
func TestADanglingTargetWithDotDotIsJudgedByTheFilesystemNotPredicted(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"a", "b"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link701(t, root, "../b/deep", "a/jump")
	link701(t, root, "a/jump/../new", "link")
	// b/deep is absent, so `..` after a/jump cannot be walked and link stays a
	// dangling link: staging stops with nothing written. The same when the
	// plan's first create makes b/deep: link is then b/new, which is not made
	// until link/x is staged.
	for _, first := range []string{"b/new/seed", "b/deep/seed"} {
		before := tree701(t, root)
		res, _ := Apply(root, []Input{create701(first), create701("link/x"), create701("b/new/x")}, Options{Force: true})
		if res.Applied {
			t.Fatalf("%s: the plan applied: %+v", first, res.Hunks)
		}
		if after := tree701(t, root); strings.Join(after, "\n") != strings.Join(before, "\n") {
			t.Errorf("%s: the plan changed the root:\nbefore %v\nafter  %v", first, before, after)
		}
	}
}

// A create and a rename to one new place, spelled d/x and l/x with l -> d.
func TestACreateAndARenameToOneNewPlaceThroughALinkAreRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	link701(t, root, "d", "l")
	write(t, root, "src", "s\n")
	refusedAtStaging(t, root, []Input{
		create701("d/x"),
		{Path: "src", Op: "rename", Body: []string{"l/x"}, Lines: -1},
	}, "reach the same place")
}

// Two creates to one new place through a link.
func TestTwoCreatesToOneNewPlaceThroughALinkAreRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	link701(t, root, "d", "l")
	refusedAtStaging(t, root, []Input{create701("d/x"), create701("l/x")}, "reach the same place")
}

// A rename destination under a directory a create's staging makes: the
// destination exists once the directories are staged.
func TestARenameDestinationThatStagingMakesAnEntryUnderIsRefused(t *testing.T) {
	root := t.TempDir()
	link701(t, root, "d", "l")
	write(t, root, "src", "s\n")
	refusedAtStaging(t, root, []Input{
		create701("d/y"),
		{Path: "src", Op: "rename", Body: []string{"l/x"}, Lines: -1},
		create701("d/x/z"),
	}, "exists now")
}

// Codex P2 (second round): on a case-sensitive filesystem left -> A and right
// -> a are different directories, and left/x with right/x is a valid plan.
func TestLinksToDirectoriesThatDifferOnlyByCaseDoNotCollideOnACaseSensitiveFilesystem(t *testing.T) {
	root := t.TempDir()
	if caseInsensitiveFS(t, root) {
		t.Skip("this filesystem folds case, so A and a are one directory")
	}
	for _, d := range []string{"A", "a"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link701(t, root, "A", "left")
	link701(t, root, "a", "right")
	res, err := Apply(root, []Input{create701("left/x"), create701("right/x")}, Options{})
	if err != nil || !res.Applied {
		t.Fatalf("a valid plan was refused: %v %+v", err, res.Hunks)
	}
}
