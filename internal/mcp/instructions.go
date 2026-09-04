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

WHEN TO REACH FOR IT. Use mrw_read and mrw_write when the task touches %s.
Below that, use your ordinary editor — one edit in one file costs the same two
calls and prints more bytes than the file holds. It is for scattered sites.

THE TWO RULES THAT PRODUCE MOST REFUSALS.
1. Read before you write, and it is enforced per LINE, not per file. Being
   served lines 10-12 does not license an edit at line 50. mrw_read is what
   records the lines; a read through any other tool licenses nothing.
2. A plan is all or nothing. If any hunk fails, NOTHING is written and the
   siblings report skipped, never ok.

READING. mrw_read takes specs: a bare path, path:N, path:N-M, path:$ for the
last line, or path:/regexp/ — so the read finds the site and no separate search
call is needed. Example: %v

WRITING. mrw_write takes one plan document. Each hunk is a header line

    @@ <path> <address> <op> [guards]

followed by its body lines. Ops are replace, insert-after, insert-before,
delete and create. Addresses in a PLAN are line numbers or N-M ranges only — a
regexp works in a read, never here — and every address resolves against the
ORIGINAL file, so several hunks in one file need no offset arithmetic. Paths
are relative to the server's root; an absolute path is refused by name.

Guards are optional and checked on every op, insertions included: sha=<hex> for
the whole file, lines=<n> for the addressed span, anchor="<text>" for the first
addressed line. If a BODY line itself begins with @@, declare body=<n> and
raw=true or the plan is refused.

A worked plan — two hunks across two files, one guard:

%s
Pass dry_run true to validate and get the same receipt without writing.

A refusal is the tool working. It names the file, the plan line and the reason.
Read it and fix the plan rather than reaching for a bigger hammer.
`, triggerRule, exampleReadSpecs, examplePlan)
}
