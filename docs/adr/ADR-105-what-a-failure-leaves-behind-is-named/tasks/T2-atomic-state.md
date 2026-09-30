# Task ADR-105-T2: state files are replaced whole

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `state.Write`; the 8 state writes through it; contract §201
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a state file is replaced, never rewritten in place`, `a refused rename falls back`, `no state write bypasses it`, `a contract row drives the binary`

## Goal

`state.Write(name, data, perm)` writes a temp file beside `name` and renames it over; a refused rename removes the
temp and writes in place. Every state write in `internal/` goes through it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/write.go` | add | `Write` and its `renameFn` seam |
| `internal/state/write105_test.go` | add | the tests below |
| `internal/state/state.go` | edit | the root marker (`:76`) and the migration copy (`:135`) |
| `internal/seen/seen.go` | edit | `save` (`:440`) |
| `internal/authoring/authoring.go` | edit | `:202`, `:288`, `:405` |
| `internal/iter/iter.go` | edit | `:135` |
| `internal/mcp/ack.go` | edit | `:450` |
| `scripts/contract.sh` | edit | §201 |

## Ordered Steps

1. [S1] Write the failing tests `TestAStateFileIsReplacedWholeNeverRewrittenInPlace` and `TestNoStateWriteBypassesTheAtomicWriter`; confirm RED. [proof: mutation]
2. [S2] `state.Write`: `CreateTemp(dir, "."+base+".tmp-*")`, write, close, chmod, rename; on a refused rename, remove the temp and `os.WriteFile`; a read-only existing file is written in place, so it stays refused. [proof: mutation] Mutants: the rename replaced by an in-place write; the fallback leaves the temp; a read-only file is replaced.
3. [S3] Route the 8 sites through it. [proof: mutation] Mutant: `seen.save` back to `os.WriteFile`.
4. [S4] Contract §201, driving `$MRW`: a write replaces the ledger (its inode changes) and leaves no temp beside it; the pair, the next write, is licensed by the replaced ledger. [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §201's rows printed]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/state/ -count=1 -timeout 300s -run 'TestAStateFileIsReplacedWholeNeverRewrittenInPlace|TestNoStateWriteBypassesTheAtomicWriter' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAStateFileIsReplacedWholeNeverRewrittenInPlace \(' "$out" \
  && grep -qE '^--- PASS: TestNoStateWriteBypassesTheAtomicWriter \(' "$out" \
  && grep -q '^# 201\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/lines internal/rooted internal/read internal/check internal/links \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/lines internal/rooted internal/read internal/check internal/links)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAStateFileIsReplacedWholeNeverRewrittenInPlace` | `internal/state/write105_test.go` | after `Write` over an existing file the name holds the new bytes and is a different file (`os.SameFile` false), with the given mode, and no temp remains; with `renameFn` refusing, the name holds the new bytes and no temp remains; a `0444` file is refused and unchanged (not as uid 0) | — | S1, S2 |
| `TestNoStateWriteBypassesTheAtomicWriter` | `internal/state/write105_test.go` | no non-test source in `internal/seen`, `internal/authoring`, `internal/iter`, `internal/mcp`, `internal/state` calls `os.WriteFile`, except `state.Write`'s own fallback | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/state/write.go` |
| 2 — something selects it | every state write in `internal/` (the class test) |
| 3 — the caller can discover it | the README matrix (T4) |
| 4 — it is used | every mrw run saves the ledger |

## Mutation Log
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/state/write.go` · the rename replaced by an in-place write · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · covers:a state file is replaced, never rewritten in place
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/state/write.go` · the fallback leaves its temp · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · covers:a refused rename falls back
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/seen/seen.go` · seen.save writes the ledger in place again · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · covers:no state write bypasses it
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/state/write.go` · a read-only state file is replaced · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · covers:a state file is replaced, never rewritten in place

## Invariants

- State file formats are unchanged; an older binary reads them.
- Exit codes keep their meanings.

## Risks

- A rename over a file another process holds open fails on Windows: the fallback writes in place, as before.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- Syncing before the rename (deferred: T3 in this record, `docs/adr/ADR-105-what-a-failure-leaves-behind-is-named/tasks/T3-fsync-measured.md`)

## Verification Log
- 2026-09-30 · 1bd8690* · exit 1 · `set -o pipefail …` · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · ms:141 · test-lock-sha256:66fb65bc50ba2aab21c8af90e809eecd8da96effbee2079b9a356d88a4bb5866 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3N0YXRlL3dyaXRlMTA1X3Rlc3QuZ28JVGVzdEFTdGF0ZUZpbGVJc1JlcGxhY2VkV2hvbGVOZXZlclJld3JpdHRlbkluUGxhY2UJNGNhNTVjNWVjZTIxOGE5MDM1MGMwNjU4MTBiMTY3YTRiNjg2OTVjZmRlNjhkMTlkMzY2OWVmMTI0MDc3NjI3ZQpib2R5CWludGVybmFsL3N0YXRlL3dyaXRlMTA1X3Rlc3QuZ28JVGVzdE5vU3RhdGVXcml0ZUJ5cGFzc2VzVGhlQXRvbWljV3JpdGVyCTA2YzYxYjg5YjkzMWQ0MDUxYmMyM2U0NjY5N2FlM2U1N2M1YjA4M2VmODA5NDU0NzE0N2IyNDRmNjA1NGZkZTc
  ```
  --- last 8 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state.test]
  internal/state/write105_test.go:45:12: undefined: Write
  internal/state/write105_test.go:65:10: undefined: renameFn
  internal/state/write105_test.go:66:21: undefined: renameFn
  internal/state/write105_test.go:67:2: undefined: renameFn
  internal/state/write105_test.go:68:12: undefined: Write
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state [build failed]
  FAIL
  ```
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · ms:357
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · ms:310
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · ms:362
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · ms:283
- 2026-09-30 · 1bd8690* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:acb680f2892087e1fe13a8c98f1cda366dbf853daeb77ae674d91bdcd42173db · ms:0 · test-lock-sha256:67c7b3911faaed1476378dac04698451db3de70edf94a7dc8163990f71079ac0 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3N0YXRlL3dyaXRlMTA1X3Rlc3QuZ28JVGVzdEFTdGF0ZUZpbGVJc1JlcGxhY2VkV2hvbGVOZXZlclJld3JpdHRlbkluUGxhY2UJNGI5NDBkZmQ1NzE1OGU5YTkxZWIyMWE0ZDQxYjgzZTU4ZjJkNzdjMDY1YjY5ZWQzN2YwYzM5MGRhZDAwZWQ3NApib2R5CWludGVybmFsL3N0YXRlL3dyaXRlMTA1X3Rlc3QuZ28JVGVzdE5vU3RhdGVXcml0ZUJ5cGFzc2VzVGhlQXRvbWljV3JpdGVyCTA2YzYxYjg5YjkzMWQ0MDUxYmMyM2U0NjY5N2FlM2U1N2M1YjA4M2VmODA5NDU0NzE0N2IyNDRmNjA1NGZkZTc · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: after the red run TestAStateFileIsReplacedWholeNeverRewrittenInPlace gained its read-only case, because the contract run showed four ledger-failure fixtures relied on a 0444 ledger staying refused; every earlier assertion kept; approved
- 2026-09-30 · human-observed · S4 observed 2026-09-30: ./scripts/contract.sh run unpiped in the ADR-105 worktree, exit 0 (contract holds), with §201 printed: a write through the built binary replaced the ledger by rename (inode changed), left no temp beside it, and the next write was licensed
