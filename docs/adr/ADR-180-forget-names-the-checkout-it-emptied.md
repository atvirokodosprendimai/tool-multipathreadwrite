# ADR-180: `mrw seen --forget` names the checkout whose ledger it emptied

**Status:** Proposed
**Date:** 2026-10-28
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-038, ADR-150, ADR-161, ADR-175, docs/agents.md, AGENTS.md
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/forget180_test.go`, `docs/agents.md`, `AGENTS.md`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/forget180_test.go::TestSeenForgetNamesTheCheckoutItEmptied`
**Invalidates:** ADR-175 — the receipt text of its Decision 1 (`forgot N file(s): a write now needs a fresh read of the lines it touches`, and `forgot 0 file(s)`), which gains the checkout; nothing else in ADR-175 changes. Checked: ADR-150 (the newer-ledger refusal is unchanged), ADR-161 Decision 14 (the ledger is the checkout's; this record only names it), contract §300 (its exact-equality rows change, its `ok` messages, which ADR-175 T1's fence greps, do not).
**Served-path change:** `mrw seen --forget` prints `forgot N file(s) in <checkout>: a write now needs a fresh read of the lines it touches`, and `forgot 0 file(s) in <checkout>` over an empty ledger, where `<checkout>` is the absolute, link-resolved checkout top whose ledger was emptied; exit codes, the ledger and what is forgotten are unchanged.

## Context

`mrw seen --forget` empties the ledger of the checkout the root belongs to, not of the directory it was run from: `seen.Forget` resolves `state.Checkout(root)` (`internal/seen/seen.go:689`) — the nearest ancestor holding a `.git`, link-resolved (`internal/state/checkout.go:17-37`) — and empties that ledger. Its receipt names only a count (`cmd/mrw/main.go:3290-3294`). Reproduced 2026-10-28 on v1.77.1 (926aaba), two scratch git checkouts A and B, `mrw read --quiet` in each: `mrw seen --forget` run from `A/sub` printed `forgot 1 file(s): a write now needs a fresh read of the lines it touches` and named nothing; B's licence survived; a write in A needing the forgotten licence was refused, exit 1. The behaviour is right; the receipt does not say which ledger it touched. A host hook runs it with `--root "$cwd"` (ADR-175 Decision 4) and a person runs it from wherever they are, and in both the one fact that tells them whether the right checkout was cleared — and that a sibling worktree was not — is missing. Owner direction: the robustness finishing plan, item 1(c) (memoryd tool-multipathreadwrite/644d167a, 2026-10-28): *"receipt identifies the actual canonical checkout whose ledger was cleared, including zero and nonzero counts"*.

## Existing Primitives Audit

- `state.Checkout(root)` (`internal/state/checkout.go:17`): already the key `seen.Forget` and `seen.ReadPath` use, and `forget` in `cmd/mrw/main.go` already calls it for the pending stores. Reuse: the receipt prints the same value, so it cannot name a checkout other than the one emptied.
- `seen.Forget(root) (int, error)` (`internal/seen/seen.go:681`): unchanged. Its signature is not widened to return the top: the caller computes the same `state.Checkout(root)` from the same root, and keeping the engine package untouched keeps this record out of `internal/seen`.
- `mrw seen` (no flag) prints the state directory, not the checkout; not reused, since the state directory is a hash, not a path the caller recognises.

## Decision

**1. The receipt names the checkout.** `forget` in `cmd/mrw/main.go` prints `forgot N file(s) in <top>: a write now needs a fresh read of the lines it touches` when N > 0 and `forgot 0 file(s) in <top>` when N = 0, where `<top>` is `state.Checkout(root)`'s first result: an absolute path with links resolved, the checkout top when the root is inside a git checkout, the root itself otherwise. It is printed as `mrw seen` prints its paths, unquoted. The exit codes (0; 2 over a newer ledger, a missing root or `--prune`/`--dry-run`) and every refusal are unchanged.

**2. Nothing else changes.** What is forgotten (ADR-175 Decisions 1-3), the pending-store emptying, the newer-ledger refusal (ADR-150) and the documented hook (ADR-175 Decision 4) stay as they are. The hook discards stdout; a caller that matched the old line exactly must match the new one.

**What would make this decision wrong:** a consumer that parses the old exact line. None is known: the documented hook ignores stdout (`docs/agents.md` §8), and the only exact matches are this repository's contract rows, which change here.

## Alternatives Considered

- **A second line, `checkout: <top>`.** Keeps the first line byte-identical, so no existing assertion changes. Rejected: two lines for one fact, and a reader who stops at the first still does not know which checkout.
- **A suffix, `… the lines it touches (in <top>)`.** Keeps every prefix match. Rejected: the checkout is the subject of the sentence, not an afterthought, and the zero case would read `forgot 0 file(s) (in <top>)`.
- **Print the root as given.** Rejected: the plan asks for the checkout actually emptied; from `A/sub` the root is `A/sub` and the ledger is `A`'s, which is exactly the confusion being removed.
- **Return the top from `seen.Forget`.** Rejected: an engine-package signature change to carry a value the caller already computes from the same input.

## Component / Boundary Impact

- `cmd/mrw` (`forget`): the receipt line. No engine package changes; ownership unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw seen --forget` stdout | names the checkout, zero and nonzero | `cmd/mrw` | people, host hooks (which discard it) |
| contract §300 | the two exact-equality rows take the new text | `scripts/contract.sh` | CI |
| `AGENTS.md`, `docs/agents.md` | the printed form | docs | agents |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** the caller sees which ledger was emptied; a forget from a subdirectory or a worktree says it reached the checkout top.
- **Negative:** the receipt is longer by one path; a script matching the old exact line breaks.
- **Neutral:** an absolute path in the output, as `mrw seen` already prints.

## Out of Scope

- A `--json` form of the forget receipt (permanent: boundary: `mrw seen` has none and no consumer has asked; this record names the checkout in the existing line)
- Quoting a checkout path that holds a newline or a control character (permanent: boundary: `mrw seen` prints its paths unquoted and this record follows it; a checkout so named breaks every line-oriented receipt mrw prints, not this one)
- Naming the pending stores emptied (permanent: boundary: they are the root's and the top's by ADR-175 Decision 1, and naming the top names both)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A consumer matched the old exact line | Low | Low | the documented hook discards stdout; release notes name the change |
| The printed top differs from the ledger emptied | Low | Med | both come from `state.Checkout` of the same root; T1's test reads the ledgers of two checkouts, not the receipt alone |

## Rollback

Before a release: revert the line and the rows. After a release: the old text is restored in a later release; nothing persistent changes.

## Follow-ups

- [ ] none at authoring
