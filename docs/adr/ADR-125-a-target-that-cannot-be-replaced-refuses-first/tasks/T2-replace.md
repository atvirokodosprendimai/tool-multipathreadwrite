# Task ADR-125-T2: on Windows a held target fails before any rename

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `apply.replaceableFn`, the probe loop before the first rename
**Consumes:** `apply.causeOf` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `on Windows a held target fails before any rename`

## Goal

Before the first rename, every existing content target, unlink source and rename source is asked whether it can be replaced; on Windows that is an open for `DELETE` with full sharing, closed at once, and a refusal fails the plan with nothing written and a receipt naming the cause. On unix the probe answers yes. The commit rename keeps going through `os.Root.Rename`, which on Windows already replaces with POSIX semantics.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/replace.go` | edit | `replaceableFn`, the seam |
| `internal/apply/replace_windows.go` | edit | the probe: `syscall.CreateFile` for `DELETE` with full sharing |
| `internal/apply/replace_other.go` | edit | the unix probe, which answers yes |
| `internal/apply/apply.go` | edit | the probe loop before the first rename |
| `internal/apply/replace125_test.go` | edit | the loop, driven through the seam on every platform |
| `internal/apply/replace125_windows_test.go` | add | the held-handle tests on Windows |

## Ordered Steps

1. [S1] Write `TestEveryTargetIsAskedBeforeTheFirstRename`. Confirm RED. [proof: mutation]
2. [S2] The seam, the platform probes and the loop. Mutants: the loop removed; unlink and rename sources not asked. [proof: mutation]
3. [S3] The Windows tests `TestAHeldTargetRefusesBeforeAnyRename`, `TestAHolderThatSharesDeleteDoesNotBlockTheCommit` and `TestAnInvalidNameGetsAReceipt`, red on a pushed test-only commit and green on the head; and `TestTheProbeDoesNotFollowASwappedParentOutOfTheRoot` (the Codex review of #338), green on the head. [proof: human: the windows-shard run URLs, red and green]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestEveryTargetIsAskedBeforeTheFirstRename' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestEveryTargetIsAskedBeforeTheFirstRename \(' "$out" \
  && go test ./internal/apply/ ./cmd/mrw/ -count=1 -timeout 900s \
  && GOOS=windows go vet ./internal/apply/ \
  && GOOS=windows go test -c -o /dev/null ./internal/apply/ \
  && grep -q '^func TestAHeldTargetRefusesBeforeAnyRename(' internal/apply/replace125_windows_test.go \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryTargetIsAskedBeforeTheFirstRename` | `internal/apply/replace125_test.go` | a content target, an unlink source and a rename source are each asked before any rename; a refusal on the last one leaves the first unchanged, NOTHING WRITTEN, the hunk naming the cause | none | S1, S2 |
| `TestAHeldTargetRefusesBeforeAnyRename` | `internal/apply/replace125_windows_test.go` | a target held without delete sharing, last in the plan, fails it with nothing written, naming "held open" | none | S3 |
| `TestAHolderThatSharesDeleteDoesNotBlockTheCommit` | `internal/apply/replace125_windows_test.go` | a holder sharing read, write and delete does not stop the plan | none | S3 |
| `TestAnInvalidNameGetsAReceipt` | `internal/apply/replace125_windows_test.go` | `q?.txt` is refused on its hunk, naming the name | none | S3 |
| `TestTheProbeDoesNotFollowASwappedParentOutOfTheRoot` | `internal/apply/replace125_windows_test.go` | a parent swapped for a junction out of the root after validation is refused, and the file outside is never opened | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `replaceableFn` |
| 2 — something selects it | `Apply`, before the first rename of every write |
| 3 — the caller can discover it | the receipt's reason; AGENTS.md |
| 4 — it is used | the Windows peers met it on 2026-10-02; no telemetry (ADR-009) |

## Invariants

- On unix the commit is unchanged.
- The probe changes nothing on disk.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the windows-shard run shows a delete-sharing holder blocking `os.Root.Rename`.

## Out of Scope

