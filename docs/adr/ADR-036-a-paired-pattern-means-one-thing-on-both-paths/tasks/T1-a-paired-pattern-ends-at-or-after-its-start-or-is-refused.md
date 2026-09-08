# Task ADR-036-T1: A paired pattern ends at or after its start, or is refused

**Depends-on:** none
**Covers:** none — no spec
**Owner:** Zy
**Produces:** the at-or-after end, and the refusal of a paired pattern whose end never matches
**Consumes:** none
**Proof map:** v1

**Rests-on:** `an end matching the start line closing the span there`, `a paired pattern with no end being reported rather than served to EOF`, `the report saying which half of the pattern failed`, `a paired pattern that resolves normally still serving every non-nested start`, `a resolved span not suppressing a later missing end`

## Goal

Make `internal/read` resolve a paired pattern the way `internal/apply` does, in
the two respects where they differed for no reason, and leave the one where they
differ on purpose.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `j > i` becomes `j >= i`; a range whose end never matches goes to `missed` instead of defaulting to `total`. |
| `internal/read/read_test.go` | edit | The three tests below. |
| `scripts/contract.sh` | edit | §74, driving both behaviours through the built binary. |
| `README.md` | edit | The paired-pattern paragraph, and withdrawal of the divergence note ADR-035's eighth review round added. |
| `AGENTS.md` | edit | The same rule in the agent-facing copy. |
| `docs/adr/BACKLOG.md` | edit | The divergence entry is closed by this record. |

## Ordered Steps

1. [S1] Write `TestAPairedPatternWithNoEndIsRefusedRatherThanExtendedToEOF` and confirm it is RED: the range is reported unreadable and NO span is served. ⚠ Assert the absence of the span, not just the presence of the report — the defect is that content was served, and a test that only checks the message passes on a build that reports AND serves. [proof: mutation]
2. [S2] Write `TestAPairedPatternEndsAtOrAfterItsStart` and confirm it is RED: an end matching the start line yields a one-line span. [proof: mutation]
3. [S3] Write the control — a paired pattern with a normal later end still serves that span, and a start matching twice still serves BOTH. Without it, "refuse everything" passes S1 and S2, and the every-start behaviour this record deliberately KEEPS could be dropped silently. [proof: mutation]
4. [S4] Assert the report names which half failed, so the caller does not re-read the file to learn whether the start or the end missed. [proof: mutation]
5. [S5] Change the resolver. [proof: mutation]
6. [S6] Add §74 driving `$MRW`: exit 1 and no served content for the missing end, and the one-line span for the same-line end. [proof: mutation]
7. [S7] Update `README.md`, `AGENTS.md` and the BACKLOG entry, and confirm no document still describes the divergence as open. [proof: acceptance]
9. [S9] ⚠ Report a missing end INDEPENDENTLY of whether an earlier start resolved. `found` is range-wide, so the first cut returned `START … END … START … EOF` as one span with no problem and exit 0 — a start the address named, silently dropped, which is the same failure this record removes moved one case along. Reported by review of PR #146. Keep the resolved spans AND report the unresolved start; either alone is a lie by omission. [proof: mutation]
8. [S8] Run every gate, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ -count=1 -v \
  -run 'TestAPairedPatternEndsAtOrAfterItsStart|TestAPairedPatternWithNoEndIsRefusedRatherThanExtendedToEOF|TestAResolvedSpanDoesNotSuppressALaterMissingEnd' 2>&1 | tee /tmp/adr036-t1.out \
  && grep -q '^--- PASS: TestAPairedPatternEndsAtOrAfterItsStart' /tmp/adr036-t1.out \
  && grep -q '^--- PASS: TestAPairedPatternWithNoEndIsRefusedRatherThanExtendedToEOF' /tmp/adr036-t1.out \
  && grep -q '^--- PASS: TestAResolvedSpanDoesNotSuppressALaterMissingEnd' /tmp/adr036-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr036-t1.out \
  && grep -q '^# 74\. ' scripts/contract.sh \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr036-t1c.out \
  && grep -q '^contract holds$' /tmp/adr036-t1c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPairedPatternWithNoEndIsRefusedRatherThanExtendedToEOF` | `internal/read/read_test.go` | The range is reported unreadable, the report names the END as the half that failed, and NO span is served — the absence asserted, because serving-and-reporting would otherwise pass | — | S1, S4, S5 |
| `TestAPairedPatternEndsAtOrAfterItsStart` | `internal/read/read_test.go` | An end matching the start line closes the span there, one line, as a write does; and a normal later end, and a start matching twice, both still behave as before | — | S2, S3, S5 |
| `TestAResolvedSpanDoesNotSuppressALaterMissingEnd` | `internal/read/read_test.go` | `START … END … START … EOF` reports the unresolved start AND still serves the span that resolved — both, because either alone is a lie by omission | — | S9 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests above |
| 2 — something selects it | the resolver runs for every paired-pattern read; the S5 mutation restores `j > i` and `end := total` and both go red |
| 3 — the caller can discover it | the `!! no match for <spec>` line names the range and the failing half, and the run exits 1 |
| 4 — it is used | §74 drives both behaviours through the built binary; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-08 · 33003e3* · mutant killed · exit 1 · `internal/read/read.go` · the end is searched strictly AFTER the start again, so an end matching the start line no longer closes the span there and read disagrees with apply.go:748 exactly as it did before this record · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · covers:an end matching the start line closing the span there
- 2026-09-08 · 33003e3* · mutant killed · exit 1 · `internal/read/read.go` · a paired pattern whose end never matches silently runs to the end of the file and reports SUCCESS again — the defect this record exists for, and the one that is invisible because a read that served everything looks exactly like a read that worked · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · covers:a paired pattern with no end being reported rather than served to EOF
- 2026-09-08 · 33003e3* · mutant killed · exit 1 · `internal/read/read.go` · the report drops which HALF missed, so a caller told only that the range matched nothing re-reads the file to learn whether it was the start or the end — and the assertion that catches this had to be strengthened first, because the spec text echoes the end pattern and the original check could not fail · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · covers:the report saying which half of the pattern failed
- 2026-09-08 · 33003e3* · mutant killed · exit 1 · `internal/read/read.go` · the resolver stops after the first start, so a read no longer serves a span for EVERY match — the behaviour this record deliberately KEEPS, and the one an over-eager alignment with the write path would have removed in silence · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · covers:a paired pattern that resolves normally still serving every start
- 2026-09-08 · 25dffb0* · mutant killed · exit 1 · `internal/read/read.go` · the missing-end report is gated on !found again, so a resolved earlier span suppresses it entirely and START END START EOF comes back as one span with exit 0 — a start the address named, silently dropped · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · covers:a resolved span not suppressing a later missing end
- 2026-09-08 · 25dffb0* · mutant killed · exit 1 · `internal/read/read.go` · the end is searched strictly AFTER the start again, so an end on the start line no longer closes the span there and read disagrees with apply.go:748 · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · covers:an end matching the start line closing the span there
- 2026-09-08 · 25dffb0* · mutant killed · exit 1 · `internal/read/read.go` · a paired pattern whose end never matches runs silently to EOF and reports SUCCESS again — the defect this record exists for · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · covers:a paired pattern with no end being reported rather than served to EOF
- 2026-09-08 · 25dffb0* · mutant killed · exit 1 · `internal/read/read.go` · the report drops which HALF missed, so a caller re-reads the file to learn whether it was the start or the end · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · covers:the report saying which half of the pattern failed
- 2026-09-08 · 25dffb0* · mutant killed · exit 1 · `internal/read/read.go` · the resolver stops after the first start, so a read no longer serves a span for every non-nested match — the behaviour this record deliberately keeps · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · covers:a paired pattern that resolves normally still serving every non-nested start

## Invariants

- `read` still serves a span for every match of the start that is not already inside a span it served. That difference from `write` is kept on purpose and is asserted by the control in S3; the nesting limit is asserted by the fixture's gap.
- `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against this record's baseline `33003e3` — NOT the branch merge-base `3245e1a`, against which `internal/apply` legitimately differs under ADR-035. `go.mod` declares exactly one requirement.
- Exit codes are unchanged: an unreadable range was already exit 1.

