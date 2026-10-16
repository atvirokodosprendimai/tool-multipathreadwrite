# Task ADR-157-T2: the stale notice claims no discard, help is silent, seen names the ledger

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `seen.Header` and the reworded `seen.StaleNotice` in `internal/seen/seen.go`; the ledger line of `mrw seen`
**Consumes:** the root `Before` as T1 leaves it (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the stale notice claims no discard`, `help does not consult the ledger`, `seen names the ledger's format and writer`, `ADR-136's test still holds`, `contract §261 drives the built binary`, `the old sentence is gone from the source`

## Goal

Decisions 4 and 5 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `StaleNotice`'s words; `Header(root)` returns the format, its number and the writer from the first line |
| `cmd/mrw/main.go` | edit | the root `Before` returns before the ledger checks for `help`, `h` and an empty verb; `seenCmd` prints the ledger line after the state-directory count |
| `cmd/mrw/notice157_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §261 |

## Ordered Steps

1. [S1] Write the failing tests and see them red. The class of tests to update is found by `mrw read --grep 'StaleNotice|has been discarded|older mrw' cmd/ internal/` and every test it names is listed in the commit.
   - `TestOnlyCommandsThatUseTheLedgerTellItIsStale`: with a `#mrw-seen v3` ledger, `mrw help`, `mrw h` and a bare `mrw` print nothing about the ledger and leave its bytes unchanged. A refused write (nothing recorded) prints the new sentence and leaves the ledger as it was, which "the next recorded read or write" allows. `mrw read` prints it once and replaces the ledger. The new sentence holds `it is not used, and the next recorded read or write replaces it` and not `has been discarded`, and stays within ADR-136's 40 words.
   - `TestSeenNamesTheLedgersFormatAndWriter`: `mrw seen` prints:
     - `# ledger format v5, written by <this mrw>` after a read;
     - `# ledger format v4, written by unknown` for a v4 ledger, and the same for a writerless v5;
     - `# ledger written by a newer mrw (v9.9.9), format v6: left untouched, licenses nothing` for a v6;
     - `# ledger in an older format: not used; mrw replaces it the next time it records a read or a write` for a v3;
     - no ledger line when no ledger exists.

     In every case the state directory is still the first line.
2. [S2] `StaleNotice` reworded. [proof: mutation]
3. [S3] `Before` skips the ledger for help, `h` and a bare `mrw`. [proof: mutation]
4. [S4] `seen.Header` and the `mrw seen` line. [proof: mutation]
5. [S5] Confirm no locked test changes: `cmd/mrw/quiet136_test.go` and `internal/seen/ledgerpath_test.go` pin only "written by an older mrw" and "line endings", which the new sentence keeps (read 2026-10-14). If a test the S1 search names pins more, change it here and re-lock its task with `adr-verify <its task> --relock --replace-hashes` after confirming the diff is the sentence alone. [proof: human: the executor reads the S1 search's output and names each test in the commit]
6. [S6] Contract §261: `mrw help` under a v3 ledger leaves stderr and the ledger untouched; `mrw seen` names the writer; the pair, `mrw read` under the same v3 ledger, prints the new sentence. [proof: human: the executor runs ./scripts/contract.sh once and reads the §261 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ ./internal/seen/ -count=1 -timeout 600s -run 'TestOnlyCommandsThatUseTheLedgerTellItIsStale|TestSeenNamesTheLedgersFormatAndWriter|TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestOnlyCommandsThatUseTheLedgerTellItIsStale \(' "$out" \
  && grep -qE '^--- PASS: TestSeenNamesTheLedgersFormatAndWriter \(' "$out" \
  && grep -qE '^--- PASS: TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions \(' "$out" \
  && grep -q '^# 261\. ' scripts/contract.sh \
  && ! grep -q 'it has been discarded. Read the files' internal/seen/seen.go \
  && fmtout=$(gofmt -l internal cmd) && [ -z "$fmtout" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestOnlyCommandsThatUseTheLedgerTellItIsStale` | `cmd/mrw/notice157_test.go` | help, `h` and a bare `mrw` are silent and leave the ledger; a read tells the new sentence and replaces it | none | S1, S2, S3 |
| `TestSeenNamesTheLedgersFormatAndWriter` | `cmd/mrw/notice157_test.go` | the ledger line for current, writerless, newer, older and absent ledgers; the state directory stays first | none | S1, S4 |
| `TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions` | `cmd/mrw/quiet136_test.go` | ADR-136's test, unchanged: the new sentence keeps both causes | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the new sentence; `seen.Header`; the ledger line |
| 2 — something selects it | the root `Before` on every command; `seenCmd`; contract §261 drives the built binary |
| 3 — the caller can discover it | stderr; `mrw seen`'s output |
| 4 — it is used | nothing measures this; built because all six Windows sessions read the ledger with their shell to check its writer, and one found help announcing a discard (2026-10-14) |

## Mutation Log
(empty until execute)
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/seen/seen.go` · the stale notice claims a discard again · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `cmd/mrw/main.go` · help and a bare mrw consult the ledger again and announce a discard · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `cmd/mrw/main.go` · mrw seen stops naming the ledger · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/seen/seen.go` · a header that names no writer is shown with an empty name · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba

## Verification Log
(empty until execute)
- 2026-10-14 · be76e27f* · exit 1 · `set -o pipefail …` · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba · ms:1095 · test-lock-sha256:74ff935127329e9baf038794857ec004e7083c9235c3466aecf9505d213bb787 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9ub3RpY2UxNTdfdGVzdC5nbwlUZXN0T25seUNvbW1hbmRzVGhhdFVzZVRoZUxlZGdlclRlbGxJdElzU3RhbGUJMmFiYmE5MWVjZDQxZmM0ZmE0OGVmMTE4MzVlYTNmYjc3NDdhYWY0MDFjZDg2ZTc0NDYzZmFhODIxMjQwZDc4Nwpib2R5CWNtZC9tcncvbm90aWNlMTU3X3Rlc3QuZ28JVGVzdFNlZW5OYW1lc1RoZUxlZGdlcnNGb3JtYXRBbmRXcml0ZXIJOTE2ZGI4NDk1ZTNjYzFkMGM4OTFlZTY1MjYwN2MwMmNjYzEyNzcwZTUwYmZiMzAxZGViZDkwMjhmMGY5MzNhYwpib2R5CWNtZC9tcncvcXVpZXQxMzZfdGVzdC5nbwlUZXN0VGhlU3RhbGVOb3RpY2VJc09uZVNlbnRlbmNlQW5kU2lsZW50T25WZXJzaW9uU3RhdHNJbnN0cnVjdGlvbnMJMGE3Y2FiYTcxYjdjNGEyYTMxMDVjZWUyMDdmMWFhOTNiN2I0YWIxYmI5NWZmZWRmN2Q4NGRjYjc0NmRkYzJkMA
  ```
  --- last 10 line(s) of stdout (of 40 after folding 40 raw)
          0 file(s) seen
  --- FAIL: TestSeenNamesTheLedgersFormatAndWriter (0.01s)
  === RUN   TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions
  --- PASS: TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions (0.03s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.289s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	0.178s [no tests to run]
  FAIL
  ```
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba · ms:1340
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba · ms:1380
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba · ms:1365
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba · ms:1348
- 2026-10-14 · human-observed · observed 2026-10-14: ./scripts/contract.sh exit 0, last line contract holds; the 261 rows PASS; the S1 search names quiet136_test.go and ledgerpath_test.go, which pin only phrases the new sentence keeps, so no test changed and nothing was re-locked
- 2026-10-16 · 8c4b3f31* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ec3adb9fa45ffd409b972490cbd15324d28cc9941b3f3bb7a953754b56df83ba · ms:0 · test-lock-sha256:f168795278bff28fe1f7b13fc674062b782f198885712f2fd111ee99295ca770 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9ub3RpY2UxNTdfdGVzdC5nbwlUZXN0T25seUNvbW1hbmRzVGhhdFVzZVRoZUxlZGdlclRlbGxJdElzU3RhbGUJYjY0Nzg5OWI0NTUzNGViNDljMTAyY2Q5NjY0MmYwNDBiYTkxNmU4MGVkYjlkOTUyY2ExNmMyN2Q0ZjRlZmYzYQpib2R5CWNtZC9tcncvbm90aWNlMTU3X3Rlc3QuZ28JVGVzdFNlZW5OYW1lc1RoZUxlZGdlcnNGb3JtYXRBbmRXcml0ZXIJYjdhYjI4ZTljNjAyYWMxMGU1MWU2NDk5MjZlN2Q0NGQyZDA4ODFiZmE3MTU1OGIyMzJkZmUzZmNlYjhkM2Y1NQpib2R5CWNtZC9tcncvcXVpZXQxMzZfdGVzdC5nbwlUZXN0VGhlU3RhbGVOb3RpY2VJc09uZVNlbnRlbmNlQW5kU2lsZW50T25WZXJzaW9uU3RhdHNJbnN0cnVjdGlvbnMJMGE3Y2FiYTcxYjdjNGEyYTMxMDVjZWUyMDdmMWFhOTNiN2I0YWIxYmI5NWZmZWRmN2Q4NGRjYjc0NmRkYzJkMA · test-lock-kind:replace

## Invariants

- `mrw seen`'s first line is the state directory (contract §71).
- read, write, check, iter, seen and mcp still tell a stale ledger (ADR-136); version, instructions and stats still do not read it.
- A v4 ledger stays current, so a writerless header says `unknown`, never older.

## Risks

- None beyond the record's.

## Out of Scope

- The damage notice's own discard sentence (permanent: boundary: the record's Out of Scope)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
