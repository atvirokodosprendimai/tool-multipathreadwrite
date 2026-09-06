# Task ADR-026-T6: One scanner for every delimiter, and no arithmetic that wraps

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** M (one shared scanner adopted by four callers, one arithmetic site, and the tests that reach them)
**Owner:** Zy
**Produces:** `addr.ClosingDelim` as the single delimiter scanner, the overflow-safe `-C`, and read-path boundary tests
**Consumes:** `internal/addr.CutRelative` (T4), the scanning base check (T5)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `one delimiter scanner with backslash parity`, `a context that cannot overflow`, `the read path's own boundary regression`

## Goal

Make every place that finds the end of a pattern use one function that counts backslash parity, stop
`-C` wrapping the same way the relative end did, and put the reproducer on the path it was found on.

## Context this task exists for

The third Codex review of PR #125 confirmed all five earlier findings fixed and raised these.
Reproduced against a binary built from `219dfcd`:

    mrw read -C 9223372036854775807 'f.txt:/two/'
    @@ 1--9223372036854775807
    EXIT=0

`-C` reaches the same `i + 1 + ctx` arithmetic the relative end had just been protected from, one
line away, and the flag refuses only a NEGATIVE value — so a caller can type this. Same silent
success: an inverted span, no lines, no problems.

    mrw read 'bs.txt:/\\/,+1'            → served
    @@ bs.txt /\\/,+1 replace            → "pattern is never closed"

`/\\/` is a pattern matching one literal backslash, and its final slash is NOT escaped — the
backslash before it is itself escaped. **Four scanners in this repository each had their own idea of
where a pattern ends**, and all four tested only the preceding byte:
`internal/addr.closingSlash`, `internal/read.splitRanges`, `internal/plan.parsePattern`, and
`internal/plan.splitHeader`. The fourth was the one that made the two paths disagree, and it did two
things wrong: it unescaped `\\` to `\` inside a pattern, and its `\/` case fired on the SECOND
backslash of a pair and swallowed the closing slash, absorbing the rest of the header — the op
included — into the address.

⚠ **The third finding is about my own gate.** `TestARelativeEndAtTheIntegerBoundaryDoesNotWrap`
builds an `apply.Input`, so it never reached either read branch — a regression fixture that describes
the defect without executing it, which is exactly what `testing.md` names as this repository's
failure mode. The reproducer was a READ.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/addr/addr.go` | edit | `closingSlash` becomes the exported `ClosingDelim`, counting the parity of the backslash run before a candidate delimiter. Exported because the agreement has to be shared, not re-implemented. |
| `internal/read/read.go` | edit | `splitRanges` calls it, and the `-C` arithmetic compares before it adds. |
| `internal/plan/plan.go` | edit | `parsePattern` calls it; `splitHeader` stops unescaping inside a pattern and consumes any escape pair as a unit, which is what gives parity for free. |
| `internal/read/read_test.go` | edit | The boundary reproducer on the READ path — both branches — plus `-C` at the boundary and the backslash pattern. |
| `scripts/contract.sh` | edit | §64 drives the boundary and the backslash pattern through the built binary, on both paths, asserting the ABSENCE of an inverted span as well as the presence of real lines. |
| `docs/adr/ADR-026-…md`, `T4`, `T5` | edit | Three stale claims: the MCP surface needing "no change of its own", a rollback naming three commits, and a `§NN` placeholder. T4's "same address strings" narrowed to the relative-end strings it actually shares. |

## Ordered Steps

