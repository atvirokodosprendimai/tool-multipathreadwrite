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

// sha= promised "hex" in its own error message while checking only the length,
// so sha=zzzzzzzz parsed and failed later as a CONTENT mismatch — sending the
// caller to look at their file instead of at their plan.
func TestShaMustActuallyBeHexadecimal(t *testing.T) {
	for _, v := range []string{"zzzzzzzz", "deadbeeg", "abcdefg1", "12345 78"} {
		if _, err := Parse(strings.NewReader("@@ f.txt 1 delete sha=" + v + "\n")); err == nil {
			t.Errorf("sha=%s parsed clean", v)
		}
	}
	// Case is not part of the alphabet question: a sha is hex either way.
	for _, v := range []string{"deadbeef", "DEADBEEF", "DeadBeef00"} {
		if _, err := Parse(strings.NewReader("@@ f.txt 1 delete sha=" + v + "\n")); err != nil {
			t.Errorf("sha=%s was rejected: %v", v, err)
		}
	}
}

// `delete` is the only op that CONSUMES A RANGE and can be written without a
// body. That is the criterion the receipt's bounds field rests on
// (cmd/mrw/main.go, ADR-008), and until now it lived only in a comment.
//
// Four consecutive rewrites of that comment each asserted something new that
// did not survive being run — the last of them "delete is the only op that can
// be written without a body", which `create` falsifies: an empty one is
// accepted and writes a zero-byte file. A claim gated by nothing goes stale
// exactly this way, so it is enumerated here over EVERY op instead. A new op
// that takes an empty body will fail this test rather than quietly make the
// comment wrong.
func TestDeleteIsTheOnlyRangeConsumingOpThatNeedsNoBody(t *testing.T) {
	for _, tc := range []struct {
		op            string
		addr          string
		consumesRange bool
		emptyBodyOK   bool
	}{
		{op: "create", addr: "-", consumesRange: false, emptyBodyOK: true},
		{op: "replace", addr: "2", consumesRange: true, emptyBodyOK: false},
		{op: "delete", addr: "2", consumesRange: true, emptyBodyOK: true},
		{op: "insert-after", addr: "2", consumesRange: false, emptyBodyOK: false},
		{op: "insert-before", addr: "2", consumesRange: false, emptyBodyOK: false},
	} {
		t.Run(tc.op, func(t *testing.T) {
			_, err := Parse(strings.NewReader("@@ f.txt " + tc.addr + " " + tc.op + "\n"))
			if tc.emptyBodyOK && err != nil {
				t.Errorf("an empty body was refused: %v", err)
			}
			if !tc.emptyBodyOK && err == nil {
				t.Error("an empty body was accepted")
			}
			// The conjunction is the claim. Either half alone is false of
			// some op: create takes an empty body, replace consumes a range.
			if only := tc.consumesRange && tc.emptyBodyOK; only != (tc.op == "delete") {
				t.Errorf("%s satisfies both halves; the receipt's bounds field is keyed on delete alone", tc.op)
			}
		})
	}
}

// apply.go states the principle this enforces: "a guard that is parsed and then
// discarded would be worse than no guard at all — the caller believes the edit
// is pinned". The parser was doing exactly that: a repeated key was last-wins
// and silent, so `anchor="NOPE" anchor="a"` applied at exit 0 with the false
// guard gone. Refused rather than resolved — two guards on one hunk are two
// different claims, and picking one silently is how the caller keeps believing
// the other.
func TestARepeatedGuardKeyIsRefused(t *testing.T) {
	for _, opts := range []string{
		`anchor="NOPE" anchor="a"`,
		`sha=aaaaaaaa sha=bbbbbbbb`,
		`lines=1 lines=2`,
		`body=1 body=2`,
		`raw=true raw=true`,
	} {
		_, err := Parse(strings.NewReader("@@ f.txt 1 replace " + opts + "\nX\n"))
		if err == nil {
			t.Errorf("%s: accepted, want a parse error", opts)
			continue
		}
		if !strings.Contains(err.Error(), "given twice") {
			t.Errorf("%s: refused for the wrong reason: %v", opts, err)
		}
	}
}

// One of each is still fine — the check must refuse repetition, not guards.
func TestDistinctGuardKeysStillParse(t *testing.T) {
	if _, err := Parse(strings.NewReader(`@@ f.txt 1 replace sha=aaaaaaaa lines=1 anchor="a"` + "\nX\n")); err != nil {
		t.Errorf("three DIFFERENT guards were refused: %v", err)
	}
}

// raw= only switches off the valid-header check inside a COUNTED body, so
// without body= the caller has written a guard that cannot fire.
func TestRawWithoutBodyIsRefused(t *testing.T) {
	_, err := Parse(strings.NewReader("@@ f.txt 1 replace raw=true\nX\n"))
	if err == nil {
		t.Fatal("raw=true without body= was accepted; it guards nothing")
	}
	if !strings.Contains(err.Error(), "without body=") {
		t.Errorf("refused for the wrong reason: %v", err)
	}
	// And the legitimate pairing still parses.
	if _, err := Parse(strings.NewReader("@@ f.txt 1 replace body=1 raw=true\n@@ not a header\n")); err != nil {
		t.Errorf("body= with raw=true was refused: %v", err)
	}
}

// Windows PowerShell 5.1 — the powershell.exe that ships with Windows — writes
// a UTF-8 BOM for `-Encoding utf8` and has no BOM-less option (utf8NoBOM
// arrived in PowerShell 7). So the most obvious way to author a plan in the
// native Windows shell produced a file mrw refused, with a message that said
// "text before the first @@ header" about a line that IS a header.
func TestAUTF8BOMDoesNotDisqualifyTheFirstHeader(t *testing.T) {
	hunks, err := Parse(strings.NewReader("\ufeff@@ f.txt 1 replace\nX\n"))
	if err != nil {
		t.Fatalf("a plan with a UTF-8 BOM was refused: %v", err)
	}
	if len(hunks) != 1 || hunks[0].Path != "f.txt" {
		t.Fatalf("got %+v, want one hunk for f.txt", hunks)
	}
	if len(hunks[0].Body) != 1 || hunks[0].Body[0] != "X" {
		t.Errorf("body = %q, want [X] — the body must not absorb the header", hunks[0].Body)
	}
}

// Only at offset 0. A BOM anywhere else is real content: stripping it there
// would silently edit a caller's body, which is the class of thing this format
// refuses to do.
func TestABOMInsideAPlanIsContentNotSyntax(t *testing.T) {
	hunks, err := Parse(strings.NewReader("@@ f.txt 1 replace\n\ufeffX\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(hunks) != 1 || len(hunks[0].Body) != 1 || hunks[0].Body[0] != "\ufeffX" {
		t.Errorf("body = %q, want the BOM preserved as content", hunks[0].Body)
	}
}
