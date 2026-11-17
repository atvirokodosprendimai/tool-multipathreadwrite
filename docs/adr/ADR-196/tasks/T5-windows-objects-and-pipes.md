# Task ADR-196-T5: past a \\.\ or \\?\ prefix only a drive, a share, a volume or a pipe is a plan or list path, and a pipe is opened once

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `judgeOn`, `pipeOn`, `windowsPipe`; `TestAPlanOrListNamingAWindowsObjectIsRefused`, `TestAPlanOrListServedOverAPipeIsOpenedOnce`
**Consumes:** `keepsOn` (T4)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an object past the prefix is refused`, `a pipe skips the state check`

## Goal

Decision 8, from the Windows re-probe of v1.86.1 (2026-11-11; four Windows 11 peers, one shell each):
- `\\.\GLOBALROOT\Device\ConDrv\Console` waited on the console as `write CON` did.
- `\\.\pipe\..\GLOBALROOT\Device\Null` and `\\?\GLOBALROOT\Device\Null` read the null device.
- A plan served by a one-shot pipe failed "All pipe instances are busy", because the state check (ADR-077, ADR-134) opened the path before the read did.

Past a device prefix, the first component is now judged by its kind, after `.` and `..` are resolved as Windows resolves them (not under `\\?\`): a drive, `UNC`, `Volume{…}` or `pipe` is accepted, and anything else is refused. A pipe skips the state check, and each open site judges the path before it touches it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/keep196.go` | edit | `judgeOn` classifies the namespace after the prefix; `keepsOn`, `pipeOn` and `windowsPipe` are views of it |
| `cmd/mrw/main.go` | edit | both open sites: `windowsKeeps` first, then the state check unless `windowsPipe` |
| `cmd/mrw/keep196t5_test.go` | add | the rule on every platform |
| `cmd/mrw/pipe196_windows_test.go` | add | a one-instance pipe serves a plan and a list, on Windows |

## Ordered Steps

1. [S1] Write `TestAPlanOrListNamingAWindowsObjectIsRefused`: GLOBALROOT and other objects, through `\\.\`, `\\?\` and a pipe's `..`, are refused; a volume GUID, a drive reached through `..` and a pipe are accepted; `pipeOn` tells a pipe from a path that leaves one. Write `TestAPlanOrListServedOverAPipeIsOpenedOnce` (Windows). See it RED (no `pipeOn`).
2. [S2] Add `judgeOn`, `pipeOn` and `windowsPipe`, and reorder both open sites. Mutants, each killed: the default (object) branch accepting; `pipeOn` always false. [proof: mutation]
3. [S3] `GOOS=windows go vet ./cmd/mrw`; the Windows CI job runs the pipe test, which no other platform can. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestAPlanOrListNamingAWindowsObjectIsRefused|TestAPlanOrListServedOverAPipeIsOpenedOnce|TestAPlanOrListNamingADeviceOrAStreamIsRefused|TestAWindowsAliasOfAPlanOrListIsRefused)$' 2>&1) \
  && grep -qE '^--- PASS: TestAPlanOrListNamingAWindowsObjectIsRefused \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAPlanOrListNamingADeviceOrAStreamIsRefused \(' <<<"$out" \
  && [ "$(grep -c 'windowsPipe(' cmd/mrw/main.go)" -eq 2 ] \
  && GOOS=windows go vet ./cmd/mrw
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanOrListNamingAWindowsObjectIsRefused` | `cmd/mrw/keep196t5_test.go` | the namespace rule and `pipeOn` on every platform | none | S1, S2 |
| `TestAPlanOrListServedOverAPipeIsOpenedOnce` | `cmd/mrw/pipe196_windows_test.go` | a one-instance pipe delivers a plan and a list | none | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `judgeOn` |
| 2 — something selects it | `windowsKeeps` and `windowsPipe` at both open sites; the fence counts `windowsPipe` |
| 3 — the caller can discover it | the refusal names the component and says to name the file by its drive or share path |
| 4 — it is used | four Windows peers met the hang, the device read and the busy pipe on 2026-11-11; no telemetry (ADR-009) |

