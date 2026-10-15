# Task ADR-159-T1: the rename task graded wrongly is superseded, and a finished stage's fence reads the corpus it ran

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `superseded_by` on `bench/tasks/cac-rename-post-send/task.json`; `bench/tasks/cac-rename-post-send-2/`; ADR-158 T3's fence pinned to `edb65629`
**Consumes:** `superseded_by` and `SelectTasks` (ADR-158 Decision 3, already on main)
**Data dependency:** `crossagentschat` at `ae4f7a9c`, on the machine that runs the fence
**Proof map:** v1
**Rests-on:** `the replacement's oracle passes a fix that leaves the test helper`, `the replacement's oracle fails a fix that renames the helper too`, `the old task is not run by default`

## Goal

Decisions 1 and 3 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/tasks/cac-rename-post-send/task.json` | edit | `superseded_by` only |
| `bench/tasks/cac-rename-post-send-2/task.json`, `reference.patch` | add | the replacement |
| `internal/bench/oracle_test.go` | edit | `TestThePostSendRenameLeavesTheTestHelper` |
| `docs/adr/ADR-158-stage-1b-closes-the-gaps-stage-1-found/tasks/T3-stage-1b-sonnet.md` | edit | its fence compares `run.json` with the corpus at `edb65629` |

## Ordered Steps

1. [S1] Write `TestThePostSendRenameLeavesTheTestHelper` and see it red: on `cac-rename-post-send-2`'s tree, the reference fix is solved, and the old reference fix (which also renames the helper) is not; it fails while the task is missing.
2. [S2] The replacement: instruction naming the handler method and the helper to leave; the reference fix is the old one's `httpapi.go` and `send.go` hunks; `exact`; require `sendMessage` in both files and the helper's definition in `send_test.go`; forbid `postSend` in both files and `sendMessage` in `send_test.go`. [proof: mutation]
3. [S3] `superseded_by` on the old task; the shape counts stay as registered (8 active `multi-site`). [proof: acceptance]
4. [S4] ADR-158 T3's fence reads the task ids from `git show edb65629:bench/tasks/<id>/task.json`, re-verified, with a mutant. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ -count=1 -timeout 900s -run 'TestThePostSendRenameLeavesTheTestHelper|TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit|TestTheCorpusHasTheRegisteredShapeCounts|TestASupersededTaskIsNotRunByDefault' -v > "$out" 2>&1 \
  && grep -qE '^--- PASS: TestThePostSendRenameLeavesTheTestHelper \(' "$out" \
  && ! grep -q -- '--- SKIP: TestThePostSendRenameLeavesTheTestHelper' "$out" \
  && grep -qE '^    --- PASS: TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit/cac-rename-post-send-2 \(' "$out" \
  && grep -qE '^--- PASS: TestTheCorpusHasTheRegisteredShapeCounts \(' "$out" \
  && grep -q '"superseded_by": "cac-rename-post-send-2"' bench/tasks/cac-rename-post-send/task.json
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestThePostSendRenameLeavesTheTestHelper` | `internal/bench/oracle_test.go` | the replacement passes the method-only fix and refuses the fix that renames the helper too | none | S1, S2 |
| `TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit` | `internal/bench/oracle_test.go` | the replacement's oracle, as every task's | none | S2 |
| `TestTheCorpusHasTheRegisteredShapeCounts` | `internal/bench/oracle_test.go` | 8 active multi-site tasks still | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the task |
| 2 — something selects it | `bench run`'s default plan |
| 3 — the caller can discover it | `bench/tasks/`; `docs/bench/README.md`'s `superseded_by` line |
| 4 — it is used | the next stage (BACKLOG) |

## Mutation Log
(empty until execute)
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `bench/tasks/cac-rename-post-send-2/task.json` · S2: the replacement oracle refuses the method-only fix it asks for · acceptance-sha256:4115fd324865a6349ff845a2da1bfededbafecff571315f5eaecc0f01c334fca
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `bench/tasks/cac-rename-post-send/task.json` · S3: the old task counted as active, so the shape counts break · acceptance-sha256:4115fd324865a6349ff845a2da1bfededbafecff571315f5eaecc0f01c334fca

