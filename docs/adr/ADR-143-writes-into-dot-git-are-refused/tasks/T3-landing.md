# Task ADR-143-T3: A path reopened by name is judged by where it lands now

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `tree.landsInGit`, `dotGitError`
**Consumes:** `rooted.HasGitComponent` (T2), `rooted.RealAsFarAsItExists`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a parent swapped for a link to .git after staging does not carry a rename into .git, and a .git refusal while staging is a failed hunk`

## Goal

Two findings of the Codex review of PR #388, both confirmed by a red test: `tree.rel` judged the lexical path, so a parent swapped for a RELATIVE link to `.git` after staging carried a rename destination into `.git`; and the refusal it returned while staging was an unclassified error, exit 2, where the record promises exit 1.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/tree.go` | edit | `landsInGit`: the parent resolved now plus the literal leaf; `dotGitError` |
| `internal/apply/apply.go` | edit | `targetCause` names `dotGitError` the target's cause, so a refusal while staging is a failed hunk |
| `internal/apply/swap143_test.go` | edit | the rename test, and the staging assertion on the create row |

## Ordered Steps

1. [S1] Write `TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut` and the `staging` assertion on `TestAWriteThroughAParentSwappedForDotGitStaysOut`'s create row. Confirm RED. [proof: acceptance]
2. [S2] `landsInGit` and its call in `tree.rel`. Mutant: it reports false. [proof: mutation]
3. [S3] `dotGitError` and its place in `targetCause`. Mutant: `targetCause` does not name it. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut|TestAWriteThroughAParentSwappedForDotGitStaysOut' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut \(' "$out" \
  && grep -qE '^--- PASS: TestAWriteThroughAParentSwappedForDotGitStaysOut \(' "$out" \
  && go test ./internal/apply/ ./internal/rooted/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut` | `internal/apply/swap143_test.go` | a rename destination under a parent swapped for a relative link to `.git` right before the commit rename writes nothing into `.git` and is not applied | none | S2 |
| `TestAWriteThroughAParentSwappedForDotGitStaysOut` | `internal/apply/swap143_test.go` | the create refused while staging comes back as a failed hunk and no error | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `tree.landsInGit`, `dotGitError` |
| 2 — something selects it | `tree.rel`, which every staging, rename and removal passes; `targetCause`, which every staging refusal asks |
| 3 — the caller can discover it | the refusal names `.git` and git as the tool |
| 4 — it is used | reproduced with ADR-106's seams; no telemetry (ADR-009). No contract row: the swap cannot be driven through the built binary, and validation's own refusal is §246 |

## Invariants

- ADR-106's confinement is unchanged: a path that leaves the root is refused as before.
- A name that is not `.git` is never refused by this check, and an entry that IS a link is judged by where its parent lands, not by where it leads: a rename or an unlink acts on the entry.

## Risks

- `landsInGit` resolves the parent on every `tree.rel` call, one more resolution per reopen. Writes are few and the cost is a `Lstat` per component.

## Stop Condition

Stop and ask if `tree.rel` is reached by a path mrw must legitimately open inside a `.git` (it is not: validation refuses every write there first).

## Out of Scope

- The instant between the check and the syscall (permanent: boundary: the finest grain mrw has; see the record)

## Mutation Log

## Verification Log
- 2026-10-10 · 409bc4e* · exit 1 · `set -o pipefail …` · acceptance-sha256:d24c11f8e148d4352fed62e1b544e5270e2f182339346d9ae72ea10135c9ef3a · ms:740 · test-lock-sha256:242b50a98bac26e031a006f1082e8c1171a18091e6478862bea020308addd241 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvc3dhcDE0M190ZXN0LmdvCVRlc3RBUmVuYW1lVGhyb3VnaEFQYXJlbnRTd2FwcGVkRm9yRG90R2l0QWZ0ZXJTdGFnaW5nU3RheXNPdXQJNzYzYzRlMGExNTJlZGNjNzg0NzM1YTQxNGQyNTk2MGNhMTY4OWFiYzA3YzZkMGQ4YmJiZTYzNjhlOGQ4NzY3ZApib2R5CWludGVybmFsL2FwcGx5L3N3YXAxNDNfdGVzdC5nbwlUZXN0QVdyaXRlVGhyb3VnaEFQYXJlbnRTd2FwcGVkRm9yRG90R2l0U3RheXNPdXQJNWJiZTViMzY0ZDQ0OTEyMzEyZjY1ZDQ0OGE3MTM3M2NhNjBkNDQ2NmQwYzA4MDYwNTRiMGEyY2MzN2FiMGMyYw
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
      --- PASS: TestAWriteThroughAParentSwappedForDotGitStaysOut/an_unlink (0.00s)
      --- PASS: TestAWriteThroughAParentSwappedForDotGitStaysOut/a_rename's_destination (0.00s)
  === RUN   TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut
      swap143_test.go:87: the rename changed .git: ["." "config" "hooks" "hooks/keep"] -> ["." "config" "hooks" "hooks/keep" "hooks/moved.txt"] (err <nil>)
      swap143_test.go:90: a rename through a parent swapped for .git was reported applied: <nil> {Root:/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut3487306280/001 DryRun:false Applied:true Files:[{Path:b.txt Created:false Written:true SHABefore:c150e5a8a604acebd8d15bd7bf8ea96b2874bdcc91dee6319977d353251283b0 SHAAfter: LinesFrom:1 LinesTo:0 Removed:true RenamedTo:sub/hooks/moved.txt Target:} {Path:sub/hooks/moved.txt Created:true Written:true SHABefore: SHAAfter:c150e5a8a604acebd8d15bd7bf8ea96b2874bdcc91dee6319977d353251283b0 LinesFrom:0 LinesTo:1 Removed:false RenamedTo: Target:}] Hunks:[{singleLineCode:false wrapTail:false Path:b.txt Addr:- Op:rename Status:ok Reason: Removed:0 Added:0 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Closer: Kind:}] Failed:0 Advisories:0 Hints:0 DirsCreated:[] LeftBehind:[] StrictSingleLine:0 StrictWouldRefuse:0}
      swap143_test.go:92: open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut3487306280/001/b.txt: no such file or directory
  --- FAIL: TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.156s
  FAIL
  ```
- 2026-10-10 · 409bc4e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d24c11f8e148d4352fed62e1b544e5270e2f182339346d9ae72ea10135c9ef3a · ms:1942
