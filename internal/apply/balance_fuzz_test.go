package apply

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ── ADR-054 arm 2 under stress ─────────────────────────────────────────────
//
// balanceDelta is arithmetic, and arithmetic is what a fuzzer is for. The
// oracle below is written independently of the implementation (strings.Count
// per family rather than a rune walk), and every property here is one the
// Decision states: empty iff every family's net matches; each mismatching
// family reported once, in `{ ( [` order, as `X +a → +b`; swapping the two
// sides reverses every arrow; never a panic on any bytes.

// refNets is the independent oracle: opens minus closes per family.
func refNets(lines []string) [3]int {
	s := strings.Join(lines, "\n")
	return [3]int{
		strings.Count(s, "{") - strings.Count(s, "}"),
		strings.Count(s, "(") - strings.Count(s, ")"),
		strings.Count(s, "[") - strings.Count(s, "]"),
	}
}

// parseDelta reads `{ +1 → +0; ( -2 → +3` back into (family, before, after)
// triples, so the string form is checked as data and not by substring.
func parseDelta(t testing.TB, s string) map[rune][2]int {
	t.Helper()
	out := map[rune][2]int{}
	if s == "" {
		return out
	}
	for _, part := range strings.Split(s, "; ") {
		fields := strings.Fields(part)
		if len(fields) != 4 || fields[2] != "→" || len([]rune(fields[0])) != 1 {
			t.Fatalf("delta part %q is not `X +a → +b`", part)
		}
		before, err1 := strconv.Atoi(fields[1])
		after, err2 := strconv.Atoi(fields[3])
		if err1 != nil || err2 != nil {
			t.Fatalf("delta part %q has non-integer nets", part)
		}
		fam := []rune(fields[0])[0]
		if _, dup := out[fam]; dup {
			t.Fatalf("family %c reported twice in %q", fam, s)
		}
		out[fam] = [2]int{before, after}
	}
	return out
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// checkDeltaProperties asserts every Decision property of one call.
func checkDeltaProperties(t testing.TB, consumed, body []string) {
	t.Helper()
	got := balanceDelta(consumed, body)
	before, after := refNets(consumed), refNets(body)
	want := before != after
	if (got != "") != want {
		t.Fatalf("balanceDelta(%q, %q) = %q; nets %v vs %v — empty iff equal is violated", consumed, body, got, before, after)
	}
	parsed := parseDelta(t, got)
	fams := []rune{'{', '(', '['}
	order := []rune{}
	for _, part := range strings.Split(got, "; ") {
		if part != "" {
			order = append(order, []rune(part)[0])
		}
	}
	// Reported families are exactly the mismatching ones, in `{ ( [` order.
	pos := 0
	for i, f := range fams {
		if before[i] == after[i] {
			if _, ok := parsed[f]; ok {
				t.Fatalf("family %c matched (%d) yet was reported in %q", f, before[i], got)
			}
			continue
		}
		p, ok := parsed[f]
		if !ok {
			t.Fatalf("family %c differs (%d → %d) yet is absent from %q", f, before[i], after[i], got)
		}
		if p != [2]int{before[i], after[i]} {
			t.Fatalf("family %c reported %v, oracle says %d → %d (in %q)", f, p, before[i], after[i], got)
		}
		if pos >= len(order) || order[pos] != f {
			t.Fatalf("families out of `{ ( [` order in %q", got)
		}
		pos++
	}
	// Swapping the sides reverses every arrow.
	rev := parseDelta(t, balanceDelta(body, consumed))
	if len(rev) != len(parsed) {
		t.Fatalf("swap reported %d families, forward reported %d", len(rev), len(parsed))
	}
	for f, p := range parsed {
		if rev[f] != [2]int{p[1], p[0]} {
			t.Fatalf("swap of %c is %v, want %v", f, rev[f], [2]int{p[1], p[0]})
		}
	}
}

func FuzzBalanceDelta(f *testing.F) {
	seeds := [][2]string{
		{"func A() {", "func A() { return }"},
		{"", "}"},
		{"x := f(a[0], b{1})", "x := f(a[0], b{1})"},
		{"s := \"{\"", "s := \"\""},
		{"｛fullwidth｝", "{ascii}"},
		{"((((", "))))"},
		{"a\nb\nc", "a\nb"},
		{"{\n{\n{", "}\n}\n}"},
		{"\r\n{\r\n", "\r\n"},
		{"日本語 { 括弧 }", "日本語"},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, consumed, body string) {
		checkDeltaProperties(t, splitLines(consumed), splitLines(body))
	})
}

