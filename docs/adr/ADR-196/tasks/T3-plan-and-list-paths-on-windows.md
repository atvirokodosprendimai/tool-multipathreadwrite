# Task ADR-196-T3: a plan file or a --files-from list spelled as a Windows alias is refused

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `windowsKeeps`, `keepsOn`; `TestAWindowsAliasOfAPlanOrListIsRefused`
**Consumes:** the guards' removal (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the rule refuses a trailing dot or space on Windows only`, `both open sites call it`

## Goal

Decision 6: on Windows, a plan file or a `--files-from` list whose path has a component ending in a dot or a space is refused before it is opened, so it never applies or reads the file Win32 would open instead.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/keep196.go` | add | `windowsKeeps` and `keepsOn`, reusing `unkeptComponent` (ADR-168) |
| `cmd/mrw/main.go` | edit | `windowsKeeps` before the plan file's `os.Open` and before the `--files-from` list's |
| `cmd/mrw/keep196_test.go` | add | the rule on every platform; on Windows, through the commands |

## Ordered Steps

1. [S1] Write `TestAWindowsAliasOfAPlanOrListIsRefused`: `keepsOn("windows", …)` refuses `p.mrw `, `p.mrw.`, `d.\p.mrw` and `d /p.mrw` and accepts `p.mrw`, `d\p.mrw`, `./p.mrw`, `..\p.mrw` and ` p.mrw`; `keepsOn("linux", "p.mrw ")` accepts; on Windows, `write 'p.mrw '` exits non-zero and does not apply `p.mrw`, `read --files-from 'list '` and `--files-from='list '` do not read `list`, and the exact spellings do. See it RED (no `keepsOn`).
2. [S2] Add `keep196.go` and the two calls. Mutants, each killed: the GOOS test inverted so the rule never fires; one call site removed (the fence counts both). [proof: mutation]
3. [S3] `GOOS=windows go vet ./cmd/mrw`; the Windows CI job runs the command half, which no other platform can. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^TestAWindowsAliasOfAPlanOrListIsRefused$' 2>&1) \
  && grep -qE '^--- PASS: TestAWindowsAliasOfAPlanOrListIsRefused \(' <<<"$out" \
  && [ "$(grep -c 'windowsKeeps(' cmd/mrw/main.go)" -eq 2 ] \
  && GOOS=windows go vet ./cmd/mrw
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWindowsAliasOfAPlanOrListIsRefused` | `cmd/mrw/keep196_test.go` | the rule for Windows on every platform; the commands on Windows | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `windowsKeeps` |
| 2 — something selects it | `mrw write PLAN` and `mrw read --files-from LIST` on Windows; the fence counts both calls |
| 3 — the caller can discover it | the refusal names the component and the fix |
| 4 — it is used | the Windows CI job; no telemetry (ADR-009) |

## Mutation Log
- 2026-11-10 · 012f56e3* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · the rule never fires, on Windows too, so the windows cases must go red · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · covers:the rule refuses a trailing dot or space on Windows only
- 2026-11-10 · 012f56e3* · mutant killed · exit 1 · `cmd/mrw/main.go` · the list is opened without the rule, so the call count must go red · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · covers:both open sites call it

## Verification Log
- 2026-11-10 · 012f56e3* · exit 1 · `set -o pipefail …` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:303 · test-lock-sha256:66b9c53e1d5e88a096608bbab9ed9e6986aab9a4970e504c62244cc45ba6bde5 · test-lock-file:docs/adr/locks/66b9c53e1d5e88a096608bbab9ed9e6986aab9a4970e504c62244cc45ba6bde5.lock
  ```
  ```
- 2026-11-10 · 012f56e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:685
- 2026-11-10 · 012f56e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:1795
- 2026-11-10 · 012f56e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:1860
- 2026-11-10 · 012f56e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:599
- 2026-11-10 · 47338f2c* · exit 0 · `set -o pipefail …` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:415
- 2026-11-10 · 47338f2c* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:0 · test-lock-sha256:6505f1d4c8da24829b3a4623b59af2feab9a6f89b9d055718cf7c58040dbe769 · test-lock-file:docs/adr/locks/6505f1d4c8da24829b3a4623b59af2feab9a6f89b9d055718cf7c58040dbe769.lock · test-lock-kind:replace
- 2026-11-10 · a988f79c* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d4cd53bcba7f77f24bc39391e645793ffef0405ebdf1cbdde3f7e168b2ab86df · ms:0 · test-lock-sha256:2c75549c6e989ff596562e7db63f070fcfc2c05942dfa960da00e0522d578b68 · test-lock-file:docs/adr/locks/2c75549c6e989ff596562e7db63f070fcfc2c05942dfa960da00e0522d578b68.lock · test-lock-kind:replace

## Invariants

- Off Windows nothing changes: a trailing space is part of a name there.
- A plan file and a list may still live outside the root.

## Risks

- The command half runs only on the Windows CI job; macOS proves the rule and the call count, not the calls' effect.

## Stop Condition

The fence exits 0, and the Windows CI job passes `TestAWindowsAliasOfAPlanOrListIsRefused`.

## Out of Scope

- An NTFS stream spelling (`p.mrw:x`) of a plan or list (permanent: boundary: it opens a stream of that file, empty unless written, not another file; `unkeptComponent` judges only what Win32 drops)
