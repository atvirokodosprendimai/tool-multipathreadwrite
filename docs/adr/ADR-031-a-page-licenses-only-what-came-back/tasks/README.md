# ADR-031 Tasks

Implementation tasks for ADR-031: A page licenses only the part of it that came back. See the parent
ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Checkpoints and the pending store | done | — | `go test ./internal/mcp/ -run 'TestACheckpointCoversTheSpanItBrackets\|TestAPendingRecordReachesNoLedger\|TestThePendingStoreIsBounded' …` |
| T2 | `ack` promotes only what it names | done | — | `go test ./internal/mcp/ -run 'TestOnlyAckedSegmentsAreRecorded' …` |
| T3 | The contract drives the server, and the instructions teach it | done | — | `grep -q '^# 68\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the checkpoint format and the pending store | T2, T3 | T1 first — there is nothing to promote until checkpoints exist |
| T2 | `ack` on both tools, and promotion | T3 | T2 before T3 — §68 drives the built server, so the field must exist before the row can be red for the right reason |

## Notes

- ⚠ **The fixture must cut the MIDDLE.** The measured host truncation kept lines 1-90 and 2644-2727 and discarded everything between, so a fixture that drops the TAIL is green against the single-token design this record explicitly rejects, and proves nothing.
- ⚠ The checkpoint is RANDOM, not a content hash and not a counter. A caller that received nothing must be unable to produce it.
- The engine go/no-go applies in full: this record owns `internal/mcp` only. `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base.
- ⚠ Breaking for MCP callers, deliberately and safely: a paged read licenses nothing until acked, and the refusal names the remedy.