- A holder that takes the file after the probe (permanent: boundary: ADR-066's PARTIALLY APPLIED reports it)

## Mutation Log
- 2026-10-06 · cc32ee4* · mutant inconclusive · exit 1 · `internal/apply/apply.go` · S2: content targets not asked · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-06 · cc32ee4* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: unlink and rename sources not asked · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c
- 2026-10-06 · cc32ee4* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: content targets not asked · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c
- 2026-10-06 · edfd38c* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: content targets not asked · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c
- 2026-10-06 · edfd38c* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: unlink and rename sources not asked · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c

## Verification Log
- 2026-10-06 · cc32ee4* · exit 1 · `set -o pipefail …` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:140 · test-lock-sha256:99938bc4c8aade5d0fcfa4aee535f2e221d289b6f049d730783264df05c30ea3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV90ZXN0LmdvCVRlc3RBblVucmVhZGFibGVUYXJnZXRHZXRzQVJlY2VpcHQJODhjMDQyM2YzMzEyODU5YzdjY2E1ZTU0YjU5YWU5MjA2MjUxNmIwY2RjZmY0MjU5NzRlMjJmZmQ3NGNiZjg3Ywpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfdGVzdC5nbwlUZXN0RXZlcnlUYXJnZXRJc0Fza2VkQmVmb3JlVGhlRmlyc3RSZW5hbWUJNGZjZmNmZjdlYTJkMjA2YWNjYzdiYjAwOGRlMmE1YmY5NDg3ZWJkODFhOTQ2OTQ0ZDMwZjk3MTA1YTllMDU0Ngpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfdGVzdC5nbwlUZXN0VGhlSWRlbnRpdHlSZWZ1c2FsTmFtZXNJdHNDYXVzZQliNDE4MmZkYjM1MGFkZTAyN2EzYmYzOWEzOTE5NjI2NzljYzUxNjMzOTZmNjExNGI0ZmE3NWY5NjUwNDljN2E2CnVucHJvdmVuCWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfd2luZG93c190ZXN0LmdvCVRlc3RBSGVsZFRhcmdldFJlZnVzZXNCZWZvcmVBbnlSZW5hbWUKdW5wcm92ZW4JaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV93aW5kb3dzX3Rlc3QuZ28JVGVzdEFIb2xkZXJUaGF0U2hhcmVzRGVsZXRlRG9lc05vdEJsb2NrVGhlQ29tbWl0CnVucHJvdmVuCWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfd2luZG93c190ZXN0LmdvCVRlc3RBbkludmFsaWROYW1lR2V0c0FSZWNlaXB0
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply.test]
  internal/apply/replace125_test.go:91:10: undefined: replaceableFn
  internal/apply/replace125_test.go:92:21: undefined: replaceableFn
  internal/apply/replace125_test.go:94:2: undefined: replaceableFn
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:37009
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:37928
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:38648
- 2026-10-06 · edfd38c* · exit 0 · `set -o pipefail …` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:40384
- 2026-10-06 · edfd38c* · exit 0 · `set -o pipefail …` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:38823
- 2026-10-06 · human-observed · observed: S3 on the windows shards — red run 37444912631 (tests alone on main, draft #336): TestAHeldTargetRefusesBeforeAnyRename failed (a.txt written before the held target) and TestAnInvalidNameGetsAReceipt failed (bare error); green run 37451927819 on head 110a2a7: windows-shard (5) ok for internal/apply, normal and race, with all four Windows tests unable to skip (TestTheProbeDoesNotFollowASwappedParentOutOfTheRoot fails on a mklink error).
- 2026-10-06 · 110a2a7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:0 · test-lock-sha256:95d77f28f916620a10e9754cba50c09109b6b73f871e076bdcfd65d766b8e627 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV90ZXN0LmdvCVRlc3RBblVucmVhZGFibGVUYXJnZXRHZXRzQVJlY2VpcHQJODhjMDQyM2YzMzEyODU5YzdjY2E1ZTU0YjU5YWU5MjA2MjUxNmIwY2RjZmY0MjU5NzRlMjJmZmQ3NGNiZjg3Ywpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfdGVzdC5nbwlUZXN0RXZlcnlUYXJnZXRJc0Fza2VkQmVmb3JlVGhlRmlyc3RSZW5hbWUJMWFkYzhjNmJmMTJiYWNhOWZkMGI2NDQxODE3ZWYzMTVmODRmMDM5MmJhNDJiODBlNzk3ZWI5Y2JhYmJhMDAwYwpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfdGVzdC5nbwlUZXN0VGhlSWRlbnRpdHlSZWZ1c2FsTmFtZXNJdHNDYXVzZQliNDE4MmZkYjM1MGFkZTAyN2EzYmYzOWEzOTE5NjI2NzljYzUxNjMzOTZmNjExNGI0ZmE3NWY5NjUwNDljN2E2CmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV93aW5kb3dzX3Rlc3QuZ28JVGVzdEFIZWxkVGFyZ2V0UmVmdXNlc0JlZm9yZUFueVJlbmFtZQk0NjIwZTY1MTViZmYzYmI2YzI3N2E3ODA5ZDBiZDUyZjQ0NDBkNDUwMTM3NmE3ODU4NTFmZTg4ODBiODQ3NTEyCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV93aW5kb3dzX3Rlc3QuZ28JVGVzdEFIb2xkZXJUaGF0U2hhcmVzRGVsZXRlRG9lc05vdEJsb2NrVGhlQ29tbWl0CWExYmNiYmFhNzNkOTkyOTY4ZjE3N2ExYzc0ZGU1ZmM5OGJmOWRjZTJjZGI2ZjA3MGIyMDg0M2NiZGM5MmUxNjUKYm9keQlpbnRlcm5hbC9hcHBseS9yZXBsYWNlMTI1X3dpbmRvd3NfdGVzdC5nbwlUZXN0QW5JbnZhbGlkTmFtZUdldHNBUmVjZWlwdAkwYTljZDg5M2E5Nzk3YTdlNTY3NWQ2NDM0ODM5YWViNzIyZTdjZjlmZGNlMGI0ZTM3YTdiYmE2ZTJkOThmMDkzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV93aW5kb3dzX3Rlc3QuZ28JVGVzdFRoZVByb2JlRG9lc05vdEZvbGxvd0FTd2FwcGVkUGFyZW50T3V0T2ZUaGVSb290CWUyYjM4OWVkNjQ3OGYyNDJjZDM2YTZjODNhYzI1NGE1MmY5ZDhmMTRkMjQ1OWZhY2JiZjhkMzNlOTAwYjYzZWM · test-lock-kind:replace
- 2026-10-06 · d493df7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:265c73d2b70193753b81a3990e6fee4febb7c301ee00b7b744c1a23b70020f0c · ms:0 · test-lock-sha256:f71b15613dee59d2d473ad51b24e642d1f2b3062bd14600cdf81cd7e6968ab19 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV90ZXN0LmdvCVRlc3RBblVucmVhZGFibGVUYXJnZXRHZXRzQVJlY2VpcHQJODhjMDQyM2YzMzEyODU5YzdjY2E1ZTU0YjU5YWU5MjA2MjUxNmIwY2RjZmY0MjU5NzRlMjJmZmQ3NGNiZjg3Ywpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfdGVzdC5nbwlUZXN0RXZlcnlUYXJnZXRJc0Fza2VkQmVmb3JlVGhlRmlyc3RSZW5hbWUJYThiZmZkYmE3OGIyNmQxY2JlZjgwZGIyYTAwODNmOWI5Mjk1OTRhYmUxODBkNTYzMDc1NWMxOWI4OWYzZDU4Ygpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfdGVzdC5nbwlUZXN0VGhlRm9yY2VDbGF1c2VJc0N1dEZyb21UaGVXb3Jkc05vdFRoZUNhbGxlcnNQYXRoCTUzNjM2OTZhMzM1MWFmNjFjMTNjZGUwZDdhZDc0NGRiZjU0ODY2NTY0ZDc5YjY0NzQ2YzFhYWNmMmRlYjA3OTMKYm9keQlpbnRlcm5hbC9hcHBseS9yZXBsYWNlMTI1X3Rlc3QuZ28JVGVzdFRoZUlkZW50aXR5UmVmdXNhbE5hbWVzSXRzQ2F1c2UJYjQxODJmZGIzNTBhZGUwMjdhM2JmMzlhMzkxOTYyNjc5Y2M1MTYzMzk2ZjYxMTRiNGZhNzVmOTY1MDQ5YzdhNgpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfd2luZG93c190ZXN0LmdvCVRlc3RBSGVsZFRhcmdldFJlZnVzZXNCZWZvcmVBbnlSZW5hbWUJNGFhMWU2NGExZTU2ZWQ1Yjc0YTY0NzlkOTIzNjQ5OTFkMWMxZjkzM2Q0M2I4MWIzZjU2NTA2ODMxMTY1NmZjZQpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfd2luZG93c190ZXN0LmdvCVRlc3RBSG9sZGVyVGhhdFNoYXJlc0RlbGV0ZURvZXNOb3RCbG9ja1RoZUNvbW1pdAlhMWJjYmJhYTczZDk5Mjk2OGYxNzdhMWM3NGRlNWZjOThiZjlkY2UyY2RiNmYwNzBiMjA4NDNjYmRjOTJlMTY1CmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV93aW5kb3dzX3Rlc3QuZ28JVGVzdEFuSW52YWxpZE5hbWVHZXRzQVJlY2VpcHQJMGE5Y2Q4OTNhOTc5N2E3ZTU2NzVkNjQzNDgzOWFlYjcyMmU3Y2Y5ZmRjZTBiNGUzN2E3YmJhNmUyZDk4ZjA5Mwpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfd2luZG93c190ZXN0LmdvCVRlc3RUaGVQcm9iZURvZXNOb3RGb2xsb3dBU3dhcHBlZFBhcmVudE91dE9mVGhlUm9vdAkwNzU0ZDllZDkyNjkyYmI4NzE1YzU5MWExYTk2MmFhZGJjNDJmMDE1NzQzMWQwYTA1ZDQ0MzNlMTk1OTQ3NDU4 · test-lock-kind:replace