## Risks

- The MCP read path shares this resolver, so a change here reaches it; it is covered by existing rows and tests that predate this record. `--grep` does not share it — `walk.go:219` builds single-pattern ranges with no `ReEnd` — which was checked rather than assumed.
- A test could pass because the read failed for an unrelated reason. S1 asserts the served output contains no content line from the fixture, which a missing file or a bad spec cannot fake.

## Stop Condition

Stop and ask if any existing test or contract row depends on the silent EOF
extension: that would mean the behaviour is load-bearing somewhere this record
did not look, and the direction of the change deserves re-checking before the
row is edited to match.

## Out of Scope

- The every-start difference (permanent: boundary: a plan must resolve to one site and an exploratory read must not be forced to — ADR-001)
- `internal/apply`'s resolver (permanent: boundary: the write side is correct and ADR-035's tasks pin it byte-identical)

## Verification Log
- 2026-09-08 · 33003e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · ms:35588
- 2026-09-08 · 33003e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · ms:35326
- 2026-09-08 · 33003e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · ms:35026
- 2026-09-08 · 33003e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · ms:39565
- 2026-09-08 · 33003e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · ms:36871
- 2026-09-08 · 33003e3* · exit 0 · `set -o pipefail …` · acceptance-sha256:8e322a413c038992672035589d0d40e975f0ad98f26d959d673e4144cbf7353f · ms:36016
- 2026-09-08 · 25dffb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · ms:38010
- 2026-09-08 · 25dffb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · ms:37602
- 2026-09-08 · 25dffb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · ms:34853
- 2026-09-08 · 25dffb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · ms:38350
- 2026-09-08 · 25dffb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · ms:42908
- 2026-09-08 · 25dffb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:55789a79f9f5e19ff97300e0a3ca7d6ea83dc055500a2e63df8d7fb1216cdbb2 · ms:34183
