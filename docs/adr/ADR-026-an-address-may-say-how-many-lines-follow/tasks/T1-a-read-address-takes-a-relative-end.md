# Task ADR-026-T1: A read address takes a relative end

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (single file, plus its test)
**Owner:** Zy
**Produces:** `read.Range.RelEnd` and the `A,+N` read grammar, with its two refusals (T2, T3)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the served range being start..start+N`, `the clamp at EOF`, `the refusal of a relative end with no start`, `the refusal of +0`

## Goal

Make `A,+N` one range in a read spec — the line `A` resolves to, plus the `N` lines after it —
instead of two addresses whose second one silently means the absolute line `N`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | Four sites. (a) `splitRanges`'s `case ','` branch continues the range when the comma is followed by `+`, so the suffix stays attached instead of starting a new address. The symmetrical clause in the `/` branch is deliberately NOT there — see S3. (b) `parseRange` cuts a trailing `,+N` off before it parses `A`, and stores `N` in the new field. (c) `Range` gains `RelEnd int` — 0 means "no relative end", which is safe because `+0` is refused. (d) `resolve` applies it where the file length is already known, which is where `$` is resolved and therefore where the clamp is free. |
| `internal/read/read_test.go` | edit | The four tests below. `TestARelativeEndServesTheLinesAfterTheStart` is what SELECTS the new field — nothing else in the package constructs a `Range` with `RelEnd` set. |

## Ordered Steps

1. [S1] Write the four tests in `internal/read/read_test.go` and confirm they are RED against the current code. Two go through `run`/`runObserved` so the assertion is about the lines actually SERVED and observed, not about a parse tree: `/gamma/,+2` serves the match and the two lines below it, and `4,+10` on the five-line fixture serves `4-5` at exit 0. Two go through `ParseSpec`, because a refusal has no served output to assert on: `f.txt:+3` and `f.txt:2,+0` are both errors.
2. [S2] Add `RelEnd int` to `Range` with a doc comment saying 0 means absent and why that is not ambiguous — `+0` is refused, so no caller can mean it. [proof: mutation]
3. [S3] Teach the `case ','` branch of `splitRanges` to CONTINUE the range when the comma is followed by `+`, so `A,+N` stays one address. The symmetrical clause in the `/` branch is deliberately absent: it was written, mutated on 2026-09-06, and the mutant SURVIVED — when the pattern branch does not swallow the comma, the loop's next pass reaches `case ','`, which does. It was removed rather than kept as a clause nothing can distinguish, and the code comment records the measurement. [proof: mutation]
4. [S4] Teach `parseRange` to cut a trailing `,+N` before parsing `A`, so `A` still reaches the existing number/`$`/regex parser unchanged. Refuse `+0` naming the fix (drop the suffix), and refuse a relative end whose start is empty naming the two things the caller probably meant. [proof: mutation]
5. [S5] Apply `RelEnd` in `resolve`, after the start line is known, clamping the end to the last line exactly as `2-99` already clamps. [proof: mutation]
6. [S6] Confirm the forms that must NOT change are unchanged by running the existing address tests: `TestParseSpec`, `TestRegexpRange`, `TestDollarIsTheLastLineNotTheWholeFile`, `TestDollarAsAnEndStillRunsToEOF`, `TestAReversedRangeWrittenWithDollarIsNotServed`, `TestOverlappingRangesAreMergedNotRepeated`. [proof: acceptance]
7. [S7] Run the package, `gofmt` and `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ -run 'TestARelativeEnd' -count=1 -v 2>&1 | tee /tmp/adr026-t1.out \
  && grep -q '^--- PASS: TestARelativeEndServesTheLinesAfterTheStart' /tmp/adr026-t1.out \
  && grep -q '^--- PASS: TestARelativeEndPastTheLastLineClamps' /tmp/adr026-t1.out \
  && grep -q '^--- PASS: TestARelativeEndWithNothingBeforeItIsRefused' /tmp/adr026-t1.out \
  && grep -q '^--- PASS: TestARelativeEndOfZeroIsRefused' /tmp/adr026-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr026-t1.out \
  && go test ./internal/read/ -count=1 -run 'TestParseSpec|TestRegexpRange|TestDollarIsTheLastLineNotTheWholeFile|TestDollarAsAnEndStillRunsToEOF|TestAReversedRangeWrittenWithDollarIsNotServed|TestOverlappingRangesAreMergedNotRepeated' \
  && go test ./internal/read/... ./internal/adversarial/ -count=1 \
  && [ -z "$(gofmt -l internal/read)" ] \
  && go vet ./internal/read/...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARelativeEndServesTheLinesAfterTheStart` | `internal/read/read_test.go` | `/gamma/,+2` and `2,+1` each serve one span running from the start line to start+N, and observe exactly those lines | — | S1, S2, S3, S4, S5 |
| `TestARelativeEndPastTheLastLineClamps` | `internal/read/read_test.go` | `4,+10` on the five-line fixture serves `4-5` at exit 0, as `2-99` does, rather than reporting no match | — | S1, S5 |
| `TestARelativeEndWithNothingBeforeItIsRefused` | `internal/read/read_test.go` | `f.txt:+3` is a parse error whose message names both fixes | — | S1, S4 |
| `TestARelativeEndOfZeroIsRefused` | `internal/read/read_test.go` | `f.txt:2,+0` is a parse error naming the fix | — | S1, S4 |
| `TestParseSpec` | `internal/read/read_test.go` | Unchanged: every existing address form parses as it did | — | S6 |
| `TestOverlappingRangesAreMergedNotRepeated` | `internal/read/read_test.go` | Unchanged: a comma that is not followed by `+` or `/` still separates two addresses | — | S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestARelativeEndServesTheLinesAfterTheStart` |
| 2 — something selects it | `splitRanges` and `parseRange` are on the only path `mrw read` and `mrw_read` take to an address; the S3 mutation removes the `+` continue from the `case ','` branch so the suffix becomes a second address again, and the fence must go red |
| 3 — the caller can discover it | The address grammar in `README.md` and `AGENTS.md`, written in T3, and the two refusal messages, which name the fix in the caller's own words per ADR-015 |
| 4 — it is used | Contract §64 (T3) drives the form through the built binary on every contract run; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · 0cb2ad1* · mutant survived · exit 0 · `internal/read/read.go` · splitRanges stops holding `,+` to the pattern it follows, so `/gamma/,+2` becomes two addresses again and serves the match and line 2 — the exact pre-ADR-026 behaviour · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the served range being start..start+N
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · resolve stops widening the span, so a relative end parses and is then discarded — the form looks supported and serves one line · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the served range being start..start+N
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · parseRange stops guarding a bare `+N`, so `f.txt:+3` parses as the absolute line 3 through Atoi and the caller is served a line they did not ask for, at exit 0 · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the refusal of a relative end with no start
- 2026-09-06 · 0cb2ad1* · mutant survived · exit 0 · `internal/read/read.go` · probe · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the served range being start..start+N
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · splitRanges stops continuing the range at a `,+`, so the suffix becomes a second address and `2,+1` serves lines 2 and 1 — the pre-ADR-026 reading · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the served range being start..start+N
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · the pattern branch parses the suffix and then drops it, so `/func Foo/,+2` serves only the matching line while the numeric form still works — the half-wired shape a single-form test would miss · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the served range being start..start+N
- 2026-09-06 · 0cb2ad1* · mutant survived · exit 0 · `internal/read/read.go` · parseRange stops refusing `,+0`, so a suffix that says exactly what the start alone says is taken instead of named · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the refusal of +0
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · parseRange accepts `,+0`, so the one suffix that means exactly what the start alone means is taken silently instead of refused with the fix named · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the refusal of +0
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · the end stops clamping at the last line, so `4,+10` resolves past EOF instead of serving 4-5 at exit 0 — the rule a relative end inherits from `2-99` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · covers:the clamp at EOF

