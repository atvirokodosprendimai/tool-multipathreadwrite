# Task ADR-007-T1: Decide what a walk may descend into

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** unassigned
**Produces:** `rooted.Descendable(absRoot, path string) (bool, error)`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Give `internal/rooted` the one thing a walk needs and `Resolve` does not answer:
whether a directory entry may be descended into.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | add | `Descendable` — the descend half of the ADR-006 boundary |
| `internal/rooted/rooted_test.go` | edit | its tests, including the loop case |

Nothing selects `Descendable` yet: T2 is its only caller and its Affected Files
table carries that line. Until then this is a function with tests and no call
site, which is the shape this repository keeps finding — it is acceptable here
only because T2 is in the same ADR and lands next.

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a plain subdirectory is
   descendable, a symlinked directory is not, a directory whose resolved path
   leaves the root is not, a self-referential symlink terminates rather than
   recursing, and a root that is ITSELF a symlink is still usable.
2. [S2] Implement `Descendable(absRoot, path string) (bool, error)`: `false` for
   anything `os.Lstat` reports as a symlink, `false` for a path
   `rooted.Resolve` refuses, `true` for a directory inside the root. An entry
   that is neither — a regular file — is not this function's question and
   returns `false` with no error.
   The ROOT is not subject to this rule: `rooted.Resolve` already canonicalises
   it once with `EvalSymlinks`, so a symlinked root is accepted and every
   candidate is compared against its real path. `Descendable` is asked about
   entries found INSIDE that root, never about the root itself.
3. [S3] Document in the function comment WHY a symlinked directory is refused
   rather than resolved and checked: following it can leave the tree and can
   loop, and refusing a loop after walking it is not a refusal. [proof: human: the reason cannot be asserted by a test, only read — the tests prove the behaviour, this proves someone can maintain it]

## Acceptance

```bash
set -o pipefail
go test ./internal/rooted/ -run 'TestDescend' -v 2>&1 | tee /tmp/adr007-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr007-t1.out \
  && go test ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestDescendableAcceptsADirectoryInsideTheRoot` | `internal/rooted/rooted_test.go` | the ordinary case | — | S1, S2 |
| `TestDescendableRefusesASymlinkedDirectory` | `internal/rooted/rooted_test.go` | rule 3 of the Decision — a symlinked DIRECTORY is never descended | — | S1, S2 |
| `TestDescendableRefusesADirectoryOutsideTheRoot` | `internal/rooted/rooted_test.go` | the ADR-006 boundary applies to descent | — | S1, S2 |
| `TestDescendableTerminatesOnASelfReferentialLink` | `internal/rooted/rooted_test.go` | a loop cannot hang a walk | — | S1, S2 |
| `TestASymlinkedRootIsStillUsable` | `internal/rooted/rooted_test.go` | the root itself is canonicalised, not refused | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests above |
| 2 — something selects it | nothing yet — T2 is the caller, and its mutation proves the walk consults this |
| 3 — the caller can discover it | n/a: no declared interface, internal package |
| 4 — it is used | nothing measures this yet |

## Mutation Log

- 2026-09-01 · d8550be* · mutant survived · exit 0 · `internal/rooted/rooted.go` · rule 3: a symlinked directory must never be descended · acceptance-sha256:2706ef7f57006313f833e1d89c345b4e04d29d524ee4e0840a33c55703cfac93
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-01 · d8550be* · mutant inconclusive · exit 1 · `internal/rooted/rooted.go` · rule 3: a symlinked directory is never descended, and the refusal must be the rule rather than Lstat semantics · acceptance-sha256:2706ef7f57006313f833e1d89c345b4e04d29d524ee4e0840a33c55703cfac93
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-01 · d8550be* · mutant killed · exit 1 · `internal/rooted/rooted.go` · rule 3: a symlinked directory is never descended — the mutant keeps lst used and compiles, so only the rule itself is disabled · acceptance-sha256:2706ef7f57006313f833e1d89c345b4e04d29d524ee4e0840a33c55703cfac93

## Invariants

- `Resolve`'s behaviour is unchanged: this task adds a function and edits none.
- A symlinked directory is refused without being resolved, so nothing outside
  the root is stat'd on the descend path.

## Risks

- A platform where `os.Lstat` does not report symlinks as such would make the
  refusal silently pass. Mitigation: the tests skip rather than pass when
  `os.Symlink` fails, as the existing suite already does.

## Stop Condition

Stop if `Descendable` cannot be written without a second boundary rule — if the
walk needs to know about ignore files or binary content to answer this, the
runner-up (`--files-from`) is the decision and this ADR should be withdrawn.

## Out of Scope

- The walk itself — T2 owns it.
- Any change to `Resolve`.

## Verification Log
- 2026-09-01 · d8550be* · exit 1 · `set -o pipefail …` · acceptance-sha256:2706ef7f57006313f833e1d89c345b4e04d29d524ee4e0840a33c55703cfac93 · ms:217
  ```
  --- last 9 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted.test]
  internal/rooted/rooted_test.go:102:14: undefined: Descendable
  internal/rooted/rooted_test.go:121:16: undefined: Descendable
  internal/rooted/rooted_test.go:138:14: undefined: Descendable
  internal/rooted/rooted_test.go:152:14: undefined: Descendable
  internal/rooted/rooted_test.go:169:14: undefined: Descendable
  internal/rooted/rooted_test.go:192:13: undefined: Descendable
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted [build failed]
  FAIL
  ```
- 2026-09-01 · d8550be* · exit 0 · `set -o pipefail …` · acceptance-sha256:2706ef7f57006313f833e1d89c345b4e04d29d524ee4e0840a33c55703cfac93 · ms:656
- 2026-09-01 · d8550be* · exit 0 · `set -o pipefail …` · acceptance-sha256:2706ef7f57006313f833e1d89c345b4e04d29d524ee4e0840a33c55703cfac93 · ms:485
- 2026-09-01 · d8550be* · exit 0 · `set -o pipefail …` · acceptance-sha256:2706ef7f57006313f833e1d89c345b4e04d29d524ee4e0840a33c55703cfac93 · ms:332