## Verification Log
(empty until execute)
- 2026-10-15 · ffbcfd04* · exit 1 · `set -o pipefail …` · acceptance-sha256:4115fd324865a6349ff845a2da1bfededbafecff571315f5eaecc0f01c334fca · ms:11521 · test-lock-sha256:5ae2ce024306a0d254c89e60bf755651c0f751d60e33f31b54e9f2472d200c2e · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEFEZWNsYXJlZFNlY3Rpb25Jc0Ryb3BwZWRBbmRBTWlzc2luZ09uZVJlZnVzZXMJZWZmYjkwZjAyZWZjMGQ5MzBjMjkwZjg3NzE2M2M2OTljMWJkMDJjMTFmZDJkZTJlY2U2MTViOGNjYTNlNmFhZgpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RBZ2VudHNtZW1vcnlzVHJlZUxvc2VzQm90aERlY2xhcmVkU2VjdGlvbnMJMDVmMzA0NGJjMjA2M2Q2ZjFiMjY1NWVkNDg5MGI4YzFlMjdiMjczYTU0MjQ5ZDY0NTIwNTI1NDAyYTdhODBkYgpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RFdmVyeU9yYWNsZUZhaWxzT25CYXNlUGFzc2VzT25GaXhBbmRSZWZ1c2VzQVN0cmF5RWRpdAkwOTA0ODJjNDAzNzA3Y2IyZWM0MjFjYjkwZmY0OTA0ZWYxYzQzYjk5NTdlYmVmOTI3ZGY0ZjdhNzg3MWIwM2M0CmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5VmFyaWVkTXVsdGlTaXRlVGFza0hhc0FSZXBlYXRlZEFuY2hvckFjcm9zc1R3b0ZpbGVzCWQyMzY1YmIwNzRhMmNjNmQyZjYzMjMxNTlmNzVhZjYyZGFmMWY3ZGI5YTU3NzQxNDVmMGUyZjU1ZGY3NmEzNDEKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCTU4NTAxMDNjNmJhMDZhOTNhMWQyMWRhMjIxNGY2YjAxZjhjNTdlZjFkYjdmYWM1N2RiMWU0NjMzMDg0MDI0ODYKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAk0OTlkMWRiNmY0NzlkMTY2Mzk1OGVjYWY3MTIxYzIwMDk3NmU5OGU2ZTYxYjQ5MGEyZjZiZjhmNTBkMzYwY2RhCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZVBvc3RTZW5kUmVuYW1lTGVhdmVzVGhlVGVzdEhlbHBlcgkyMWVmZDQ0YmM4MjE1ZDRlNGRjZmIxNTEwYzU3ZTRiMWZmMjg0MTg5MGJkMTEyN2ZhNGQyYmRjMGY5YWE1YzA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZVByZXBhcmVkVHJlZVJlZGFjdHNJbnN0cnVjdGlvbnNBbmRTdHJpcHNDb25maWd1cmF0aW9uCWIwNjBkMTZlYmRjNTVmZmVlYTkwZjFmMDRjNDI5ZjY0YmQ1ODFmYzU2MTI2ZmM0NmZlZDRmZTFjM2EyZGZkZmYKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlVmFyaWVkT3JhY2xlc0dyYWRlV2hhdFRoZVJldmlld09mNDE3Rm91bmQJZDEyZThhMmM0ZWM2MzIzYTU1NzQ3NDQxZWYzOGFmZDUwYjQ3ZmE2NWY1YWU5OGNjZDNiYTc2M2RjMTEwMjM1MQ
  ```
  ```
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:4115fd324865a6349ff845a2da1bfededbafecff571315f5eaecc0f01c334fca · ms:11746
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:4115fd324865a6349ff845a2da1bfededbafecff571315f5eaecc0f01c334fca · ms:11998

## Invariants

- The old task's instruction, oracle and reference fix are byte-for-byte unchanged; it gains `superseded_by` only.
- No published receipt changes.

## Risks

- The replacement's instruction now tells the agent which function to leave, so it is easier than the old one: it is a new task with a new id, measured from the next stage on.

## Out of Scope

- Running the replacement (deferred: docs/adr/BACKLOG.md "From ADR-159")

## Stop Condition

The fence exits 0; or the replacement cannot be given an oracle that fails on base and passes on its fix.
