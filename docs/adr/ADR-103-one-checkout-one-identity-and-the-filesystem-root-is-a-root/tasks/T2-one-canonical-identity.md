# Task ADR-103-T2: one canonical identity for a checkout

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `internal/links` (`Through`, `FS`, `OS`, `Real`, `IsRooted`, `Follow`); `state.absReal`, `check.confine`/`placed` and `read.astGrepRel` canonicalising through it
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every spelling keys one state directory`, `the walk moved unchanged`, `only the owned engine packages change`

## Goal

The link walk moves from `internal/rooted` to the leaf `internal/links`; `rooted` forwards `Real` and `IsRooted`
and keeps `win32Alias`/`win32Device`. `state.absReal` canonicalises through `links.Real`; `check.confine`,
`check.placed` and `read.astGrepRel` through `rooted.Real`. On Windows a junction to a checkout and the checkout
key one state directory.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/links/links.go`, `follow_windows.go`, `follow_other.go` | add | the walk, moved from `internal/rooted/links.go` (`:13-98`), with the seam's fields exported |
| `internal/links/links_test.go` | add | the test below |
| `internal/rooted/rooted.go`, `links.go`, `links_test.go` | edit | forward `Real`/`IsRooted`/`throughLinks`; keep the win32 refusals; the walk's tests stay, their fake helper builds a `links.FS`; drop `links_windows.go`/`links_other.go` |
| `internal/state/state.go` | edit | `absReal` through `links.Real` |
| `internal/check/check.go` | edit | `confine` (`:461-467`), `placed` (`:636-642`) through `rooted.Real` |
| `internal/read/astgrep.go` | edit | `astGrepRel` (`:229`) through `rooted.Real` |
| `cmd/mrw/junction103_windows_test.go` | add | the Windows test below |

## Ordered Steps

