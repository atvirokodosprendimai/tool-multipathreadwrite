# Task ADR-096-T3: `iter add` names the real reason a path cannot be statted

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `iter add`'s refusal for an unstatable path; contract §185
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a loop is named as a loop`, `a denied directory is named as denied`, `a missing path keeps its sentence and hint`, `a reachable path is still added`, `nothing is added by a refused call`, `the existing iter refusals hold`, `a contract row drives the binary`, `the tree is gofmt-clean`, `no engine package changes`, `go.mod declares one requirement`

## Goal

`mrw iter add loop`, where `loop` is a link loop, refuses with the error the OS gave ("too many levels
of symbolic links") instead of "no such file: loop (quote a spec containing spaces)"; a path that is
simply not there keeps that sentence, whose hint is about exactly that case.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `iter add` (`:1821-1831`): `errors.Is(err, fs.ErrNotExist)` keeps the path in `missing`; any other `Stat` error is reported as the OS worded it, at exit 2, and nothing is added |
| `cmd/mrw/iter096_test.go` | add | the two tests |
| `scripts/contract.sh` | edit | §185 |

**What selects it:** the `add` case of the `change` closure (`cmd/mrw/main.go:1791-1833`; the closure
opens at `:1789`), the only production `os.Stat` in `cmd/mrw` (enumerated in the record). No new flag,
no new caller.

## Ordered Steps

1. [S1] Write the two tests below and confirm `TestIterAddNamesALinkLoopAsALoop` RED at `8cbb89e` on
   Linux and macOS (it gets "no such file"); `TestIterAddStillSaysNoSuchFileForAPathThatIsNotThere`
   passes there and must stay green. On Windows `rooted.Resolve` follows links itself and refuses the
   loop before any `Stat` ("leads through more than 255 links", `internal/rooted/links.go:77`), so
   the loop case is not red there and pins that boundary reason instead. [proof: mutation]
2. [S2] Split the `Stat` error at `cmd/mrw/main.go:1821`: `fs.ErrNotExist` keeps today's path and
   sentence; any other error is collected and reported with its own text (it names the path), after
   the boundary refusals and before the missing list, at exit 2, with nothing added. [proof: mutation]
   Mutants: every `Stat` error filed as missing (today's line); a missing path reported by its OS text,
   losing the hint; the valid sibling in `iter add a.go loop` added before the refusal.
3. [S3] Contract §185, driving `$MRW`: `iter add loop185` exits 2 naming "too many levels of symbolic
   links" and not "no such file"; paired, `iter add nosuch185` exits 2 with "no such file: nosuch185",
   and `iter add a.go` exits 0 and lists `a.go`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestIterAddNamesALinkLoopAsALoop|TestIterAddStillSaysNoSuchFileForAPathThatIsNotThere' -v 2>&1 | tee /tmp/adr096-T3.out \
  && missing=$(for t in TestIterAddNamesALinkLoopAsALoop TestIterAddStillSaysNoSuchFileForAPathThatIsNotThere; do grep -qE "^--- PASS: $t \(" /tmp/adr096-T3.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestIterAdd|TestTheIterRefusalKeepsTheVerb|TestAnIterVerbIsNotJudgedAsAPath' \
  && grep -q '^# 185\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

`internal/read` is left out of the engine clause on purpose: T1 and T2 own it, and T3 may run after
them.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestIterAddNamesALinkLoopAsALoop` | `cmd/mrw/iter096_test.go` | `iter add loop` fails without "no such file", naming "too many levels of symbolic links" where `rooted` does not follow links itself and "leads through more than 255 links" on Windows, where the boundary refuses it first; `iter add a.go loop` fails the same way and adds neither; where the process is not uid 0 and not on Windows, `iter add noperm/f` under a mode-000 directory names "permission denied"; the working set is unchanged after each | — | S1, S2 |
| `TestIterAddStillSaysNoSuchFileForAPathThatIsNotThere` | `cmd/mrw/iter096_test.go` | `iter add nosuch` still fails with "no such file: nosuch (quote a spec containing spaces)"; `iter add a.go` still adds | — | S1, S2 |

