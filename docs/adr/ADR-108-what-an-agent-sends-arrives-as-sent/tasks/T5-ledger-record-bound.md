# Task ADR-108-T5: the ledger reads back every record it saves

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the ledger reads back every record it saves
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the ledger reads back every record it saves`

## Goal

The ledger has a record bound, `maxRecordBytes` (16 MiB): `save` leaves out a longer record, and `Load` reads lines up to it; a longer line discards the ledger.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `maxRecordBytes`; `save` and `Load` |
| `internal/seen/record108_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test(s) `TestALedgerRecordTheWriterSavesTheLoaderReads`; confirm RED. [proof: mutation]
2. [S2] The bound on both sides. Mutants: the loader's buffer back to the default; save writes a record over the bound. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/seen/ -count=1 -timeout 300s -run 'TestALedgerRecordTheWriterSavesTheLoaderReads' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestALedgerRecordTheWriterSavesTheLoaderReads \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestALedgerRecordTheWriterSavesTheLoaderReads` | `internal/seen/record108_test.go` | a record of 10,000 disjoint spans (over 64 KiB) saves, loads, and a further `Record` succeeds; with `maxRecordBytes` set small, a longer record is left out on save while other records stay, and a ledger file holding a line over the bound loads empty with no error | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write or acknowledgement that reaches it |
| 3 — the caller can discover it | the refusal or the receipt names it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review |

## Mutation Log
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/seen/seen.go` · the loader buffer back to 64 KiB: a record over a smaller bound still loads · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67
- 2026-10-01 · 98feab5* · mutant survived · exit 0 · `internal/seen/seen.go` · save writes a record over the bound, so the loader drops the whole ledger · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/seen/seen.go` · save writes a record over the bound, so the next Load discards the whole ledger and b.txt with it · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/seen/seen.go` · the loader buffer back to 64 KiB: a record over a smaller bound still loads · acceptance-sha256:0fc376e40f207a8fce707f7e0dc2dc2ce18e11ac103b755efea078a927044a37
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/seen/seen.go` · save writes a record over the bound, so the next Load discards the whole ledger and b.txt with it · acceptance-sha256:0fc376e40f207a8fce707f7e0dc2dc2ce18e11ac103b755efea078a927044a37

## Invariants

- Exit codes keep their meanings; inputs the defect did not touch behave as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 98feab5* · exit 1 · `set -o pipefail …` · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67 · ms:341 · test-lock-sha256:2fafb463e7fbceaee52971ede018e4e49a47291f3f34689189104d499cfaa067 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3NlZW4vcmVjb3JkMTA4X3Rlc3QuZ28JVGVzdEFMZWRnZXJSZWNvcmRUaGVXcml0ZXJTYXZlc1RoZUxvYWRlclJlYWRzCTRlMzU1OGIzZGVjNzI5MTRhMTg5MzczNmI1NDZmYjRjYjA2YmExMDc3ODYyZjYxNzdkNmY4MmJkYWI2MmY3YTk
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen.test]
  internal/seen/record108_test.go:36:9: undefined: maxRecordBytes
  internal/seen/record108_test.go:37:21: undefined: maxRecordBytes
  internal/seen/record108_test.go:38:2: undefined: maxRecordBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [build failed]
  FAIL
  ```
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67 · ms:360
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67 · ms:372
- 2026-10-01 · 98feab5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67 · ms:0 · test-lock-sha256:d9b40055da3b72de392c15b7269e3ec9e3d0bf0cb97dead8529debf4c491e2b2 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3NlZW4vcmVjb3JkMTA4X3Rlc3QuZ28JVGVzdEFMZWRnZXJSZWNvcmRUaGVXcml0ZXJTYXZlc1RoZUxvYWRlclJlYWRzCWRhODYxZTE2OWRjYTVkMDhmNTVmMGQ0NjgyYWNhYTU3NDUzNmU3NjhlN2FjNjI0Zjg3YTFmMTExODA4Y2Q3ZTQ · test-lock-kind:replace
- 2026-10-01 · human-observed · Zy's session observed the relock: after red the bounded case records b.txt before the long a.txt, so a save that writes the long record loses b.txt; recorded the other way round the save mutant survived
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:d22e5beddcd12870c9eed13faebe47754bd2ae403918e76f9e4ab676f62b5c67 · ms:362
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:0fc376e40f207a8fce707f7e0dc2dc2ce18e11ac103b755efea078a927044a37 · ms:433
  ```
  --- last 4 line(s) of stdout
  === RUN   TestALedgerRecordTheWriterSavesTheLoaderReads
  --- PASS: TestALedgerRecordTheWriterSavesTheLoaderReads (0.03s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	0.200s
  ```
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:0fc376e40f207a8fce707f7e0dc2dc2ce18e11ac103b755efea078a927044a37 · ms:362
  ```
  --- last 4 line(s) of stdout
  === RUN   TestALedgerRecordTheWriterSavesTheLoaderReads
  --- PASS: TestALedgerRecordTheWriterSavesTheLoaderReads (0.02s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	0.098s
  ```
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0fc376e40f207a8fce707f7e0dc2dc2ce18e11ac103b755efea078a927044a37 · ms:379
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0fc376e40f207a8fce707f7e0dc2dc2ce18e11ac103b755efea078a927044a37 · ms:334