1. [S1] Write the failing test `TestRealFollowsAJunctionWhenTheWalkIsOn` (it names `links.Real`'s seam, which does not exist yet); confirm RED. [proof: mutation]
2. [S2] Move the walk; route the four callers. [proof: mutation] Mutants: `realVia` skips the walk; `state.absReal` back to `EvalSymlinks` alone (killed on the windows CI job only).
3. [S3] `TestAJunctionSpellingSharesTheCheckoutsState` passes on the windows CI job of the PR. [proof: human: the PR's windows jobs ran it green; no local Windows]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/links/ ./internal/rooted/ ./internal/state/ ./internal/check/ ./internal/read/ ./cmd/mrw/ -count=1 -timeout 300s -run 'TestRealFollowsAJunctionWhenTheWalkIsOn|TestTheLinkWalkReplacesALinkWithItsTarget|TestAJunctionSpellingSharesTheCheckoutsState' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestRealFollowsAJunctionWhenTheWalkIsOn \(' "$out" \
  && grep -qE '^--- PASS: TestTheLinkWalkReplacesALinkWithItsTarget \(' "$out" \
  && [ ! -e internal/rooted/links_other.go ] \
  && grep -q 'links.Real' internal/state/state.go \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestRealFollowsAJunctionWhenTheWalkIsOn` | `internal/links/links_test.go` | on a fake filesystem, `realVia` with the walk on resolves a junction spelling to its target, and with it off leaves it | — | S1, S2 |
| `TestTheLinkWalkReplacesALinkWithItsTarget` | `internal/rooted/links_test.go` | (existing, unchanged) the walk, now `links.Through`, replaces a link with its target | — | S2 |
| `TestAJunctionSpellingSharesTheCheckoutsState` | `cmd/mrw/junction103_windows_test.go` | (Windows) a junction to a checkout and the checkout give the same `state.Path(…, "seen.write.lock")` | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write and check resolves through `rooted`; every state access keys through `state.absReal` |
| 3 — the caller can discover it | the refusal no longer appears; one state directory per checkout (`mrw seen`) |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/links/links.go` · realVia skips the walk · acceptance-sha256:3332fba550b31d658f9107b7dd3507eac63e80120806c31d25487a2ae97fce9b · covers:every spelling keys one state directory
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package this record does not own changed · acceptance-sha256:3332fba550b31d658f9107b7dd3507eac63e80120806c31d25487a2ae97fce9b · covers:only the owned engine packages change
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/links/links.go` · realVia skips the walk · acceptance-sha256:afc8e1a64d241d800f34f678d138c9b4051905a9305e8b350a0bc27f1a661947 · covers:every spelling keys one state directory
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package this record does not own changed · acceptance-sha256:afc8e1a64d241d800f34f678d138c9b4051905a9305e8b350a0bc27f1a661947 · covers:only the owned engine packages change

## Invariants

- Off Windows, the state key for every root is unchanged.
- `/repo` still does not contain `/repo-backup`.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-103 task.

## Verification Log
- 2026-09-30 · f1d5996* · exit 1 · `set -o pipefail …` · acceptance-sha256:3332fba550b31d658f9107b7dd3507eac63e80120806c31d25487a2ae97fce9b · ms:653 · test-lock-sha256:8c256ace775f5e78e9e88b91b365000117dd3747d0d062ffd4963f91857254f6 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanVuY3Rpb24xMDNfd2luZG93c190ZXN0LmdvCVRlc3RBSnVuY3Rpb25TcGVsbGluZ1NoYXJlc1RoZUNoZWNrb3V0c1N0YXRlCTkyNzViMTllODMwNjRkZTE1Y2ZjODNiODIwZjQ3Y2M1ZDk3MGY5ZjlmZmM4YzhjYmExY2JiNjFiOTdhYmZlODgKYm9keQlpbnRlcm5hbC9saW5rcy9saW5rc190ZXN0LmdvCVRlc3RSZWFsRm9sbG93c0FKdW5jdGlvbldoZW5UaGVXYWxrSXNPbglhNzAwOWNmYWY0ZTRlZGIxMjg2ZDRlY2VlMzRjZjA4N2RjNWEyZjRhMmUzMTQ5ZjAzODgwYzE0MTNlOGRhZmZjCnVucHJvdmVuCWludGVybmFsL2xpbmtzL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrUmVwbGFjZXNBTGlua1dpdGhJdHNUYXJnZXQ
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state	0.199s [no tests to run]
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	0.200s [no tests to run]
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.191s [no tests to run]
  FAIL
  ```
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:3332fba550b31d658f9107b7dd3507eac63e80120806c31d25487a2ae97fce9b · ms:325
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:3332fba550b31d658f9107b7dd3507eac63e80120806c31d25487a2ae97fce9b · ms:337
- 2026-09-30 · human-observed · relock 2026-09-30: the Tests table first placed TestTheLinkWalkReplacesALinkWithItsTarget in internal/links; it stays in internal/rooted/links_test.go (the repo's record_test caught the wrong path). The table is corrected; no test body changed; approved
- 2026-09-30 · f1d5996* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3332fba550b31d658f9107b7dd3507eac63e80120806c31d25487a2ae97fce9b · ms:0 · test-lock-sha256:dd59166d6014c03620708e165d3d3dde74c75227bd975dd7ab64d7b2cc541364 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanVuY3Rpb24xMDNfd2luZG93c190ZXN0LmdvCVRlc3RBSnVuY3Rpb25TcGVsbGluZ1NoYXJlc1RoZUNoZWNrb3V0c1N0YXRlCTkyNzViMTllODMwNjRkZTE1Y2ZjODNiODIwZjQ3Y2M1ZDk3MGY5ZjlmZmM4YzhjYmExY2JiNjFiOTdhYmZlODgKYm9keQlpbnRlcm5hbC9saW5rcy9saW5rc190ZXN0LmdvCVRlc3RSZWFsRm9sbG93c0FKdW5jdGlvbldoZW5UaGVXYWxrSXNPbglhNzAwOWNmYWY0ZTRlZGIxMjg2ZDRlY2VlMzRjZjA4N2RjNWEyZjRhMmUzMTQ5ZjAzODgwYzE0MTNlOGRhZmZjCmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrQm91bmRzQUxvb3AJZTQ2YTViZDI5YTdmZWRjNzk5NGY5ZDVmYWMzMGIwZDMyMjYwNTBmYmJkYzUwYmIwMzE4Njc5NmUwZGI2ZWYzNgpib2R5CWludGVybmFsL3Jvb3RlZC9saW5rc190ZXN0LmdvCVRlc3RUaGVMaW5rV2Fsa0VuZHNBdEFOYW1lVGhhdENhbm5vdEV4aXN0CTZlZDNiZjY2NjZkMGU2ZjQwMjEwN2U1MDkyYmQ3NjFkYjc4YzdkYjMxYjhlZjdmYzdjYTEyMDhjZmVmZWYxOTcKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0VGhlTGlua1dhbGtMZWF2ZXNBUGxhY2Vob2xkZXJBbG9uZQk0N2E4N2RkYjhiY2U1ODhkNzljZTdhYTlhMTNkNGRhZTFlODkzYWFiYjY5YTYxODQ3MzlmZmM4NTg2MTQ1NTk2CmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrUmVmdXNlc0FDb21wb25lbnRJdENhbm5vdEV4YW1pbmUJYjE5YjM2N2VmOTg5Mzc1NDE2MjEyOGJlNGE1NTY0NDgyNWE1YTA1Y2YyYTc1OWMzZTliYmQ2YjczM2RkZjRmNwpib2R5CWludGVybmFsL3Jvb3RlZC9saW5rc190ZXN0LmdvCVRlc3RUaGVMaW5rV2Fsa1JlZnVzZXNBTGlua0l0Q2Fubm90UmVhZAkzM2U3MjMxMGI1MDc0YTBjMTY2MDNhMWQ1OGRmMTA3OWQzNDViMmFkNzg0OGJiMTQxMTcxMGJiYzQ0YjUxZDZiCmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrUmVmdXNlc0FTeW1saW5rSXRDYW5ub3RSZWFkCThkYmIwYzk4Y2NlMjUxODRhODQ4NjZhYTlmMmU1NGQ4NTBhZWFiOWUzMDZjZDkwMjE3NmJlNTNkMTVmZjhhZTEKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0VGhlTGlua1dhbGtSZXBsYWNlc0FMaW5rV2l0aEl0c1RhcmdldAliY2RjNGJmMjRlMmM4NDAzZjRhMzQ5ODA3YmZiZGM1ZDNmNjgzOTUyYmYyNzMwNTY2NmI3NzVkMWNjNGI4NTJjCmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrU3RvcHNBdEFNaXNzaW5nQ29tcG9uZW50CTFjZGEyYTFjNTFkNmU1N2Q0ZDAwZDQ3MjFjMGI3NWYxMGVlNWFmNWFiOGY0NWJiNTE5YWJmM2Q5NTY2NDRhNGUKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0V2luMzJBbGlhc0xlYXZlc0RvdEFuZERvdERvdEFsb25lCTA0YmIzY2Y5Nzg5ZGYxNGE1MzIxMDI5ZjJlZjkyNmExNDVlYTY0NGU3ZTQxMjY5M2IwZWQ4MzZlYTRlYjk2MTAKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0V2luMzJBbGlhc05hbWVzVGhlQ29tcG9uZW50V2luZG93c1dvdWxkUmVtYXAJYTQwY2I4YTdiZWJhZDFiNTE1MDI1MjdjM2M1NWM4NDU4MTY3MDU5NjU4ZDE0ODQ0NTMwMjMwMWRiMDMxZDUzZQ · test-lock-kind:replace
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:3332fba550b31d658f9107b7dd3507eac63e80120806c31d25487a2ae97fce9b · ms:394
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:afc8e1a64d241d800f34f678d138c9b4051905a9305e8b350a0bc27f1a661947 · ms:709
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:afc8e1a64d241d800f34f678d138c9b4051905a9305e8b350a0bc27f1a661947 · ms:403
