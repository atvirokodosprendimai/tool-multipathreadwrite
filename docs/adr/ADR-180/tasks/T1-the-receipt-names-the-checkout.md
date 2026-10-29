# Task ADR-180-T1: `mrw seen --forget` names the checkout whose ledger it emptied

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the receipt `forgot N file(s) in <top>…`; contract §315; `TestSeenForgetNamesTheCheckoutItEmptied`, `TestSeenForgetWithRootNamesThatCheckoutAndLeavesTheOtherAlone`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Decision 1: the receipt of `mrw seen --forget` names `state.Checkout(root)`'s top for both a nonzero and a zero count; what is forgotten and every exit code stay as ADR-175 left them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/forget180_test.go` | new | the red tests, two real git checkouts |
| `cmd/mrw/main.go` | edit | `forget` (`:3273-3296`) prints the top it already computes; the `seen` Description (`:753-756`) says the receipt names the checkout |
| `cmd/mrw/forget175_test.go` | edit | `:31` asserts the old full sentence; it takes the new one, which changes the body `TestSeenForgetMakesTheNextWriteNeedAFreshRead` is locked at in ADR-175 T1 |
| `docs/adr/ADR-175/tasks/T1-seen-forget.md` | edit | its test lock, rewritten by `adr-verify docs/adr/ADR-175/tasks/T1-seen-forget.md --relock --replace-hashes` |
| `scripts/contract.sh` | edit | §300's two exact-equality rows (`:10262`, `:10272`) take the new text, their `ok` messages unchanged; §315 the two-checkout rows |
| `AGENTS.md` | edit | `:750` the printed form |
| `docs/agents.md` | edit | `:180` the printed form |

## Ordered Steps

1. [S1] Write `TestSeenForgetNamesTheCheckoutItEmptied` (two git checkouts A and B, each with a licensed file; `seen --forget` run from `A/sub` prints `forgot 1 file(s) in <A's real path>: …`, A's ledger is empty, B's still licenses B's write, a write in A needing the forgotten licence is refused exit 1 and writes nothing; a second forget prints `forgot 0 file(s) in <A's real path>`; run from a `git worktree` of A, whose `.git` is a file, it names the worktree's own top and leaves A's ledger alone) and `TestSeenForgetWithRootNamesThatCheckoutAndLeavesTheOtherAlone` (from A's cwd, `mrw --root B seen --forget` names B, empties B, leaves A's licence), and see both RED on the current receipt.
2. [S2] `forget` prints the top. Mutant: print `root` instead of the top — the subdirectory case must go red. [proof: mutation]
3. [S3] Mutant: the zero-count line drops the checkout — the zero case must go red. [proof: mutation]
4. [S4] Contract §300's exact rows and §315 (two checkouts: forget from `A/sub` names A and B's write still applies; `--root B` names B and A's write still applies). [proof: acceptance]
5. [S5] `AGENTS.md`, `docs/agents.md`, the `seen` Description; `forget175_test.go:31`; `adr-verify docs/adr/ADR-175/tasks/T1-seen-forget.md --relock --replace-hashes`, and `adr-lint` over every record whose lock moved. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestSeenForgetNamesTheCheckoutItEmptied|TestSeenForgetWithRootNamesThatCheckoutAndLeavesTheOtherAlone|TestSeenForgetMakesTheNextWriteNeedAFreshRead|TestSeenForgetFromASubdirectoryForgetsTheCheckout)$' 2>&1) \
  && grep -qE '^--- PASS: TestSeenForgetNamesTheCheckoutItEmptied \(' <<<"$out" \
  && grep -qE '^--- PASS: TestSeenForgetWithRootNamesThatCheckoutAndLeavesTheOtherAlone \(' <<<"$out" \
  && grep -qE '^--- PASS: TestSeenForgetMakesTheNextWriteNeedAFreshRead \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 315\. ' scripts/contract.sh \
  && grep -qF 'PASS  mrw seen --forget from a subdirectory names the checkout it emptied' <<<"$c" \
  && grep -qF 'PASS  the pair: the other checkout still licenses its write' <<<"$c" \
  && grep -qF 'PASS  mrw seen --forget says how many files the ledger held' <<<"$c" \
  && grep -qF 'PASS  the pair: an empty ledger is forgot 0 file(s)' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestSeenForgetNamesTheCheckoutItEmptied` | `cmd/mrw/forget180_test.go` | from `A/sub`: the receipt names A's top, zero and nonzero; A emptied, B kept, A's write refused | none | S1, S2, S3 |
| `TestSeenForgetWithRootNamesThatCheckoutAndLeavesTheOtherAlone` | `cmd/mrw/forget180_test.go` | `--root B` from A names and empties B, A kept | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | every `mrw seen --forget` |
| 3 — the caller can discover it | the receipt itself; AGENTS.md, `docs/agents.md` |
| 4 — it is used | the ADR-175 hook runs it on every compaction; the reproduction of 2026-10-28 is the evidence for building it |

## Mutation Log
- 2026-10-29 · 15a28a65* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the receipt names the root it was given, not the checkout emptied · acceptance-sha256:b17bfb233dd1e4b8b86685718f5e87783cd3ac6e1776b6190ccd4f39d4d1e8c1
- 2026-10-29 · 15a28a65* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: the zero-count line drops the checkout · acceptance-sha256:b17bfb233dd1e4b8b86685718f5e87783cd3ac6e1776b6190ccd4f39d4d1e8c1

## Invariants

- What is forgotten, the pending-store emptying and every exit code are ADR-175's.
- The printed checkout is the one whose ledger was emptied: both are `state.Checkout` of the same root.
- No engine package changes.

## Risks

- None beyond ADR-180's.

## Stop Condition

The fence exits 0, or a consumer that parses the old exact line is found: stop and bring it to Zy.

## Out of Scope

- Changing what is forgotten (permanent: boundary: ADR-175 owns it)

## Verification Log
- 2026-10-29 · 15a28a65* · exit 1 · `set -o pipefail …` · acceptance-sha256:b17bfb233dd1e4b8b86685718f5e87783cd3ac6e1776b6190ccd4f39d4d1e8c1 · ms:94249 · test-lock-sha256:f1cb41ce0b3adc1b994b73f885e3cef693a19ba37b78492dbc452a5e6f8539ff · test-lock-file:docs/adr/locks/f1cb41ce0b3adc1b994b73f885e3cef693a19ba37b78492dbc452a5e6f8539ff.lock
  ```
  ```
- 2026-10-29 · 15a28a65* · exit 1 · `set -o pipefail …` · acceptance-sha256:b17bfb233dd1e4b8b86685718f5e87783cd3ac6e1776b6190ccd4f39d4d1e8c1 · ms:88778
  ```
  ```
- 2026-10-29 · 15a28a65* · exit 0 · `set -o pipefail …` · acceptance-sha256:b17bfb233dd1e4b8b86685718f5e87783cd3ac6e1776b6190ccd4f39d4d1e8c1 · ms:82687
- 2026-10-29 · 15a28a65* · exit 0 · `set -o pipefail …` · acceptance-sha256:b17bfb233dd1e4b8b86685718f5e87783cd3ac6e1776b6190ccd4f39d4d1e8c1 · ms:82346
- 2026-10-29 · 15a28a65* · exit 0 · `set -o pipefail …` · acceptance-sha256:b17bfb233dd1e4b8b86685718f5e87783cd3ac6e1776b6190ccd4f39d4d1e8c1 · ms:79882
