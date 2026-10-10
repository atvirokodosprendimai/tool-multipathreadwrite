# Task ADR-143-T2: A path reopened by name is judged for `.git` again

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `rooted.HasGitComponent`, the check in `tree.rel`
**Consumes:** `inGit` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a directory swapped for a link to .git after validation does not carry a create, an unlink or a rename destination into .git`

## Goal

Decision 6 of the record, with a test that fails before it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/gitdir.go` | edit | `HasGitComponent` |
| `internal/apply/tree.go` | edit | `tree.rel` refuses a path with a `.git` component |
| `internal/apply/swap143_test.go` | add | the test, on ADR-106's swap seams |

## Ordered Steps

1. [S1] Write `TestAWriteThroughAParentSwappedForDotGitStaysOut` (a create, an unlink and a rename destination, each after `stageFileFn` swaps `sub` for a link to `.git`). Confirm RED.
2. [S2] `HasGitComponent` and its call in `tree.rel`. Mutant: the check removed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAWriteThroughAParentSwappedForDotGitStaysOut' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteThroughAParentSwappedForDotGitStaysOut \(' "$out" \
  && go test ./internal/apply/ ./internal/rooted/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteThroughAParentSwappedForDotGitStaysOut` | `internal/apply/swap143_test.go` | after a parent is swapped for a link to `.git`, a create, an unlink and a rename destination write nothing into `.git` and the plan is not applied | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.HasGitComponent` |
| 2 — something selects it | `tree.rel`, which every staging, rename and removal passes |
| 3 — the caller can discover it | the refusal names `.git` and git as the tool |
| 4 — it is used | reproduced in the test with ADR-106's seams; no telemetry (ADR-009) |

## Invariants

- ADR-106's confinement is unchanged: a path that leaves the root is refused as before.
- A name that is not `.git` is never refused by this check.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if `tree.rel` is reached by a path mrw must legitimately open inside a `.git` (it is not: validation refuses every write there first).

## Out of Scope

- The instant between the check and the syscall (permanent: boundary: the finest grain mrw has; see the record)

## Mutation Log
- 2026-10-10 · daa372a* · mutant killed · exit 1 · `internal/apply/tree.go` · S2: tree.rel does not refuse a path with a .git component · acceptance-sha256:3ba8bd5505ffd39adc91154a417b47cdc44457997a1c6278c5d94e74c7a17e52 · covers:a directory swapped for a link to .git after validation does not carry a create, an unlink or a rename destination into .git

## Verification Log
- 2026-10-10 · daa372a* · exit 1 · `set -o pipefail …` · acceptance-sha256:3ba8bd5505ffd39adc91154a417b47cdc44457997a1c6278c5d94e74c7a17e52 · ms:1205 · test-lock-sha256:f2ddeaf4d1206422548b6095f28d5c3887696e290abb8e489755f19fe8ab2df4 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvc3dhcDE0M190ZXN0LmdvCVRlc3RBV3JpdGVUaHJvdWdoQVBhcmVudFN3YXBwZWRGb3JEb3RHaXRTdGF5c091dAkyNjU4NmE0OTQ5Y2RjZjUyY2MyODk1OGMxOTM2MGFjNjlhZjNkYzAwYzlkMmE3N2M3YmFhMjQzZjY2OWU4ZjJl
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  === RUN   TestAWriteThroughAParentSwappedForDotGitStaysOut/a_rename's_destination
      swap143_test.go:37: the plan changed .git: ["." "config" "hooks" "hooks/keep"] -> ["." "config" "hooks" "hooks/keep" "hooks/moved.txt"] (err <nil>)
      swap143_test.go:46: a write through a parent swapped for .git was reported applied: <nil> {Root:/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAWriteThroughAParentSwappedForDotGitStaysOuta_renames_dest4086425000/001 DryRun:false Applied:true Files:[{Path:x.txt Created:false Written:true SHABefore:86dc03602dcf385217216784784a8ecf20e6400decc3208170b12fcb0afb6698 SHAAfter:f7d380dc845bc8c024347c911213f790730ccb6eccf173248bdf8f7417314aa9 LinesFrom:5 LinesTo:5 Removed:false RenamedTo: Target:} {Path:b.txt Created:false Written:true SHABefore:c150e5a8a604acebd8d15bd7bf8ea96b2874bdcc91dee6319977d353251283b0 SHAAfter: LinesFrom:1 LinesTo:0 Removed:true RenamedTo:sub/hooks/moved.txt Target:} {Path:sub/hooks/moved.txt Created:true Written:true SHABefore: SHAAfter:c150e5a8a604acebd8d15bd7bf8ea96b2874bdcc91dee6319977d353251283b0 LinesFrom:0 LinesTo:1 Removed:false RenamedTo: Target:}] Hunks:[{singleLineCode:false wrapTail:false Path:x.txt Addr:1 Op:replace Status:ok Reason: Removed:1 Added:1 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Closer: Kind:} {singleLineCode:false wrapTail:false Path:b.txt Addr:- Op:rename Status:ok Reason: Removed:0 Added:0 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Closer: Kind:}] Failed:0 Advisories:0 Hints:0 DirsCreated:[] LeftBehind:[] StrictSingleLine:0 StrictWouldRefuse:0}
  --- FAIL: TestAWriteThroughAParentSwappedForDotGitStaysOut (0.01s)
      --- FAIL: TestAWriteThroughAParentSwappedForDotGitStaysOut/a_create (0.01s)
      --- FAIL: TestAWriteThroughAParentSwappedForDotGitStaysOut/an_unlink (0.00s)
      --- FAIL: TestAWriteThroughAParentSwappedForDotGitStaysOut/a_rename's_destination (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.323s
  FAIL
  ```
- 2026-10-10 · daa372a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ba8bd5505ffd39adc91154a417b47cdc44457997a1c6278c5d94e74c7a17e52 · ms:2411
