# Task ADR-023-T2: file the Desktop envelope probe recipe

**Depends-on:** none
**Covers:** F-6, F-16, UC3-S1, UC3-S2
**Estimated scope:** S
**Owner:** unassigned
**Produces:** ADR-023 Verification Log recipe
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `Verification Log names the Desktop envelope probe`, `BACKLOG other-hosts stays not blocking`

## Goal

ADR-023 carries a Verification Log recipe for one Desktop `mrw_read` of a two-line fixture. An unrun probe is not a miss rate: BACKLOG stays not blocking.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/adr/ADR-023-a-reads-answer-is-the-served-text.md` | edit | `## Verification Log` recipe. |
| `internal/adversarial/desktop_probe_test.go` | edit | Recipe exists; BACKLOG not-blocking. |

## Ordered Steps

1. [S1] Confirm the failing tests for `Covers:` IDs exist. [proof: mutation]
2. [S2] File the recipe. S1 GREEN. [proof: mutation]
3. [S3] Leave BACKLOG "ADR-023: other hosts" open and not blocking. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/adversarial/ -count=1 -v \
  -run 'TestTheDesktopEnvelopeProbeRecipeIsFiled|TestAnUnrunDesktopProbeIsNotAMissRate' 2>&1 | tee /tmp/adr023-t2.out \
  && grep -q '^--- PASS: TestTheDesktopEnvelopeProbeRecipeIsFiled' /tmp/adr023-t2.out \
  && grep -q '^--- PASS: TestAnUnrunDesktopProbeIsNotAMissRate' /tmp/adr023-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr023-t2.out \
  && grep -q '## Verification Log' docs/adr/ADR-023-a-reads-answer-is-the-served-text.md \
  && grep -q 'Desktop envelope probe' docs/adr/ADR-023-a-reads-answer-is-the-served-text.md \
  && grep -q 'two-line' docs/adr/ADR-023-a-reads-answer-is-the-served-text.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheDesktopEnvelopeProbeRecipeIsFiled` | `internal/adversarial/desktop_probe_test.go` | recipe is on ADR-023 | F-6, UC3-S1 | S1, S2 |
| `TestAnUnrunDesktopProbeIsNotAMissRate` | `internal/adversarial/desktop_probe_test.go` | BACKLOG stays not blocking | F-16, UC3-S2 | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | deleting the Verification Log heading leaves S1 red |
| 3 — the caller can discover it | ADR-023 Verification Log |
| 4 — it is used | human-observed when a Desktop session is at hand |

## Mutation Log
- 2026-09-15 · 9ff8e35* · mutant killed · exit 1 · `docs/adr/ADR-023-a-reads-answer-is-the-served-text.md` · the Desktop probe recipe must live under Verification Log or it has no filed home · acceptance-sha256:1eb5a3f3aa491687d58d15aace5e910100c5222b89f1e3acebd555663b4b650a · covers:Verification Log names the Desktop envelope probe

## Invariants

- No engine change.
- Reading 12 stays VOID.
- Unrun is not a miss rate.

## Risks

- none

## Stop Condition

Stop if the probe finds a host defect that needs an engine ADR — that is a new record.

## Out of Scope

- Running the probe in this task (human-observed when a session is at hand).

## Verification Log
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:1eb5a3f3aa491687d58d15aace5e910100c5222b89f1e3acebd555663b4b650a · ms:768
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:1eb5a3f3aa491687d58d15aace5e910100c5222b89f1e3acebd555663b4b650a · ms:374
- 2026-09-15 · 9ff8e35* · exit 1 · `set -o pipefail …` · acceptance-sha256:1eb5a3f3aa491687d58d15aace5e910100c5222b89f1e3acebd555663b4b650a · ms:878 · test-lock-sha256:2a094004eee3f815c107308a82b9257a31d9d7dce4a35c255061978f3f347d46 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FkdmVyc2FyaWFsL2Rlc2t0b3BfcHJvYmVfdGVzdC5nbwlUZXN0QW5VbnJ1bkRlc2t0b3BQcm9iZUlzTm90QU1pc3NSYXRlCTg4MDFiMzI3YWI5MWIyNjM2MzNiYTBmMDk2MTkzNjY0YWFhNDBmOTdkOTgxNTEwZjU2ZTRhMzVlMzFmNDU1YWIKYm9keQlpbnRlcm5hbC9hZHZlcnNhcmlhbC9kZXNrdG9wX3Byb2JlX3Rlc3QuZ28JVGVzdFRoZURlc2t0b3BFbnZlbG9wZVByb2JlUmVjaXBlSXNGaWxlZAk2MmFmNGYwMjRhOTZjYzMxODU1ZGRhZjAwMmM1OTFkMTJjMzM4ZGIyMWRjMWNhZjVjOTlkNmNjZjgxNGI5ZWIy
  ```
  --- last 9 line(s) of stdout
  === RUN   TestTheDesktopEnvelopeProbeRecipeIsFiled
      desktop_probe_test.go:20: ADR-023 has no Verification Log section for the Desktop probe
      desktop_probe_test.go:23: ADR-023 Verification Log does not name the Desktop envelope probe
  --- FAIL: TestTheDesktopEnvelopeProbeRecipeIsFiled (0.00s)
  === RUN   TestAnUnrunDesktopProbeIsNotAMissRate
  --- PASS: TestAnUnrunDesktopProbeIsNotAMissRate (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.389s
  FAIL
  ```
