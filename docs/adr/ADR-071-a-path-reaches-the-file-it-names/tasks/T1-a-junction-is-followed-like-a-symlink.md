# Task ADR-071-T1: A junction is followed like a symlink

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `rooted.throughLinks`, `rooted.linkFS`, the Windows-only call in `Resolve` and `Abs`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a link is replaced by its target`, `a placeholder redirects nothing`, `an unreadable link refuses`, `a missing component ends the walk`, `a link loop is bounded`, `the Windows tests compile and exist`, `the engine packages are unchanged`

## Goal

On Windows a junction under `--root` escaped it: `EvalSymlinks` no longer follows a mount point
(Go 1.23, `winsymlink=1`), `rooted.Resolve` read the resulting ENOTDIR as a missing leaf, and the
lexical ancestor it fell back to was inside. Walk the path component by component, replacing each
symlink or junction with its `os.Readlink` target, before the boundary compares.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/links.go` | new | `linkFS`, `throughLinks`: the walk, driven through injected `Lstat`/`Readlink` |
| `internal/rooted/links_windows.go` | new | `followLinks = true`, the OS-backed `linkFS` (`ModeSymlink` or `ModeIrregular`) |
| `internal/rooted/links_other.go` | new | `followLinks = false`: POSIX is unchanged |
| `internal/rooted/rooted.go` | edit | `Resolve` and `Abs` walk first when `followLinks` |
| `internal/rooted/rooted.go` | edit | `Real`: an absolute argument resolved the way `Abs` resolves a root (review of #228) |
| `internal/read/read.go`, `internal/read/walk.go` | edit | the absolute-path pre-screens call `rooted.Real` |
| `internal/rooted/links_test.go` | new | the walk against a fake filesystem, on every platform |
| `internal/rooted/links_windows_test.go` | new | a real junction (`mklink /J`) through `Resolve` |
| `cmd/mrw/junction_windows_test.go` | new | read, replace, create, rename, unlink through a junction, each refused |

## Ordered Steps

1. [S1] Write the fake-filesystem tests and the two Windows tests; the fence goes RED (the walk does not exist). [proof: mutation]
2. [S2] Add the walk and wire it into `Resolve` and `Abs` behind `followLinks`; GREEN; `GOOS=windows go vet` compiles the Windows tests. [proof: mutation]
   Mutants: the target substitution dropped; the placeholder rule dropped (every `ENOENT` refuses); the refusal on an unreadable link dropped (it continues lexically); the hop bound dropped.

## Acceptance

```bash
set -o pipefail
go test ./internal/rooted/ ./cmd/mrw/ -count=1 -timeout 120s -run 'TestTheLinkWalk|TestResolveRefusesAJunctionOutOfTheRoot|TestAJunctionCannotCarryAnyOpOutOfTheRoot|TestAnAbsolutePathUnderAJunctionedRootIsServed' -v 2>&1 | tee /tmp/adr071-T1.out \
  && missing=$(for t in TestTheLinkWalkReplacesALinkWithItsTarget TestTheLinkWalkLeavesAPlaceholderAlone TestTheLinkWalkRefusesALinkItCannotRead TestTheLinkWalkStopsAtAMissingComponent TestTheLinkWalkBoundsALoop TestTheLinkWalkRefusesASymlinkItCannotRead TestTheLinkWalkRefusesAComponentItCannotExamine; do grep -qE "^--- PASS: $t \(" /tmp/adr071-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && GOOS=windows go vet ./internal/rooted/ ./cmd/mrw/ \
  && grep -q '^func TestResolveRefusesAJunctionOutOfTheRoot(' internal/rooted/links_windows_test.go \
  && grep -q '^func TestAJunctionCannotCarryAnyOpOutOfTheRoot(' cmd/mrw/junction_windows_test.go \
  && grep -q '^func TestAnAbsolutePathUnderAJunctionedRootIsServed(' cmd/mrw/junction_windows_test.go \
&& git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheLinkWalkReplacesALinkWithItsTarget` | `internal/rooted/links_test.go` | a junction to a directory outside is replaced by its target, and so is a link inside the target | — | S1, S2 |
| `TestTheLinkWalkLeavesAPlaceholderAlone` | `internal/rooted/links_test.go` | an irregular entry whose `Readlink` says "not a link" stays as named | — | S1, S2 |
| `TestTheLinkWalkRefusesALinkItCannotRead` | `internal/rooted/links_test.go` | any other `Readlink` failure refuses | — | S1, S2 |
| `TestTheLinkWalkStopsAtAMissingComponent` | `internal/rooted/links_test.go` | a create's missing tail is kept lexically, after the links above it are followed | — | S1, S2 |
| `TestTheLinkWalkBoundsALoop` | `internal/rooted/links_test.go` | two links pointing at each other refuse rather than spin | — | S1, S2 |
| `TestResolveRefusesAJunctionOutOfTheRoot` | `internal/rooted/links_windows_test.go` | a real junction out is refused; one that stays inside resolves | — | S1, S2 |
| `TestAJunctionCannotCarryAnyOpOutOfTheRoot` | `cmd/mrw/junction_windows_test.go` | read, replace, create, rename, unlink through it: refused, the outside file unchanged | — | S1, S2 |
| `TestAnAbsolutePathUnderAJunctionedRootIsServed` | `cmd/mrw/junction_windows_test.go` | a root reached through a junction still serves an absolute path inside it (review of #228, S1) | — | S2 |
| `TestTheLinkWalkRefusesASymlinkItCannotRead` | `internal/rooted/links_test.go` | a symlink whose Readlink says ENOENT is refused, not kept as a placeholder (review A2) | — | S2 |
| `TestTheLinkWalkRefusesAComponentItCannotExamine` | `internal/rooted/links_test.go` | only a missing component ends the walk; one Lstat cannot examine is refused (review A1) | — | S2 |
| `TestTheLinkWalkKeepsAMountedVolumeAsTheFolderItIs` | `internal/rooted/links_test.go` | a folder with a whole volume mounted on it is kept, not refused (review A4) | — | S2 |
| `TestTheLinkWalkKnowsAWholeVolumeFromADirectoryOnIt` | `internal/rooted/links_test.go` | only `\\?\Volume{GUID}\` is kept; a directory on another volume is followed | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `throughLinks` and its tests |
| 2 — something selects it | `Resolve` and `Abs` call it on Windows, which every read, write, check, `iter add` and `body=@` pass through |
| 3 — the caller can discover it | the refusal names the path and the root |
| 4 — it is used | three Windows sessions reproduced the escape it closes; a peer re-runs the repro on the release asset |

## Verification Log
(empty until execute)
- 2026-09-25 · 77a408a* · exit 1 · `set -o pipefail …` · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · ms:276 · test-lock-sha256:ea074756496fd59a5c73ff8e354579bcb076a9527e0814b6469804f49865d97d · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanVuY3Rpb25fd2luZG93c190ZXN0LmdvCVRlc3RBSnVuY3Rpb25DYW5ub3RDYXJyeUFueU9wT3V0T2ZUaGVSb290CTBkOTI0ZDU5N2FiMmI1YjdlZWM3ZjU0NGYxY2EyYTg1NjJiMTM5YjZhNjZmNDNiOTllM2ZlMTNlODE1ODc5ODAKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3NfdGVzdC5nbwlUZXN0VGhlTGlua1dhbGtCb3VuZHNBTG9vcAllNDZhNWJkMjlhN2ZlZGM3OTk0ZjlkNWZhYzMwYjBkMzIyNjA1MGZiYmRjNTBiYjAzMTg2Nzk2ZTBkYjZlZjM2CmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrTGVhdmVzQVBsYWNlaG9sZGVyQWxvbmUJNDdhODdkZGI4YmNlNTg4ZDc5Y2U3YWE5YTEzZDRkYWUxZTg5M2FhYmI2OWE2MTg0NzM5ZmZjODU4NjE0NTU5Ngpib2R5CWludGVybmFsL3Jvb3RlZC9saW5rc190ZXN0LmdvCVRlc3RUaGVMaW5rV2Fsa1JlZnVzZXNBTGlua0l0Q2Fubm90UmVhZAkzM2U3MjMxMGI1MDc0YTBjMTY2MDNhMWQ1OGRmMTA3OWQzNDViMmFkNzg0OGJiMTQxMTcxMGJiYzQ0YjUxZDZiCmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFRoZUxpbmtXYWxrUmVwbGFjZXNBTGlua1dpdGhJdHNUYXJnZXQJYmNkYzRiZjI0ZTJjODQwM2Y0YTM0OTgwN2JmYmRjNWQzZjY4Mzk1MmJmMjczMDU2NjZiNzc1ZDFjYzRiODUyYwpib2R5CWludGVybmFsL3Jvb3RlZC9saW5rc190ZXN0LmdvCVRlc3RUaGVMaW5rV2Fsa1N0b3BzQXRBTWlzc2luZ0NvbXBvbmVudAkxY2RhMmExYzUxZDZlNTdkNGQwMGQ0NzIxYzBiNzVmMTBlZTVhZjVhYjhmNDViYjUxOWFiZjNkOTU2NjQ0YTRlCmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFdpbjMyQWxpYXNMZWF2ZXNEb3RBbmREb3REb3RBbG9uZQkwNGJiM2NmOTc4OWRmMTRhNTMyMTAyOWYyZWY5MjZhMTQ1ZWE2NDRlN2U0MTI2OTNiMGVkODM2ZWE0ZWI5NjEwCmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3Rlc3QuZ28JVGVzdFdpbjMyQWxpYXNOYW1lc1RoZUNvbXBvbmVudFdpbmRvd3NXb3VsZFJlbWFwCWE0MGNiOGE3YmViYWQxYjUxNTAyNTI3YzNjNTVjODQ1ODE2NzA1OTY1OGQxNDg0NDUzMDIzMDFkYjAzMWQ1M2UKYm9keQlpbnRlcm5hbC9yb290ZWQvbGlua3Nfd2luZG93c190ZXN0LmdvCVRlc3RSZXNvbHZlUmVmdXNlc0FKdW5jdGlvbk91dE9mVGhlUm9vdAk1MGY5ZjI3ZGM4ODFkMjA5MTNhMWI5NWY1ODc4YmViY2Q4ZTgyNDRmYzBiNDBhYWVlZDkxMzNiYzE3NzkwNjk0CmJvZHkJaW50ZXJuYWwvcm9vdGVkL2xpbmtzX3dpbmRvd3NfdGVzdC5nbwlUZXN0UmVzb2x2ZVJlZnVzZXNBV2luMzJBbGlhcwkzZWI0ZDJjMDMyZDg5MjdiZTcyNzNkODFlNWUxNzU3YmY5NWQ1ZGZiZjViMmM5NGI3M2FkN2U5N2ZmZjgyNmQ5
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
  internal/rooted/links_test.go:42:9: undefined: linkFS
  internal/rooted/links_test.go:89:15: undefined: throughLinks
  internal/rooted/links_test.go:110:14: undefined: throughLinks
  internal/rooted/links_test.go:126:12: undefined: throughLinks
  internal/rooted/links_test.go:143:14: undefined: throughLinks
  internal/rooted/links_test.go:159:15: undefined: throughLinks
  internal/rooted/links_test.go:176:18: undefined: win32Alias
  internal/rooted/links_test.go:187:17: undefined: win32Alias
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted [build failed]
  FAIL
  ```
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · ms:5830
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · ms:1041
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · ms:367
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · ms:375
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · ms:896
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:815dc87fe558148019204ef3a88e1d7801aa472d226168aa99996edb1faa3b95 · ms:1005
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · ms:1129
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · ms:654
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · ms:528
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · ms:490
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · ms:1516
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · ms:949
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:1247
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:413
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:429
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:433
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:397
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:397
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:508
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:396
- 2026-09-25 · 52e730c* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:1405
- 2026-09-25 · 52e730c* · exit 0 · `set -o pipefail …` · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · ms:506

## Mutation Log
(empty until execute)
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · a link is kept as written instead of replaced by its target · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · covers:a link is replaced by its target
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · every Readlink ENOENT refuses, so a OneDrive placeholder is refused as a link · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · covers:a placeholder redirects nothing
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · an unreadable link is walked past lexically instead of refused · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · covers:an unreadable link refuses
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · the hop bound is gone, so two links pointing at each other spin until the test timeout · acceptance-sha256:09a4ab66605f0ee5e8eb0dc0292898b4d2649ee76711f71d22f0c40eea34ddd9 · covers:a link loop is bounded
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · a link is kept as written instead of replaced by its target · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · covers:a link is replaced by its target
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · every Readlink ENOENT refuses, so a OneDrive placeholder is refused as a link · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · covers:a placeholder redirects nothing
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · an unreadable link is walked past lexically instead of refused · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · covers:an unreadable link refuses
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · the hop bound is gone, so two links pointing at each other spin until the test timeout · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · covers:a link loop is bounded
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/rooted/links.go` · a missing component refuses the whole path, so a create through a followed junction is refused as unreadable · acceptance-sha256:1bb7d3efd60543a9e01454df1230b7c631dfb3cc4a839516ea4232521752f923 · covers:a missing component ends the walk
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · a link is kept as written instead of replaced by its target · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:a link is replaced by its target
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · every Readlink ENOENT refuses, so a OneDrive placeholder is refused as a link · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:a placeholder redirects nothing
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · a symlink whose Readlink says ENOENT is kept as a placeholder instead of refused · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:an unreadable link refuses
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · an unreadable link is walked past lexically instead of refused · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:an unreadable link refuses
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · the hop bound is gone, so two links pointing at each other spin until the test timeout · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:a link loop is bounded
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · a missing component refuses the whole path, so a create through a followed junction is refused · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:a missing component ends the walk
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/rooted/links.go` · any Lstat error ends the walk, so a component that cannot be examined is judged by its spelling · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:a missing component ends the walk
- 2026-09-25 · 52e730c* · mutant killed · exit 1 · `internal/rooted/links.go` · a folder with a whole volume mounted on it is followed to its GUID spelling, and every path under it is refused · acceptance-sha256:456cb74ed1be95a43fb3946be9046c4c3b461bb620a6e6a50c503af4f93c90c1 · covers:a link is replaced by its target

## Invariants

- On a platform where `followLinks` is false, `Resolve` and `Abs` are unchanged.
- A junction that stays inside the root resolves.

## Risks

- The Windows tests run only on the `windows-shard` CI jobs; the mutants are killed by the fake-filesystem tests, which run everywhere.

## Out of Scope

- `os.Root` for every file access (permanent: boundary: ADR-071 Alternatives)

## Stop Condition

Stop if the walk needs a change outside `internal/rooted`.
