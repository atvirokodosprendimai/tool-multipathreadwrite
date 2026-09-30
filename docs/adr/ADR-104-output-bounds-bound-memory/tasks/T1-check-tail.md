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
| `scripts/contract.sh` | edit | §200 |

## Ordered Steps

1. [S1] Write the failing test `TestTheCheckTailReadsALogInBoundedMemory`; confirm RED on `f911dd7`. [proof: mutation]
2. [S2] Stream the tail. [proof: mutation] Mutants: the per-line cap removed; the ring keeps the first lines instead of the last.
3. [S3] Contract §200, driving `$MRW`: a check whose last line is 10,000 characters shows it capped with ` … [` in the receipt's tail; the pair, a short last line, is shown whole. [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §200's rows printed]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ -count=1 -timeout 300s -run 'TestTheCheckTailReadsALogInBoundedMemory' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCheckTailReadsALogInBoundedMemory \(' "$out" \
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
