# Task ADR-081-T1: every reserved device name is refused by name on Windows

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the by-name refusal in `rooted.Resolve`
**Consumes:** `win32Device`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a reserved name is a candidate by Go's rule`, `Windows refuses every candidate`, `no OS query remains`, `every component counts`, `the packages vet for Windows`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

On Windows 11, mrw v1.27.0 created `con`, `nul.txt` and `COM1.txt` as files that its own unlink and PowerShell 5 could not reach.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | edit | refuse every candidate, no OS query |
| `internal/rooted/links_windows.go`, `internal/rooted/links_other.go` | edit | `opensDevice` removed |
| `cmd/mrw/device_windows_test.go` | edit | `con`, `nul.txt`, `COM1.txt`, `aux.go`, `CONOUT$` refused; `console.txt` created |
| `internal/rooted/paths076_test.go` | edit | every component counts |
| `internal/rooted/links.go` | edit | `win32Device` scans every component |
| `internal/read/read.go`, `internal/apply/apply.go` | edit | `ErrDeviceName` printed without the `--root` advice |
| `internal/read/astgrep.go` | edit | each hit passes `rooted.Resolve` before the CR-only probe |
| `internal/apply/encoding_test.go` | edit | fixture `nulbyte.bin`, not the device name `nul.bin` |
| `AGENTS.md`, `docs/adr/ADR-076-a-path-means-what-it-says.md` | edit | the rule, and the pointer |

## Ordered Steps

