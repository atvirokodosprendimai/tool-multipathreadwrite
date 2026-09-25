# Task ADR-068-T1: `parseLine` keeps a path's edge spaces; contract §127

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** exact ledger keys for paths with a trailing space or carriage return
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a loaded path keeps its spaces`, `a read of x-space does not license x`, `the binary refuses the trimmed sibling`, `no engine file changes but seen`

## Goal

The ledger reader trims nothing: `Load` and `IsStale` split on `\n` alone (`scanLF`), and
`parseLine` (`internal/seen/seen.go`) keeps the path. An observation of `x ` or `x\r` loads under
its own key and licenses nothing else. After `mrw read -- "x "`, a write to `x` is refused as unread
(exit 1, nothing written), and a write to `x ` applies. The same holds for `x\r`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `parseLine` trims nothing; `scanLF` for `Load` and `IsStale` |
| `internal/seen/ledgerpath_test.go` | new | `TestALedgerPathKeepsItsSurroundingSpaces` |
| `cmd/mrw/ledgerpath_test.go` | new | `TestAReadOfATrailingSpacePathDoesNotLicenseItsTrimmedSibling` |
| `scripts/contract.sh` | edit | §127 |
| `docs/adr/BACKLOG.md` | edit | the deferred `internal/iter` sibling |

## Ordered Steps

1. [S1] Write the tests and confirm each RED on the current tree on an assertion. [proof: mutation]
   - `TestALedgerPathKeepsItsSurroundingSpaces`: `Record` observations for `x `, ` y`, `a  b` and
     `z\r`, then `Load`. Each loads under its exact key; `x`, `y` and `z` are absent.
   - `TestAReadOfATrailingSpacePathDoesNotLicenseItsTrimmedSibling`, one subtest each for `x ` and
     `x\r`: files `x` and the name with identical bytes; the CLI reads only the name (after `--`).
     A write to `x` exits 1 with `x` unchanged; a write to the name (quoted in the plan) exits 0.
2. [S2] Change `parseLine`; confirm GREEN, and that every `internal/seen` and `cmd/mrw` test stays
   green. [proof: mutation]
   Mutant: `TrimSpace` restored, which kills both tests.
3. [S3] §127 through the binary, the pair above. RED against v1.24.0 in a mini-harness, GREEN in the
   full `./scripts/contract.sh`. [proof: mutation]
4. [S4] BACKLOG row for the `internal/iter` sibling; `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 127\. ' scripts/contract.sh \
  && go test ./internal/seen/ ./cmd/mrw/ -count=1 -v \
    -run 'TestALedgerPathKeepsItsSurroundingSpaces|TestAReadOfATrailingSpacePathDoesNotLicenseItsTrimmedSibling' 2>&1 | tee /tmp/adr068-t1.out \
  && grep -q '^--- PASS: TestALedgerPathKeepsItsSurroundingSpaces ' /tmp/adr068-t1.out \
  && grep -q '^--- PASS: TestAReadOfATrailingSpacePathDoesNotLicenseItsTrimmedSibling ' /tmp/adr068-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr068-t1.out \
  && ./scripts/contract.sh > /tmp/adr068-t1-contract.out 2>&1 \
  && grep -q '^  PASS  a read of a trailing-space path does not license its trimmed sibling' /tmp/adr068-t1-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/seen cmd/mrw)" ] \
  && go vet ./internal/seen/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestALedgerPathKeepsItsSurroundingSpaces` | `internal/seen/ledgerpath_test.go` | a loaded path keeps its edge and inner spaces | — | S1, S2 |
| `TestAReadOfATrailingSpacePathDoesNotLicenseItsTrimmedSibling` | `cmd/mrw/ledgerpath_test.go` | ADR-002 end to end: the trimmed sibling is refused, the read file is writable | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §127 |
| 2 — something selects it | `Load` calls `parseLine` for every ledger line; `apply` consults the loaded map |
| 3 — the caller can discover it | the refusal names the unread file |
| 4 — it is used | every write consults the ledger |

## Verification Log
(empty until execute)
- 2026-09-25 · 9efc095* · exit 1 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:37 · test-lock-sha256:a5c368f0aea562605b0ebffe990215b24135509976eda96c7852c9ba6be1738b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGVkZ2VycGF0aF90ZXN0LmdvCVRlc3RBUmVhZE9mQVRyYWlsaW5nU3BhY2VQYXRoRG9lc05vdExpY2Vuc2VJdHNUcmltbWVkU2libGluZwllYWM4NjBiZTIwMDE0Y2U5YTFiZjgwZWIwOGI2YmY2OGJhOTcxMzYzNWZkMTc3ODU2OGZhN2M2ZDgyOGE5OWIzCmJvZHkJaW50ZXJuYWwvc2Vlbi9sZWRnZXJwYXRoX3Rlc3QuZ28JVGVzdEFMZWRnZXJQYXRoS2VlcHNJdHNTdXJyb3VuZGluZ1NwYWNlcwk2MDhiOGViMzM0OTBkYTBlYmNlMTllOWI0ZWUzMGI4Yzg2ZTdhNmE2MmFlMWE5YmFkZGE1NjhhNzhjNmM3MTVi
  ```
  ```
