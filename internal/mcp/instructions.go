package mcp

import "fmt"

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

// triggerRule is the threshold, quoted from this project's own agent
// instructions so the wire says what the repository says. The duplication is
// deliberate — a host cannot read a file in this checkout — and it is asserted
// rather than trusted: contract §43 greps the same sentence out of both.
const triggerRule = "3 or more edits, 2 or more files, or several ranges you need to read"

// maxInstructionsChars bounds the handshake document. A host that supports the
// field puts it in front of the model once per session, whether or not a tool
// is ever called, so the length is paid by every session rather than by the
// callers who benefit. The bound is what keeps this from growing into a second
// copy of the repository's agent instructions.
const maxInstructionsChars = 4096

// examplePlan is a worked plan: two hunks, two files, one guard. Two rather
// than one because a single edit in a single file is the case a caller should
// not have reached for mrw at all — showing it as the example would teach the
// wrong trigger alongside the right grammar.
const examplePlan = `@@ internal/store/store.go 42-44 replace anchor="func (s *Store) Get"
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
// It is a function rather than a constant because it interpolates the same
// examplePlan the tool schema publishes: one worked plan, quoted twice, so the
// two copies cannot disagree about a format that has no second source.
func instructionsText() string {
	return fmt.Sprintf(`mrw reads many file ranges and applies many edits in ONE call, and it reports a
verdict for EVERY edit. The failure it exists to prevent: a read that finds
nothing is obvious, a write that changes nothing is not.

WHICH SURFACE. Reach for mrw when the task touches %s. Below that use your
editor: same two calls, more bytes than the file holds.

Then choose. The CLI is broader — only it has --files-from, --check (the
project's tests, scoped to your writes), and check, iter, seen and stats. `+"`mrw --root DIR read`"+` points it at ANY checkout; --root goes
BEFORE the subcommand, since after `+"`read`"+` the short -C is the context flag.
This surface returns structured JSON (a read's receipt: 2nd text block); one
server is one writer to the ledger while parallel CLI processes race. With
a shell prefer the CLI; prefer this one with none, or when callers sharing
ONE fixed checkout want writes serialized.

THE TWO RULES THAT PRODUCE MOST REFUSALS.
1. Read before you write, per LINE not per file: served lines 10-12 do not
   license an edit at line 50. Only mrw_read serves lines; ack on either tool records them.
2. A plan is all or nothing: if any hunk fails NOTHING is written and the
   siblings report skipped, never ok.

READING. mrw_read takes specs: a bare path, path:N, path:N-M, path:A,+N (A plus
the N lines after it), path:$ for the last line, or path:/regexp/ — the read
finds its own site. Example: %v
To find files you cannot NAME, set grep to a regexp: mrw walks your paths (or the
root) and serves every match. Too large? An INDEX — one spec per file,
no content — send back as specs. exclude skips globs; no range with grep.

A read too large comes back as a PAGE: the lines that fit, a
-- PARTIAL: line, next_read for the rest, and markers BRACKETING each run:
"-- ck <id> open lines A-B (N lines follow)", the lines, "-- ck <id> close".
Repeat until next_read is absent; stopping early leaves part of a file.
A PAGE LICENSES NOTHING UNTIL YOU ACKNOWLEDGE IT.
%s
An id you omit leaves its lines unwritable: a host can cut a page before you see
it and mrw cannot tell, but you can.

WRITING. mrw_write takes one plan. Each hunk is a header line

    @@ <path> <address> <op> [guards]
then its body lines. Ops: replace, insert-after, insert-before, delete, create.
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
dry_run true: same receipt, no write. A refusal is the tool working:
it names the file, the plan line and the reason.

Both tools cap the ENCODED answer at the ceiling _meta names. An oversized
write receipt drops successes then UNWRITTEN files and says so in elided;
smaller still, one sentence and no receipt.
`, triggerRule, exampleReadSpecs, AckRule, examplePlan)
}
