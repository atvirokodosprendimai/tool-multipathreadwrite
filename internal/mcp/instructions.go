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

// maxInstructionsChars bounds the handshake document. A host that supports the
// field puts it in front of the model once per session, whether or not a tool
// is ever called, so the length is paid by every session rather than by the
// callers who benefit. The bound is what keeps this from growing into a second
// copy of the repository's agent instructions.
const maxInstructionsChars = 4096

// examplePlan is a worked plan: two hunks, two files, one guard. Two rather
// than one so the example is a plan, not a single hunk.
const examplePlan = `@@ internal/store/store.go 42-44 replace anchor="func (s *Store) Get" body=4
func (s *Store) Get(id string) (Row, bool) {
	r, ok := s.rows[id]
	return r, ok
}
@@ cmd/app/main.go 12 insert-after
	"example.com/app/internal/store"
`

// exampleReadSpecs shows the three address forms in one call: a line range, a
// regexp that finds its own line, and $ for the last line.
var exampleReadSpecs = []string{
	"internal/store/store.go:40-60",
	`internal/store/store.go:/^func (s \*Store) Put/`,
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
	return guide.Shared() + "\n\n" + guide.WhyAllOrNothing() + "\n\n" + fmt.Sprintf(`WHICH SURFACE. With a shell prefer the CLI.
CLI has --files-from, --check, and
check, iter, seen and stats. `+"`mrw --root DIR read`"+` points it at ANY checkout; --root goes
BEFORE the subcommand, since after `+"`read`"+` the short -C is the context flag.
This surface serves ONE fixed checkout, chosen at launch with `+"`--root DIR mcp`"+`,
and returns structured JSON. With a shell prefer the CLI; prefer this one with
none. Writers take turns on either surface: one writer per checkout.

Only mrw_read serves lines; ack records them. Lines 10-12 do not license line 50.

READING. mrw_read takes specs: a bare path, path:N, path:N-M, path:A,+N (A plus
the N lines after it), path:$ for the last line, or path:/regexp/ — the read
finds its own site. Example: %v
To find files you cannot NAME, set grep (regexp) or ast_grep (structural; missing
ast-grep is named). Too large? An INDEX — one spec per file, no content. exclude
skips globs; no range with either.

A read too large comes back as a PAGE: the lines that fit, a
-- PARTIAL: line, next_read for the rest, and markers BRACKETING each run:
"-- ck <id> open lines A-B (N lines follow)", the lines, "-- ck <id> close".
Repeat until next_read is absent; stopping early leaves part of a file.
A SERVE LICENSES NOTHING UNTIL YOU ACKNOWLEDGE IT.
%s
An id you omit leaves its lines unwritable: a host can cut a result before you
see it and mrw cannot tell, but you can.

WRITING. mrw_write takes one plan and those ck ids as ack. Optional format: plan (default), apply_patch, or search_replace. git is refused. Each hunk is a header line

    @@ <path> <address> <op> [guards]
then its body lines. Ops: replace, insert-after, insert-before, delete, create, unlink, rename.
An address is a line number, an N-M range, A,+N (ONE start
plus the N lines after; a read clamps at the last line, a write refuses past
it), $ for the last, or a PATTERN — /regexp/ for one line, /from/,/to/ for
a range. The START must match EXACTLY ONE line: none or several fails that hunk
and the refusal names its matches. Every address resolves against the
ORIGINAL file, so hunks need no offset arithmetic; a pattern is NOT a way to
edit a file you have not read — the line it resolves to must still have been
served. Paths are relative to the server's root; an absolute one is
refused by name, and two spellings of ONE file (case, symlink) are one, so
a plan naming both is refused.

Guards, checked on every op: sha=<hex> whole file, lines=<n> the span,
anchor="<text>" first addressed line. A BODY line beginning with @@ needs
body=<n> and raw=true, or the plan is refused.

A worked plan:

%s
dry_run true: same receipt, no write. A refusal is the tool working.

Both tools cap the ENCODED answer at the ceiling _meta names. An oversized
write receipt drops successes then UNWRITTEN files and says so in elided;
smaller still, one sentence and no receipt.
`, exampleReadSpecs, AckRule, examplePlan)
}