- 2026-09-25 · 9efc095* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:31203
- 2026-09-25 · 9efc095* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:30578
- 2026-09-25 · 9efc095* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:30892
- 2026-09-25 · 9efc095* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:30920
- 2026-09-25 · 9efc095* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:0 · test-lock-sha256:cb99ac20b580d5d40c1fd616f53c0afe19221b1ba7da768bf2d54eab24eef440 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGVkZ2VycGF0aF90ZXN0LmdvCVRlc3RBUmVhZE9mQVRyYWlsaW5nU3BhY2VQYXRoRG9lc05vdExpY2Vuc2VJdHNUcmltbWVkU2libGluZwk0YTg4MmI3OTAzNDE1MzExNWU0NDVhZjZiNGM5ZjZhMTBhOGNhNjBmMDdmYjA2ODZlZWQ0MjQ4NGUxMGYzMjUwCmJvZHkJaW50ZXJuYWwvc2Vlbi9sZWRnZXJwYXRoX3Rlc3QuZ28JVGVzdEFMZWRnZXJQYXRoS2VlcHNJdHNTdXJyb3VuZGluZ1NwYWNlcwk2MDhiOGViMzM0OTBkYTBlYmNlMTllOWI0ZWUzMGI4Yzg2ZTdhNmE2MmFlMWE5YmFkZGE1NjhhNzhjNmM3MTVi · test-lock-kind:replace
- 2026-09-25 · 9efc095* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:31280
- 2026-09-25 · 01b07d7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:0 · test-lock-sha256:485223008a6d290672fc150b20f40305bbadee9c223f9a152c3df9b5549847d8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGVkZ2VycGF0aF90ZXN0LmdvCVRlc3RBUmVhZE9mQVRyYWlsaW5nU3BhY2VQYXRoRG9lc05vdExpY2Vuc2VJdHNUcmltbWVkU2libGluZwk2OTc4Y2IzNzg1ZjdkNTc1MTNlYWFmMzUxZWQ5NzdmODgwODNlMDliYTgxOWM4ZGNiOWVhNmJjNTljNTg0OTM5CmJvZHkJaW50ZXJuYWwvc2Vlbi9sZWRnZXJwYXRoX3Rlc3QuZ28JVGVzdEFMZWRnZXJQYXRoS2VlcHNJdHNTdXJyb3VuZGluZ1NwYWNlcwkyZDQxYzJkNWVjZDdjZTIyZWViMTA2YTU0YmU4NjMzYTJjZTQ0MmI3NWZmMmU3YjBjNGIxNTMxNDU4MmI4NDQy · test-lock-kind:replace
- 2026-09-25 · 01b07d7* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:32479
- 2026-09-25 · 01b07d7* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:31821
- 2026-09-25 · 01b07d7* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:31699
- 2026-09-25 · 4d558fc* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · ms:35157

## Mutation Log
(empty until execute)
- 2026-09-25 · 9efc095* · mutant killed · exit 1 · `internal/seen/seen.go` · parseLine trims the whole line again: both tests and §127 must go red · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · covers:the binary refuses the trimmed sibling
- 2026-09-25 · 9efc095* · mutant killed · exit 1 · `internal/seen/seen.go` · parseLine trims the whole line again: both tests and §127 must go red · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · covers:a loaded path keeps its spaces
- 2026-09-25 · 9efc095* · mutant killed · exit 1 · `internal/seen/seen.go` · parseLine trims the whole line again: both tests and §127 must go red · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · covers:a read of x-space does not license x
- 2026-09-25 · 9efc095* · mutant killed · exit 1 · `internal/iter/iter.go` · an engine file outside internal/seen changes: the go/no-go guard must go red · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · covers:no engine file changes but seen
- 2026-09-25 · 01b07d7* · mutant killed · exit 1 · `internal/seen/seen.go` · scanLF drops a CR before the newline, as bufio.ScanLines did: the z\r key and the x\r subtest must go red · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · covers:a loaded path keeps its spaces
- 2026-09-25 · 01b07d7* · mutant killed · exit 1 · `internal/seen/seen.go` · scanLF drops a CR before the newline, as bufio.ScanLines did: the x\r subtest and §127 CR row must go red · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · covers:the binary refuses the trimmed sibling
- 2026-09-25 · 01b07d7* · mutant killed · exit 1 · `internal/seen/seen.go` · parseLine trims the whole line again: the x-space key, subtest and §127 must go red · acceptance-sha256:ba7596f570227a692e8ac9e9ec6a1bf32c7379a8512945bf6e3e99bc77bc4345 · covers:a read of x-space does not license x

## Invariants

- The ledger format and header are unchanged; no ledger mrw wrote is discarded.
- A path with no edge spaces loads exactly as before.

## Risks

- Windows cannot hold both names: Win32 strips a trailing space from a short path, and `\r` is not a legal name character. The CLI subtests skip there, visibly; `internal/seen`'s test covers the ledger on every platform.

## Out of Scope

- The `internal/iter` working set trimming its lines (deferred: docs/adr/BACKLOG.md)

## Stop Condition

Stop if keeping the path exactly requires a ledger format change.
