# Task ADR-161-T3: mrw instructions lead with the recipes; --core names the sub write

**Depends-on:** T1, T2, T6, T8
**Covers:** none — no spec
**Estimated scope:** S (text and its tests)
**Owner:** Zy
**Produces:** the `Recipes:` block in `guide.CLI()`, `guide.Core()` rule 1
**Consumes:** `read --sub`, `write --sub` (T1), `--rewrite` (T2), `read --quiet` (T6), `read -` (T8)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the recipes follow the shared sentences`, `the core names the sub write`, `the recipes run`

## Goal

An agent that reads only `mrw instructions` meets the context-clean recipes before anything else, and they work against the binary.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `Recipes:` after `Shared()`; `Core()` rule 1 |
| `internal/guide/guide_test.go` | edit | `TestInstructionsLeadWithTheRecipes` |
| `cmd/mrw/core141_test.go` | edit | `TestTheCoreNamesTheSubWrite` |
| `scripts/contract.sh` | edit | §265: recipes 1 and 3 taken from `$MRW instructions` and run |

## Ordered Steps

1. [S1] Write the failing tests and the contract row and see them red. [proof: acceptance]
2. [S2] The `Recipes:` block of Decision 4 after `Shared()`, each recipe a label line and a command line; the reference keeps every pinned phrase and the whole stays over 900 words. [proof: mutation]
3. [S3] `Core()` rule 1 names `mrw write --grep P --sub R`; the eight stay under 300 words and name only real flags. [proof: mutation]
4. [S4] Contract §265 takes the command lines of recipes 1 and 3 from `$MRW instructions` — the indented line after `1. ` and after `3. `, each required to start `mrw write --` before it runs — and runs them in a scratch tree (3 with the fake ast-grep), requiring the result and exit 0; a missing or misshapen line fails the row. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ ./cmd/mrw/ -count=1 -v 2>&1) \
  && for n in TestInstructionsLeadWithTheRecipes TestTheCoreNamesTheSubWrite; do grep -qE "^--- PASS: $n " <<<"$out" || exit 1; done \
  && test -z "$(gofmt -l .)" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  the recipes mrw instructions prints run as written' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the test the fence names joins this table in the change that writes it, since `TestEveryTestNamedByARecordExists` holds every named test to exist | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the text |
| 2 — something selects it | `mrw instructions`, `--core` |
| 3 — the caller can discover it | it is what they print; configuration B reads it |
| 4 — it is used | T5 |

## Mutation Log
(empty until execute)

## Invariants

- `Shared()` unchanged and first; every pinned phrase present.

## Risks

- A recipe line too long to read; each is one command.

## Stop Condition

A pinned phrase cannot stay without the reference staying as long as it is.

## Out of Scope

- The MCP handshake text (permanent: boundary: T9 carries it, with the MCP arguments)

## Verification Log
(empty until execute)
