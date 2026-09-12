# Task ADR-051-T2: `--format=apply_patch` on write; contract §82

**Depends-on:** T1
**Covers:** F-19, F-20, F-21, F-22, F-23, UC1-S1, UC3-S1, UC3-S2
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `write --format=apply_patch` (T2), contract §82 (T2)
**Consumes:** `ingest.CompileApplyPatch` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `write --format selecting the compiler`, `§82 driving the built binary`, `a git patch not compiling as apply_patch`

## Goal

`mrw write --format=apply_patch` is the served path. Contract §82 drives the built binary: a two-hunk apply_patch with one unread line writes nothing (exit 1, FAIL+skip). The same document without the flag is a bad native plan (exit 2). A git-shaped patch under the flag is refused.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `--format` on `writeCmd`; `plan` default; `apply_patch` calls `CompileApplyPatch` then `plan.Parse`. The selector. |
| `cmd/mrw/writehelp_test.go` | edit | `TestWriteHelpNamesApplyPatchFormat` — help names the flag and that a git patch is not one. |
| `cmd/mrw/writeformat_test.go` | add | `TestWriteFormatApplyPatchUnreadWritesNothing` through `rootCommand`. |
| `scripts/contract.sh` | edit | **§82** — reserve with `grep -oE '^# [0-9]+\. ' scripts/contract.sh \| sort -k2,2n \| tail -1`. Pair unread (exit 1, FAIL+skip, unchanged) with served (exit 0) and no-flag (exit 2). |

## Ordered Steps

1. [S1] Write `TestWriteHelpNamesApplyPatchFormat` and confirm it is RED: `write --help` / Description does not name `--format=apply_patch`. [proof: mutation]
2. [S2] Write `TestWriteFormatApplyPatchUnreadWritesNothing` through `rootCommand` and confirm it is RED. [proof: mutation]
3. [S3] Wire `--format` on `writeCmd`. `apply_patch` compiles then Parses. Unknown format and `git` are usage (exit 2). Confirm S1–S2 GREEN. Deleting the `CompileApplyPatch` call must fail S2 — that is rung 2. [proof: mutation]
4. [S4] Write §82 against a binary that does not yet honour `--format` and confirm it is RED (unread case must be exit 1 with FAIL+skip, not exit 2), then rebuild and confirm it is GREEN. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 82\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v \
    -run 'TestWriteHelpNamesApplyPatchFormat|TestWriteFormatApplyPatchUnreadWritesNothing' 2>&1 | tee /tmp/adr051-t2.out \
  && grep -q '^--- PASS: TestWriteHelpNamesApplyPatchFormat' /tmp/adr051-t2.out \
  && grep -q '^--- PASS: TestWriteFormatApplyPatchUnreadWritesNothing' /tmp/adr051-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr051-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWriteHelpNamesApplyPatchFormat` | `cmd/mrw/writehelp_test.go` | `write --help` names `--format=apply_patch` and that a git patch is not one | — | S1, S3 |
| `TestWriteFormatApplyPatchUnreadWritesNothing` | `cmd/mrw/writeformat_test.go` | The CLI flag compiles then Apply refuses the unread sibling; file unchanged; exit 1 | — | S2, S3 |
| `§82` | `scripts/contract.sh` | Built binary: unread apply_patch writes nothing (exit 1, FAIL+skip); served applies; no flag is exit 2 | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §82 |
| 2 — something selects it | `writeCmd` `--format`; deleting the `CompileApplyPatch` call fails S2 and §82 |
| 3 — the caller can discover it | `mrw write --help` |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 9f5c2bf* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · skip the compiler: --format=apply_patch parses the foreign document as a native plan and the unread case exits 2 · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-12 · 9f5c2bf* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · skip the compiler: --format=apply_patch parses the foreign document as a native plan and the unread case exits 2 · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-12 · 9f5c2bf* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · skip the compiler: --format=apply_patch parses the foreign document as a native plan and the unread case exits 2 · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-12 · 9f5c2bf* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · compiler is a pass-through: write parses apply_patch as a native plan and the unread case exits 2 · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd

## Invariants

- Default `--format` is `plan`. Native write is unchanged.
- Compile refusal is exit 2. An unread compiled hunk is exit 1.
- `maxInstructionsChars` stays 4096. Do not put `--format` in `guide.Shared()`.
- ADR-019 pick A stands.

## Risks

- §82's unread half passes on exit 2 (parse fail) when `--format` is ignored. Mitigation: assert exit 1, `FAIL`, `skip`, and `has not been read`.
- Teaching on Shared() overflows 4096. Mitigation: Description / Usage only.

## Stop Condition

Stop if the proposed fix is auto-detect, `--format=git`, raising 4096, or applying without Parse.

## Out of Scope

- Aider SEARCH/REPLACE
- MCP cargo tools
- `*** Delete File:` / `*** Move to:`
- README / AGENTS.md beyond `write --help` (the flag is the served path)

## Verification Log
- 2026-09-12 · 9f5c2bf* · exit 0 · `set -o pipefail …` · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd · ms:577
- 2026-09-12 · 9f5c2bf* · exit 0 · `set -o pipefail …` · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd · ms:675
- 2026-09-12 · 9f5c2bf* · exit 0 · `set -o pipefail …` · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd · ms:502
- 2026-09-12 · 9f5c2bf* · exit 0 · `set -o pipefail …` · acceptance-sha256:1ce3ddaba6d29e7541d303ae88967d949ccfb339d3165c6ef5c7d0709f2504bd · ms:670
