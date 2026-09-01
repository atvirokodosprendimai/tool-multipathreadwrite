package plan

import (
	"strings"
	"testing"
)

func TestParseAddr(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Addr
		bad  bool
	}{
		{in: "5", want: Addr{5, 5}},
		{in: "3-6", want: Addr{3, 6}},
		{in: "3-", want: Addr{3, EOF}},
		{in: "0", want: Addr{0, 0}},
		{in: "$", want: Addr{EOF, EOF}},
		{in: "10-$", want: Addr{10, EOF}},
		{in: "-", want: Addr{0, 0}},
		{in: "", bad: true},
		{in: "x", bad: true},
		{in: "3-x", bad: true},
	} {
		got, err := ParseAddr(tc.in)
		if tc.bad {
			if err == nil {
				t.Errorf("ParseAddr(%q) = %v, want error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAddr(%q): %v", tc.in, err)
		} else if got != tc.want {
			t.Errorf("ParseAddr(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseMultipleFilesAndOps(t *testing.T) {
	const doc = `# a comment
@@ a.go 3-6 replace anchor=func
new one
new two
@@ a.go 42 insert-after
added
@@ b.go 10-12 delete
@@ c.go - create
hello
`
	hunks, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(hunks) != 4 {
		t.Fatalf("got %d hunks, want 4", len(hunks))
	}
	if h := hunks[0]; h.Path != "a.go" || h.Op != OpReplace || h.Addr != (Addr{3, 6}) ||
		h.Anchor != "func" || len(h.Body) != 2 {
		t.Errorf("hunk 0 = %+v", h)
	}
	if h := hunks[1]; h.Op != OpInsertAfter || h.Addr != (Addr{42, 42}) || len(h.Body) != 1 {
		t.Errorf("hunk 1 = %+v", h)
	}
	if h := hunks[2]; h.Op != OpDelete || len(h.Body) != 0 {
		t.Errorf("hunk 2 = %+v", h)
	}
	if h := hunks[3]; h.Op != OpCreate || h.Body[0] != "hello" {
		t.Errorf("hunk 3 = %+v", h)
	}
}

// A body line may itself start with "@@ " when body= states the length, which
// is what makes the format safe for editing diffs and Markdown.
func TestExplicitBodyLengthProtectsHeaderLikeLines(t *testing.T) {
	const doc = "@@ a.md 1 replace body=2\n@@ not a header\nsecond\n@@ b.md 1 delete\n"
	hunks, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(hunks) != 2 {
		t.Fatalf("got %d hunks, want 2", len(hunks))
	}
	if got := hunks[0].Body; len(got) != 2 || got[0] != "@@ not a header" {
		t.Errorf("body = %q", got)
	}
	if hunks[1].Path != "b.md" {
		t.Errorf("second hunk = %+v", hunks[1])
	}
}

func TestParseRejectsBadPlans(t *testing.T) {
	for name, doc := range map[string]string{
		"empty":            "",
		"unknown op":       "@@ a.go 1 frobnicate\n",
		"short header":     "@@ a.go 1\n",
		"create with addr": "@@ a.go 1 create\nx\n",
		"insert range":     "@@ a.go 1-3 insert-after\nx\n",
		"empty insert":     "@@ a.go 1 insert-after\n",
		"reversed range":   "@@ a.go 6-3 replace\nx\n",
		"bad option":       "@@ a.go 1 delete wat=1\n",
		"short sha":        "@@ a.go 1 delete sha=abc\n",
		"loose text":       "hello\n@@ a.go 1 delete\n",
		"unterminated":     "@@ \"a.go 1 delete\n",
	} {
		if _, err := Parse(strings.NewReader(doc)); err == nil {
			t.Errorf("%s: Parse succeeded, want error", name)
		}
	}
}

// Every error in one round trip: re-emitting a plan is the expensive part, so
// the parser must not stop at the first mistake.
func TestParseReportsEveryError(t *testing.T) {
	_, err := Parse(strings.NewReader("@@ a.go 1 frobnicate\n@@ b.go 1 delete wat=1\n"))
	if err == nil {
		t.Fatal("Parse succeeded, want error")
	}
	if !strings.Contains(err.Error(), "frobnicate") || !strings.Contains(err.Error(), "wat") {
		t.Errorf("error names only some problems: %v", err)
	}
}

func TestQuotedFieldsSurvive(t *testing.T) {
	hunks, err := Parse(strings.NewReader("@@ \"my file.go\" 1 replace anchor=\"func Foo(\"\nx\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if hunks[0].Path != "my file.go" || hunks[0].Anchor != "func Foo(" {
		t.Errorf("hunk = %+v", hunks[0])
	}
}

// ADR-008 T2. A body on a `delete` used to be a hard parse error. It now means
// "these are the lines I expect to remove" — the one assertion only the caller
// can make, because it is their picture of the file rather than the file's own
// bytes.
func TestDeleteBodyIsNoLongerAParseError(t *testing.T) {
	hunks, err := Parse(strings.NewReader("@@ f.txt 2-3 delete\nb\nc\n"))
	if err != nil {
		t.Fatalf("a delete with an expected body was rejected: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("want 1 hunk, got %d", len(hunks))
	}
	if got, want := hunks[0].Body, []string{"b", "c"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// The rest of validate's rules are untouched: this task removes one rejection,
// not the section it lived in.
func TestDeleteBodyDoesNotWeakenTheOtherOps(t *testing.T) {
	for _, doc := range []string{
		"@@ f.txt 2 replace\n",      // ADR-006: an empty replace is a delete in disguise
		"@@ f.txt 2 insert-after\n", // an insertion with nothing to insert
		"@@ f.txt 2 create\nx\n",    // create takes no address
	} {
		if _, err := Parse(strings.NewReader(doc)); err == nil {
			t.Errorf("still-invalid plan parsed clean: %q", doc)
		}
	}
}
