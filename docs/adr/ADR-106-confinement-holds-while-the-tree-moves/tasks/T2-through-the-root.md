# Task ADR-106-T2: the write path goes through `os.Root`

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** Zy
**Produces:** `tree` (`internal/apply/tree.go`); the 15 sites routed through it; the seams taking `*tree`
**Consumes:** the swap fixtures (T1)
**Data dependency:** hermetic; the Windows CI shards are the only place junctions and root handles are exercised
**Proof map:** v1
**Rests-on:** `a swapped parent writes nothing outside the root`, `an in-root link stays writable`, `only the owned engine packages change`

## Goal

`Apply` opens the canonical root once as an `os.Root`, and every tree-changing operation after validation goes
through it with a resolved, root-relative path; the seams take the tree, so the failure-injection suite exercises
it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/tree.go` | add | `tree`: `openTree`, `rel`, `mkdirAll`, `createExcl`, `createTemp`, `rename`, `remove`, `lstat`, `readDir` |
| `internal/apply/apply.go` | edit | open the tree in `apply`; staging, probe, commit, cleanup and `noteIfLeft` through it; seams take `*tree` |
| `internal/apply/pathop.go` | edit | `commitPathOps` takes the tree; placeholder, aside, destination and undo through it; resolved parents |
| `internal/apply/inroot106_test.go` | add | the test below |
| `internal/apply/*_test.go` | edit | seam overrides gain `tr *tree` and pass it through |

## Ordered Steps

1. [S1] Write the failing test `TestInRootLinksStayWritableThroughTheRoot`; confirm it is green today and stays the guard for the change (an in-root relative directory link and an in-root absolute file link are written through; a junction on Windows). [proof: acceptance]
2. [S2] `tree` and the routing: every one of the 16 sites through the tree, paths resolved first; the Windows attributes set through the staging handle. [proof: mutation] Mutants: the unlink's rename bypasses the tree (killed by T1's late unlink swap); the commit-time destination `MkdirAll` bypasses the tree (T1's late rename swap); the staging-time destination `MkdirAll` bypasses the tree (T1's early rename swap). The attribute handle is Windows-only and is exercised by the Windows shards' ADR-076 attribute tests.
3. [S3] Seams take `*tree`; every override passes it through; the moved locks relocked with a note. [proof: human: the adr-lint run over every record after the relocks, and the list of relocked tasks in the PR]
4. [S4] Push; the six Windows CI shards pass before T3 starts. [proof: human: the CI run id and its six Windows shard verdicts]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAWriteThroughASwappedParentStaysInTheRoot|TestInRootLinksStayWritableThroughTheRoot' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteThroughASwappedParentStaysInTheRoot \(' "$out" \
  && grep -qE '^--- PASS: TestInRootLinksStayWritableThroughTheRoot \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteThroughASwappedParentStaysInTheRoot` | `internal/apply/swap106_test.go` | (T1) nothing outside the root changes through a swapped parent | — | S2 |
| `TestInRootLinksStayWritableThroughTheRoot` | `internal/apply/inroot106_test.go` | an edit through an in-root relative directory link, an edit of a file reached by an in-root absolute symlink, and an unlink and a rename beneath the directory link all apply, change the real files, and keep the links; on Windows the same through an in-root junction | — | S1, S2 |
| `TestATempMovedByAnotherProcessIsNotClaimed` | `internal/apply/inroot106_test.go` | a staged temp whose directory another process renames away before a failed commit is not named in `left_behind`, nothing outside the root is touched, and the moved temp stays where the other process put it (the record's Out of Scope boundary) | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/apply/tree.go` |
| 2 — something selects it | every write after validation goes through it |
| 3 — the caller can discover it | the refusal names the path |
| 4 — it is used | every plan that writes |

## Mutation Log
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/pathop.go` · the unlink's rename bypasses the root · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · covers:a swapped parent writes nothing outside the root
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/pathop.go` · the commit-time destination MkdirAll bypasses the root · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · covers:a swapped parent writes nothing outside the root
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/apply.go` · the staging-time destination MkdirAll bypasses the root · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · covers:a swapped parent writes nothing outside the root

## Invariants

- Receipts and exit codes are unchanged for every plan that did not meet a swap.
- The whole existing suite passes, on Linux, macOS and Windows.

## Risks

- `os.Root` behaves differently on Windows: S4 exists for it.

## Stop Condition

Stop and ask if an in-root link or junction cannot be written through the root, or a Windows shard fails for a reason the record did not foresee.

## Out of Scope

- The identity recheck (deferred: T3 in this record, `docs/adr/ADR-106-confinement-holds-while-the-tree-moves/tasks/T3-identity-recheck.md`)

## Verification Log
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:280 · test-lock-sha256:d9cf49768d448720e858d2becde445af2f54ea74f4a62bf8edcb7126f9152320 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2lucm9vdDEwNl90ZXN0LmdvCVRlc3RBVGVtcE1vdmVkQnlBbm90aGVyUHJvY2Vzc0lzTm90Q2xhaW1lZAlkYzc0MDg5YjhkMTgxZDA4MzVmNGRmNzE0OTFiN2RhYTU1OTg0MWMzOWViMzJlNGYwMTZjMjc4NTg4NzFjYTgxCmJvZHkJaW50ZXJuYWwvYXBwbHkvaW5yb290MTA2X3Rlc3QuZ28JVGVzdEluUm9vdExpbmtzU3RheVdyaXRhYmxlVGhyb3VnaFRoZVJvb3QJZDVjNDQ2ZWMxYmIxMjQzNGE2Njk0MWZlM2IzNTkzMWE0YmMyY2VkYTEyOTIxMzFhYTZhOWNhMWU0ODdjZDg3Mwpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlUZXN0QVdyaXRlVGhyb3VnaEFTd2FwcGVkUGFyZW50U3RheXNJblRoZVJvb3QJMzVjZmNhYWYyMTZhMjQzY2FkOGM1M2FlZTYxOTYzMzM2Njk1NjViOWFkYzU5ZDNmNjQxYWQ5ZTE2YzBhMWU5YQpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlhIHJlbmFtZSBzd2FwcGVkIGp1c3QgYmVmb3JlIGl0cyBjb21taXQJYjNjYjJiOTYxZDcyMzI5YTRjOTZlOWYxOTFjZDE4YmZhZWE2NGQ0NzJkNjAwYzk4NjRmMmIwODVjNjhmYjc2ZQpib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlhbiB1bmxpbmsJYjgzZmI3NDg0N2E4YTIyNWRkMDExMTIyYmIyZWE5NTc1NWY0NzM2OTM1Y2NkNDNlZWM0NGU1MjYzNzgzYzBiZApib2R5CWludGVybmFsL2FwcGx5L3N3YXAxMDZfdGVzdC5nbwlhbiB1bmxpbmsgc3dhcHBlZCBqdXN0IGJlZm9yZSBpdHMgcmVuYW1lCWZlNzNkM2FiMGQ2MmIzNTUzOTMzNjFiOGZiMTNjNTViZGUxZTg1ZjNjYTkyODJjY2QxZGRlZmFmZjdhMTUzNjk
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit
  --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot (0.01s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.083s
  ```
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:304
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit
  --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot (0.01s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.085s
  ```
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:334
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit
  --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot (0.01s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.073s
  ```
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:324
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit
  --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot (0.01s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.074s
  ```
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:366
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename
  === RUN   TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit
  --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot (0.01s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename's_destination (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/an_unlink_swapped_just_before_its_rename (0.00s)
      --- PASS: TestAWriteThroughASwappedParentStaysInTheRoot/a_rename_swapped_just_before_its_commit (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.074s
  ```
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:591
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:286
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:cec4e09569da6aa7b7ee2836fef2f41fe0bb7ce67ccb62e5d2d4a70064ad8a12 · ms:318
- 2026-09-30 · human-observed · S3 observed 2026-09-30: after the seam-signature change, adr-lint was run over every record in docs/adr; the moved locks were in ADR-052 T1/T2, ADR-054 T2, ADR-055 T1/T3, ADR-056 T2, ADR-066 T3, ADR-071 T2, ADR-076 T2/T3, ADR-086 T2 and ADR-105 T1, each relocked with a reviewed note naming this change; the PR lists them
