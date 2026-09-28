# ADR-090: The shipped examples run on a tree they did not write

**Status:** Accepted
**Accepted:** 2026-09-28 by Zy — "C2 now, release after C3", then "Accept" on the record as drafted
**Date:** 2026-09-28
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-012, ADR-013, ADR-023, ADR-037, ADR-052, ADR-070, docs/adr/BACKLOG.md
**Governs:** `internal/mcp/instructions.go`, `internal/mcp/testdata/example/**`, `internal/mcp/conformance_test.go`, `internal/mcp/examples_test.go`
**Enforced-by:** `internal/mcp/conformance_test.go::TestEveryEmbeddedExamplePlanReallyApplies`
**Served-path change:** two example strings. The worked plan in the `initialize` instructions and in `mrw_write`'s `plan.examples` addresses its second hunk by pattern — `@@ cmd/app/main.go /^import \($/ insert-after` where it said `12 insert-after`; and the read example in the instructions and in `mrw_read`'s `specs.examples` escapes its receiver, `/^func \(s \*Store\) Put/`, which as shipped matched nothing. Thirteen bytes more on a 3,974-byte handshake. Nothing else a caller sees changes.

## Context

The MCP surface ships two worked examples, because no model has mrw's formats in training data
(ADR-012): a plan (`examplePlan`) and a read spec list (`exampleReadSpecs`). Both are proved by
tests that cannot fail for the reasons that matter to a caller who copies them.

- **The plan is proved on a tree built from itself.** `treeFor` (`conformance_test.go`) sizes each
  file to the highest line the plan names and plants every `anchor=` on the line it guards. So the
  anchor cannot fail — changing its text survived as a mutant on 2026-09-04 — and a line number
  cannot be out of range. Contract §43 rebuilds the same tree in Python.
- **No shipped example uses a pattern address.** ADR-013 taught `/regexp/` in prose, and its
  follow-up (ADR-013:210) was blocked on `treeFor`, which reads `Addr.Start`/`End` — zero for a
  pattern — and §43's header regex, which cannot parse one.
- **The read example is never executed.** §43 checks that `specs.examples[0]` is a list of more than
  one item; nothing reads it. Run for the first time by this record, its regexp spec
  `/^func (s \*Store) Put/` matched no Go ever written: the receiver's parentheses were a regexp
  group, not literals. It had shipped in every handshake and `tools/list` since ADR-012 (#72, first released in v0.1.0).
- **The plan test accepts too little.** `dryRunExample` passes a one-hunk, one-file plan, and reads
  `failed` through an unchecked assertion, so a receipt without the field reads as zero (ADR-012
  follow-up (a), Codex review of 96c5098).
- **The description walk has one sentinel.** The floor of 20 properties survives the loss of whole
  containers; only `mrw_write:hunks.status` is named (ADR-012 follow-up (b)).

## Existing Primitives Audit

- `dryRunExample` already drives the real `mrw_read` → ack → `mrw_write dry_run` round trip; only
  the tree it runs on changes.
- `readSchema()` (`schema_test.go`) still describes the read receipt for
  `TestEveryOutputSchemaPropertyIsDescribed`; the sentinel test reuses `describedPaths`.
- Contract §43 already builds the binary's published plan and dry-runs it; only its tree builder
  is replaced.

## Decision

1. **A hand-written fixture.** `internal/mcp/testdata/example/` holds `internal/store/store.go` and
   `cmd/app/main.go`, gofmt-clean Go written for the examples: `func (s *Store) Get` on line 42,
   one `func (s *Store) Put`, one `import (` in an eleven-line `main.go`. `treeFor` is deleted.
2. **The plan's second hunk is pattern-addressed:** `@@ cmd/app/main.go /^import \($/ insert-after`.
   The first hunk keeps its line range, `anchor=` and `body=4` (ADR-070). The shipped plan then
   shows both address forms.
3. **Every shipped example runs on a copy of the fixture.** Each shipped plan must hold at least two
   hunks over at least two paths, and dry-run with a `failed` field that is present and zero. The
   read specs are sent to `mrw_read` as one call and every spec must be served with no problem.
   The two examples are independent: the plan is licensed by a whole-file read of the fixture, not
   by the read example, whose `$` serves one line.
4. **The description walk names its sentinels:** `mrw_write:hunks.status`,
   `mrw_write:files.written`, `mrw_write:pattern.fires` and `mrw_read receipt:observed.Spans`, each
   required by exact path in a new test, so the three locks on
   `TestEveryOutputSchemaPropertyIsDescribed` stay intact.
5. **Contract §43 copies the fixture from `$SRC`** in place of its Python tree builder, reads the
   published spec list through the binary, and dry-runs the published plan. Its pair: the published
   plan with a pattern that matches nothing is refused, exit 1. §44 names `files.written` beside
   `hunks.status`.

## Alternatives Considered

- **Teach `treeFor` patterns.** Rejected: a tree built from the plan still cannot fail the plan's
  own guards, which is the defect.
- **Keep `12 insert-after` and only swap the tree.** Rejected: ADR-013's follow-up is that no shipped
  example exercises a pattern, and the handshake has the room.
- **Pattern-address both hunks.** Rejected: a caller should see the line range beside the pattern,
  and the first hunk carries the `anchor=` that ADR-035 requires on a multi-line replace.

## Component / Boundary Impact

`internal/mcp` (instructions, tests, testdata) and `scripts/contract.sh` §43/§44. No engine package
changes: `internal/read`, `apply`, `plan`, `seen`, `check` and `state` stay byte-identical.

## Wiring & Contract Changes

The handshake text and `mrw_write`'s `plan.examples[0]` change by one address; `legacy_golden.jsonl`
is regenerated and `era_test.go` records why. The exit codes, the tools and their schemas are
unchanged.

## Inter-task Contracts

T2 uses nothing T1 produces.

## Implementation

See `tasks/`.

## Consequences

- A shipped example that stops matching real-looking code — an anchor, a line past the end, a pattern
  that matches nothing or twice — goes red in `go test` and in §43.
- An example a later record adds must address the fixture, or extend it.
- ADR-013:210 and ADR-012's follow-ups (a) and (b) close.
- The read example's receiver parentheses are escaped, `/^func \(s \*Store\) Put/`: a caller who
  copied the old spec got "no match" on the very code it describes.

## Out of Scope

- A second MCP host that surfaces `instructions` (deferred: ADR-012 follow-up, first item)
- Compiling the fixture (permanent: boundary: `testdata` is outside the build; the fixture needs to be real-looking to the examples, not buildable — gofmt holds its shape)
- The CLI's own `--help` and `mrw instructions` examples (permanent: boundary: `internal/guide` is proved by its own tests; this record owns the MCP examples)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later example edit forgets the fixture | Medium | Low | the dry run fails and names the hunk |
| The handshake bound | Low | Medium | 3,987 of 4,096 bytes, measured on the built binary; the bound is asserted in three places |

## Rollback

Revert the PR and regenerate `legacy_golden.jsonl`.

## Follow-ups

None.