The loop case skips where the platform cannot create a link; the permission case skips under uid 0,
which ignores the mode (contract.sh's `skip` has the same reason).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the split in the `add` verb |
| 2 — something selects it | `mrw iter add`, driven by both tests through `rootCommand()` and by §185 through the built binary |
| 3 — the caller can discover it | the refusal is the OS's own sentence, naming the path |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-29 survey |

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: every Stat error filed as missing, as v1.31.0 did · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:a loop is named as a loop
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: a denied directory is filed as missing · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:a denied directory is named as denied
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: a missing path is reported by its OS text, losing the hint · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:a missing path keeps its sentence and hint
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the valid sibling in iter add a.go loop is added, and the call succeeds · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:nothing is added by a refused call
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · a reachable path is no longer added · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:a reachable path is still added
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · the boundary refusal is skipped, so a path outside the root is added · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:the existing iter refusals hold
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · main.go is not gofmt-clean · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:the tree is gofmt-clean
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/rooted/rooted.go` · an engine package changes · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · covers:no engine package changes

## Invariants

- A path outside the root is still refused with the boundary's reason, before any `Stat`
  (`TestIterAddRefusesAPathOutsideTheRoot`, `TestIterAddNamesTheBoundarysOwnReason`).
- A refused `iter add` adds nothing, whichever reason refused it.
- `internal/iter` is unchanged.

## Risks

- The OS words `ELOOP` and `EACCES` itself; the tests match the phrases Go's `syscall` table uses on
  Linux and macOS, and the permission case skips where the mode is not enforced. On Windows the loop
  never reaches `Stat`: `rooted.Resolve` refuses it with its own reason (`links.go:77`), which the
  test pins there rather than changing the boundary.

## Stop Condition

Stop and ask if an existing `iter` test must change to pass.

## Out of Scope

- `iter add` accepting a directory — the record's Out of Scope.
- The finders — T1 and T2.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:344 · test-lock-sha256:dfbe9862175515ce53692e440afd9b08e896032c1784e42504e5119b52c4bc8a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvaXRlcjA5Nl90ZXN0LmdvCVRlc3RJdGVyQWRkTmFtZXNBTGlua0xvb3BBc0FMb29wCWM0NmMzYTcyNzIzYTZjMDQ5NjBhMDA2ZTJmYWQzNjU2NTkyNDM3Y2VjY2U3N2U4ZDFjMzdjYmE4NGMyNTVkMzgKYm9keQljbWQvbXJ3L2l0ZXIwOTZfdGVzdC5nbwlUZXN0SXRlckFkZFN0aWxsU2F5c05vU3VjaEZpbGVGb3JBUGF0aFRoYXRJc05vdFRoZXJlCWIwMGJiYTkwYmU0ZmMzOTRiYzJiYmFkNTJlNDZiY2QyOWUyZWIxMTAyOTliZGUyMThiY2IwNjdmNjAzZmJiNjc
  ```
  --- last 10 line(s) of stdout (of 13 after folding 13 raw)
      iter096_test.go:40: iter add [a.go loop]: want "too many levels of symbolic links" and not "no such file":
          no such file: loop (quote a spec containing spaces)
      iter096_test.go:60: iter add noperm/f: exit 2, want 2 naming permission denied:
          no such file: noperm/f (quote a spec containing spaces)
  --- FAIL: TestIterAddNamesALinkLoopAsALoop (0.01s)
  === RUN   TestIterAddStillSaysNoSuchFileForAPathThatIsNotThere
  --- PASS: TestIterAddStillSaysNoSuchFileForAPathThatIsNotThere (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.135s
  FAIL
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:852
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:738
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:742
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:737
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:671
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:814
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:963
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:4df03b2e8babda892e91b3de4fc3f72054ebe39775265fae8e26475b56c4cf31 · ms:878
