# ADR-084: What blind reading 03 taught

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — approved the backlog plan that names this record, having said "we can take our own decisison besides M, since we own this now"
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-037, ADR-063, docs/blind/blind-03-result.md, docs/adr/BACKLOG.md
**Governs:** `internal/plan/plan.go`, `internal/guide/guide.go`, `cmd/mrw/main.go`, `AGENTS.md`
**Enforced-by:** `internal/plan/minus084_test.go::TestAWriteAddressThatStartsWithMinusNamesTheForm`
**Served-path change:** a write address of `-` and a positive number (`@@ f -2 delete`) is refused naming the form a write takes (`1-2`) instead of `bad line number ""`, and `--2`, `-0` or an overflowing bound is refused without a recommendation, still exit 2; `mrw instructions` says a write exits 1 when a hunk fails and 2 on a usage or filesystem failure; `mrw instructions`, `read --help` and AGENTS.md say a bare directory name in `--exclude` met below where the walk starts prunes that whole subtree, a named path is walked anyway, and `--ast-grep` filters its hits afterwards.

## Context

**What was observed** (blind reading 03, `docs/blind/blind-03-result.md`, "Teaching leads the agents
reported"; filed in BACKLOG as "Teaching leads from blind reading 03, not yet acted on"):

- `@@ f -2 delete` fails as `bad line number ""`, exit 2: a parse error that does not name the form.
  Runs s1 and s3 reported it, and reading 01 found it too. `mrw instructions` already says a write
  address takes no `-M` (ADR-063), so the teaching exists and the refusal is what fails the caller.
- `mrw instructions` explains exit 3 and the read's exit 1, but not a write's exit 1 or exit 2; run
  s2 learned them from the output.
- All three Sonnet runs avoided `--exclude` because the text does not say whether a bare directory
  name prunes the subtree. It does (`internal/read/walk.go`: an excluded directory returns
  `fs.SkipDir`); the Haiku runs that used it got it right.

BACKLOG said each needed its own decision because the handshake and the CLI text are budgeted. The
MCP handshake (`internal/mcp/instructions.go`: `Shared()`, `WhyAllOrNothing()` and MCP-specific
text) is bounded at 4096 characters as a whole and is not touched here; `CLI()` carries no size cap.

## Existing Primitives Audit

- `ParseAddr` is the one plan-address parser (`internal/plan`); `addr.CutRelative` handles `A,+N`
  before it, so the new refusal sits in one place.
- `guide.CLI()` is the document `mrw instructions` prints; AGENTS.md is the authored mirror.

## Decision

1. A plan address of `-` followed by a positive number is refused naming the write form, `1-M`, and
   saying `-M` is a read range; a `-` followed by any other digit string (`-0`, an overflow) or a
   second `-` is refused without recommending a form. Exit 2, as any parse refusal.
2. `CLI()` says: a write exits 1 when a hunk fails validation, and nothing is written; 2 on a usage
   or filesystem failure.
3. `CLI()`, the `--exclude` flag help and AGENTS.md say a bare directory name met below where the
   walk starts prunes that whole subtree, that a path you name is walked anyway, and (`CLI()`,
   AGENTS.md) that `--ast-grep` drops excluded hits after the binary runs.

## Alternatives Considered

- **Accept `-M` in a write as `1-M`.** Rejected: a write address that silently widens to line 1 is
  exactly the unread-range surprise ADR-002 guards against; naming the form costs one retry.
- **Teach these in the MCP handshake too.** Rejected: the whole handshake is bounded at 4096
  characters and carries the five ADR-037 sentences; the MCP surface returns the same refusal text.

## Component / Boundary Impact

`internal/plan` (the refusal), `internal/guide` and `cmd/mrw` (text), AGENTS.md.

## Wiring & Contract Changes

A refusal's wording changes; its exit code does not. Contract §166 drives the binary.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- The centralised `mrw` skill mirrors AGENTS.md and needs the `--exclude` sentence after merge.

## Out of Scope

- The regex substring lead from the same reading (permanent: boundary: `/regexp/` is documented as a regex and anchoring is the caller's; no run was harmed by it)
- The MCP handshake (permanent: boundary: `Shared()` stays the five ADR-037 sentences)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller matched on the old `bad line number ""` text | Low | Low | exit 2 is unchanged; no test or doc quoted the old text |

## Rollback

Revert the task.

## Follow-ups

- [ ] Mirror the `--exclude` sentence into the centralised `mrw` skill after merge.