// Fullwidth and other look-alike brackets are NOT counted: the three families
// are ASCII, and a delta on `｛` would be a report about nothing a compiler
// sees. Pinned so a "helpful" widening of the set has to change this test.
func TestBalanceCountsOnlyASCIIBrackets(t *testing.T) {
	if got := balanceDelta([]string{"｛ （ ［ 「 «"}, []string{"｝ ） ］ 」 »"}); got != "" {
		t.Errorf("look-alike brackets produced a delta: %q", got)
	}
}

// A brace inside a string literal counts. That is the documented false
// positive (ADR-048: no lexer), and the test exists so nobody "fixes" it into
// a parser without retiring the record.
func TestBalanceCountsBracesInsideStrings(t *testing.T) {
	if got := balanceDelta([]string{`s := "{"`}, []string{`s := ""`}); got == "" {
		t.Error("a brace inside a string literal was not counted; the arm has grown a lexer")
	}
}

// ── Randomised Apply: the field on the receipt, not the function ───────────
//
// Random files, random ops, random paths from a pool that mixes prose, code,
// data and no-extension names. The reference splice is independent of Apply,
// and the Balance expectation is the Decision's rule stated directly: insert
// consumes nothing, delete's body net is zero, prose reports nothing. The
// seed is printed on failure and settable via MRW_SEED for a replay.

var pathPool = []string{
	"a.go", "b.rs", "c.py", "Makefile", "data.jsonl", "conf.toml", "x.yml",
	"notes.md", "README.MD", "spec.Markdown", "plain.txt", "doc.rst", "page.adoc",
	"dir.md/inner.go", "weird.md.go", "weird.go.md",
}

var linePool = []string{
	"func A() {", "}", "	return x", "x := f(a[0], b{1})", "s := \"{\"", "(", ")", "[", "]",
	"plain", "", "use {packages} here", "```", "- item (one)", "{{ template }}", "if a { b() }",
}

func randomLines(r *rand.Rand, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = linePool[r.Intn(len(linePool))]
	}
	return out
}

// refSplice is the independent model of what the file should hold after one
// hunk at 1-based [start,end].
func refSplice(orig []string, op string, start, end int, body []string) []string {
	var out []string
	switch op {
	case "insert-before":
		out = append(out, orig[:start-1]...)
		out = append(out, body...)
		out = append(out, orig[start-1:]...)
	case "insert-after":
		out = append(out, orig[:start]...)
		out = append(out, body...)
		out = append(out, orig[start:]...)
	case "replace":
		out = append(out, orig[:start-1]...)
		out = append(out, body...)
		out = append(out, orig[end:]...)
	case "delete":
		out = append(out, orig[:start-1]...)
		out = append(out, orig[end:]...)
	}
	return out
}

