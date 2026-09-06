# Task ADR-026-T7: The header does not rewrite the pattern it carries

**Depends-on:** T6
**Covers:** none — no spec
**Estimated scope:** M (the header tokeniser, the read pattern parser, and the rows that pair them)
**Owner:** Zy
**Produces:** a header that leaves a pattern verbatim, and a read pattern parser that scans
**Consumes:** `addr.ClosingDelim` (T6)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a pattern reaching the parser as the caller wrote it`, `the two grammars refusing the same malformed patterns`

## Goal

Stop the plan header rewriting the regex inside it, and make the read path refuse the malformed
pattern addresses the plan path already refuses.

## Context this task exists for

The fourth Codex review of PR #125 confirmed every finding of the third fixed, and found two that
predate this branch. Reproduced against a binary built from `d2a9f55`:

    $ printf '@@ f.txt /^"foo"$/ replace\nX\n' | mrw --root . write -
    ok   f.txt /^foo$/ replace  -1 +1

**A silent wrong write.** `splitHeader` toggles on `"` to keep a quoted anchor together, and it did
that INSIDE a pattern — consuming both quotes. `/^"foo"$/` reached the parser as `/^foo$/`, a
different expression that matched a different line, and the receipt echoed the mutation rather than
what the caller wrote. An odd quote (`/"/`) produced "unterminated quote in header" for a legal
regex. Every pattern test in this repository calls `ParseAddr` directly, so none of them came
through the splitter — the same blind spot the header's own `inPat` comment records from PR #74.

    read: f.txt:/            exit 0, served every line     plan: refused
    read: f.txt:/a/garbage   compiled as `a/garbage`       plan: refused
    read: f.txt:/a/,/b/,/c/  silently became /a/           plan: refused

**And the two grammars still disagreed.** `parseRange` reached its pattern by `TrimPrefix` +
`TrimSuffix` + `Split`, so an unclosed pattern became an empty regexp matching every line, trailing
bytes were compiled into the expression, and a third endpoint was dropped. This record claims the two
paths agree about what an address is; on these four inputs they did not.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | The quote branch is gated on `!inPat`, so a quote inside a pattern is an ordinary regexp character; and a pattern still open at the end of a header is reported instead of silently absorbing the rest of the line. |
| `internal/plan/plan_test.go` | edit | `TestAQuoteInsideAPatternSurvivesTheHeader` goes through `Parse`, not `ParseAddr` — the splitter is the thing under test, and calling the parser directly is what hid this. |
| `internal/read/read.go` | edit | `parseRange` scans the pattern with `addr.ClosingDelim`: a missing close, an empty body, trailing bytes and a third endpoint are each refused, naming what was written. |
| `internal/read/read_test.go` | edit | The four malformed shapes and the three legal controls. |
| `scripts/contract.sh` | edit | §64 drives the four malformed shapes through BOTH surfaces of the built binary and asserts both refuse, plus a quoted pattern applying and its receipt keeping the quotes. |
| `T4`, `T6` | edit | Two claims the review called overbroad: T4's "the same string" is narrowed to the relative-end strings actually shared, and T6's "the only delimiter scan" now says what is true — every address parser, not `splitHeader`. |

## Ordered Steps

