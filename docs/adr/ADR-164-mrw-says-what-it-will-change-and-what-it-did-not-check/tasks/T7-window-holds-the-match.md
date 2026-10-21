# Task ADR-164-T7: a --max-cols window holds the match

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the window placement in `internal/read/maxcols.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a match that fits is whole`, `a longer match starts the window`, `no match starts the line`, `the binary prints it`

## Goal

Decision 7: `window` centres on the match itself, so a match no longer than N is never cut.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/maxcols.go` | edit | `window` takes the match's end too and centres on its middle; a match longer than N starts the window |
| `internal/read/maxcols164_test.go` | add | the test |
| `internal/read/maxcols137_test.go` | read only | `TestTheWindowCentresOnTheMatch` already asserts a mid-line, front, back and no-pattern case; it must stay green |
| `scripts/contract.sh` | edit | §279 |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestAMatchThatFitsIsWholeInItsWindow (`window("Alpha NEEDLE x", 8, NEEDLE)` holds `NEEDLE` whole; a 30-character match with N 20 starts the window at the match; with N equal to the match's length the window is exactly the match; an empty match does not panic). [proof: acceptance]
2. [S2] Centre on the match. [proof: mutation]
3. [S3] Contract §279: `mrw read --grep NEEDLE --max-cols 8 f` prints `NEEDLE` whole; the pair: the line is still not licensed (a plan to it is refused). [proof: human: the executor runs ./scripts/contract.sh once and reads the §279 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAMatchThatFitsIsWholeInItsWindow$|TestTheWindowCentresOnTheMatch$|TestAnOverlongLineIsCutToAWindowAndLicensesNothing$|TestTheWindowCentresOnTheEndOfAStartEndRange$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAMatchThatFitsIsWholeInItsWindow \(' "$out" \
  && grep -qE '^--- PASS: TestTheWindowCentresOnTheMatch \(' "$out" \
  && grep -qE '^--- PASS: TestAnOverlongLineIsCutToAWindowAndLicensesNothing \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep '^# 279\. ' scripts/contract.sh > /tmp/adr164-t7-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWindowCentresOnTheMatch` | `internal/read/maxcols137_test.go` | ADR-137's window cases unchanged | none | — |
| `TestAnOverlongLineIsCutToAWindowAndLicensesNothing` | `internal/read/maxcols137_test.go` | a cut line still licenses nothing | none | — |
| `TestAMatchThatFitsIsWholeInItsWindow` | `internal/read/maxcols164_test.go` | a match that fits is whole; a longer one starts the window; an empty one does not panic | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `window`'s placement |
| 2 — something selects it | every `mrw read --max-cols N` with a pattern; contract §279 drives the binary |
| 3 — the caller can discover it | the `[cols A-B of C]` marker |
| 4 — it is used | nothing measures this; met in this session (2026-10-20) |

## Mutation Log
(empty until execute)
- 2026-10-20 · 497e9ffc* · mutant killed · exit 1 · `internal/read/maxcols.go` · centring on the match start instead of its middle cuts a match that fits · acceptance-sha256:b4bf02be1270c43a2ce39e4fc20a5cd5589dfb59a24c983fb050252a3c6f25e9

## Verification Log
(empty until execute)
- 2026-10-20 · 497e9ffc* · exit 1 · `set -o pipefail …` · acceptance-sha256:b4bf02be1270c43a2ce39e4fc20a5cd5589dfb59a24c983fb050252a3c6f25e9 · ms:1263 · test-lock-sha256:9c8bd7b49e2b92ac6e022687a5e11ed134603428292178eba01b3a3ee3f8905d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9tYXhjb2xzMTM3X3Rlc3QuZ28JVGVzdEFuT3ZlcmxvbmdMaW5lSXNDdXRUb0FXaW5kb3dBbmRMaWNlbnNlc05vdGhpbmcJZTljYTU1NTY5ZjZlZjZlOGQ2MzI3ZDUyMGEzOGJlMmQ2ODBlZWJlMzFhMTc5NGNiNzhkMzg5N2FhNmM5OWEzOQpib2R5CWludGVybmFsL3JlYWQvbWF4Y29sczEzN190ZXN0LmdvCVRlc3RUaGVXaW5kb3dDZW50cmVzT25UaGVNYXRjaAk4MGIwNmQ3ZDZiYmFiMmY5ZjNmNDJhMGJiNTA0ZGUxYzJjZTNlNWVjYzE1OTRhMTYzODVmNmNjZWRmMTAxZDVi
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  === RUN   TestTheWindowCentresOnTheMatch
  --- PASS: TestTheWindowCentresOnTheMatch (0.00s)
  === RUN   TestAMatchThatFitsIsWholeInItsWindow
      maxcols164_test.go:17: a 6-character match in an 8-column window must be whole: "…pha NEED…  [cols 3-10 of 14]"
      maxcols164_test.go:21: a 30-character match in a 20-column window must start the window: "…aaaaaaaaaaMMMMMMMMMM…  [cols 41-60 of 130]"
      maxcols164_test.go:24: a window as wide as the match must be exactly the match: "…aaaNEE…  [cols 48-53 of 106]"
  --- FAIL: TestAMatchThatFitsIsWholeInItsWindow (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.256s
  FAIL
  ```
- 2026-10-20 · 497e9ffc* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4bf02be1270c43a2ce39e4fc20a5cd5589dfb59a24c983fb050252a3c6f25e9 · ms:3593
- 2026-10-20 · human-observed · Claude observed: ./scripts/contract.sh exit 0 at the T7+T8 tree (2026-10-20), §279 all 4 rows PASS: --max-cols 8 serves NEEDLE whole in its window, and a plan to the cut line is still refused exit 1 with the file unchanged · acceptance-sha256:b4bf02be1270c43a2ce39e4fc20a5cd5589dfb59a24c983fb050252a3c6f25e9
- 2026-10-20 · human-observed · Claude observed, restated 2026-10-20 so the note reads as the verdict it is: ./scripts/contract.sh exit 0 at the T7+T8 tree, §279 all 4 rows PASS: --max-cols 8 serves NEEDLE whole in its window, and a plan to the cut line exits 1 with the file unchanged · acceptance-sha256:b4bf02be1270c43a2ce39e4fc20a5cd5589dfb59a24c983fb050252a3c6f25e9

## Invariants

- The window is N characters wide whenever the line is longer than N; a cut line licenses nothing (ADR-137); `mrw_read`'s `max_cols` (ADR-154) gets the same placement.

## Risks

- None beyond the record's.

## Out of Scope

- Windows round every match on a line (permanent: boundary: ADR-137 cuts round the first match)

## Stop Condition

Stop and bring it to Zy if `TestTheWindowCentresOnTheEndOfAStartEndRange` (a `/a/,/b/` range centres on its end) cannot hold beside a centred match.
