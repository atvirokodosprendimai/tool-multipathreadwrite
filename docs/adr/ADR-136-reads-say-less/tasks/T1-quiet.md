# Task ADR-136-T1: the neighbour note once, the stale notice short, silent on commands without a ledger

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** none new — three existing behaviours change
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a read prints the neighbour note once, the stale notice is one sentence, and version, instructions and stats print none`

## Goal

Decisions 1–3 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | the note is printed at the first qualifying range of a call |
| `internal/seen/seen.go` | edit | `StaleNotice` is one sentence |
| `cmd/mrw/main.go` | edit | the root `Before` skips `version`, `instructions`, `stats` |
| `internal/read/quiet136_test.go`, `cmd/mrw/quiet136_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §237 |
| `AGENTS.md` | edit | the read paragraph |

## Ordered Steps

1. [S1] Write `TestTheNeighbourNoteIsPrintedOncePerRead` (a read of three multi-line ranges prints the note once; a single range still prints it) and `TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions` (a stale ledger: `read` and `write` print the notice, which is one line naming both causes; `version`, `instructions` and `stats` print none). Confirm RED.
2. [S2] The local in `Run`, the shorter constant, the condition in `Before`. Mutants: the note printed per range again; the skip list emptied (the notice on `version`). [proof: mutation]
3. [S3] Contract §237 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestTheNeighbourNoteIsPrintedOncePerRead' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheNeighbourNoteIsPrintedOncePerRead \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions \(' "$out" \
  && go test ./internal/read/ ./internal/seen/ ./internal/adversarial/ -count=1 -timeout 900s \
  && grep -q '^# 237\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheNeighbourNoteIsPrintedOncePerRead` | `internal/read/quiet136_test.go` | several qualifying ranges print the note once; one range still prints it | none | S1, S2 |
| `TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions` | `cmd/mrw/quiet136_test.go` | the notice is one line naming both causes and appears on `read`/`write` only | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the local, the constant, the condition |
| 2 — something selects it | `Run` on every read; the root `Before` on every command |
| 3 — the caller can discover it | the output itself; AGENTS.md |
| 4 — it is used | the 2026-10-09 survey (14 sessions); no telemetry (ADR-009) |

## Invariants

- A write is refused exactly as before; every exit code and receipt is unchanged.
- A single-range read prints the note as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a caller's test or script needs the note at every range: the record would then become "keep it per range".

## Out of Scope

- A look mode, a newer-ledger message (deferred: docs/adr/BACKLOG.md)

## Mutation Log
- 2026-10-10 · 675dfad · mutant inconclusive · exit 1 · `internal/read/read.go` · S2: the note is printed at every qualifying range again · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · covers:a read prints the neighbour note once, the stale notice is one sentence, and version, instructions and stats print none
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-10 · 675dfad* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the skip list is emptied — version, instructions and stats print the stale notice · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · covers:a read prints the neighbour note once, the stale notice is one sentence, and version, instructions and stats print none
- 2026-10-10 · 675dfad* · mutant killed · exit 1 · `internal/read/read.go` · S2: the note is printed at every qualifying range again · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · covers:a read prints the neighbour note once, the stale notice is one sentence, and version, instructions and stats print none
- 2026-10-10 · 675dfad* · mutant killed · exit 1 · `internal/seen/seen.go` · S2: the notice carries the version history again · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · covers:a read prints the neighbour note once, the stale notice is one sentence, and version, instructions and stats print none

## Verification Log
- 2026-10-10 · 675dfad · exit 0 · `set -o pipefail …` · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · ms:25112
- 2026-10-10 · 675dfad* · exit 0 · `set -o pipefail …` · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · ms:23762
- 2026-10-10 · 675dfad* · exit 0 · `set -o pipefail …` · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · ms:23600
- 2026-10-10 · 675dfad* · exit 0 · `set -o pipefail …` · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · ms:24871
- 2026-10-10 · 675dfad* · exit 1 · `set -o pipefail …` · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · ms:370 · test-lock-sha256:03b34df6cf35e031042599f28f6ff2201e2f71ba0ce1bdfc3691bf3575db74af · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9xdWlldDEzNl90ZXN0LmdvCVRlc3RUaGVTdGFsZU5vdGljZUlzT25lU2VudGVuY2VBbmRTaWxlbnRPblZlcnNpb25TdGF0c0luc3RydWN0aW9ucwkwYTdjYWJhNzFiN2M0YTJhMzEwNWNlZTIwN2YxYWE5M2I3YjRhYjFiYjk1ZmZlZGY3ZDg0ZGNiNzQ2ZGRjMmQwCmJvZHkJaW50ZXJuYWwvcmVhZC9xdWlldDEzNl90ZXN0LmdvCVRlc3RUaGVOZWlnaGJvdXJOb3RlSXNQcmludGVkT25jZVBlclJlYWQJOTgxZTQwYWRmNzY5NmZhMjY3OGMwNmQyN2RmOTU5NTBlNjIxZmU0ZDRmYTE5YmJlNTQyYjk0ZjdlMzM0ZjU2ZQ
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
              5| }
          ==> a.go  9L  61B  sha 1c739edc
          @@ 7-8
          -- note: a multi-line replace of 7-8 needs a served line after 8
              7| func Bar() {
              8| 	return
  --- FAIL: TestTheNeighbourNoteIsPrintedOncePerRead (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.179s
  FAIL
  ```
- 2026-10-10 · 675dfad* · exit 0 · `set -o pipefail …` · acceptance-sha256:91efcee894eebfa8564c8f73e3484451be487702ecafe4cdf0fb218b7ef44533 · ms:22679
