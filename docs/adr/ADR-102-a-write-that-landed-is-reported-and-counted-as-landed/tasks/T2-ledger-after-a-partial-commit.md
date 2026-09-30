# Task ADR-102-T2: the ledger records a partial commit

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** a partial commit's written files recorded in the ledger; the BACKLOG entry closed
**Consumes:** `writer.MutationOf(res)` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a partial commit licenses what it wrote`, `a removed file is dropped`, `no engine package changes`

## Goal

After a partial commit `writer.Apply` records each written file as wholly known at its new sha and drops each
removed one, as after a complete commit; a ledger failure there is joined to the commit error and is not a
`LedgerError`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/writer.go` | edit | `Apply` records whenever any file was written (not only when `Applied`), still skipping dry runs; `Drop` before `Record` |
| `internal/writer/partial102_test.go` | add | the test below |
| `docs/adr/BACKLOG.md` | edit | "The ledger after a partial commit" closed |

## Ordered Steps

1. [S1] Write the failing test `TestAPartialCommitRecordsWhatLanded`; confirm RED. [proof: mutation]
2. [S2] Record the partial commit. [proof: mutation] Mutant: the partial branch records nothing.
3. [S3] BACKLOG entry closed by ADR-102. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/writer/ -count=1 -timeout 300s -run 'TestAPartialCommitRecordsWhatLanded|TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPartialCommitRecordsWhatLanded \(' "$out" \
  && grep -qE '^--- PASS: TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError \(' "$out" \
  && grep -q 'Closed\*\* by ADR-102' docs/adr/BACKLOG.md \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPartialCommitRecordsWhatLanded` | `internal/writer/partial102_test.go` | after the read-only-directory partial commit the ledger holds `a.txt` wholly known at its `SHAAfter`, and `writer.Apply` returns a commit error that is not a `LedgerError`; an unlink of `c` with a rename `b`→`c` in one applied plan leaves `c` wholly known (Drop runs before Record) | — | S1, S2 |
| `TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError` | `internal/writer/partial102_test.go` | with `d/` and the ledger both read-only, the partial commit returns the commit error with "the ledger could not record what landed" added, and not a `LedgerError` (added after its mutant survived the first fence) | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every `mrw write` and `mrw_write` goes through `writer.Apply` and the tallies; the tests drive `rootCommand` or `Serve` |
| 3 — the caller can discover it | the receipt, `stats`, and the ledger's licence on the next write |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/writer/writer.go` · a partial commit records nothing again · acceptance-sha256:bb56ac423f26ec2d51112e96470f40b4b8bb4a33382522262e34ac5ed57f7fa0 · covers:a partial commit licenses what it wrote
- 2026-09-30 · f1d5996* · mutant survived · exit 0 · `internal/writer/writer.go` · a partial commit whose ledger fails returns a LedgerError · acceptance-sha256:bb56ac423f26ec2d51112e96470f40b4b8bb4a33382522262e34ac5ed57f7fa0 · covers:a partial commit licenses what it wrote
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:bb56ac423f26ec2d51112e96470f40b4b8bb4a33382522262e34ac5ed57f7fa0 · covers:no engine package changes
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/writer/writer.go` · a partial commit records nothing again · acceptance-sha256:435486eea701b3e42e712394bc4c4c704d8df9e17e764aa63d8132a74bbc5879 · covers:a partial commit licenses what it wrote
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/writer/writer.go` · a partial commit whose ledger fails returns a LedgerError · acceptance-sha256:435486eea701b3e42e712394bc4c4c704d8df9e17e764aa63d8132a74bbc5879 · covers:a partial commit licenses what it wrote
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:435486eea701b3e42e712394bc4c4c704d8df9e17e764aa63d8132a74bbc5879 · covers:no engine package changes

## Invariants

- Exit codes and hunk statuses are unchanged.
- A write that landed whole is counted and recorded exactly as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-102 tasks, each in its own file.

