# Task ADR-175-T1: `mrw seen --forget` empties this checkout's ledger

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `seen.Forget`; `mrw seen --forget`; contract §300
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the forget tests run`, `the forget unit test passed`, `the forget binary test passed`, `the newer-ledger notice is said once`, `the subdirectory forget test passed`, `the pending test passed`, `the pending rows pass`, `the contract run`, `section 300 exists`, `the forget row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

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
5. [S5] The review of #527: `TestSeenForgetFromASubdirectoryForgetsTheCheckout` and §300's MCP rows (the pending store forgotten at the root, at the checkout top from a subdirectory, and left alone when a newer ledger is refused), red first. Mutants: `Forget` reads and saves `root` not the checkout top; `forget` drops its `ForgetPending` calls; the newer-ledger check moved after them. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/seen/ ./internal/mcp/ ./cmd/mrw/ -count=1 -v -run '^(TestForgetDropsEveryLicenceInTheCheckout|TestForgetDropsPendingAcknowledgements|TestSeenForgetMakesTheNextWriteNeedAFreshRead|TestSeenForgetOverANewerLedgerSaysSoOnce|TestSeenForgetFromASubdirectoryForgetsTheCheckout)$' 2>&1) \
  && grep -qE '^--- PASS: TestForgetDropsEveryLicenceInTheCheckout \(' <<<"$out" \
  && grep -qE '^--- PASS: TestSeenForgetMakesTheNextWriteNeedAFreshRead \(' <<<"$out" \
  && grep -qE '^--- PASS: TestForgetDropsPendingAcknowledgements \(' <<<"$out" \
  && grep -qE '^--- PASS: TestSeenForgetOverANewerLedgerSaysSoOnce \(' <<<"$out" \
  && grep -qE '^--- PASS: TestSeenForgetFromASubdirectoryForgetsTheCheckout \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 300\.' scripts/contract.sh \
  && grep -qF 'PASS  after mrw seen --forget a write needs a fresh read' <<<"$c" \
  && grep -qF 'PASS  the ack served before --forget licenses no write' <<<"$c" \
  && grep -qF "PASS  a server at the checkout's top is forgotten when --forget runs from a subdirectory" <<<"$c" \
  && grep -qF 'PASS  and the refusal left the pending acknowledgements alone' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestForgetDropsEveryLicenceInTheCheckout` | `internal/seen/forget175_test.go` | read and written observations gone, header kept, count, newer ledger untouched | none | S1, S2, S3 |
| `TestSeenForgetMakesTheNextWriteNeedAFreshRead` | `cmd/mrw/forget175_test.go` | the flag through the binary; the refusal; the fresh read; the usage errors | none | S1, S2 |
| `TestSeenForgetOverANewerLedgerSaysSoOnce` | `cmd/mrw/forget175_test.go` | over a newer mrw's ledger the refusal says so once (review of #527) | none | S2 |
| `TestSeenForgetFromASubdirectoryForgetsTheCheckout` | `cmd/mrw/forget175_test.go` | a forget run from `sub/` of a git checkout empties the ledger a read at the top recorded (review of #527) | none | S5 |
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
- 2026-10-24 · 2199256e* · mutant survived · exit 0 · `cmd/mrw/main.go` · seen --forget over a newer ledger repeats the notice · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-24 · 2199256e* · mutant survived · exit 0 · `cmd/mrw/main.go` · seen --forget over a newer ledger repeats the notice · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-24 · 2199256e* · mutant survived · exit 0 · `cmd/mrw/main.go` · seen --forget over a newer ledger repeats the notice · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-24 · 2199256e* · mutant killed · exit 1 · `cmd/mrw/main.go` · seen --forget over a newer ledger repeats the notice · acceptance-sha256:f0e896bd791e609b7755ec9ad3f0f8f40c83142042efa752c3668ff23050c2bc
- 2026-10-27 · 39d6ff31* · mutant killed · exit 1 · `internal/seen/seen.go` · S5: Forget reads and saves the root, not the checkout top · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · covers:the subdirectory forget test passed
- 2026-10-27 · 39d6ff31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5: forget never calls ForgetPending · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · covers:the pending rows pass
- 2026-10-27 · 39d6ff31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5: the checkout top's pending store is not forgotten · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · covers:the pending rows pass
- 2026-10-27 · 39d6ff31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5: the newer-ledger check comes after the pending store is emptied · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · covers:the pending rows pass
- 2026-10-27 · 39d6ff31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5: -forget is not the flag · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · covers:the newer-ledger notice is said once
- 2026-10-27 · 39d6ff31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5: the refusal keeps its own mrw: prefix · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · covers:the newer-ledger notice is said once

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
- 2026-10-24 · 2199256e* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:77173
- 2026-10-24 · 2199256e* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:77580
- 2026-10-24 · 2199256e* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2d95dc6c5227e1716eafc060e22b47cc1a77d4a5117d9f6b566513870330753 · ms:76891
- 2026-10-24 · 2199256e* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0e896bd791e609b7755ec9ad3f0f8f40c83142042efa752c3668ff23050c2bc · ms:80722
- 2026-10-24 · 2199256e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f0e896bd791e609b7755ec9ad3f0f8f40c83142042efa752c3668ff23050c2bc · ms:0 · test-lock-sha256:c2a898aae8608c8c3d4bcb0d120db34db4c751133601f9ec2b07421687286362 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9mb3JnZXQxNzVfdGVzdC5nbwlUZXN0U2VlbkZvcmdldE1ha2VzVGhlTmV4dFdyaXRlTmVlZEFGcmVzaFJlYWQJZTk3ZTY0YmE5NWM0MzUyNTIwNjc0ZTM3YjM2M2UwNTk3YzlkNTAzYjk0MTIyNDVmY2RlYTYzZmQ2ZjkxNTBjOApib2R5CWNtZC9tcncvZm9yZ2V0MTc1X3Rlc3QuZ28JVGVzdFNlZW5Gb3JnZXRPdmVyQU5ld2VyTGVkZ2VyU2F5c1NvT25jZQlkZTVmZDM4NjdmNzlhMjQwZWI1MmE1ZjU5YjExMDhlNDQ0MTk0N2Q3MWFjNzc0OWY1ZTZiMDIyZTNiNmNjMTY4CmJvZHkJaW50ZXJuYWwvbWNwL2ZvcmdldDE3NV90ZXN0LmdvCVRlc3RGb3JnZXREcm9wc1BlbmRpbmdBY2tub3dsZWRnZW1lbnRzCTZmNzZlN2UzNGE3OWY2M2U1NjViNWFlNTljNTYwZjQ5MmIzMWI0YmM3ZDVmZGRhMzYzZTZlMjY1OTM0MzNlNTcKYm9keQlpbnRlcm5hbC9zZWVuL2ZvcmdldDE3NV90ZXN0LmdvCVRlc3RGb3JnZXREcm9wc0V2ZXJ5TGljZW5jZUluVGhlQ2hlY2tvdXQJYWNlNWMwODUxYmZmODJjNzYxMjA1OTk0MDhhOTQwODE0NTRiNDY1OTgyMzMyM2YyY2QzM2RjZGE1YWFhM2NjZQ · test-lock-kind:replace
- 2026-10-24 · 2199256e* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0e896bd791e609b7755ec9ad3f0f8f40c83142042efa752c3668ff23050c2bc · ms:78903
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:84887
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:174501
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:100943
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:88720
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:86633
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:117878
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:112310
- 2026-10-27 · 39d6ff31* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ae8957d3314fd4fb8336dbe48080212a51db9eb4e8d8bbf27c9216030888ebb8 · ms:0 · test-lock-sha256:58a0bb0d569d82384011d135c7658db63d550ae60db279aede28c31f1fec65c1 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9mb3JnZXQxNzVfdGVzdC5nbwlUZXN0U2VlbkZvcmdldEZyb21BU3ViZGlyZWN0b3J5Rm9yZ2V0c1RoZUNoZWNrb3V0CWU1NGE4YmE1NmMzYmY4MDQwYjliNGM1ZjliMGY5NzM2NDI4NTQ1MmM3YWE3N2M5NmRkZjU2ODhkY2M2N2E1N2IKYm9keQljbWQvbXJ3L2ZvcmdldDE3NV90ZXN0LmdvCVRlc3RTZWVuRm9yZ2V0TWFrZXNUaGVOZXh0V3JpdGVOZWVkQUZyZXNoUmVhZAllOTdlNjRiYTk1YzQzNTI1MjA2NzRlMzdiMzYzZTA1OTdjOWQ1MDNiOTQxMjI0NWZjZGVhNjNmZDZmOTE1MGM4CmJvZHkJY21kL21ydy9mb3JnZXQxNzVfdGVzdC5nbwlUZXN0U2VlbkZvcmdldE92ZXJBTmV3ZXJMZWRnZXJTYXlzU29PbmNlCTNlZGY2NjhjYTMzNWU3NmQ2NzI0ZGZiMGYwMjQ0NmVmZmVlNmIxNDgzNjgwZDIwNzZjYWEzYzM1MmEyMzEzZjEKYm9keQlpbnRlcm5hbC9tY3AvZm9yZ2V0MTc1X3Rlc3QuZ28JVGVzdEZvcmdldERyb3BzUGVuZGluZ0Fja25vd2xlZGdlbWVudHMJNmY3NmU3ZTM0YTc5ZjYzZTU2NWI1YWU1OWM1NjBmNDkyYjMxYjRiYzdkNWZkZGEzNjNlNmUyNjU5MzQzM2U1Nwpib2R5CWludGVybmFsL3NlZW4vZm9yZ2V0MTc1X3Rlc3QuZ28JVGVzdEZvcmdldERyb3BzRXZlcnlMaWNlbmNlSW5UaGVDaGVja291dAlhY2U1YzA4NTFiZmY4MmM3NjEyMDU5OTQwOGE5NDA4MTQ1NGI0NjU5ODIzMzIzZjJjZDMzZGNkYTVhYWEzY2Nl · test-lock-kind:replace
