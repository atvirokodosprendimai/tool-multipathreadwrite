# Task ADR-105-T3: the licence files are synced; the tree sync is measured

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `state.WriteSynced`; the ledger and the acknowledgement store synced; the tree-sync measurement
**Consumes:** `state.Write` (T2)
**Data dependency:** the tree measurement needs a quiet machine (load below the core count); the fence is hermetic
**Proof map:** v1
**Rests-on:** `the licence files are synced before the rename`, `the measurement is recorded against the registered bar`

## Goal

`state.WriteSynced` syncs the temp file before the rename, and the files that carry licences use it: the ledger
(`seen.save`), a legacy ledger migrated into the state directory (`state.Migrate`), and the MCP acknowledgement
store (`ack.go`). The tree-file sync is measured against the bar the plan registered (the record's Decision 3) and
recorded.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/write.go` | edit | `WriteSynced` and its `syncFn` seam |
| `internal/state/sync105_test.go` | add | the test below |
| `internal/seen/seen.go` | edit | `save` uses `WriteSynced` |
| `internal/mcp/ack.go` | edit | the store uses `WriteSynced` |
| `internal/state/state.go` | edit | `Migrate` uses `WriteSynced` (the review of the record) |

## Ordered Steps

1. [S1] Write the failing test `TestTheLicenceFilesAreSyncedBeforeTheyAreRenamed`; confirm RED. [proof: mutation]
2. [S2] `WriteSynced`; `seen.save`, `Migrate` and the ack store use it. [proof: mutation] Mutants: the sync removed; `seen.save` uses `Write`; `Migrate` uses `Write`.
3. [S3] Measure the tree sync: a fixture of 500 files of 100 lines in 20 directories, a plan of 10 single-line replaces per file (5,000 hunks), every file read first, then `mrw write --no-check` timed alone, each run on a fresh copy with its own `XDG_STATE_HOME`; a build with `tmp.Sync()` added in `stageFile` (a probe build, not committed) against one without, 5 alternating runs each; record the medians and the load in a sign-off. [proof: human: the medians, the load average and the machine, against the ≤10 % bar]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/state/ -count=1 -timeout 300s -run 'TestTheLicenceFilesAreSyncedBeforeTheyAreRenamed' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheLicenceFilesAreSyncedBeforeTheyAreRenamed \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/lines internal/rooted internal/read internal/check internal/links \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/lines internal/rooted internal/read internal/check internal/links)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheLicenceFilesAreSyncedBeforeTheyAreRenamed` | `internal/state/sync105_test.go` | through `syncFn` and `renameFn`, `WriteSynced` syncs the temp file and only then renames it, and `Write` does not sync; the non-test source of `internal/seen/seen.go` and `internal/mcp/ack.go` writes through `WriteSynced` alone, and `Migrate` writes through `WriteSynced` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/state/write.go` |
| 2 — something selects it | `seen.save` on every run; the ack store on every MCP read |
| 3 — the caller can discover it | the README matrix (T4) |
| 4 — it is used | every mrw run saves the ledger |

## Mutation Log
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/state/write.go` · WriteSynced does not sync · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · covers:the licence files are synced before the rename
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/seen/seen.go` · the ledger is written unsynced · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · covers:the licence files are synced before the rename
- 2026-09-30 · 1e77123* · mutant killed · exit 1 · `internal/state/state.go` · a migrated legacy ledger is written unsynced · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · covers:the licence files are synced before the rename

## Invariants

- The tally, ring, pricing, working set and marker are not synced (the record's Decision 3).
- Exit codes keep their meanings.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the tree measurement passes the bar: syncing staged files then needs its own task, written before
any code. On a fail, T4 states the power-loss window with the numbers.

## Out of Scope

- Syncing the tally, ring, pricing, working set and marker (permanent: boundary: measurement and convenience files; the record's Decision 3)

## Verification Log
- 2026-09-30 · 1bd8690* · exit 1 · `set -o pipefail …` · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · ms:114 · test-lock-sha256:4571a48e2a56d044ccb1863a9cb0e1dae18038f6bb6d8438690a53991136f8e9 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3N0YXRlL3N5bmMxMDVfdGVzdC5nbwlUZXN0VGhlTGljZW5jZUZpbGVzQXJlU3luY2VkQmVmb3JlVGhleUFyZVJlbmFtZWQJZWMwYWIwMTJiM2JiY2QwZmM0NDdlOWQ0NTNiZTcyYzRmYzc3M2I3MTM4ZWFjMDNlMTc4ZGJhMDJlMjgxNTZkMA
  ```
  --- last 7 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state.test]
  internal/state/sync105_test.go:19:26: undefined: syncFn
  internal/state/sync105_test.go:20:21: undefined: syncFn
  internal/state/sync105_test.go:21:2: undefined: syncFn
  internal/state/sync105_test.go:25:12: undefined: WriteSynced
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state [build failed]
  FAIL
  ```
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · ms:334
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · ms:278
- 2026-09-30 · human-observed · S3 observed 2026-09-30 on the owner's Mac (APFS, 10 cores): a 500-file, 5,000-hunk write, 5 alternating runs each of a build with and without a File.Sync in stageFile (a probe build in scratchpad, not committed, now deleted): medians 0.138 s unsynced and 1.98 s synced, +1335 % against the ≤10 % bar, so staging stays unsynced and T4 states the power-loss window. Confound named: the load average was 12 to 14, above the core count the protocol asks for; the sync cost is disk-bound and a quieter machine shrinks only the CPU-bound baseline, so it can only widen the ratio; observed
- 2026-09-30 · 1e77123* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · ms:0 · test-lock-sha256:5f6005f2682965f80d86e77aac22f73f7718b4f8d5402f0a7e9f11eccbeebde1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3N0YXRlL3N5bmMxMDVfdGVzdC5nbwlUZXN0VGhlTGljZW5jZUZpbGVzQXJlU3luY2VkQmVmb3JlVGhleUFyZVJlbmFtZWQJYzhhYWFlN2UxYzVmNGVjZTg5MzAzOGNkNTYzNTIzNDUwYzAzZGU1NmU0MzQzMDg4NTgwNDYzMTJmYzcyMzA1Mw · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: the Codex review of the record found migration wrote the legacy ledger unsynced, so TestTheLicenceFilesAreSyncedBeforeTheyAreRenamed also checks that Migrate writes through WriteSynced; every earlier assertion kept; approved
- 2026-09-30 · 1e77123* · exit 0 · `set -o pipefail …` · acceptance-sha256:35e88215f6ad5744c7914394661b235dff58413a748530e40565552ffb2b0a55 · ms:268
