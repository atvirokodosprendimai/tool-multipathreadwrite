# Task ADR-196-T4: a plan file or a --files-from list named as a device or an NTFS stream is refused on Windows

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `keepsOn` judges `:` and device names; `rooted.DeviceBase`; `TestAPlanOrListNamingADeviceOrAStreamIsRefused`
**Consumes:** `windowsKeeps`, `keepsOn` (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a device name is refused on Windows only`, `a stream is refused, a verbatim path included`

## Goal

Decision 6, widened by the Windows field test of v1.86.0 (2026-11-10, four Windows 11 peers: cmd, pwsh 7, Windows PowerShell 5.1, Git Bash): `write CON` waited on the console until it was killed, `write NUL` and `--files-from NUL` read the null device, and `write p.mrw:alt` applied a plan held in an NTFS stream that `type p.mrw`, `dir` and git do not show. A path inside the root is refused for each (ADR-071, ADR-081); a plan file and a `--files-from` list now are too.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/keep196.go` | edit | `keepsOn` refuses a component holding `:` (a `\\?\` path too, since the filesystem parses streams) and, outside `\\?\`, a device name |
| `internal/rooted/links.go` | edit | `DeviceBase` exports `isReservedBase`, so the list of device names stays in one place |
| `cmd/mrw/keep196t4_test.go` | add | the rule on every platform; on Windows, through the commands |

## Ordered Steps

1. [S1] Write `TestAPlanOrListNamingADeviceOrAStreamIsRefused`: `keepsOn("windows", …)` refuses `CON`, `NUL.txt`, `COM1`, `aux .mrw`, `\\.\CON` as a device and `p.mrw:alt`, `p.mrw::$DATA`, `\\?\C:\t\p.mrw:alt` as a stream; accepts `C:p.mrw`, `\\?\C:\t\CON`, `\\.\C:\t\p.mrw`, `COM10`, a UNC path; off Windows accepts both; on Windows `write NUL`, `--files-from NUL`, `write p.mrw:alt` and `--files-from list:alt` are refused and apply or read nothing. See it RED.
2. [S2] Widen `keepsOn` and add `rooted.DeviceBase`. Mutants, each killed: the device check removed; the stream check skipped for a `\\?\` path. [proof: mutation]
3. [S3] `GOOS=windows go vet ./cmd/mrw`; the Windows CI job runs the command half. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestAPlanOrListNamingADeviceOrAStreamIsRefused|TestAWindowsAliasOfAPlanOrListIsRefused)$' 2>&1) \
  && grep -qE '^--- PASS: TestAPlanOrListNamingADeviceOrAStreamIsRefused \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAWindowsAliasOfAPlanOrListIsRefused \(' <<<"$out" \
  && [ "$(grep -c 'windowsKeeps(' cmd/mrw/main.go)" -eq 2 ] \
  && GOOS=windows go vet ./cmd/mrw
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanOrListNamingADeviceOrAStreamIsRefused` | `cmd/mrw/keep196t4_test.go` | the rule for Windows on every platform; the commands on Windows | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `keepsOn` |
| 2 — something selects it | `windowsKeeps` before both opens (T3); the fence counts both calls |
| 3 — the caller can discover it | the refusal names the component and why |
| 4 — it is used | four Windows peers met the hang and the stream on 2026-11-10; no telemetry (ADR-009) |

## Mutation Log
- 2026-11-10 · 3da3f32b* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · the device check removed, so CON must go red · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · covers:a device name is refused on Windows only
- 2026-11-10 · 3da3f32b* · mutant survived · exit 0 · `cmd/mrw/keep196.go` · placeholder · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-10 · 3da3f32b* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · the stream check skipped for a \\?\ path, so \\?\C:\t\p.mrw:alt must go red · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · covers:a stream is refused, a verbatim path included
- 2026-11-10 · faaf2678* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · the device check removed, so CON must go red (after the #630 review) · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · covers:a device name is refused on Windows only
- 2026-11-10 · faaf2678* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · the stream check skipped for a \\?\ path (after the #630 review) · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · covers:a stream is refused, a verbatim path included
- 2026-11-10 · 2ae3f86b* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · a pipe path exempt even with a .. that leaves the pipe namespace, so \\.\pipe\..\NUL must go red (Codex re-review of #630) · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · covers:a device name is refused on Windows only

## Verification Log
- 2026-11-10 · 3da3f32b* · exit 1 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:2468 · test-lock-sha256:821ee80700bb28428d41d98b07203a8537ab8102f943b3c30ddda6772f9dd094 · test-lock-file:docs/adr/locks/821ee80700bb28428d41d98b07203a8537ab8102f943b3c30ddda6772f9dd094.lock
  ```
  ```
- 2026-11-10 · 3da3f32b* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:769
- 2026-11-10 · 3da3f32b* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:3718
- 2026-11-10 · 3da3f32b* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:3054
- 2026-11-10 · 3da3f32b* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:2026
- 2026-11-10 · faaf2678* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:0 · test-lock-sha256:8eaf00397bbe957294e69fce9296118785b2dda158312f6452f419d97f4dc148 · test-lock-file:docs/adr/locks/8eaf00397bbe957294e69fce9296118785b2dda158312f6452f419d97f4dc148.lock · test-lock-kind:replace
- 2026-11-10 · faaf2678* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:1585
- 2026-11-10 · faaf2678* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:1995
- 2026-11-10 · faaf2678* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:2056
- 2026-11-10 · 2ae3f86b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:0 · test-lock-sha256:9738dccb51e3ed829a7297580682021c24d980e275fd740684f5c93befcf802e · test-lock-file:docs/adr/locks/9738dccb51e3ed829a7297580682021c24d980e275fd740684f5c93befcf802e.lock · test-lock-kind:replace
- 2026-11-10 · 2ae3f86b* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:2452
- 2026-11-10 · 2ae3f86b* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:619
- 2026-11-10 · 9fdd9e61* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:0 · test-lock-sha256:5728f00d80cbe9af6e3eb78a6d51f6f44aef276b6e36cbd0a4c448cacda79f19 · test-lock-file:docs/adr/locks/5728f00d80cbe9af6e3eb78a6d51f6f44aef276b6e36cbd0a4c448cacda79f19.lock · test-lock-kind:replace
- 2026-11-10 · 9fdd9e61* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:476
- 2026-11-10 · 9fdd9e61* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:0 · test-lock-sha256:5728f00d80cbe9af6e3eb78a6d51f6f44aef276b6e36cbd0a4c448cacda79f19 · test-lock-file:docs/adr/locks/5728f00d80cbe9af6e3eb78a6d51f6f44aef276b6e36cbd0a4c448cacda79f19.lock · test-lock-kind:replace
- 2026-11-10 · 9fdd9e61* · exit 0 · `set -o pipefail …` · acceptance-sha256:b2b6d6520c8ac80135f1fc2febb5f3a5fbfc00eda91d71bb434a301af64b9fee · ms:553

## Invariants

- Off Windows nothing changes: `CON` and `p:alt` are names there.
- A `\\?\` path keeps its trailing dots, spaces and device names, which Win32 does not map there; a stream is still a stream.
- `-` still reads standard input.

## Risks

- The command half runs only on the Windows CI job; macOS proves the rule and the call count.

## Stop Condition

The fence exits 0, and the Windows CI job passes both tests.

## Out of Scope

- A byte that is not valid UTF-8 in a plan or list path (permanent: boundary: Windows hands mrw its command line as UTF-16, so the path Go builds from it is valid UTF-8)
- An 8.3 short name or another case of the same file (permanent: boundary: Win32 opens the file the caller named, by another spelling of it, not another file)