## Invariants

- A comma followed by neither `+` nor `/` still separates two addresses of one file.
- `/a/,/b/` is unchanged, and so is every numeric, `$` and regex address that carries no `,+`.
- `$` still resolves to the last line and `$-1` is still reversed and refused.
- A relative end never widens a range backwards: the start is exactly the line `A` resolved to.
- `internal/apply`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base. `internal/read` and `internal/plan` are the two this record OWNS (ADR-026 `Governs:`), so the usual engine go/no-go is lifted for them and for them only.
- `go.mod` still declares exactly one requirement.

## Risks

- `splitRanges` is scanning bytes, and a `,+` inside a regex must not be caught by the new `case ','` clause. The existing loop consumes a `/…/` wholesale before it looks at the next byte, so a `,+` inside a pattern is never seen by the comma case — asserted by keeping `TestRegexpRange` and `TestParseSpec` in the fence, and by `isDigits`, which refuses to read `3/` as a count.
- `internal/adversarial/record_test.go` turns `go test ./...` red the moment a Tests table names a test that does not exist, so this task's table is red until S1 lands. That is red-first working; the fence runs `./internal/adversarial/` so the task's own gate sees it.

## Stop Condition

Stop and ask if honouring `,+N` requires changing what a bare comma means, or if the clamp cannot be
applied in `resolve` without moving where `$` is resolved. Either means the form does not fit the
parser this record assumed, and the Decision should be revisited rather than the parser reshaped
around it.

## Out of Scope

- The plan side — that is T2's job
- The contract row and the documented grammar — that is T3's job
- A backwards relative address (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1454
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1319
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1000
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1406
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1597
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1207
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1040
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1342
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1392
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:21e1c87b9807d30c69ddd2b7345a61d53afe98e5a3c351110284c5c9d8d3cf8b · ms:1389
