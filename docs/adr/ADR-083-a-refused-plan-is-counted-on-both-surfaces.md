# ADR-083: A plan refused after it parsed is counted on both surfaces

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — approved the backlog plan that names this record, having said "we can take our own decisison besides M, since we own this now"
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-079, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`
**Enforced-by:** `cmd/mrw/tally083_test.go::TestAPlanRefusedAfterItParsedIsOneRefusal`
**Served-path change:** `mrw stats` now counts, as `refused_apply`, a plan the CLI refused after it parsed and before anything landed: a path that names a directory, a pointer that resolves to no file or to several, a malformed `.quality-harness.json`, an unreadable working set or ledger. `mrw_write` already counted its equivalent. Exit codes and receipts are unchanged.

## Context

**What was observed** (the review of #240, filed in BACKLOG under "Filesystem-error refusals are
tallied on one surface"; reproduced 2026-09-27 against v1.28.0): a plan naming a directory exits 2
on the CLI, `--dry-run` or not, and `mrw stats` afterwards says "no plans recorded yet", while
`mrw_write` counts the same refusal as `refused_apply` (`internal/mcp/tools.go`, the tally switch:
`applyErr != nil || res.Failed > 0`). The CLI returned before its ADR-009 tally switch.

**The class**, audited 2026-09-27 with
`mrw read --grep 'return refuse\(|return refuseWith\(|return cli.Exit\(' cmd/mrw/main.go` over the
write action: after the plan parsed, five refusals return before the tally — the working set
cannot be loaded, a pointer hunk path does not resolve or names more than one entry, the harness
config does not load, the ledger snapshot cannot be read, and the filesystem error from apply.
Refusals BEFORE the plan parsed (an unreadable plan file, an unknown `--format`, a plan inside
mrw's state directory) are left out on purpose: no document was judged, and ADR-009 counts
documents.

## Existing Primitives Audit

- `refuseWith` already counts a landed plan (`Applied`) with a `tallied` guard so nothing is
  counted twice; the same guard carries the new count.
- `authoring.RefusedApply` is ADR-009's bucket for "parsed, and did not apply"; no new name.

## Decision

1. A plan that parsed and was refused before anything landed is one `refused_apply` on the CLI,
   recorded by `refuseWith`, whatever the reason.
2. A filesystem error from apply is one `refused_apply`, `--dry-run` or not, under `--json` or not —
   what `mrw_write` records (ADR-079: a refused dry run is one refusal).
3. A refusal before the plan parsed is not counted.

## Alternatives Considered

- **Count only the filesystem-error branch, as BACKLOG named it.** Rejected: the other four
  post-parse refusals would stay uncounted on the CLI for the same reason, which is the drift this
  record exists to remove.
- **A new bucket for environment faults (unreadable ledger, bad harness config).** Rejected by
  ADR-009's own rule: splitting `refused_apply` means classifying by message text.

## Component / Boundary Impact

`cmd/mrw` only. `internal/authoring` and `internal/mcp` are unchanged.

## Wiring & Contract Changes

`mrw stats` counts more plans; the vocabulary is unchanged. Contract §164 drives the binary.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- A tally taken before this change undercounts CLI refusals; `refused_apply` rises for callers who
  hit these paths. The denominator (`plans`) rises with it, so no rate is inflated.

## Out of Scope

- A refusal before the plan parsed (permanent: boundary: ADR-009 counts documents, and none was judged)
- The pattern ring and pricing (permanent: boundary: they count landed writes only, ADR-055 and ADR-056)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A refusal is counted twice | Low | Low | the `tallied` guard; the test asserts `plans` after each run |

## Rollback

Revert the task.

## Follow-ups

None.
