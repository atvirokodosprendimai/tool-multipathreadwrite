# Task ADR-175-T1: `mrw seen --forget` empties this checkout's ledger

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `seen.Forget`; `mrw seen --forget`; contract §300
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the forget tests run`, `the forget unit test passed`, `the forget binary test passed`, `the pending test passed`, `the contract run`, `section 300 exists`, `the forget row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision 1: `mrw seen --forget` removes every observation in this checkout's ledger under `seen.lock`, keeps the header, empties this root's pending MCP acknowledgements first, says how many files it held, refuses a newer mrw's ledger, and refuses `--prune`/`--dry-run` beside it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/forget175_test.go` | new | the red unit test |
| `cmd/mrw/forget175_test.go` | new | the red test driving the built binary |
| `internal/seen/seen.go` | edit | `Forget(root) (int, error)` beside `Drop` (`:651-673`) |
| `internal/mcp/ack.go` | edit | `ForgetPending(root)`: empty `pending-v3.json` under `pendingLock` (`:72-83`) |
| `internal/mcp/forget175_test.go` | new | an MCP read served but not acknowledged, then `ForgetPending`, then the old `ack` id licenses nothing and the write is refused |
| `cmd/mrw/main.go` | edit | the `--forget` flag and its action in `seenCmd` (`:707-760`), the Description |
| `scripts/contract.sh` | edit | §300: a read licenses a write; after `--forget` the same write is refused naming the unread lines; a fresh read licenses it again |

## Ordered Steps

1. [S1] Write `TestForgetDropsEveryLicenceInTheCheckout` (unit: read and written observations both gone, header kept, count returned, a newer ledger untouched and reported), `TestForgetDropsPendingAcknowledgements` (an `mrw_read` serve left unacknowledged, then `ForgetPending`, then its `ack` id licenses nothing and the write is refused) and `TestSeenForgetMakesTheNextWriteNeedAFreshRead` (binary: read, `seen --forget` prints `forgot 1 file(s)`, the plan is refused, a fresh read lets it apply; `--forget --prune` exits 2) and see all three RED.
2. [S2] `seen.Forget` and the flag. Mutant: `Forget` keeps observations whose `Written` is set. [proof: mutation]
3. [S3] Mutant: the ADR-150 newer-ledger guard dropped — the newer-ledger case must go red. [proof: mutation]
4. [S4] Contract §300. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/seen/ ./internal/mcp/ ./cmd/mrw/ -count=1 -v -run '^(TestForgetDropsEveryLicenceInTheCheckout|TestForgetDropsPendingAcknowledgements|TestSeenForgetMakesTheNextWriteNeedAFreshRead)$' 2>&1) \
  && grep -qE '^--- PASS: TestForgetDropsEveryLicenceInTheCheckout \(' <<<"$out" \
  && grep -qE '^--- PASS: TestSeenForgetMakesTheNextWriteNeedAFreshRead \(' <<<"$out" \
  && grep -qE '^--- PASS: TestForgetDropsPendingAcknowledgements \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 300\.' scripts/contract.sh \
  && grep -qF 'PASS  after mrw seen --forget a write needs a fresh read' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestForgetDropsEveryLicenceInTheCheckout` | `internal/seen/forget175_test.go` | read and written observations gone, header kept, count, newer ledger untouched | none | S1, S2, S3 |
| `TestSeenForgetMakesTheNextWriteNeedAFreshRead` | `cmd/mrw/forget175_test.go` | the flag through the binary; the refusal; the fresh read; the usage errors | none | S1, S2 |
| `TestForgetDropsPendingAcknowledgements` | `internal/mcp/forget175_test.go` | a pending checkpoint served before `--forget` licenses nothing after it | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | `--forget` on `mrw seen` |
| 3 — the caller can discover it | `mrw seen --help`; README, AGENTS.md, `docs/agents.md` (T2) |
| 4 — it is used | not yet: the documented hook (T2) is the first consumer; MRW-12 is the evidence for building it |

## Mutation Log
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/seen/seen.go` · S2: Forget keeps observations whose Written is set · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/seen/seen.go` · S3: the ADR-150 newer-ledger guard dropped · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/mcp/ack.go` · S2: ForgetPending leaves the pending store intact · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753

## Invariants

- The ledger format and header are unchanged (ADR-150).
- A newer mrw's ledger is never written over.
- The tally, the working set and the state directory are untouched.

## Risks

- None beyond ADR-175's.

## Stop Condition

The fence exits 0.

## Out of Scope

- Forgetting one file (permanent: boundary: a compaction loses the whole context; `Drop` already serves an unlink)

## Verification Log
- 2026-10-24 · 8278fcda* · exit 1 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:380 · test-lock-sha256:ee9033d076640376e9193b801387cc65bd83c473cca9c0f1327958ff342f3fb8 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9mb3JnZXQxNzVfdGVzdC5nbwlUZXN0U2VlbkZvcmdldE1ha2VzVGhlTmV4dFdyaXRlTmVlZEFGcmVzaFJlYWQJZTk3ZTY0YmE5NWM0MzUyNTIwNjc0ZTM3YjM2M2UwNTk3YzlkNTAzYjk0MTIyNDVmY2RlYTYzZmQ2ZjkxNTBjOApib2R5CWludGVybmFsL21jcC9mb3JnZXQxNzVfdGVzdC5nbwlUZXN0Rm9yZ2V0RHJvcHNQZW5kaW5nQWNrbm93bGVkZ2VtZW50cwk2Zjc2ZTdlMzRhNzlmNjNlNTY1YjVhZTU5YzU2MGY0OTJiMzFiNGJjN2Q1ZmRkYTM2M2U2ZTI2NTkzNDMzZTU3CmJvZHkJaW50ZXJuYWwvc2Vlbi9mb3JnZXQxNzVfdGVzdC5nbwlUZXN0Rm9yZ2V0RHJvcHNFdmVyeUxpY2VuY2VJblRoZUNoZWNrb3V0CWFjZTVjMDg1MWJmZjgyYzc2MTIwNTk5NDA4YTk0MDgxNDU0YjQ2NTk4MjMzMjNmMmNkMzNkY2RhNWFhYTNjY2U
  ```
  ```
- 2026-10-24 · 8278fcda* · exit 1 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:78831
  ```
  ```
- 2026-10-24 · 8278fcda* · exit 1 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:79984
  ```
  ```
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:102315
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:93473
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:94020