1. [S1] Write `TestAQuoteInsideAPatternSurvivesTheHeader` through `plan.Parse` and the malformed-address table through `read.ParseSpec`, and confirm both are RED. [proof: mutation]
2. [S2] Gate the header's quote branch on `!inPat`, keeping the quoted-anchor control green. [proof: mutation]
3. [S3] Report an unterminated pattern in a header rather than letting `inPat` survive to the end of the line. [proof: acceptance]
4. [S4] Scan the read pattern with `ClosingDelim`, refusing a missing close, an empty body, trailing bytes and a third endpoint. [proof: mutation]
5. [S5] Add the §64 rows: the four malformed shapes on both surfaces, and the quoted pattern applying with its receipt intact. [proof: acceptance]
6. [S6] Correct T4's and T6's claims to what is true. [proof: human: read T6's rung 2 against `grep -rn 'ClosingDelim' internal/` and confirm the list of callers is exactly what the row names, and that `splitHeader` is named as the exception rather than omitted]
7. [S7] Run every gate including `go test -race ./...`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ ./internal/plan/ -count=1 -v \
  -run 'TestAMalformedPatternAddressIsRefused|TestAQuoteInsideAPatternSurvivesTheHeader' 2>&1 | tee /tmp/adr026-t7.out \
  && grep -q '^--- PASS: TestAMalformedPatternAddressIsRefused' /tmp/adr026-t7.out \
  && grep -q '^--- PASS: TestAQuoteInsideAPatternSurvivesTheHeader' /tmp/adr026-t7.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr026-t7.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr026-t7c.out \
  && grep -q '^contract holds$' /tmp/adr026-t7c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAQuoteInsideAPatternSurvivesTheHeader` | `internal/plan/plan_test.go` | `/^"foo"$/` survives `Parse` intact, `/"/ ` is not an unterminated header, a quoted anchor still parses, and an unclosed pattern is reported | — | S1, S2, S3 |
| `TestAMalformedPatternAddressIsRefused` | `internal/read/read_test.go` | `/`, `//`, `/a/garbage`, `/a/,/b/,/c/` and `/a/,/b` are refused naming what was written; `/a/`, `/a/,/b/` and `/a/,+2` still parse | — | S1, S4 |
| `§64` | `scripts/contract.sh` | The built binary refuses all four malformed shapes on BOTH surfaces, applies a quoted pattern, and echoes it with its quotes | — | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAQuoteInsideAPatternSurvivesTheHeader` |
| 2 — something selects it | `splitHeader` is on the only path a plan hunk's header takes, and `parseRange` on the only path a read spec's address takes; §64 drives both through the built binary, and the S2 mutation restores the quote toggle so the receipt shows a mutated address again |
| 3 — the caller can discover it | Unchanged by this task |
| 4 — it is used | §64 runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · d2a9f55* · mutant killed · exit 1 · `internal/plan/plan.go` · the header toggles on a quote inside a pattern again, so /^"foo"$/ reaches the parser as /^foo$/ — a different expression, a different line, and a receipt echoing the mutation · acceptance-sha256:f3c09e49ec2333e5ab59c5cc6e46cf74ee7265aea915a056e2643cc76643a775 · covers:a pattern reaching the parser as the caller wrote it
- 2026-09-06 · d2a9f55* · mutant killed · exit 1 · `internal/read/read.go` · the read pattern is found by search rather than scanned, so /a/garbage compiles as a/garbage and the two grammars disagree again on what is malformed · acceptance-sha256:f3c09e49ec2333e5ab59c5cc6e46cf74ee7265aea915a056e2643cc76643a775 · covers:the two grammars refusing the same malformed patterns

## Invariants

- A quoted anchor still works: `anchor="a b"` keeps its space, which is what the toggle exists for.
- `\/` inside a pattern is still a literal slash, and `/\\/` is still one literal backslash.
- The two legal pattern forms — `/re/` and `/from/,/to/` — parse on both paths, and the read path keeps the richer grammar the plan path does not have. This task narrows what is malformed, not what is allowed.
- `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- Refusing trailing bytes after a pattern could break a caller who relied on `/a/garbage` compiling as `a/garbage`. Judged safe: it is undocumented, the plan path already refuses it, and the read receipt never showed the caller what it had actually compiled.
- Gating the quote toggle on `!inPat` means a quote can no longer group inside a pattern. That is correct — a pattern is delimited by slashes — and the quoted-anchor control is kept in the same test and the same contract row.

## Stop Condition

Stop and ask if making the two grammars agree on malformed input requires making them agree on
LEGAL input too. Read's grammar is deliberately richer, and collapsing that is a different decision
than this record made.

## Out of Scope

- Unifying the header tokeniser with the address parsers (permanent: boundary: a header is a line of fields and an address is one field; one scanner for both is the grammar this record already declined to build)
- A backwards relative address (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-06 · d2a9f55* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3c09e49ec2333e5ab59c5cc6e46cf74ee7265aea915a056e2643cc76643a775 · ms:33351
- 2026-09-06 · d2a9f55* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3c09e49ec2333e5ab59c5cc6e46cf74ee7265aea915a056e2643cc76643a775 · ms:39660
- 2026-09-06 · d2a9f55* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3c09e49ec2333e5ab59c5cc6e46cf74ee7265aea915a056e2643cc76643a775 · ms:31638
