package mcp

import (
	"fmt"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// This file is the only documentation an MCP caller has.
//
// A host driving `mrw mcp` is in a checkout it did not clone from here: it has
// no AGENTS.md, no README.md, and no model has this plan format in training
// data — that premise is ADR-009's, and over MCP this file is where it gets
// answered. So nothing below may point at a file in this repository. A
// reference the reader cannot open reads as help and is not.
//
// The examples are not illustrations. TestEveryEmbeddedExamplePlanReallyApplies
// parses each one and dry-run applies it against a tree built from the plan
// itself, because an example asserted to be PRESENT stays green long after it
// has stopped being VALID, and the example is the one thing a caller copies
// verbatim. ADR-012 records the decision.

// triggerRule is Shared's first sentence, quoted so the wire says what the
// repository says. The duplication is deliberate — a host cannot read a file
// in this checkout — and it is asserted rather than trusted: contract §43
// greps the same sentence out of both. ADR-062: this is always + plan, not a
// 3+ threshold.
const triggerRule = "Use mrw always: plan the activity as one read of every site, then one plan, then one write."

// examplePlan is a worked plan: two hunks, two files, one guard, and both
// address forms — a line range and a pattern. It is proved on the
// hand-written tree in testdata/example (ADR-090), so its anchor, its range
// and its pattern have to match real code rather than a tree built from them.
const examplePlan = `@@ internal/store/store.go 42-44 replace anchor="func (s *Store) Get" body=4
func (s *Store) Get(id string) (Row, bool) {
	r, ok := s.rows[id]
	return r, ok
}
@@ cmd/app/main.go /^import \($/ insert-after
	"example.com/app/internal/store"
`

// exampleReadSpecs shows the three address forms in one call: a line range, a
// regexp that finds its own line, and $ for the last line. The receiver's
// parentheses are escaped: unescaped they are a regexp group, and the spec
// matched no Go ever written — shipped that way until ADR-090 ran it.
var exampleReadSpecs = []string{
	"internal/store/store.go:40-60",
	`internal/store/store.go:/^func \(s \*Store\) Put/`,
	"cmd/app/main.go:$",
}

// instructionsText is the `instructions` field of the initialize result — the
// place the 2025-06-18 lifecycle provides for a server to say how it is meant
// to be driven.
//
// It is a function rather than a constant because it prepends guide.Shared
// (ADR-037) and WhyAllOrNothing, then interpolates the same examplePlan
// the tool schema publishes:
// one worked plan, quoted twice, so the two copies cannot disagree about a
// format that has no second source.
func instructionsText() string {
	return guide.Shared() + "\n\n" + guide.WhyAllOrNothing() + "\n\n" + fmt.Sprintf(`Recipes:
1. the same change at many sites, none in context: mrw_write with grep and sub (grep \bOld\b, sub New)
2. see each change: the same with dry_run true
3. a change by syntax: mrw_write with ast_grep and rewrite
4. look first: mrw_read with grep and count, then specs f.go:/P/
5. different changes: one mrw_read of the sites, one mrw_write plan
6. a step after the write: mrw_write with then [NAME]
7. license known lines unseen: mrw_read with quiet true
8. long lines: mrw_read with grep, max_cols 200
mrw_read with sub or rewrite returns the plan, licensed, no ack: send it as plan.

WHICH SURFACE. CLI has --context, --check on prose and more;
`+"`mrw --root DIR read`"+` reaches ANY checkout. This surface serves ONE fixed checkout,
chosen at launch with `+"`--root DIR mcp`"+`. With a shell prefer the CLI; prefer this one
with none; one writer per checkout.

READING. mrw_read takes specs: a path, path:N-M, path:A,+N (A plus the N lines
after it), path:$ (the last line) or path:/regexp/ — the read finds its own
site. Example: %v
To find files you cannot NAME, set grep (regexp) or ast_grep (structural; missing
ast-grep is named). Too large? An INDEX — one spec per file, no content. exclude
skips globs, grep ignored and binary files too; no range with either.

A read too large comes back as a PAGE: the lines that fit, a
-- PARTIAL: line, next_read for the rest, and markers BRACKETING each run:
"-- ck <id> open lines A-B (N lines follow)", the lines, "-- ck <id> close".
Repeat until next_read is absent; stopping early leaves part of a file.
A SERVE LICENSES NOTHING UNTIL YOU ACKNOWLEDGE IT.
%s
An id you omit leaves its lines unwritable: a host can cut a result unseen.

WRITING. mrw_write takes one plan and those ck ids as ack. Optional format: plan (default), apply_patch, or search_replace. git is refused. Each hunk is a header line

    @@ <path> <address> <op> [guards]
then its body lines. Ops: replace, insert-after, insert-before, delete, create, unlink, rename.
An address is a line number, an N-M range, A,+N (ONE start
plus the N lines after; a read clamps at the last line, a write refuses past
it), $ for the last, or a PATTERN — /regexp/ for one line, /from/,/to/ for
a range. The START must match EXACTLY ONE line: none or several fails that hunk
and the refusal names its matches. Every address resolves against the ORIGINAL
file; a pattern is NOT a way to edit a file you have not read — its line must
have been served. Paths are relative to the root; two spellings of ONE file
(case, symlink) are one: a plan naming both is refused.

Guards: sha=<hex> whole file, on every op; lines=<n> and anchor="<text>",
refused on create, unlink and rename. A BODY line starting @@ needs body=<n> and raw=true.

A worked plan:

%s
dry_run true: same receipt, no write.

Both tools cap the ENCODED answer at the ceiling _meta names; an oversized write
receipt drops the check's tail, then ok and skipped verdicts, then UNWRITTEN files, and says so in elided.
`, exampleReadSpecs, AckRule, examplePlan)
}
