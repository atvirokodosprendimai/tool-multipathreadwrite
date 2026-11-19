# Task ADR-205-T1: check: true runs the check

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `mrw_write` selecting `CheckDemand`; `CheckAuto` removed; the tool and schema descriptions; contract §351; the sentences in `AGENTS.md` and `README.md`
**Consumes:** `writer.CheckDemand`, the receipt's `check` object
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `check: true runs the check after a prose-only write`, `a tree with no check command is reported, not skipped`, `without check: true no check runs`, `a failed check is still not an error`, `the automatic mode is gone`, `the markers exist`

## Goal

ADR-205 Decisions 1 to 4. In a tree whose `.quality-harness.json` declares a check that writes a marker file:

| `mrw_write` call | Outcome |
|------------------|---------|
| a plan editing only `notes.md`, `check: true` | the marker exists; the receipt's `check` says it ran and passed |
| the same plan, no `check` | no marker; no `check` in the receipt |
| a plan editing `a.go`, `check: true`, the check made to fail | `check` says it ran and failed; the call is not `isError`; the edit is on disk |
| `check: true` with `dry_run: true` | no marker |
| `check: true` and `then: ["static"]` on a prose-only plan | the check runs, then the step |

And in a tree with no check declared and none inferable: a prose-only plan with `check: true` returns `check` with `ran: false` and a reason; the call is not `isError`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `CheckDemand` |
| `internal/writer/flow.go` | edit | `CheckAuto` and its branch removed |
| `internal/mcp/mcp.go`, `internal/mcp/schema.go` | edit | the descriptions |
| `internal/mcp/check205_test.go` | new | the Goal's table |
| existing tests naming `CheckAuto` or the old rule | edit | listed in the pull request |
| `scripts/contract.sh` | edit | §351 |
| `AGENTS.md`, `README.md` | edit | the sentences on `mrw_write`'s check |

## Ordered Steps

1. [S1] `TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan`, RED on the first and last rows. [proof: acceptance]
2. [S2] The mode and the removal. Mutants, each killed: `check: true` selects no check; a prose-only write skips it; a tree with no command returns no `check`; a failed check is `isError`; a dry run runs it. [proof: mutation]
3. [S3] Contract §351 drives `mrw mcp --enable` over stdio: a prose-only write with `check: true` leaves the marker, paired with the same without it leaving none. The descriptions and documents. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/mcp -run '^TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan$' > "${TMPDIR:-/tmp}/adr205-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr205-fence.out")" -eq 1 ] \
  && grep -q '^# 351\. ' scripts/contract.sh \
  && ! grep -rq 'CheckAuto' internal cmd \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan` | `internal/mcp/check205_test.go` | every row of the Goal's table, by the marker file and the receipt | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §351 |
| 2 — something selects it | `mrw_write`'s `check` argument |
| 3 — the caller can discover it | the tool's description and `check`'s schema description, which a host reads; `AGENTS.md`, `README.md` |
| 4 — it is used | telemetry is refused by ADR-009; the follow-up in the record |

## Mutation Log

## Verification Log

## Invariants

- `mrw write` on the CLI behaves exactly as before.
- No receipt key is added, removed or renamed.

## Risks

- The fence's `# 351.` clause is a marker; the contract gate proves the section.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes with CI's Python suites and `scripts/measure.sh`, and the record is Accepted by Zy.

## Out of Scope

- The acknowledgement protocol and paging of `mrw_read` (deferred: docs/adr/BACKLOG.md)
