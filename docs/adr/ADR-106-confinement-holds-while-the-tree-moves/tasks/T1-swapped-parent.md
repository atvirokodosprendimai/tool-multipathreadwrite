# Task ADR-106-T1: a swapped parent escapes the root today

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the swap fixtures in `internal/apply/swap106_test.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a swapped parent writes nothing outside the root`

## Goal

A test that swaps a validated parent directory for a link out of the root — through `stageFileFn` after the
first file stages, and through `commitRenameFn` immediately before the affected operation — and asserts nothing
outside the root changes: red on the current code for an unlink and for a rename's destination, at both moments.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/swap106_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestAWriteThroughASwappedParentStaysInTheRoot`; confirm RED on `321e066`: the unlink removes the outside file, and the rename builds its directory and lands outside. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAWriteThroughASwappedParentStaysInTheRoot' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteThroughASwappedParentStaysInTheRoot \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteThroughASwappedParentStaysInTheRoot` | `internal/apply/swap106_test.go` | a plan editing `x.txt` and unlinking `sub/a.txt`, with `sub` swapped after `x.txt` stages for a link to an outside directory holding `a.txt`, fails and leaves the outside `a.txt` byte-identical; a plan editing `x.txt` and renaming `b.txt` to `sub/deep/b.txt`, swapped the same way, fails and creates nothing outside; skipped where symlinks cannot be made | — | S1 |
| `TestAWriteThroughASwappedParentStaysInTheRoot` | `internal/apply/swap106_test.go` | with `sub` swapped for a link to an outside directory — after `x.txt` stages, and again immediately before the unlink's rename or the rename's commit `MkdirAll` (the review of the record: a bypass of the root at those operations is reached only by the later swap) — a plan editing `x.txt` and unlinking `sub/a.txt` leaves the outside `a.txt` byte-identical, and a plan editing `x.txt` and renaming `b.txt` to `sub/deep/b.txt` creates nothing outside; each fails; skipped where symlinks cannot be made | — | S1 |
## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test file |
| 2 — something selects it | the fence |
| 3 — the caller can discover it | not applicable: a test |
| 4 — it is used | the fence runs it |

## Mutation Log
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/pathop.go` · the unlink's rename bypasses the root: the late unlink swap reaches outside · acceptance-sha256:88684f58f8d9e7fd1d523472c8ebc19cafc39851d30b2c3bd88d7d743ca20f86 · covers:a swapped parent writes nothing outside the root

## Invariants

- No production code changes in this task.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the test is green on `321e066`: the escape the record describes would then not be reached.

## Out of Scope

- The other ADR-106 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:88684f58f8d9e7fd1d523472c8ebc19cafc39851d30b2c3bd88d7d743ca20f86 · ms:269 · test-lock-sha256:50ffa48df3d993b8850735d1b7c26e46e3680995fb6287412a9e0c7ab701cbf3 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlUZXN0QVdyaXRlVGhyb3VnaEFTd2FwcGVkUGFyZW50U3RheXNJblRoZVJvb3QJZmI4MTU5ZDM1MDYzZjcyMTE0YzVhZTkyMTY4MTI3YWUyMDg0YjA2Yzc5ZTUzNWUyYWI0MDRhNDZiZmIzZGNhYgpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlhbiB1bmxpbmsJYjgzZmI3NDg0N2E4YTIyNWRkMDExMTIyYmIyZWE5NTc1NWY0NzM2OTM1Y2NkNDNlZWM0NGU1MjYzNzgzYzBiZA
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
      swap106_test.go:71: open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAWriteThroughASwappedParentStaysInTheRootan_unlink430453314/002/a.txt: no such file or directory
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination
      swap106_test.go:94: the rename reached outside the root: ["."] -> ["." "deep" "deep/b.txt"] (err <nil>)
      swap106_test.go:97: a rename through a swapped parent was reported applied: <nil> {Root:/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAWriteThroughASwappedParentStaysInTheRoota_renames_destina1440400466/001 DryRun:false Applied:true Files:[{Path:x.txt Created:false Written:true SHABefore:86dc03602dcf385217216784784a8ecf20e6400decc3208170b12fcb0afb6698 SHAAfter:f7d380dc845bc8c024347c911213f790730ccb6eccf173248bdf8f7417314aa9 LinesFrom:5 LinesTo:5 Removed:false RenamedTo: Target:} {Path:b.txt Created:false Written:true SHABefore:c150e5a8a604acebd8d15bd7bf8ea96b2874bdcc91dee6319977d353251283b0 SHAAfter: LinesFrom:1 LinesTo:0 Removed:true RenamedTo:sub/deep/b.txt Target:} {Path:sub/deep/b.txt Created:true Written:true SHABefore: SHAAfter:c150e5a8a604acebd8d15bd7bf8ea96b2874bdcc91dee6319977d353251283b0 LinesFrom:0 LinesTo:1 Removed:false RenamedTo: Target:}] Hunks:[{singleLineCode:false wrapTail:false Path:x.txt Addr:1 Op:replace Status:ok Reason: Removed:1 Added:1 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Kind:} {singleLineCode:false wrapTail:false Path:b.txt Addr:- Op:rename Status:ok Reason: Removed:0 Added:0 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Kind:}] Failed:0 Advisories:0 DirsCreated:[sub/deep] LeftBehind:[] StrictSingleLine:0 StrictWouldRefuse:0}
  --- FAIL: TestAWriteThroughASwappedParentStaysInTheRoot (0.00s)
      --- FAIL: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink (0.00s)
      --- FAIL: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.072s
  FAIL
  ```
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:88684f58f8d9e7fd1d523472c8ebc19cafc39851d30b2c3bd88d7d743ca20f86 · ms:304
- 2026-09-30 · 321e066* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:88684f58f8d9e7fd1d523472c8ebc19cafc39851d30b2c3bd88d7d743ca20f86 · ms:0 · test-lock-sha256:af6f036a92719ef7fc577bc52248aac61d27ff75d485df9c9105e20e9a27e329 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlUZXN0QVdyaXRlVGhyb3VnaEFTd2FwcGVkUGFyZW50U3RheXNJblRoZVJvb3QJMzVjZmNhYWYyMTZhMjQzY2FkOGM1M2FlZTYxOTYzMzM2Njk1NjViOWFkYzU5ZDNmNjQxYWQ5ZTE2YzBhMWU5YQpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlhIHJlbmFtZSBzd2FwcGVkIGp1c3QgYmVmb3JlIGl0cyBjb21taXQJYjNjYjJiOTYxZDcyMzI5YTRjOTZlOWYxOTFjZDE4YmZhZWE2NGQ0NzJkNjAwYzk4NjRmMmIwODVjNjhmYjc2ZQpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlhbiB1bmxpbmsJYjgzZmI3NDg0N2E4YTIyNWRkMDExMTIyYmIyZWE5NTc1NWY0NzM2OTM1Y2NkNDNlZWM0NGU1MjYzNzgzYzBiZApib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlhbiB1bmxpbmsgc3dhcHBlZCBqdXN0IGJlZm9yZSBpdHMgcmVuYW1lCWZlNzNkM2FiMGQ2MmIzNTUzOTMzNjFiOGZiMTNjNTViZGUxZTg1ZjNjYTkyODJjY2QxZGRlZmFmZjdhMTUzNjk · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: after the red run TestAWriteThroughASwappedParentStaysInTheRoot gained the two late swaps the Codex review of the record asked for (immediately before the unlink's rename and before the rename's commit), its stageFileFn helper takes the tree, and the swap moved into swapSub; every earlier assertion kept; approved
