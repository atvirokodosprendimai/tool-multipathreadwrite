package adversarial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// ADR-035 requires an anchor= on a replace addressing more than one line. A
// documented example that does not carry one teaches a plan mrw refuses, and
// nothing else in this repository reads README.md or AGENTS.md.
//
// ⚠ THIS IS A GO TEST BECAUSE THE CHECK NEEDS THE PARSER, AND THREE SHELL CUTS
// PROVED IT. A contract row can drive the built binary but it cannot tokenise a
// plan header, and every attempt to approximate `splitHeader` with a regex was
// defeated by a header the parser accepts:
//
//	awk fields                took the FIRST field equal to `replace`, so a
//	                          QUOTED path was read as the op
//	greedy `.* replace`       backtracked into a quoted path when the real op
//	                          ended the line, then accepted an anchor= that sat
//	                          in the ADDRESS
//	space-only separators     missed `@@ f.go 2-3<TAB>replace`; splitHeader
//	                          accepts tabs
//	counting bare `replace`   missed `@@ f.go 2-3 "replace"`, because
//	                          splitHeader strips quotes from ANY field, and
//	                          missed a BOM-prefixed header
//
// Each of those was found by a review round rather than by reading, which is
// the signal that the approach was wrong and not merely incomplete. Calling
// plan.Parse makes the question parser-equivalent by construction: whatever
// splitHeader accepts, this sees the same way, including forms nobody has
// thought of yet.
func TestEveryDocumentedReplaceCarriesItsAnchor(t *testing.T) {
	repo := repoRoot(t)
	for _, name := range []string{"README.md", "AGENTS.md"} {
		b, err := os.ReadFile(filepath.Join(repo, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			why := documentedReplaceNeedsAnchor(line)
			if why != "" {
				t.Errorf("%s:%d teaches a plan mrw refuses — %s:\n\t%s",
					name, i+1, why, line)
			}
		}
	}
}

// documentedReplaceNeedsAnchor returns why the line is a documented replace
// that ADR-035 would refuse, or "" if it is not one or carries its anchor.
//
// A line that does not parse as exactly one hunk is not a plan header — read
// output such as `@@ 3-3` is the common case — and is left alone. That is not
// leniency about malformed plans: this check's only claim is about replace
// headers, and a non-hunk is not one.
func documentedReplaceNeedsAnchor(line string) string {
	line = strings.TrimPrefix(line, "\ufeff")
	if !strings.HasPrefix(line, "@@") {
		return ""
	}
	// A replace needs a body or plan.validate refuses it for the wrong reason,
	// and this check is not about that rule.
	//
	// ⚠ THE BODY LENGTH IS SEARCHED, NOT ASSUMED. A header may declare `body=N`
	// (ADR-027's counted body, used by any documented plan whose content itself
	// begins with @@), and a declared count that does not match what follows is
	// a parse error — which this function would then read as "not a plan
	// header" and skip SILENTLY. That is the same shape as the four shell cuts
	// it replaces: a real replace passing unseen. Trying successive lengths
	// costs nothing here and needs no regex over the header to find the count.
	var h plan.Hunk
	found := false
	for n := 1; n <= 8 && !found; n++ {
		hs, err := plan.Parse(strings.NewReader(line + "\n" + strings.Repeat("BODY\n", n)))
		if err == nil && len(hs) == 1 {
			h, found = hs[0], true
		}
	}
	if !found {
		return ""
	}
	if h.Op != plan.OpReplace || h.Anchor != "" {
		return ""
	}
	a := h.Addr
	// Single-line spellings need no anchor. Asked of the PARSED address, so a
	// pattern, a relative end and an open-ended range are all multi-line here
	// without this test knowing how any of them are written.
	if a.StartPat == nil && a.EndPat == nil && a.RelEnd == 0 && a.Start == a.End {
		return ""
	}
	return "a replace addressing more than one line, with no anchor="
}

// TestTheDocumentedPlanCheckRejectsWhatItMustReject is the gate on the gate.
// Without it the test above passes on a classifier that returns "" for
// everything — and the three shell cuts it replaces were each green against the
// real documentation while blind to a header the parser accepts. Every case
// below defeated one of them.
func TestTheDocumentedPlanCheckRejectsWhatItMustReject(t *testing.T) {
	mustFlag := []string{
		`@@ f.go 2-3 replace`,
		`@@ f.go 2- replace`,
		`@@ f.go 2-$ replace`,
		`@@ f.go 12,+2 replace`,
		`@@ f.go /^a$/,/^b$/ replace`,
		`@@ f.go /anchor="x"/,/^b$/ replace`,
		"@@ f.go 2-3\treplace",
		`@@ f.go 2-3 "replace"`,
		"\ufeff@@ f.go 2-3 replace",
		`@@ "foo replace bar.go" 2-3 replace`,
		// A counted body: the header declares how many lines follow, so a check
		// that appends a fixed one gets a parse error and skips it in silence.
		`@@ f.go 2-5 replace body=2 raw=true`,
	}
	for _, line := range mustFlag {
		if documentedReplaceNeedsAnchor(line) == "" {
			t.Errorf("not flagged, but mrw would refuse it: %q", line)
		}
	}

	mustPass := []string{
		`@@ f.go 12 replace`,
		`@@ f.go $ replace`,
		"@@ f.go 12\treplace",
		`@@ f.go 42-58 replace anchor="func Apply" lines=17`,
		`@@ f.go 2-3 replace anchor=x`,
		`@@ internal/store/store.go /^func \(s \*Store\) Get/,/^\}/ replace anchor="func (s *Store) Get"`,
		`@@ f.go 2-3 delete`,
		`@@ f.go 2 insert-after`,
		`@@ new.md - create`,
		`@@ 3-3`,
	}
	for _, line := range mustPass {
		if why := documentedReplaceNeedsAnchor(line); why != "" {
			t.Errorf("flagged (%s), but mrw accepts it: %q", why, line)
		}
	}
}
