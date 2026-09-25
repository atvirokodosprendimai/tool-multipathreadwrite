package apply

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// onDisk lists what a root holds, so a refused plan can be shown to have left
// nothing behind rather than only to have said so.
func onDisk(t *testing.T, root string) []string {
	t.Helper()
	es, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// indexed numbers a plan's inputs the way the CLI does.
func indexed(in []Input) []Input {
	for i := range in {
		in[i].Index = i
		in[i].Lines = -1
	}
	return in
}

// failedReasons returns the reason of every failed hunk.
func failedReasons(res Result) []string {
	var out []string
	for _, h := range res.Hunks {
		if h.Status == StatusFailed {
			out = append(out, h.Reason)
		}
	}
	return out
}

// ADR-071 T2. The v1.25.1 round created n.txt and N.TXT in one plan on APFS and
// NTFS: both hunks said "created", one file remained, and the first body was
// gone, exit 0. ADR-021's check asks os.SameFile, and a create has no inode to
// ask about. Names that differ only by case are refused whatever the
// filesystem, because validation cannot tell which kind it is without a probe.
func TestTwoCreatesThatCouldBeOneFileAreRefused(t *testing.T) {
	for _, tc := range []struct {
		why  string
		in   []Input
		a, b string
	}{
		{"two case spellings of a new file", []Input{
			{Path: "n.txt", Op: "create", Body: []string{"one"}, SrcLine: 1},
			{Path: "N.TXT", Op: "create", Body: []string{"two"}, SrcLine: 3},
		}, "n.txt", "N.TXT"},
		{"a create beside a rename destination of another case", []Input{
			{Path: "src.txt", Op: "rename", Body: []string{"Dest.txt"}, SrcLine: 1},
			{Path: "dest.txt", Op: "create", Body: []string{"x"}, SrcLine: 3},
		}, "Dest.txt", "dest.txt"},
	} {
		t.Run(tc.why, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "src.txt", "s\n")
			res, err := Apply(root, indexed(tc.in), Options{Seen: map[string]Seen{"src.txt": {SHA: shaOfFile(t, root, "src.txt")}}})
			if err != nil {
				t.Fatal(err)
			}
			if res.Applied {
				t.Fatalf("a plan naming %s and %s APPLIED: %+v", tc.a, tc.b, res.Hunks)
			}
			reasons := failedReasons(res)
			if len(reasons) != 1 || !strings.Contains(reasons[0], tc.a) || !strings.Contains(reasons[0], tc.b) {
				t.Fatalf("want one refusal naming both %s and %s, got %q", tc.a, tc.b, reasons)
			}
			if got := onDisk(t, root); strings.Join(got, ",") != "src.txt" {
				t.Fatalf("a refused plan left %v on disk", got)
			}
		})
	}
}

// Two creates of one spelling were two inserts into one new file: both bodies
// landed and both hunks said ok.
func TestAPathCreatedTwiceIsRefused(t *testing.T) {
	root := t.TempDir()
	res, err := Apply(root, indexed([]Input{
		{Path: "a.txt", Op: "create", Body: []string{"one"}, SrcLine: 1},
		{Path: "a.txt", Op: "create", Body: []string{"two"}, SrcLine: 3},
	}), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatalf("a path created twice APPLIED: %+v", res.Hunks)
	}
	reasons := failedReasons(res)
	if len(reasons) != 1 || !strings.Contains(reasons[0], "created twice") ||
		!strings.Contains(reasons[0], "1") || !strings.Contains(reasons[0], "3") {
		t.Fatalf("want one refusal naming both plan lines, got %q", reasons)
	}
	if got := onDisk(t, root); len(got) != 0 {
		t.Fatalf("a refused plan left %v on disk", got)
	}
}

// An edit of A.txt beside a create of a.txt: on a case-insensitive filesystem
// they are one file and ADR-021 refuses them; on a case-sensitive one they are
// two, and ADR-071 refuses them anyway. Either way A.txt is untouched.
func TestACreateBesideAnExistingFileOfAnotherCaseIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "A.txt", "one\n")
	res, err := Apply(root, indexed([]Input{
		{Path: "A.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, SrcLine: 1},
		{Path: "a.txt", Op: "create", Body: []string{"new"}, SrcLine: 3},
	}), Options{Seen: map[string]Seen{"A.txt": {SHA: shaOfFile(t, root, "A.txt")}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatalf("an edit of A.txt beside a create of a.txt APPLIED: %+v", res.Hunks)
	}
	reasons := failedReasons(res)
	if len(reasons) != 1 || !strings.Contains(reasons[0], "A.txt") || !strings.Contains(reasons[0], "a.txt") {
		t.Fatalf("want one refusal naming both spellings, got %q", reasons)
	}
	if got := read(t, root, "A.txt"); got != "one\n" {
		t.Fatalf("A.txt changed under a refused plan: %q", got)
	}
}

// The pairs: two creates of different names still land, and on a
// case-sensitive filesystem a case-only rename is not compared with its own
// source.
func TestTwoCreatesOfDifferentNamesStillApply(t *testing.T) {
	root := t.TempDir()
	res, err := Apply(root, indexed([]Input{
		{Path: "a.txt", Op: "create", Body: []string{"one"}, SrcLine: 1},
		{Path: "b.txt", Op: "create", Body: []string{"two"}, SrcLine: 3},
	}), Options{})
	if err != nil || !res.Applied {
		t.Fatalf("two creates of different names were refused: %v %+v", err, res.Hunks)
	}
	if got := onDisk(t, root); strings.Join(got, ",") != "a.txt,b.txt" {
		t.Fatalf("want a.txt and b.txt on disk, got %v", got)
	}

	sensitive := t.TempDir()
	if caseInsensitiveFS(t, sensitive) {
		return // a case-only rename onto a name that already answers is ADR-066's refusal
	}
	write(t, sensitive, "r.txt", "r\n")
	res, err = Apply(sensitive, indexed([]Input{
		{Path: "r.txt", Op: "rename", Body: []string{"R.txt"}, SrcLine: 1},
	}), Options{Seen: map[string]Seen{"r.txt": {SHA: shaOfFile(t, sensitive, "r.txt")}}})
	if err != nil || !res.Applied {
		t.Fatalf("a case-only rename was refused: %v %+v", err, res.Hunks)
	}
	if _, err := os.Stat(filepath.Join(sensitive, "R.txt")); err != nil {
		t.Fatalf("R.txt is not there after the rename: %v", err)
	}
}

// The rule itself, independent of the filesystem the suite runs on: two names
// that differ by case are compared only when one of them does not exist yet.
// Two EXISTING files are left to os.SameFile, which on a case-sensitive
// filesystem says they are two, and they both apply. A case-insensitive runner
// cannot hold two such files, so this drives foldClashes directly.
func TestTheCaseComparisonLeavesTwoExistingFilesToSameFile(t *testing.T) {
	in := indexed([]Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, SrcLine: 1},
		{Path: "A.txt", Start: 1, End: 1, Op: "replace", Body: []string{"B"}, SrcLine: 3},
	})
	if got := foldClashes(in, func(string) bool { return true }); len(got) != 0 {
		t.Fatalf("two existing files were compared by case: %v", got)
	}
	got := foldClashes(in, func(name string) bool { return name == "a.txt" })
	if reason := got[1]; !strings.Contains(reason, "A.txt may name the same file as a.txt (plan line 1)") {
		t.Fatalf("a name that does not exist yet was not compared: %v", got)
	}
}
