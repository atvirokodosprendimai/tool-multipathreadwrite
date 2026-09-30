# Task ADR-104-T1: the check tail in bounded memory

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the streaming `lastLines`; contract §200
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the tail is read in bounded memory`, `the tail keeps its meaning`, `a contract row drives the binary`, `only the owned engine packages change`

## Goal

`lastLines` streams the check log, keeps a ring of the last `n` lines and at most 4 KiB of each (a longer line ends
` … [N more bytes]`), and returns the same lines and `truncated_lines` as the whole-file split did for every
input under the cap.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `lastLines` (`:746-759`) streams; `maxTailLineBytes` |
| `internal/check/tail104_test.go` | add | the test below |
| `cmd/mrw/main.go` | edit | `reportSteps`: a passing step whose tail cut a line names its kept log (the review of #295) |
| `cmd/mrw/steplog104_test.go` | add | the test for that receipt |
| `scripts/contract.sh` | edit | §200 |

## Ordered Steps

1. [S1] Write the failing test `TestTheCheckTailReadsALogInBoundedMemory`; confirm RED on `f911dd7`. [proof: mutation]
2. [S2] Stream the tail. [proof: mutation] Mutants: the per-line cap removed; the ring keeps the first lines instead of the last.
3. [S3] Contract §200, driving `$MRW`: a check whose last line is 10,000 characters shows it capped with ` … [` in the receipt's tail; the pair, a short last line, is shown whole. [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §200's rows printed]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ -count=1 -timeout 300s -run 'TestTheCheckTailReadsALogInBoundedMemory|TestTheTailCountsWhatItCutAndKeepsTheLog' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCheckTailReadsALogInBoundedMemory \(' "$out" \
  && grep -qE '^--- PASS: TestTheTailCountsWhatItCutAndKeepsTheLog \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAPassingStepNamesTheLogItKept' -v 2>&1 | tee -a "$out" \
  && grep -qE '^--- PASS: TestAPassingStepNamesTheLogItKept \(' "$out" \
  && grep -q '^# 200\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCheckTailReadsALogInBoundedMemory` | `internal/check/tail104_test.go` | on edge inputs (empty, one empty line, no trailing newline, blank lines, CRLF, fewer lines than the tail) the result equals the old whole-file split; a 50 MB log gives the last 30 lines and the right count while allocating under 16 MB; a 1 MB line comes back as 4 KiB plus ` … [N more bytes]` | — | S1, S2 |
| `TestTheTailCountsWhatItCutAndKeepsTheLog` | `internal/check/tail104_test.go` | (the review of #295) a line cut inside a rune counts the stray byte (`[2 more bytes]`); `tail_lines` 1<<30 on a one-line log allocates under 1 MB; a passing check whose tail cut a line keeps its log, holding the whole line | — | S2 |
| `TestAPassingStepNamesTheLogItKept` | `cmd/mrw/steplog104_test.go` | (the second review of #295) a passing `--then-sh` step whose one line was cut prints `full output: <log>` after the cut line, and that log holds the whole line | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every check, every MCP request, every read and ast-grep run goes through it |
| 3 — the caller can discover it | the refusal or the marker names the limit |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/check/check.go` · the per-line cap removed · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · covers:the tail is read in bounded memory
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/check/check.go` · the ring keeps the first lines instead of the last · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · covers:the tail keeps its meaning
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/check/check.go` · every line allocates a large string · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · covers:the tail is read in bounded memory
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package this record does not own changed · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · covers:only the owned engine packages change
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/check/check.go` · the per-line cap removed · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · covers:the tail is read in bounded memory
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/check/check.go` · the ring keeps the first lines instead of the last · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · covers:the tail keeps its meaning
- 2026-09-30 · f2b0ee2* · mutant inconclusive · exit 1 · `internal/check/check.go` · a passing check with a cut line deletes its log again · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · covers:the tail keeps its meaning
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/check/check.go` · the marker undercounts a split rune again · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · covers:the tail keeps its meaning
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package this record does not own changed · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · covers:only the owned engine packages change
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/check/check.go` · a passing check with a cut line deletes its log again · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · covers:the tail keeps its meaning
- 2026-09-30 · dfaebc9* · mutant killed · exit 1 · `cmd/mrw/main.go` · a passing step whose tail cut a line names no log (the second review of #295) · acceptance-sha256:a1dd3f253c04daaad76cf1c11f53b3eb3d871549c1470ca0da8e3f39751b2c3e

## Invariants

- Under the limit, every answer is byte-identical to before.
- Exit codes keep their meanings.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-104 tasks, each in its own file.

## Verification Log
- 2026-09-30 · f911dd7* · exit 1 · `set -o pipefail …` · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · ms:208 · test-lock-sha256:076fc8de84ddbfd412a2c7fc86f48b2c083eff95525c854dbcc57f67f42ea305 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL3RhaWwxMDRfdGVzdC5nbwlUZXN0VGhlQ2hlY2tUYWlsUmVhZHNBTG9nSW5Cb3VuZGVkTWVtb3J5CTEyZTFiYTVkM2UwNTI3YzEzMGFlMzQzMTAyNzExOThkMzQxMjJmZmUwYzkxN2UxYjFlODE4MjJhMzBjMTRkYWU
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check.test]
  internal/check/tail104_test.go:69:57: undefined: maxTailLineBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check [build failed]
  FAIL
  ```
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · ms:443
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · ms:393
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · ms:407
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:122f86acf047d054d0c6b672f3633541fdf2aea140be04e63bc9b4cb74bb489d · ms:426
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped in the ADR-104 worktree (base f911dd7), exit 0, with §200 printed: a failing check whose last line is 10,000 characters exits 3 and its receipt's tail shows the line capped with ' … [N more bytes]'; the pair, a short last line, is shown whole
- 2026-09-30 · human-observed · relock 2026-09-30 (Codex review of #295): lastLines now also returns whether it cut a line, so TestTheCheckTailReadsALogInBoundedMemory takes a third value at its three calls; every assertion kept. TestTheTailCountsWhatItCutAndKeepsTheLog is new; approved
- 2026-09-30 · f2b0ee2* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · ms:0 · test-lock-sha256:ccdca82504e82dd022fe774d575be39638c7325821a24ec70cb39f85f81fd712 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL3RhaWwxMDRfdGVzdC5nbwlUZXN0VGhlQ2hlY2tUYWlsUmVhZHNBTG9nSW5Cb3VuZGVkTWVtb3J5CTIwZjFmODE3ZWNjZmZiZjk0MzMwZmY1NGI4YWI2MmE3YzlhMjhhYzE3ODAwOWJlYmIxYjQzMGQ1Y2U3YzBlYTEKYm9keQlpbnRlcm5hbC9jaGVjay90YWlsMTA0X3Rlc3QuZ28JVGVzdFRoZVRhaWxDb3VudHNXaGF0SXRDdXRBbmRLZWVwc1RoZUxvZwljZmI1YmNmMGZkZmZjN2Y5MGFmOGFlMWNjYWNhNDk1YjQ4ZjBkYzE4N2IwYjU4ZjYzYTE0YmRkZTM0OTI0NTVh · test-lock-kind:replace
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · ms:500
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · ms:456
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · ms:472
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · ms:487
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · ms:499
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:68862897f9f0accf369b8f9feac3cebb9c9d7d6217c302bb07513eff1b000e14 · ms:643
- 2026-09-30 · human-observed · S3 observed again 2026-09-30 after the review of #295: ./scripts/contract.sh exit 0 with §200's new row printed — a passing check with a 10,000-character line exits 0 and keeps its log, which the cut line's marker points at
- 2026-09-30 · dfaebc9* · exit 0 · `set -o pipefail …` · acceptance-sha256:a1dd3f253c04daaad76cf1c11f53b3eb3d871549c1470ca0da8e3f39751b2c3e · ms:836
