# Task ADR-196-T6: past a device prefix, a drive, volume, share or pipe names a file only with a component after it

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `judgeOn` refuses a volume with no file; `TestAPlanOrListNamingAVolumeAndNoFileIsRefused`
**Consumes:** `judgeOn`, `pipeOn` (T5)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a volume with no file is refused`

## Goal

Decision 9, from the Windows field test of v1.86.3 (2026-11-11, four Windows 11 peers: pwsh 7, Git Bash, Windows PowerShell 5.1, cmd):
- `\\.\C:`, `\\?\C:` and `\\.\Volume{…}` passed T5's check, because a drive or volume was accepted with nothing after it. They reached os.Open on the raw volume, stopped only by "Access is denied" in shells that were not elevated.
- `\\.\pipe` opened the pipe namespace.
- `\\.\UNC\localhost\pipe\..\..\GLOBALROOT\…` passed too; it resolves to a UNC path with the server `GLOBALROOT`, which Windows treats the same way ("the network path was not found"), so no device opened. A UNC path whose `..` climbs above its share names no share at all.

Past a device prefix, a drive, a `Volume{…}`, a UNC server and share, or `pipe` must now be followed by a component, as must a plain UNC server and share. Otherwise the path is refused, exit 2, before it is opened.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/keep196.go` | edit | `judgeOn` refuses a volume, share or pipe namespace with no component after it |
| `cmd/mrw/keep196t6_test.go` | add | the rule on every platform |

## Ordered Steps

1. [S1] Write `TestAPlanOrListNamingAVolumeAndNoFileIsRefused`:
   - Refused, and not a pipe: `\\.\C:`, `\\?\C:\`, `\??\C:`, `\\.\Volume{…}`, `\\.\pipe`, `\\.\UNC\srv`, `\\?\UNC\srv\share`, a UNC whose `..` climbs above its share, `\\srv\share` and `\\srv\pipe`.
   - Accepted: the same spellings with a file after them, and a plain `C:` or `C:\`.
   - Off Windows: accepted.

   See it RED.
2. [S2] Change `judgeOn`. Mutant, killed: the drive branch accepting with nothing after it. [proof: mutation]
3. [S3] `GOOS=windows go vet ./cmd/mrw`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestAPlanOrListNamingAVolumeAndNoFileIsRefused|TestAPlanOrListNamingAWindowsObjectIsRefused|TestAPlanOrListNamingADeviceOrAStreamIsRefused|TestAWindowsAliasOfAPlanOrListIsRefused)$' 2>&1) \
  && grep -qE '^--- PASS: TestAPlanOrListNamingAVolumeAndNoFileIsRefused \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAPlanOrListNamingAWindowsObjectIsRefused \(' <<<"$out" \
  && GOOS=windows go vet ./cmd/mrw
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanOrListNamingAVolumeAndNoFileIsRefused` | `cmd/mrw/keep196t6_test.go` | a volume, share or pipe namespace with no file is refused, and is not a pipe | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `judgeOn`'s no-file refusal |
| 2 — something selects it | `windowsKeeps` and `windowsPipe` at both open sites (T3, T5) |
| 3 — the caller can discover it | the refusal names the volume and says to name a file in it |
| 4 — it is used | three Windows peers reached the raw volume on 2026-11-11; no telemetry (ADR-009) |

## Mutation Log
- 2026-11-11 · f005acca* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · a drive or volume accepted with nothing after it, so \\.\C: must go red · acceptance-sha256:f1461c05745b27472c19dcaadc9c859fa080ae55fdf5e95830a20404612cb828 · covers:a volume with no file is refused

## Verification Log
- 2026-11-11 · f005acca* · exit 1 · `set -o pipefail …` · acceptance-sha256:f1461c05745b27472c19dcaadc9c859fa080ae55fdf5e95830a20404612cb828 · ms:349 · test-lock-sha256:65cc298f946c5e8fcf21f54d0a4c7caa8096d1a9f50ddb1c245eeeeb71dcce33 · test-lock-file:docs/adr/locks/65cc298f946c5e8fcf21f54d0a4c7caa8096d1a9f50ddb1c245eeeeb71dcce33.lock
  ```
  ```
- 2026-11-11 · f005acca* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f1461c05745b27472c19dcaadc9c859fa080ae55fdf5e95830a20404612cb828 · ms:0 · test-lock-sha256:abbbc65c02f6fc064dc16086b8b279949324229f5e635aca900616c6c35353d6 · test-lock-file:docs/adr/locks/abbbc65c02f6fc064dc16086b8b279949324229f5e635aca900616c6c35353d6.lock · test-lock-kind:replace
- 2026-11-11 · f005acca* · exit 0 · `set -o pipefail …` · acceptance-sha256:f1461c05745b27472c19dcaadc9c859fa080ae55fdf5e95830a20404612cb828 · ms:1200
- 2026-11-11 · f005acca* · exit 0 · `set -o pipefail …` · acceptance-sha256:f1461c05745b27472c19dcaadc9c859fa080ae55fdf5e95830a20404612cb828 · ms:3445

## Invariants

- Off Windows nothing changes.
- A plain drive-relative `C:` or `C:\` is not a device path and is not judged by this rule: it opens a directory, which fails at the read.

## Risks

- None new beyond T5: every refused spelling names no file.

## Stop Condition

The fence exits 0, and the Windows CI job passes it.

## Out of Scope

- An elevated open of a raw volume (permanent: boundary: refused before the open, whatever the privilege; the peers that found it were unelevated and saw "Access is denied")