func TestRandomisedApplyBalanceFollowsTheDecision(t *testing.T) {
	seed := int64(54)
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = v
	}
	r := rand.New(rand.NewSource(seed))
	const iterations = 400
	for i := 0; i < iterations; i++ {
		path := pathPool[r.Intn(len(pathPool))]
		orig := randomLines(r, 3+r.Intn(8))
		ops := []string{"replace", "insert-before", "insert-after", "delete"}
		op := ops[r.Intn(len(ops))]
		start := 1 + r.Intn(len(orig))
		end := start
		if op == "replace" || op == "delete" {
			end = start + r.Intn(len(orig)-start+1)
		}
		body := randomLines(r, 1+r.Intn(4))
		anchor := strings.TrimSpace(orig[start-1])
		if anchor == "" && op == "replace" && end > start {
			// ADR-035: a multi-line replace needs anchor=, and a blank first
			// line has nothing to anchor on. The refusal is right; the
			// generator narrows the range instead of fighting it.
			end = start
		}
		in := Input{Path: path, Start: start, End: end, Op: op, Lines: -1, Anchor: anchor}
		switch op {
		case "delete":
			// Half the deletes carry ADR-008's expected body — the lines
			// they remove, verbatim. The Decision's rule for delete is
			// "body net is 0", so an expected body must not change the
			// delta.
			if r.Intn(2) == 0 {
				in.Body = append([]string(nil), orig[start-1:end]...)
			}
		default:
			in.Body = body
		}
		if in.Anchor == "" {
			in.Anchor = ""
		}

		root := t.TempDir()
		write(t, root, path, strings.Join(orig, "\n")+"\n")
		res, err := Apply(root, []Input{in}, Options{})
		label := fmt.Sprintf("seed=%d iter=%d path=%s op=%s %d-%d orig=%q body=%q", seed, i, path, op, start, end, orig, in.Body)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		h := res.Hunks[0]
		if h.Status != StatusOK {
			t.Fatalf("%s: status %s: %s", label, h.Status, h.Reason)
		}
		// The file is the reference splice, whatever Balance says.
		bodyForFile := in.Body
		if op == "delete" {
			bodyForFile = nil
		}
		spliced := refSplice(orig, op, start, end, bodyForFile)
		wantFile := ""
		if len(spliced) > 0 { // a file emptied by a delete has no terminator to keep
			wantFile = strings.Join(spliced, "\n") + "\n"
		}
		if got := read(t, root, path); got != wantFile {
			t.Fatalf("%s: file %q, want %q", label, got, wantFile)
		}
		// Balance follows the Decision.
		if IsProse(path) {
			if h.Balance != "" {
				t.Fatalf("%s: prose path carried Balance %q", label, h.Balance)
			}
			continue
		}
		var consumed, netBody []string
		switch op {
		case "replace":
			consumed, netBody = orig[start-1:end], in.Body
		case "delete":
			consumed, netBody = orig[start-1:end], nil // body net is 0 by Decision
		default:
			consumed, netBody = nil, in.Body // insert consumes nothing
		}
		wantDelta := refNets(consumed) != refNets(netBody)
		if (h.Balance != "") != wantDelta {
			t.Fatalf("%s: Balance=%q but oracle nets %v vs %v", label, h.Balance, refNets(consumed), refNets(netBody))
		}
		if wantDelta {
			p := parseDelta(t, h.Balance)
			b, a := refNets(consumed), refNets(netBody)
			for i, f := range []rune{'{', '(', '['} {
				if b[i] != a[i] && p[f] != [2]int{b[i], a[i]} {
					t.Fatalf("%s: family %c reported %v, want %d → %d", label, f, p[f], b[i], a[i])
				}
			}
		}
	}
}

// IsProse is a closed list keyed on the lowercased final extension and nothing
// else. Each row here is a way a path can look like prose without being it, or
// the reverse.
func TestIsProseIsTheClosedListOnTheFinalExtensionOnly(t *testing.T) {
	cases := map[string]bool{
		"notes.md": true, "README.MD": true, "a.Markdown": true, "x.txt": true, "d.rst": true, "p.adoc": true,
		".md":                         true, // a dotfile named .md has extension .md
		"weird.go.md":                 true,
		"weird.md.go":                 false,
		"dir.md/inner.go":             false,
		"Makefile":                    false,
		"data.jsonl":                  false,
		"conf.toml":                   false,
		"x.yml":                       false,
		"x.json":                      false,
		"x.mdx":                       false,
		"x.text":                      false,
		"x.md ":                       false, // trailing space is part of the extension
		"x.md~":                       false,
		filepath.Join("docs", "a.md"): true,
	}
	for p, want := range cases {
		if got := IsProse(p); got != want {
			t.Errorf("IsProse(%q) = %v, want %v", p, got, want)
		}
	}
}
