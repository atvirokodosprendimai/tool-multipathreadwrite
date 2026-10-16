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
out=$(go test ./internal/read/ ./cmd/mrw/ -count=1 -v -run '^TestARewrite' 2>&1) \
  && for n in TestARewritePlanGroupsSpansAndAnchorsThem TestARewriteReadAndWriteApplyAstGrepsResult TestARewriteWithoutItsPartnersIsAUsageError TestARewriteOfAPartialScanIsIncompleteAndWritesNothing TestARewriteIsRefusedWhenAFileChangesBetweenTwoRuns; do grep -qE "^--- PASS: $n " <<<"$out" || exit 1; done \
  && go build -o bin/mrw ./cmd/mrw && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a rewrite read piped to write, and a rewrite write, give ast-grep'"'"'s result' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARewritePlanGroupsSpansAndAnchorsThem` | `internal/read/rewrite161_test.go` | recorded and fake hits become anchored whole-line hunks: single- and multi-line matches, two matches on one line, touching spans merged, the anchor and path round-tripped through the header parser, each span and the line after it licensed and not shown, an empty replacement, a CRLF body, a multibyte line; a missing `replacement` or `byteOffset`, a range outside the file, a file that is not what ast-grep read, an unnameable first line, and nested (overlapping) matches each refused by name | none | S2, S3, S4 |
| `TestARewriteReadAndWriteApplyAstGrepsResult` | `cmd/mrw/rewrite161_test.go` | the built binary with a fake ast-grep: `read --ast-grep --rewrite` prints the plan on stdout and the count on stderr and passes `--rewrite=R --json=stream` on, the plan piped to `write -` and `write --ast-grep --rewrite` give the same file, a dry run writes nothing, nested matches exit 1 with nothing printed or written, a rewrite that changes no line is `incomplete:`, no match is `no match:` | none | S2, S3, S4, S5 |
| `TestARewriteWithoutItsPartnersIsAUsageError` | `cmd/mrw/rewrite161_test.go` | `--rewrite` without `--ast-grep`, beside `--grep`, `--sub`, `--count`, `--stat`, `--quiet`, `--max-cols`, `--max-lines`, `-N` or a range, and on `write` with a PLAN `-`, `--format`, `--create` or `--grep --sub`, is exit 2 | none | S2 |
| `TestARewriteOfAPartialScanIsIncompleteAndWritesNothing` | `cmd/mrw/rewrite161_test.go` | an ast-grep that answers valid hits at exit 0 and names an unreadable path on stderr (as 0.45.3 does) makes the rewrite partial: read prints the plan and exits 1 `incomplete:`, write exits 1 and writes nothing, the diagnostic named | none | S2 |
| `TestARewriteIsRefusedWhenAFileChangesBetweenTwoRuns` | `cmd/mrw/rewrite161_test.go` | the plan is bound to one version of each file: a file edited while the second ast-grep run reads it, and one whose second run finds no match, are named and nothing is written | none | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/read/rewrite.go` |
| 2 — something selects it | `--rewrite` with `--ast-grep` |
| 3 — the caller can discover it | `--help`, `mrw instructions` (T3) |
| 4 — it is used | agents where ast-grep is installed |

## Mutation Log
(empty until execute)
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/rewrite.go` · S2: the overlap refusal disabled; nested matches are applied over each other · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/rewrite.go` · S4: the line after each span is no longer licensed; a multi-line replace is refused by ADR-052 · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite licenses each span and the line after it
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/rewrite.go` · S3: spans that touch are no longer merged, so two hunks sit on adjacent lines · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: --rewrite on read is not wired to the plan builder · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: write --ast-grep --rewrite never builds the rewrite plan · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9
- 2026-10-16 · 88e9961e* · mutant survived · exit 0 · `internal/read/astgrep.go` · P1: ast-grep stderr diagnostics are no longer named, so a partial scan applies at exit 0 · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-16 · 88e9961e* · mutant survived · exit 0 · `internal/read/rewrite.go` · P2: the bytes read around the second ast-grep run are no longer compared · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/rewrite.go` · P3: a CR is trimmed from the last piece of a CRLF span too, dropping the replacement's own · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/astgrep.go` · P1: ast-grep stderr diagnostics are no longer named, so a partial scan applies at exit 0 · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/rewrite.go` · P2: the bytes read around the second ast-grep run are no longer compared · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · covers:a rewrite plan groups spans and anchors them
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/rewrite.go` · P1 (Codex round 2): two non-empty, differing hit sets keep the file and drop no Problem · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/rewrite.go` · P1 (Codex round 2): a file ast-grep silently skipped as unreadable is not named, and the rest is written · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/astgrep.go` · P1 (Codex round 3): a killed ast-grep run's valid prefix is planned and written · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9
- 2026-10-16 · 88e9961e* · mutant killed · exit 1 · `internal/read/astgrep.go` · P1 (Codex round 1): an ast-grep diagnostic on stderr is dropped, so a partial scan applies; now bound by the CLI tests in the widened fence · acceptance-sha256:c0bcce56c44cb3f1ef45ff0c4cfa940b81b55ce06c8cd39c6fffb51cf63ceffb

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
- 2026-10-16 · 88e9961e* · exit 1 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:242 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:80945
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:81590
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:81744
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:80724
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:80372
- 2026-10-16 · human-observed · S5 observed and signed off by the coordinator, 2026-10-16: ast-grep 0.45.3 (npx cache of @ast-grep/cli@0.45.3), mrw built from adr-161-t2-rewrite, scratch tree: mrw read --ast-grep 'foo($A, $B)' --rewrite 'bar($B, $A)' . printed @@ a.go 4-8 replace (body=3, anchor tab foo(1, 2)) and @@ b.go 3 replace, planned 2 sites in 2 files, exit 0; mrw write --no-check with the same flags applied 2 hunks exit 0; diff a.go lines 4-8 to bar(2, 1) / x := bar(4, 3) / s := naive-s; bar(s-hat, s) and b.go line 3 to bar(8, 7); bar(10, 9); the nested foo(foo(5, 6), 7) tree is refused, overlap named, nothing planned.
- 2026-10-16 · human-observed · S5 observed and signed off by the coordinator, 2026-10-16 (this row replaces the wording of the row above): ast-grep 0.45.3 from the npx cache of @ast-grep/cli@0.45.3 and mrw built from adr-161-t2-rewrite, in a scratch tree: read --ast-grep 'foo($A, $B)' --rewrite 'bar($B, $A)' printed two hunks (a.go 4-8, b.go 3), exit 0; write with the same flags applied both, exit 0, and the diff is the one ast-grep rewrites to; the nested foo(foo(5, 6), 7) tree is answered with its overlap named and an empty plan, exit 1, as designed.
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:88719
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:78396
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:79886
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:79159
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:81186
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:83556
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:79768
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:82629
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:134921
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae0b7d5b70bd6618de0c872e4a587c2d215a8848d7c14ce808c5fd26dec0dcd9 · ms:87559
- 2026-10-16 · 88e9961e* · exit 0 · `set -o pipefail …` · acceptance-sha256:c0bcce56c44cb3f1ef45ff0c4cfa940b81b55ce06c8cd39c6fffb51cf63ceffb · ms:108249
