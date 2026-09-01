package adversarial

import (
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// Every hunk gets its OWN verdict. apply.Input carries an Index the caller
// fills in, and the verdict map was keyed by it — so two inputs sharing an
// Index shared a slot, and one hunk's report silently became another's. The CLI
// always numbers its inputs, which is exactly why nothing caught it: the defect
// was reachable only by the package's other callers, and this suite is one.
func TestEveryHunkGetsItsOwnVerdictEvenWhenTheCallerNumbersThemBadly(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile, "b.go": goFile})

	res, err := apply.Apply(root, []apply.Input{
		{Path: "a.go", Start: 1, End: 1, Op: "replace", Body: []string{"package q"}, Lines: unset},
		{Path: "b.go", Start: 3, End: 3, Op: "replace", Body: []string{"nope"}, Anchor: "NOT HERE", Lines: unset},
	}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Hunks) != 2 {
		t.Fatalf("want a verdict per hunk, got %d", len(res.Hunks))
	}
	if got := res.Hunks[0]; got.Path != "a.go" || got.Status != "skipped" {
		t.Errorf("hunk 0 reported %+v; want a.go skipped", got)
	}
	if got := res.Hunks[1]; got.Path != "b.go" || got.Status != "failed" {
		t.Errorf("hunk 1 reported %+v; want b.go failed", got)
	}
	if res.Failed != 1 {
		t.Errorf("failed=%d, want exactly the one bad hunk", res.Failed)
	}
}

// A dry run must be indistinguishable from the real thing except that nothing
// is written — including the per-hunk verdicts, which is what makes --dry-run
// worth trusting before an apply.
func TestADryRunReportsWhatARealRunWould(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})
	in := []apply.Input{{Path: "a.go", Start: 1, End: 1, Op: "replace", Body: []string{"package q"}, Lines: unset}}

	dry, err := apply.Apply(root, in, apply.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, root, "a.go"); got != goFile {
		t.Fatalf("a dry run wrote the file:\n%s", got)
	}

	wet, err := apply.Apply(root, in, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if len(dry.Hunks) != len(wet.Hunks) || dry.Hunks[0].Status != wet.Hunks[0].Status {
		t.Errorf("dry run said %+v, real run said %+v", dry.Hunks, wet.Hunks)
	}
	if dry.Files[0].SHAAfter != wet.Files[0].SHAAfter {
		t.Errorf("dry run predicted sha %s, real run produced %s", dry.Files[0].SHAAfter, wet.Files[0].SHAAfter)
	}
}
