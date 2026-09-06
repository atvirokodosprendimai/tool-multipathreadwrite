# Task ADR-026-T2: A plan address takes the same relative end

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (four files: both parsers' consumers, plus two tests)
**Owner:** Zy
**Produces:** `plan.ParseAddr` accepting `A,+N` and resolving it against the original file (T3)
**Consumes:** the `A,+N` grammar and its two refusals (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the hunk addressing start..start+N`, `the same two refusals as the read path`

## Goal

Make `@@ f.txt /func Start/,+3 replace` address the match and the three lines after it, so a caller
who wrote a read address can paste it into a plan instead of meeting `bad line number`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `ParseAddr` cuts a trailing `,+N` before the pattern scan and the `-` cut, and records it on `Addr`. It is cut BEFORE `parsePattern` for the reason the existing comment gives about `-` and `,`: those are ordinary characters inside a regex, so the suffix must be recognised at the end of the token rather than by splitting the token. |
| `internal/apply/apply.go` | edit | Three sites. `Input` and the internal `hunk` carry `RelEnd` through to resolution, and the relative end is applied in the resolve loop AFTER the pattern and EOF sentinels have resolved — one rule for every address form. Not in the first draft of this task, which named only the parser: the parse is worthless until the resolved span reaches the splicer. |
| `cmd/mrw/main.go` and `internal/mcp/tools.go` | edit | The two places a `plan.Hunk` becomes an `apply.Input`. These are what SELECT the new field: delete either line and the CLI or the MCP server silently ignores a relative end while every unit test stays green. |
| `internal/plan/plan_test.go` | edit | `TestAPlanAddressTakesARelativeEnd` is what SELECTS the new field on the plan side — nothing else constructs an `Addr` carrying one. |

## Ordered Steps

1. [S1] Write `TestAPlanAddressTakesARelativeEnd` in `internal/plan/plan_test.go` and confirm it is RED. Four shapes in one test so no parser can satisfy it by accepting everything: `5,+3` is the address `5..8`; `/beta/,+1` resolves to the matching line and the one below it; `+3` alone is refused; `5,+0` is refused. The two refusals assert the message names the fix, not merely that an error occurred.
2. [S2] Add the relative end to `Addr` with a doc comment pointing at ADR-026 and at `read.Range.RelEnd`, so the two are findable from each other. [proof: mutation]
3. [S3] Cut the `,+N` suffix in `ParseAddr` before the pattern branch, and refuse `+0` and a suffix with no start with the same wording the read path uses. [proof: mutation]
4. [S4] Resolve the relative end where the hunk's start line is already known, against the ORIGINAL file — ADR-001's rule, so a relative end in one hunk is unaffected by what an earlier hunk did. Clamp at EOF as T1 does. [proof: mutation]
5. [S5] Confirm the plan forms that must not change are unchanged: `TestAnAmbiguousRegexAddressIsRefused` (ADR-013), the `/from/,/to/` range, `$`, `N-M`, and `N-`. [proof: acceptance]
6. [S6] Drive one relative-end hunk through `apply` end to end, so the address is proved to reach a write and not only a parse: replace `A,+2` and assert those three lines are what changed and that the line below them survived. The ledger half is asserted where it already lives — `internal/adversarial`'s `TestARegexAddressIsStillSubjectToTheLedger` — and is not re-asserted here. [proof: acceptance]
7. [S7] Run both packages, `gofmt` and `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ -run 'TestAPlanAddressTakesARelativeEnd' -count=1 -v 2>&1 | tee /tmp/adr026-t2.out \
  && grep -q '^--- PASS: TestAPlanAddressTakesARelativeEnd' /tmp/adr026-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr026-t2.out \
  && go test ./internal/apply/ -run 'TestARelativeEndAddressesTheLinesItReplaces' -count=1 -v 2>&1 | tee /tmp/adr026-t2b.out \
  && grep -q '^--- PASS: TestARelativeEndAddressesTheLinesItReplaces' /tmp/adr026-t2b.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr026-t2b.out \
  && go test ./internal/plan/ ./internal/apply/ -count=1 -run 'TestAnAmbiguousRegexAddressIsRefused|TestEarlierHunksNeverShiftLaterAddresses|TestAFileChangedBehindMrwsBackCannotBeEdited' \
  && go test ./internal/plan/... ./internal/apply/... ./internal/adversarial/ -count=1 \
  && [ -z "$(gofmt -l internal/plan internal/apply)" ] \
  && go vet ./internal/plan/... ./internal/apply/...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanAddressTakesARelativeEnd` | `internal/plan/plan_test.go` | `5,+3` and `/beta/,+1` parse to a start..start+N address; `+3` and `5,+0` are refused with a message naming the fix | — | S1, S2, S3 |
| `TestARelativeEndAddressesTheLinesItReplaces` | `internal/apply/apply_test.go` | A `A,+2` replace changes exactly those three lines, against the original file, and stops where it said | — | S1, S4, S6 |
| `TestAnAmbiguousRegexAddressIsRefused` | `internal/apply/apply_test.go` | Unchanged: ADR-013's ambiguity refusal is untouched by the new suffix | — | S5 |
| `TestEarlierHunksNeverShiftLaterAddresses` | `internal/apply/apply_test.go` | Unchanged: ADR-001 still holds with a relative end in the plan | — | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAPlanAddressTakesARelativeEnd` |
| 2 — something selects it | `ParseAddr` is the only path a plan hunk's address takes, for the CLI and for `mrw_write` alike; the S3 mutation drops the suffix cut so `5,+3` becomes `bad line number` again, and the fence must go red |
| 3 — the caller can discover it | The plan grammar in `AGENTS.md` §2 and `README.md`, written in T3 |
| 4 — it is used | Contract §64 (T3) drives a relative-end plan through the built binary; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/apply/apply.go` · apply parses the relative end and then discards it, so a plan hunk addressed 3,+1 replaces only line 3 while internal/plan tests stay green — the parse is not the behaviour · acceptance-sha256:da8158425010a7299bad25ea3178e455113e39acc9fbe234e15199e2170af05c · covers:the hunk addressing start..start+N
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/plan/plan.go` · ParseAddr stops guarding a bare `+N`, which is how `@@ a.go +3 replace` silently applied at line 3 before this record — measured against bab2128 · acceptance-sha256:da8158425010a7299bad25ea3178e455113e39acc9fbe234e15199e2170af05c · covers:the same two refusals as the read path
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/plan/plan.go` · ParseAddr accepts `,+0` while the read path refuses it, which is exactly the two-grammar divergence this record exists to remove · acceptance-sha256:da8158425010a7299bad25ea3178e455113e39acc9fbe234e15199e2170af05c · covers:the same two refusals as the read path

## Invariants

- ADR-001 holds: every address still resolves against the ORIGINAL file, so a relative end in one hunk is unaffected by an earlier hunk's edit.
- ADR-002 holds: a relative end licenses nothing. Every line it resolves to must have been served, and the refusal is still per line.
- ADR-013 holds: a pattern that matches no line or several still fails that hunk, naming the lines it matched.
- `/a/,/b/`, `$`, `N-M`, `N-` and a bare `-` are unchanged.
- `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base. `internal/read`, `internal/plan` and `internal/apply` are touched by this record and named in it.
- `go.mod` still declares exactly one requirement.

## Risks

- Cutting the suffix before `parsePattern` could eat a `,+` that is part of a regex — `/a,+b/` is a legal pattern. Mitigated by cutting only a suffix that matches `,\+[0-9]+$` AFTER a closing `/` or a numeric/`$` token, and by keeping `TestAnAmbiguousRegexAddressIsRefused` and a `/a,+b/` case in the test.
- The plan path and the read path could accept different things, which is the divergence this record exists to remove. Mitigated by T3's contract row driving both through the built binary in one section.

## Stop Condition

Stop and ask if the suffix cannot be cut without splitting a regex, or if resolving it against the
original file conflicts with how `apply` orders hunks. Either means the form needs a different
grammar on the plan side, and two grammars is what this record refused.

## Out of Scope

- The read side — T1
- The contract row and the documented grammar — T3

## Verification Log
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:da8158425010a7299bad25ea3178e455113e39acc9fbe234e15199e2170af05c · ms:1604
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:da8158425010a7299bad25ea3178e455113e39acc9fbe234e15199e2170af05c · ms:1916
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:da8158425010a7299bad25ea3178e455113e39acc9fbe234e15199e2170af05c · ms:1879
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:da8158425010a7299bad25ea3178e455113e39acc9fbe234e15199e2170af05c · ms:1546
