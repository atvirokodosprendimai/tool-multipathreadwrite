package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func create701(p string) Input {
	return Input{Path: filepath.FromSlash(p), Op: "create", Body: []string{"x"}, Lines: -1}
}

// refusedAsNested asserts the plan fails whole, dry run or not, with a hunk
// naming the nesting, and leaves nothing in the root but what it began with.
func refusedAsNested(t *testing.T, root string, in []Input, want string) {
	t.Helper()
	before, _ := os.ReadDir(root)
	for _, dry := range []bool{true, false} {
		res, err := Apply(root, in, Options{DryRun: dry, Force: true})
		if err != nil {
			t.Fatalf("dry run %v: %v", dry, err)
		}
		if res.Applied || res.Failed == 0 {
			t.Fatalf("dry run %v: the plan validated: %+v", dry, res.Hunks)
		}
		var named bool
		for _, h := range res.Hunks {
			if h.Status == StatusFailed && strings.Contains(h.Reason, want) {
				named = true
			}
		}
		if !named {
			t.Errorf("dry run %v: no failed hunk says %q: %+v", dry, want, res.Hunks)
		}
	}
	if after, _ := os.ReadDir(root); len(after) != len(before) {
		t.Errorf("the refused plan changed the root: %d entries, was %d", len(after), len(before))
	}
}

// P2-1 of the #701 review: the nesting check compared bytes, so on a
// filesystem that folds case, creates of ok.txt, New and new/child.txt passed
// a dry run and then ended PARTIALLY APPLIED.
func TestWrittenPathsThatNestOnlyByCaseAreRefusedWhereTheFilesystemFoldsCase(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem keeps case")
	}
	refusedAsNested(t, root, []Input{create701("ok.txt"), create701("New"), create701("new/child.txt")}, "sits under")
}

// And by case where the filesystem folds it: rename onto X, create x/y.
func TestARenameDestinationThatNestsOnlyByCaseIsRefusedWhereTheFilesystemFoldsCase(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem keeps case")
	}
	write(t, root, "a.txt", "a\n")
	refusedAsNested(t, root, []Input{
		{Path: "a.txt", Op: "rename", Body: []string{"X"}, Lines: -1},
		create701("x/y"),
	}, "nests with")
}

// One conflict is one refusal, on the path that sits under the other, and the
// other path's hunk is skipped, not failed. Both paths are absent, so no
// earlier guard (belowAFile) refuses first: only the nesting check can.
func TestANestingConflictFailsOnlyThePathThatSitsUnderTheOther(t *testing.T) {
	for _, dry := range []bool{true, false} {
		root := t.TempDir()
		res, err := Apply(root, []Input{
			{Path: "f", Op: "create", Body: []string{"x"}, Lines: -1, Index: 0},
			{Path: "f/c", Op: "create", Body: []string{"x"}, Lines: -1, Index: 1},
		}, Options{DryRun: dry})
		if err != nil || res.Applied || len(res.Hunks) != 2 {
			t.Fatalf("dry %v: err %v applied %v: %+v", dry, err, res.Applied, res.Hunks)
		}
		if h := res.Hunks[0]; h.Status != StatusSkipped {
			t.Errorf("dry %v: f reports %s (%s), want skip", dry, h.Status, h.Reason)
		}
		if h := res.Hunks[1]; h.Status != StatusFailed || !strings.Contains(h.Reason, "sits under") {
			t.Errorf("dry %v: f/c reports %s (%s), want the nesting refusal", dry, h.Status, h.Reason)
		}
	}
}

// Two directories in one plan: the load error returned is the first in plan
// order, not the last one seen.
func TestTwoDirectoriesInOnePlanReturnTheFirstLoadErrorInPlanOrder(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"d1", "d2"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	in := []Input{
		{Path: "d2", Op: "replace", Start: 1, End: 1, Body: []string{"z"}, Lines: -1, Index: 0},
		{Path: "d1", Op: "replace", Start: 1, End: 1, Body: []string{"z"}, Lines: -1, Index: 1},
	}
	res, err := Apply(root, in, Options{Force: true})
	if err == nil || !strings.HasPrefix(err.Error(), "d2: ") {
		t.Fatalf("the error is %v, want the first directory's (d2)", err)
	}
	if len(res.Hunks) != 2 {
		t.Errorf("the receipt holds %d verdicts, want 2", len(res.Hunks))
	}
}

// Codex P2 on #724: where the filesystem keeps case, an existing file A and an
// existing directory a are two entries, and editing A and a/b is a valid plan.
func TestExistingEntriesThatDifferOnlyByCaseDoNotNestOnACaseSensitiveFilesystem(t *testing.T) {
	root := t.TempDir()
	if caseInsensitiveFS(t, root) {
		t.Skip("this filesystem folds case, so A and a are one entry")
	}
	write(t, root, "A", "one\n")
	if err := os.Mkdir(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "a/b", "one\n")
	res, err := Apply(root, []Input{
		{Path: "A", Op: "replace", Start: 1, End: 1, Body: []string{"two"}, Lines: -1, Index: 0},
		{Path: "a/b", Op: "replace", Start: 1, End: 1, Body: []string{"two"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if err != nil || !res.Applied {
		t.Fatalf("a valid plan was refused: %v %+v", err, res.Hunks)
	}
}