## Verification Log
- 2026-09-30 · f1d5996* · exit 1 · `set -o pipefail …` · acceptance-sha256:bb56ac423f26ec2d51112e96470f40b4b8bb4a33382522262e34ac5ed57f7fa0 · ms:107 · test-lock-sha256:0ca82f02926d791701d5fc2dda28ca03f2c35b6bcdf3e2c33e84e8c02d800f4c · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3dyaXRlci9wYXJ0aWFsMTAyX3Rlc3QuZ28JVGVzdEFQYXJ0aWFsQ29tbWl0UmVjb3Jkc1doYXRMYW5kZWQJNDhiNTcyMWQwZGJhYjJiNDFiNGYwNTE1Nzk3NzEwY2M0ZjY4OGNhOWMyNTNmNWY2YjA2MTJhMjE1ZDdiOWFiNw
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/writer/mutation102_test.go:22:96: undefined: None
  internal/writer/mutation102_test.go:23:109: undefined: Partial
  internal/writer/mutation102_test.go:24:92: undefined: Complete
  internal/writer/mutation102_test.go:26:13: undefined: MutationOf
  internal/writer/partial102_test.go:50:57: undefined: MutationOf
  internal/writer/partial102_test.go:50:76: undefined: Partial
  internal/writer/partial102_test.go:51:110: undefined: MutationOf
  internal/writer/partial102_test.go:51:110: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer [build failed]
  FAIL
  ```
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb56ac423f26ec2d51112e96470f40b4b8bb4a33382522262e34ac5ed57f7fa0 · ms:306
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb56ac423f26ec2d51112e96470f40b4b8bb4a33382522262e34ac5ed57f7fa0 · ms:306
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb56ac423f26ec2d51112e96470f40b4b8bb4a33382522262e34ac5ed57f7fa0 · ms:297
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:435486eea701b3e42e712394bc4c4c704d8df9e17e764aa63d8132a74bbc5879 · ms:283
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:435486eea701b3e42e712394bc4c4c704d8df9e17e764aa63d8132a74bbc5879 · ms:347
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:435486eea701b3e42e712394bc4c4c704d8df9e17e764aa63d8132a74bbc5879 · ms:318
- 2026-09-30 · human-observed · relock 2026-09-30: after the red run, TestAPartialCommitRecordsWhatLanded's cleanup captured d before root was reassigned (it restored the wrong directory, so TempDir cleanup failed); every assertion kept. TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError was added after its mutant (a partial commit's ledger failure returned as LedgerError) survived the first fence; it now kills it
- 2026-09-30 · f1d5996* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:435486eea701b3e42e712394bc4c4c704d8df9e17e764aa63d8132a74bbc5879 · ms:0 · test-lock-sha256:49014da273d2994a93a1f21748a31ec6c288a57dd7f76e62fd4e1b0200e1cb25 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3dyaXRlci9wYXJ0aWFsMTAyX3Rlc3QuZ28JVGVzdEFQYXJ0aWFsQ29tbWl0UmVjb3Jkc1doYXRMYW5kZWQJM2NiMzMxZjJkZDcyZGM1OGYxY2U0MjZmNTUwMjJlZGE5YjQ5MWExMWM2MGQwMmM5MzE3ZGM5MmFiMDNmYTM1Ygpib2R5CWludGVybmFsL3dyaXRlci9wYXJ0aWFsMTAyX3Rlc3QuZ28JVGVzdEFQYXJ0aWFsQ29tbWl0V2hvc2VMZWRnZXJGYWlsc0tlZXBzSXRzQ29tbWl0RXJyb3IJZjA1MDljNWZhMmVmM2U5ZmE0MjVlODA3YzFkZDk1ZDg1MzZhZjNjZDg5MzIzOWUxOTU5ZDkxMDRhODdjNDgyYw · test-lock-kind:replace
