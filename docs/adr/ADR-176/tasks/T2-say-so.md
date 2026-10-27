# Task ADR-176-T2: README, AGENTS.md and `mrw instructions` say a refused piped plan is kept

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** one sentence in each of README, AGENTS.md and `mrw instructions`
**Consumes:** the stderr line and `refused_plan` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the binary builds`, `the instructions say it`, `README says it`, `AGENTS.md says it`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision 2's line is not the only place a caller learns it: the full contract, README and AGENTS.md say, in one sentence each, that a plan refused from standard input is kept in the state directory and runs again by its path.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | one sentence in `CLI()`, beside `mrw write PLAN reads a plan from a file` (`:114`) |
| `README.md` | edit | the write section |
| `AGENTS.md` | edit | §2, after "If any hunk fails validation, nothing is written at all" |

## Ordered Steps

1. [S1] See the fence RED: the sentence is in none of the three. [proof: acceptance]
2. [S2] Add the sentence to each. Mutant: the sentence removed from `CLI()`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go build -o bin/mrw ./cmd/mrw \
  && h=$(./bin/mrw instructions 2>&1) && grep -qF 'mrw-refused/' <<<"$h" \
  && grep -qF 'mrw-refused/' README.md \
  && grep -qF 'mrw-refused/' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence greps the three surfaces; ADR-037's shared-sentence test keeps running unchanged | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the sentences |
| 2 — something selects it | `mrw instructions`, README, AGENTS.md |
| 3 — the caller can discover it | those three |
| 4 — it is used | not measured (ADR-009) |

## Mutation Log
(empty until execute)
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the sentence removed from CLI() · acceptance-sha256:95cb80f818cea6d1c7bcf826fc4e9e041725ea5beee5fde7a17e5344109f2a99

## Invariants

- `Core()` and `Shared()` do not change (ADR-141, ADR-037); the sentence goes in the full contract only.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0.

## Out of Scope

- A recipe for re-running a kept plan in `Recipes()` (permanent: boundary: the stderr line carries the command at the moment it is needed)

## Verification Log
(empty until execute)
- 2026-10-24 · 8278fcda* · exit 1 · `set -o pipefail …` · acceptance-sha256:95cb80f818cea6d1c7bcf826fc4e9e041725ea5beee5fde7a17e5344109f2a99 · ms:765 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:95cb80f818cea6d1c7bcf826fc4e9e041725ea5beee5fde7a17e5344109f2a99 · ms:1428
- 2026-10-24 · 71b67559* · exit 0 · `set -o pipefail …` · acceptance-sha256:95cb80f818cea6d1c7bcf826fc4e9e041725ea5beee5fde7a17e5344109f2a99 · ms:221
