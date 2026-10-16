# Task ADR-161-T2: --rewrite with --ast-grep prints or applies ast-grep's own rewrite as a plan

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `mrw read --ast-grep P --rewrite R`, `mrw write --ast-grep P --rewrite R`
**Consumes:** the plan printing and the write's plan source (T1)
**Data dependency:** hermetic (a fake ast-grep with recorded `--json`; one manual run against a pinned real ast-grep through npx, recorded)
**Proof map:** v1
**Rests-on:** `a rewrite plan groups spans and anchors them`, `a rewrite licenses each span and the line after it`

## Goal

A change by syntax at many sites is one mrw command, built from ast-grep's rewrite, with no line in the agent's context.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/astgrep.go` | edit | `--rewrite` passed to ast-grep; `replacement` and byte offsets read from `--json` |
| `internal/read/rewrite.go` | add | replacements applied to each file's bytes, whole-line spans grouped and merged, one anchored `replace` hunk each |
| `internal/read/rewrite161_test.go` | add | spans, merging, anchors, quoting, the licence, refusals |
| `cmd/mrw/main.go` | edit | `--rewrite` on `read` and `write` |
| `scripts/contract.sh` | edit | §264: the built binary with a fake ast-grep |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] Read each match's `range.byteOffset` (UTF-8 bytes, end exclusive) and `replacement` (`""` is one; a missing field or an offset outside the file refused, never guessed); the file is read once and its sha checked against what ast-grep read; overlapping ranges — nested matches — refused by name, nothing printed or written. A minimal sentence naming `--rewrite` in `guide.CLI()`. [proof: mutation]
3. [S3] Apply the non-overlapping matches of a file to its bytes; group changed lines into whole-line spans, merging spans that touch; one `replace` hunk per span with `sha=`, `body=N raw=true` and `anchor=` its first line, quoted and escaped as `splitHeader` reads it. Fixtures: nested matches, a multibyte line, a multi-line replacement, an empty replacement, two matches on one line. [proof: mutation]
4. [S4] Record each span and the line after it as licensed and not shown (ADR-052, ADR-118); `write --ast-grep P --rewrite R` applies the plan through T1's path. [proof: mutation]
5. [S5] Contract §264 with a fake ast-grep answering recorded `--json`: `read … --rewrite … | mrw write -` and `write … --rewrite …` give the recorded result, exit 0. One run against a pinned ast-grep (`npx -y @ast-grep/cli@<version>`) on the same tree, recorded by the executor. [proof: human: the executor records the ast-grep version, the command and the diff]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ -count=1 -v -run '^TestARewritePlanGroupsSpansAndAnchorsThem$' 2>&1) \
  && grep -qE '^--- PASS: TestARewritePlanGroupsSpansAndAnchorsThem ' <<<"$out" \
  && go build -o bin/mrw ./cmd/mrw && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a rewrite read piped to write, and a rewrite write, give ast-grep'"'"'s result' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the test the fence names joins this table in the change that writes it, since `TestEveryTestNamedByARecordExists` holds every named test to exist | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/read/rewrite.go` |
| 2 — something selects it | `--rewrite` with `--ast-grep` |
| 3 — the caller can discover it | `--help`, `mrw instructions` (T3) |
| 4 — it is used | agents where ast-grep is installed |

## Mutation Log
(empty until execute)

## Invariants

- `--ast-grep` without `--rewrite` is unchanged.

## Risks

- ast-grep's JSON names a field differently across versions; the fake pins the shape and the manual run records the version.

## Stop Condition

ast-grep's `--json` with `--rewrite` does not carry both the byte range and the replacement.

## Out of Scope

- Rewrites ast-grep cannot express (permanent: boundary: the rewrite is ast-grep's)

## Verification Log
(empty until execute)