1. [S1] Write the read-path boundary tests first and confirm they are RED against the current tree — the pattern branch, the numeric branch, `-C`, and the backslash pattern. This is the finding, so it is also the fixture. [proof: mutation]
2. [S2] Export `ClosingDelim` with parity counting and adopt it in `internal/addr`, `internal/read.splitRanges` and `internal/plan.parsePattern`. [proof: mutation]
3. [S3] Stop `splitHeader` unescaping inside a pattern, and make its escape case consume the next rune whatever it is. [proof: mutation]
4. [S4] Compare before adding in the `-C` arithmetic. [proof: mutation]
5. [S5] Add the contract rows: the boundary on both branches and on `-C`, the write-side refusal at the boundary, and the backslash pattern on both paths. [proof: acceptance]
6. [S6] Correct the three stale record claims and narrow T4's. [proof: human: read the ADR's Component/Boundary and Consequences against the two MCP source files this branch changed, and confirm neither still says the surface needed no change]
7. [S7] Run every gate INCLUDING `go test -race ./...` — the full one CONTRIBUTING.md requires, not a package subset. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ -count=1 -v \
  -run 'TestARelativeEndAtTheIntegerBoundaryClampsOnARead|TestContextAtTheIntegerBoundaryClamps|TestAPatternEndingInABackslashIsClosed' 2>&1 | tee /tmp/adr026-t6.out \
  && grep -q '^--- PASS: TestARelativeEndAtTheIntegerBoundaryClampsOnARead' /tmp/adr026-t6.out \
  && grep -q '^--- PASS: TestContextAtTheIntegerBoundaryClamps' /tmp/adr026-t6.out \
  && grep -q '^--- PASS: TestAPatternEndingInABackslashIsClosed' /tmp/adr026-t6.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr026-t6.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr026-t6c.out \
  && grep -q '^contract holds$' /tmp/adr026-t6c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARelativeEndAtTheIntegerBoundaryClampsOnARead` | `internal/read/read_test.go` | Both read branches clamp at `math.MaxInt` and produce no inverted span | — | S1, S2 |
| `TestContextAtTheIntegerBoundaryClamps` | `internal/read/read_test.go` | `-C` at `math.MaxInt` clamps to the whole file rather than wrapping | — | S1, S4 |
| `TestAPatternEndingInABackslashIsClosed` | `internal/read/read_test.go` | `/\\/,+1` resolves to its match plus one on the read path | — | S1, S2 |
| `§64` | `scripts/contract.sh` | The built binary at the boundary on both branches and on `-C`, the write-side refusal, and `/\\/,+1` applying through a plan | — | S3, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAPatternEndingInABackslashIsClosed` |
| 2 — something selects it | `ClosingDelim` is now the only delimiter scan in the repository — `grep -n "!= '\\\\\\\\'" internal/` returns nothing outside it — and §64 drives both paths through the built binary |
| 3 — the caller can discover it | Unchanged by this task: the grammar passages T3 and T5 gate |
| 4 — it is used | §64 runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · 219dfcd* · mutant killed · exit 1 · `internal/addr/addr.go` · the scanner stops counting backslash parity and tests only the preceding byte again, so the closing slash of /\\/ reads as escaped and a legal pattern is refused as unclosed · acceptance-sha256:6be98cc64be73c6f25e43fbe2cc5d281776f2736b257d7f4596bbf0a40e05e6e · covers:one delimiter scanner with backslash parity
- 2026-09-06 · 219dfcd* · mutant killed · exit 1 · `internal/read/read.go` · the context is added before it is compared again, so -C at math.MaxInt wraps to a negative end and the read prints an inverted span while serving nothing at exit 0 · acceptance-sha256:6be98cc64be73c6f25e43fbe2cc5d281776f2736b257d7f4596bbf0a40e05e6e · covers:a context that cannot overflow
- 2026-09-06 · 219dfcd* · mutant killed · exit 1 · `internal/read/read.go` · the pattern branch of the read path adds before it compares, which is the exact reproducer the earlier boundary test could not reach because it built an apply.Input instead of doing a read · acceptance-sha256:6be98cc64be73c6f25e43fbe2cc5d281776f2736b257d7f4596bbf0a40e05e6e · covers:the read path's own boundary regression

## Invariants

- `\/` inside a pattern still means a literal slash: the escaped-slash address `/x\/y/` must keep working, and it is asserted beside the backslash case.
- The header splitter still unescapes `\"` and `\\` OUTSIDE a pattern, so an anchor naming code with a quote is unchanged.
- A read clamps, a write refuses — unchanged by the arithmetic fix.
- `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- Consuming any escape pair inside a pattern changes what reaches the regexp engine for inputs nobody has written. Mitigated by keeping the escaped-slash control in the same contract row as the backslash case, so the two cannot be traded for each other.
- `ClosingDelim` is exported from a package the engine depends on, which widens its surface. Judged worth it: four private copies is what produced this finding, and the alternative is a fifth.

## Stop Condition

Stop and ask if a fifth delimiter scanner turns up that cannot use `ClosingDelim` — that would mean
the delimiter rule is not actually one rule, and the record should say so rather than the code
pretending otherwise.

## Out of Scope

- A general escape grammar for the plan header beyond `\"`, `\\` and pattern-internal escapes (permanent: boundary: the header is whitespace-and-quotes, and a general grammar is a language nobody asked for)
- A backwards relative address (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-06 · 219dfcd* · exit 0 · `set -o pipefail …` · acceptance-sha256:6be98cc64be73c6f25e43fbe2cc5d281776f2736b257d7f4596bbf0a40e05e6e · ms:29866
- 2026-09-06 · 219dfcd* · exit 0 · `set -o pipefail …` · acceptance-sha256:6be98cc64be73c6f25e43fbe2cc5d281776f2736b257d7f4596bbf0a40e05e6e · ms:30323
- 2026-09-06 · 219dfcd* · exit 0 · `set -o pipefail …` · acceptance-sha256:6be98cc64be73c6f25e43fbe2cc5d281776f2736b257d7f4596bbf0a40e05e6e · ms:30616
- 2026-09-06 · 219dfcd* · exit 0 · `set -o pipefail …` · acceptance-sha256:6be98cc64be73c6f25e43fbe2cc5d281776f2736b257d7f4596bbf0a40e05e6e · ms:31382
