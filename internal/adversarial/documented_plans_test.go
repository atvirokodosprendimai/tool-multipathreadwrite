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
// ⚠ THIS GATE IS DELIBERATELY STRICTER THAN THE ENGINE, IN ONE STATED PLACE.
// It has no file, so for `/from/,/to/`, `N-`, `N-$` and `$-N` the span is not
// knowable here: each CAN resolve to one line, and mrw would then accept it
// unanchored. The gate asks for an anchor anyway rather than guessing, because
// an anchor on such a documented example is never wrong and the two that exist
// already carry one. That is policy, and it is said out loud — an earlier cut
// reported these as "a plan mrw refuses", which was simply untrue of them.
//
// ⚠ THIS IS A GO TEST BECAUSE THE CHECK NEEDS THE PARSER, AND FOUR SHELL CUTS
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
// plan.Parse settles the part the regexes kept getting wrong — TOKENIZATION.
// Whatever splitHeader accepts, this sees the same way, including forms nobody
// has thought of yet.
//
// It is not equivalent in every respect, and saying so is the point: the body
// length is searched to a finite bound (below), so a header declaring a body
// longer than that is skipped. That residue is one bounded, stated case rather
// than an open class of header spellings, which is the trade this rewrite made.
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
				t.Errorf("%s:%d — %s:\n\t%s",
					name, i+1, why, line)
			}
		}
	}
}

// documentedReplaceNeedsAnchor returns why the line is a documented replace that
// this repository's DOCUMENTATION POLICY requires an anchor on, or "" if it is
// not one or carries its anchor. Not every such line is one mrw would refuse:
// the third bucket below is deliberately stricter than the engine, and saying
// otherwise is the claim two rounds of review took out of this file.
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
	// The bound is finite and that is a real limit, not a proof: a header
	// declaring body=65 would fall out of the search and be skipped. It is not
	// closed because closing it means reading the count back out of the header,
	// which is the regex this check exists to stop writing. 64 is far past any
	// documented example, and the limit is stated here rather than papered over.
	for n := 1; n <= 64 && !found; n++ {
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
	// THREE OUTCOMES, NOT TWO — and collapsing them to two is what produced a
	// false positive twice. This check has no file, so for some address forms
	// the span is genuinely not knowable here, and calling those "multi-line"
	// states something that may be false.
	//
	// Guaranteed ONE line, whatever the file holds:
	//   a bare number, `$`, an equal literal range `N-N`, and a SINGLE pattern.
	//   apply.go:746 sets `to := from` and extends it only when EndPat is set,
	//   so `/re/` alone resolves to one line exactly as `12` does. Treating
	//   every pattern as multi-line was the round-5 defect.
	switch {
	case a.EndPat == nil && a.RelEnd == 0 && (a.StartPat != nil || a.Start == a.End):
		return ""
	// Guaranteed MORE than one line, whatever the file holds: a relative end of
	// at least 1, or a literal range whose ends are ordinary line numbers with
	// the end above the start. EOF is excluded because `$` is a sentinel, not a
	// number, and `N-$` is only knowable against a file.
	case a.RelEnd >= 1,
		a.StartPat == nil && a.EndPat == nil &&
			a.Start != plan.EOF && a.End != plan.EOF && a.End > a.Start:
		return "a replace addressing more than one line, with no anchor="
	}
	// Everything left — `/from/,/to/`, `N-`, `N-$`, `$-N` — resolves against a
	// file this check does not have, and CAN come out as one line: a pattern
	// pair whose ends match the same line, or an open range starting at the last
	// line. So the requirement is stated as what it is. An anchor on such a
	// documented example is never wrong, and the two that exist already carry
	// one.
	return "a replace whose span cannot be known without the file, and no anchor= to pin it"
}

// TestTheDocumentedPlanCheckRejectsWhatItMustReject is the gate on the gate.
// Without it the test above passes on a classifier that returns "" for
// everything — and the four shell cuts it replaces were each green against the
// real documentation while blind to a header the parser accepts. Most cases
// below defeated one of them; the counted-body case defeated the FIRST Go cut
// of this very check, which is why it is here too.
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
			t.Errorf("not flagged, but the documentation policy requires an anchor here: %q", line)
		}
	}

	mustPass := []string{
		`@@ f.go 12 replace`,
		`@@ f.go $ replace`,
		// Equal after resolution, and equal WITHOUT resolution: these cover one
		// line whatever the file holds, so requiring an anchor on them would
		// fail the build on correct documentation.
		`@@ f.go 3-3 replace`,
		`@@ f.go $-$ replace`,
		`@@ f.go $- replace`,
		"@@ f.go 12\treplace",
		`@@ f.go 42-58 replace anchor="func Apply" lines=17`,
		`@@ f.go 2-3 replace anchor=x`,
		// A SINGLE pattern resolves to one line (apply.go:746), so it needs no
		// anchor. An earlier cut flagged it and would have failed the build on a
		// legitimate documented header.
		`@@ f.go /^func A/ replace`,
		`@@ f.go /^func A/ replace lines=1`,
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
