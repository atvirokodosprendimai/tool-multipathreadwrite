# Task ADR-031-T2: `ack` promotes only what it names

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (a field on two tools, and the promotion)
**Owner:** Zy
**Produces:** `ack` on both tools, and promotion
**Consumes:** the checkpoint format and the pending store (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an acked checkpoint promoting exactly its own span`, `an unacked span licensing nothing`

## Goal

Turn a checkpoint the caller echoes into a ledger record for that span alone, and leave every span
nobody echoed unlicensed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `mrw_read` and `mrw_write` gain the optional `ack` array; the paged path stops calling `seen.Record` and files a pending record instead. |
| `internal/mcp/ack.go` | edit | `promote(root, acks)` resolves checkpoints to spans and calls `seen.Record` for those and no others. |
| `internal/mcp/mcp.go` | edit | Both advertised tool schemas gain `ack`, built from `AckRule` — a handler that accepts a field no schema declares is undiscoverable to a schema-driven host. |
| `internal/mcp/ack_test.go` | edit | `TestOnlyAckedSegmentsAreRecorded` — the record's `Enforced-by`. |

## Ordered Steps

1. [S1] Write `TestOnlyAckedSegmentsAreRecorded` and confirm it is RED. ⚠ **The fixture MUST cut the MIDDLE.** Serve a page with checkpoints, ack the FIRST and LAST and not the middle, and assert the ledger holds the two end spans and that a write to the middle is refused. `BACKLOG.md` and ADR-031's Context pre-register this: the measured host truncation kept both ends, so a fixture that drops the tail is green against the single-token design this record rejects and proves nothing. [proof: mutation]
2. [S2] Add `ack` to both tool handlers AND to both ADVERTISED schemas in `tools/list`, optional, absent meaning "I acknowledge nothing". ⚠ The first cut added it to the handlers only, so a schema-driven host could not discover a BREAKING requirement (review of PR #132); `TestBothToolsAdvertiseAck` pins it. [proof: mutation]
3. [S3] Implement `promote`: for each ack that matches a pending record, `seen.Record` that span; drop the pending entry; ignore an ack that matches nothing rather than failing the call, since a stale ack from a previous session is a caller mistake and not a reason to refuse a read. [proof: mutation]
4. [S4] Stop the paged path recording on serve. [proof: mutation]
5. [S5] Assert the refusal for an unacked span names `ack` — ADR-015's rule, that a refusal names the fix. [proof: mutation]
6. [S6] Run every gate, including `go test -race ./...` and `./scripts/contract.sh`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -v \
  -run 'TestOnlyAckedSegmentsAreRecorded' 2>&1 | tee /tmp/adr031-t2.out \
  && grep -q '^--- PASS: TestOnlyAckedSegmentsAreRecorded' /tmp/adr031-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr031-t2.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr031-t2c.out \
  && grep -q '^contract holds$' /tmp/adr031-t2c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestOnlyAckedSegmentsAreRecorded` | `internal/mcp/ack_test.go` | With the FIRST and LAST checkpoints acked and the middle not, the ledger holds exactly the two end spans — compared as an exact span set, and asserted at their EDGES, since probing an unacknowledged middle leaves an off-by-one at a boundary invisible — and the WRITES are driven, not merely the ledger inspected — the middle is refused, the refusal carries `ack` after `nameTheAck`, and an acknowledged span applies | — | S1, S2, S3, S4, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `promote` in `internal/mcp/ack.go` |
| 2 — something selects it | Both tool handlers call it before doing anything else; the S4 mutation restores the record-on-serve and the fence goes red on the middle write applying |
| 3 — the caller can discover it | The page footer and the refusal both name `ack`; T3 puts it in the instructions |
| 4 — it is used | Contract §68 (T3) drives the built server; no telemetry, per ADR-009 |

## Mutation Log

⚠ **Two P0 defects and three unasserted properties were found here by the review of PR #132, not by
this task's own gates.** Promotion keyed observations by PATH alone and let the last checkpoint's SHA
win, so a stale and a current acknowledgement together recorded old spans under the new SHA;
`ack` reached the handlers but not the advertised schemas; no test sent `ack` on a READ, so deleting
read-side promotion left everything green; the table asserted `Covers` rather than driving the
writes; and §68 sent two ids while writing only into the first. Each now has a test, and the entries
below post-date them.

⚠ **And the round that fixed those found a P0 of its own: the MECHANISM was bracketed while every
piece of caller guidance still said "the lines above it, send what you received".** A caller
following the instructions would acknowledge a span it had only half received, which is the defect
the brackets exist to prevent — the implementation was right and worth nothing. The footer, the MCP
instructions, `README.md` and `AGENTS.md` all state the both-markers-and-count rule now.

- 2026-09-07 · d95d79e* · mutant killed · exit 1 · `internal/mcp/ack.go` · promotion records a WHOLE-file observation instead of the acknowledged span, so acking one checkpoint licenses the entire file — the middle nobody received included · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · covers:an acked checkpoint promoting exactly its own span
- 2026-09-07 · d95d79e* · mutant killed · exit 1 · `internal/mcp/tools.go` · the refusal stops naming the remedy, so a caller meets "has not been read" for a page it was sent and is told nothing about ack — ADR-015 says a refusal names the fix · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · covers:an unacked span licensing nothing
- 2026-09-07 · dbe88d0* · mutant killed · exit 1 · `internal/mcp/ack.go` · promotion stops distinguishing file versions, so a stale acknowledgement and a current one merge and old spans are recorded against the current file — the second P0 the review of PR #132 found · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · covers:an acked checkpoint promoting exactly its own span

## Invariants
- An unacked span licenses nothing, and the refusal is the ledger's existing message plus the remedy.
- An ack that matches no pending record is ignored, not an error: it is a stale caller, and refusing the whole read would punish the honest half.
- ⚠ Promotion records the pending span, never the REQUESTED one. Recording what was asked for is the defect this record exists to close, wearing a new name.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base.

## Risks

- ⚠ A tail-cut fixture is green against the design this record REJECTS. S1 cuts the middle for that reason, and the S4 mutant is what proves the distinction held.
- Breaking existing MCP callers. It fails safe and the refusal names the remedy; S5 asserts that rather than assuming it.

## Stop Condition

Stop and ask if promotion needs `internal/seen` to change shape — the spans are already a set and
`merge` already unions them, so needing more means the design has drifted.

## Out of Scope

- The contract row and the instructions — T3
- Checkpoints on small whole reads (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:30972
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:32278
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:30027
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:31852
- 2026-09-07 · dbe88d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:30570
- 2026-09-07 · dbe88d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:43525
- 2026-09-07 · ba6aecd* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:30422
- 2026-09-07 · ba6aecd* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:38542
- 2026-09-07 · f0865c8* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:30602
- 2026-09-07 · 7c7b0b7* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:36407
- 2026-09-07 · c617934* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:30248
- 2026-09-07 · fc95241* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:34260
- 2026-09-07 · fc95241* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:32402
- 2026-09-07 · c9f747b* · exit 0 · `set -o pipefail …` · acceptance-sha256:56737d912f453189e61cc8dbdacd0a2f2d54685176185c78827a35eb3634bec8 · ms:35003