1. [S1] Change the Windows test; it fails on v1.27.0 wherever the OS reads `con` as a file — the Windows peer's run and the CI runner, per ADR-076. [proof: acceptance]
2. [S2] Refuse by name; drop `opensDevice`; the Windows CI shard runs the test. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/rooted/ -count=1 -timeout 120s -run 'TestADeviceNameIsACandidateByGosRule|TestADeviceCandidateNeverPanicsOnUnicode' -v 2>&1 | tee /tmp/adr081-T1.out \
  && missing=$(for t in TestADeviceNameIsACandidateByGosRule TestADeviceCandidateNeverPanicsOnUnicode; do grep -qE "^--- PASS: $t \(" /tmp/adr081-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '"create-con":' cmd/mrw/device_windows_test.go \
  && grep -q '"create-nul.txt":' cmd/mrw/device_windows_test.go \
  && ! grep -rq 'opensDevice' internal/rooted/ \
  && GOOS=windows go vet ./internal/rooted/ ./cmd/mrw/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/check internal/lines internal/iter internal/seen internal/state internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/check internal/lines internal/iter internal/seen internal/state internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestADeviceNameIsACandidateByGosRule` | `internal/rooted/paths076_test.go` | the candidates, on every platform | — | S2 |
| `TestADeviceCandidateNeverPanicsOnUnicode` | `internal/rooted/paths076_test.go` | the candidate check on Unicode | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.Resolve` |
| 2 — something selects it | every read spec, plan path and rename destination on Windows |
| 3 — the caller can discover it | the refusal names the device name, and AGENTS.md lists them |
| 4 — it is used | the Windows peer's run of v1.27.0 hit it |

## Verification Log
(empty until execute)
- 2026-09-26 · e9e3d3e* · exit 1 · `set -o pipefail …` · acceptance-sha256:6e5f86b7881a3788fd0e7f69883b7313c8001af069056b127dd09815d144f09b · ms:557 · test-lock-sha256:610354e958fcc5edd0dfdbd98613224a6b167419182ef98ecad322e6a5f79cbe · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZGV2aWNlX3dpbmRvd3NfdGVzdC5nbwlUZXN0QURldmljZU5hbWVCZWhpbmRBRG90Q29tcG9uZW50SXNSZWZ1c2VkCTY5NTZlMTVmOTI4ZTU5OTI5MDhkODg0ZGVhNjQ0NDRhY2IyYWIyNWU0OTRjNjhiOWQ3N2VmNTI2ZmM3OGUxNTcKYm9keQljbWQvbXJ3L2RldmljZV93aW5kb3dzX3Rlc3QuZ28JVGVzdEFEZXZpY2VOYW1lSXNSZWZ1c2VkTm90UmVhZEFzQW5FbXB0eUZpbGUJYjMyMjk3MmFhZDEzMTUzMTk5NjMyMDRhNmQ5YmU2ZTgyMTBlZWIwNTU2MGM5Nzc5NjJhYjk4ZWE2Njk3Y2FlMApib2R5CWludGVybmFsL3Jvb3RlZC9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBRGV2aWNlQ2FuZGlkYXRlTmV2ZXJQYW5pY3NPblVuaWNvZGUJYjE3MDNjMGM5YTJlYWMxNTg3YzZlYjc2ZDNlYTdmZmQxZWZkMTNlZDJmZjJlMjQ0NjAwM2M1NzE0M2JlNGJkOQpib2R5CWludGVybmFsL3Jvb3RlZC9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBRGV2aWNlTmFtZUlzQUNhbmRpZGF0ZUJ5R29zUnVsZQljMTNmMjE0NjI1YWJkN2IyZThkYmU2ZmVhNzNmZTUxZTYzMjA1NzBlYTNkMzYwMjM4NDllYWRhMzU5NjRkOWMzCmJvZHkJaW50ZXJuYWwvcm9vdGVkL3BhdGhzMDc2X3Rlc3QuZ28JVGVzdEFQYXRoRW5kaW5nSW5BU2VwYXJhdG9yTXVzdE5hbWVBRGlyZWN0b3J5CWM2NDczMjExYTRmOTk5MTg2MTAxNTZkOWU4YjlhN2U0MDBmMGFjZTJiMGVlZWU1MDIwMjljYWMxN2Y5NjhhYzAKYm9keQlpbnRlcm5hbC9yb290ZWQvcGF0aHMwNzZfdGVzdC5nbwlUZXN0QVJvb3RUaGF0RG9lc05vdEV4aXN0SXNOYW1lZEFzTWlzc2luZwkzZWIzNzZmOWE0NGYxODk0NmJjNWY5NjNmNjhmNTRkNWUzYWJhNWI3NTBkNjg3ZjllYmZlNTkwMDY4ZmUyMWU4
  ```
  --- last 6 line(s) of stdout
  === RUN   TestADeviceNameIsACandidateByGosRule
  --- PASS: TestADeviceNameIsACandidateByGosRule (0.00s)
  === RUN   TestADeviceCandidateNeverPanicsOnUnicode
  --- PASS: TestADeviceCandidateNeverPanicsOnUnicode (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.170s
  ```
- 2026-09-26 · e9e3d3e* · exit 1 · `set -o pipefail …` · acceptance-sha256:6e5f86b7881a3788fd0e7f69883b7313c8001af069056b127dd09815d144f09b · ms:244
  ```
  --- last 6 line(s) of stdout
  === RUN   TestADeviceNameIsACandidateByGosRule
  --- PASS: TestADeviceNameIsACandidateByGosRule (0.00s)
  === RUN   TestADeviceCandidateNeverPanicsOnUnicode
  --- PASS: TestADeviceCandidateNeverPanicsOnUnicode (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.053s
  ```
- 2026-09-26 · e9e3d3e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6e5f86b7881a3788fd0e7f69883b7313c8001af069056b127dd09815d144f09b · ms:944
- 2026-09-26 · e9e3d3e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6e5f86b7881a3788fd0e7f69883b7313c8001af069056b127dd09815d144f09b · ms:0 · test-lock-sha256:9905fdf1dc10b9853fadee3b4e5ef842f1e48bace812b4cb7a3f75f1f890e480 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3Jvb3RlZC9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBRGV2aWNlQ2FuZGlkYXRlTmV2ZXJQYW5pY3NPblVuaWNvZGUJYjE3MDNjMGM5YTJlYWMxNTg3YzZlYjc2ZDNlYTdmZmQxZWZkMTNlZDJmZjJlMjQ0NjAwM2M1NzE0M2JlNGJkOQpib2R5CWludGVybmFsL3Jvb3RlZC9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBRGV2aWNlTmFtZUlzQUNhbmRpZGF0ZUJ5R29zUnVsZQljMTNmMjE0NjI1YWJkN2IyZThkYmU2ZmVhNzNmZTUxZTYzMjA1NzBlYTNkMzYwMjM4NDllYWRhMzU5NjRkOWMzCmJvZHkJaW50ZXJuYWwvcm9vdGVkL3BhdGhzMDc2X3Rlc3QuZ28JVGVzdEFQYXRoRW5kaW5nSW5BU2VwYXJhdG9yTXVzdE5hbWVBRGlyZWN0b3J5CWM2NDczMjExYTRmOTk5MTg2MTAxNTZkOWU4YjlhN2U0MDBmMGFjZTJiMGVlZWU1MDIwMjljYWMxN2Y5NjhhYzAKYm9keQlpbnRlcm5hbC9yb290ZWQvcGF0aHMwNzZfdGVzdC5nbwlUZXN0QVJvb3RUaGF0RG9lc05vdEV4aXN0SXNOYW1lZEFzTWlzc2luZwkzZWIzNzZmOWE0NGYxODk0NmJjNWY5NjNmNjhmNTRkNWUzYWJhNWI3NTBkNjg3ZjllYmZlNTkwMDY4ZmUyMWU4 · test-lock-kind:replace
- 2026-09-26 · e9e3d3e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6e5f86b7881a3788fd0e7f69883b7313c8001af069056b127dd09815d144f09b · ms:675
- 2026-09-26 · e9e3d3e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6e5f86b7881a3788fd0e7f69883b7313c8001af069056b127dd09815d144f09b · ms:346
- 2026-09-26 · 0929742* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:493bee97294cbaed265392cff9ef7afb93da4a7b9e0241a584266202edfbf9f5 · ms:0 · test-lock-sha256:0ef16edde892cb8a632dfd5867c5a42eb497d37ebbaaf7a8accc101783a4c8b2 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3Jvb3RlZC9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBRGV2aWNlQ2FuZGlkYXRlTmV2ZXJQYW5pY3NPblVuaWNvZGUJNDU0Yzk3MzE2NmMyOGIxZjE2M2NiYWZhNTVmZjFlYjA5ODQ2ZWE2OThiZjhiMmQ5ZDE0MGYxYzZkMTY2MWY2YQpib2R5CWludGVybmFsL3Jvb3RlZC9wYXRoczA3Nl90ZXN0LmdvCVRlc3RBRGV2aWNlTmFtZUlzQUNhbmRpZGF0ZUJ5R29zUnVsZQljNGY4MGExNTQ4NGRhYWNhZGI4NDJjY2Q2N2Y2OTgwNzBlNjgzZjc1ODQ5OTJjOGRmOTQ2MWQ0Y2NlYTdkY2Q3CmJvZHkJaW50ZXJuYWwvcm9vdGVkL3BhdGhzMDc2X3Rlc3QuZ28JVGVzdEFQYXRoRW5kaW5nSW5BU2VwYXJhdG9yTXVzdE5hbWVBRGlyZWN0b3J5CWM2NDczMjExYTRmOTk5MTg2MTAxNTZkOWU4YjlhN2U0MDBmMGFjZTJiMGVlZWU1MDIwMjljYWMxN2Y5NjhhYzAKYm9keQlpbnRlcm5hbC9yb290ZWQvcGF0aHMwNzZfdGVzdC5nbwlUZXN0QVJvb3RUaGF0RG9lc05vdEV4aXN0SXNOYW1lZEFzTWlzc2luZwkzZWIzNzZmOWE0NGYxODk0NmJjNWY5NjNmNjhmNTRkNWUzYWJhNWI3NTBkNjg3ZjllYmZlNTkwMDY4ZmUyMWU4 · test-lock-kind:replace
- 2026-09-26 · 0929742* · exit 0 · `set -o pipefail …` · acceptance-sha256:493bee97294cbaed265392cff9ef7afb93da4a7b9e0241a584266202edfbf9f5 · ms:762
- 2026-09-26 · 0929742* · exit 0 · `set -o pipefail …` · acceptance-sha256:493bee97294cbaed265392cff9ef7afb93da4a7b9e0241a584266202edfbf9f5 · ms:322
- 2026-09-26 · 0929742* · exit 0 · `set -o pipefail …` · acceptance-sha256:493bee97294cbaed265392cff9ef7afb93da4a7b9e0241a584266202edfbf9f5 · ms:315

## Mutation Log
(empty until execute)
- 2026-09-26 · e9e3d3e* · mutant killed · exit 1 · `internal/rooted/links.go` · no name is a device candidate · acceptance-sha256:6e5f86b7881a3788fd0e7f69883b7313c8001af069056b127dd09815d144f09b · covers:a reserved name is a candidate by Go's rule
- 2026-09-26 · 0929742* · mutant killed · exit 1 · `internal/rooted/links.go` · only the last component is judged again · acceptance-sha256:493bee97294cbaed265392cff9ef7afb93da4a7b9e0241a584266202edfbf9f5 · covers:every component counts
- 2026-09-26 · 0929742* · mutant killed · exit 1 · `internal/rooted/links.go` · no name is a device candidate · acceptance-sha256:493bee97294cbaed265392cff9ef7afb93da4a7b9e0241a584266202edfbf9f5 · covers:a reserved name is a candidate by Go's rule

## Invariants

- On Windows mrw never creates, reads or edits a path any component of which, below the root, is a reserved device name — directly, through a link, or from an ast-grep hit.

## Risks

- The behavioural tests run only on Windows (`cmd/mrw/device_windows_test.go`: `TestADeviceNameIsRefusedNotReadAsAnEmptyFile`, `TestALinkToADeviceNameIsRefused`, `TestAnAstGrepHitOnADeviceNameIsDropped`); the local fence proves the candidate rule for every component, the Windows compile and that no OS query remains.

## Out of Scope

- A contract row (permanent: fact: `scripts/contract.sh` runs on Linux only; the Windows CI shard runs the test)

## Stop Condition

The fence exits 0 and the Windows CI shard passes `TestADeviceNameIsRefusedNotReadAsAnEmptyFile`.
