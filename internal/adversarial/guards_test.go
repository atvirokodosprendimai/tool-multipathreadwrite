package adversarial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// tree writes files into a fresh directory and returns its path. Every test
// here works in its own tree so nothing leaks between them.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func readFile(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// unset is what plan.Parse puts in Lines when the caller wrote no lines= guard.
const unset = -1

const goFile = "package p\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"

// A guard the caller wrote must be checked, whatever the op. anchor= is the
// property the README sells over Write — "a wrong address fails loudly instead
// of overwriting the wrong lines" — and an insertion at a wrong address is the
// same mistake as a replacement at one.
func TestAnchorIsCheckedOnInsertAfter(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "a.go", Start: 2, End: 2, Op: "insert-after",
		Body: []string{"// inserted"}, Anchor: "func Add", Lines: unset,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Line 2 is blank; "func Add" is on line 3. The anchor does not hold.
	if res.Failed != 1 {
		t.Errorf("anchor %q does not appear on line 2, yet the hunk was accepted: failed=%d, applied=%v",
			"func Add", res.Failed, res.Applied)
	}
	if got := readFile(t, root, "a.go"); got != goFile {
		t.Errorf("file was modified through a false anchor:\n%s", got)
	}
}

func TestAnchorIsCheckedOnInsertBefore(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "a.go", Start: 1, End: 1, Op: "insert-before",
		Body: []string{"// header"}, Anchor: "func Add", Lines: unset,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if res.Failed != 1 {
		t.Errorf("anchor %q does not appear on line 1 (%q), yet the hunk was accepted: failed=%d",
			"func Add", "package p", res.Failed)
	}
}

// lines= asserts how many lines the address covers. On an insertion the address
// is a single line, so the only honest readings are "lines=1 holds, anything
// else fails" or "lines= is rejected on this op". Accepting lines=99 silently
// is neither: the caller paid a token for an assertion that was discarded.
func TestLinesGuardIsNotDiscardedOnAnInsert(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "a.go", Start: 2, End: 2, Op: "insert-after",
		Body: []string{"// inserted"}, Lines: 99,
	}}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if res.Failed != 1 {
		t.Errorf("lines=99 on a single-line insertion was accepted: failed=%d, applied=%v",
			res.Failed, res.Applied)
	}
}

// The two boundary addresses have no line to check an anchor against:
// insert-after 0 is before the first line, and insert-before one past the last
// is after everything. mrw refuses rather than passing them vacuously, which
// means prepend and append are the two edits an anchor cannot guard. README
// says so; this holds the pair together.
func TestAnAnchorOnABoundaryAddressIsRefusedRatherThanIgnored(t *testing.T) {
	for _, tc := range []struct {
		name  string
		op    string
		start int
	}{
		{"insert-after 0", "insert-after", 0},
		{"insert-before past the end", "insert-before", 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := tree(t, map[string]string{"a.go": goFile})

			res, err := apply.Apply(root, []apply.Input{{
				Path: "a.go", Start: tc.start, End: tc.start, Op: tc.op,
				Body: []string{"// x"}, Anchor: "package p", Lines: unset,
			}}, apply.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Failed != 1 {
				t.Fatalf("an anchor at %s was ignored: failed=%d, applied=%v", tc.name, res.Failed, res.Applied)
			}
			if !strings.Contains(res.Hunks[0].Reason, "cannot be checked") {
				t.Errorf("the refusal does not say why: %s", res.Hunks[0].Reason)
			}
		})
	}
}