## Mutation Log
- 2026-11-11 · db2941aa* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · an object past the prefix accepted, so GLOBALROOT must go red · acceptance-sha256:b2f108728d2ab5704cc40e7241050afbf59dd35c06bb558966e1f4e736787d49 · covers:an object past the prefix is refused
- 2026-11-11 · db2941aa* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · pipeOn always false, so the state check opens a pipe again · acceptance-sha256:b2f108728d2ab5704cc40e7241050afbf59dd35c06bb558966e1f4e736787d49 · covers:a pipe skips the state check
- 2026-11-11 · db2941aa* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · an object past the prefix accepted, so GLOBALROOT must go red · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · covers:an object past the prefix is refused
- 2026-11-11 · db2941aa* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · pipeOn always false, so the state check opens a pipe again · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · covers:a pipe skips the state check
- 2026-11-11 · a7d829b2* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · a UNC pipe with .. judged a path again, so \\srv\pipe\..\plan must go red (Codex review of #633) · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · covers:a pipe skips the state check
- 2026-11-11 · a7d829b2* · mutant survived · exit 0 · `cmd/mrw/keep196.go` · the \??\ prefix not verbatim: \??\GLOBALROOT must still go red · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · covers:an object past the prefix is refused
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-11 · a7d829b2* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · the \??\ prefix not verbatim, so \??\C:\t\CON must go red · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · covers:an object past the prefix is refused

## Verification Log
- 2026-11-11 · db2941aa* · exit 1 · `set -o pipefail …` · acceptance-sha256:b2f108728d2ab5704cc40e7241050afbf59dd35c06bb558966e1f4e736787d49 · ms:213 · test-lock-sha256:533c0920eb27d7a136f48809252874bad17199f631b908c96af1a4d6efd2e4c9 · test-lock-file:docs/adr/locks/533c0920eb27d7a136f48809252874bad17199f631b908c96af1a4d6efd2e4c9.lock
  ```
  ```
- 2026-11-11 · db2941aa* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2f108728d2ab5704cc40e7241050afbf59dd35c06bb558966e1f4e736787d49 · ms:461
- 2026-11-11 · db2941aa* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2f108728d2ab5704cc40e7241050afbf59dd35c06bb558966e1f4e736787d49 · ms:1958
- 2026-11-11 · db2941aa* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2f108728d2ab5704cc40e7241050afbf59dd35c06bb558966e1f4e736787d49 · ms:1838
- 2026-11-11 · db2941aa* · exit 1 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:211
  ```
  ```
- 2026-11-11 · db2941aa* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:518
- 2026-11-11 · db2941aa* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:1904
- 2026-11-11 · db2941aa* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:1893
- 2026-11-11 · a7d829b2* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:0 · test-lock-sha256:c83abc8d0725b8233fe98f3f9bd4408315884892a08662c023ba37facf3bfab9 · test-lock-file:docs/adr/locks/c83abc8d0725b8233fe98f3f9bd4408315884892a08662c023ba37facf3bfab9.lock · test-lock-kind:replace
- 2026-11-11 · a7d829b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:447
- 2026-11-11 · a7d829b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:1725
- 2026-11-11 · a7d829b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:1756
- 2026-11-11 · a7d829b2* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:0 · test-lock-sha256:c578d5d249752ff77e81b09e73dfb07aafffdadb1d8d31790ed6bec667630895 · test-lock-file:docs/adr/locks/c578d5d249752ff77e81b09e73dfb07aafffdadb1d8d31790ed6bec667630895.lock · test-lock-kind:replace
- 2026-11-11 · a7d829b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:1801
- 2026-11-11 · a7d829b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:15f2c8213ff373603723868ad3407334e475eebab6cdaee2f18412412ae78f58 · ms:485

## Invariants

- Off Windows nothing changes.
- A drive, a UNC share and a `\\?\` path keep T3's and T4's rules.
- `-` still reads standard input.

## Risks

- A file reached through `\\?\GLOBALROOT\Device\HarddiskVolumeN\…` is refused, though it is a file; the caller names it by its drive letter.
- The pipe test runs only on the Windows CI job.

## Stop Condition

The fence exits 0, and the Windows CI job passes both tests.

## Out of Scope

- A file symlink to a device or stream (permanent: boundary: os.Open follows it as Windows does; creating one needs admin or Developer Mode, and a caller who made one named it)
- A `CLOCK$` or `COM0` name, which Windows 11 does not map (permanent: fact: both opened as missing files in the re-probe; citation: url `https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file`)
