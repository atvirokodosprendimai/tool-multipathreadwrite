# Task ADR-026-T5: The lexer scans, and the wire teaches the form

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** M (the shared lexer, three arithmetic sites, the engine boundary, and four wire surfaces)
**Owner:** Zy
**Produces:** the delimiter-aware base scan, the overflow-safe relative end, the engine-boundary refusal, and the form on the MCP wire
**Consumes:** `internal/addr.CutRelative` (T4)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a relative end that cannot overflow`, `a base scanned with escapes honoured`, `the engine refusing what an op would ignore`, `the MCP wire teaching the form`

## Goal

Close the second review of PR #125: make the shared lexer scan rather than
substring-search, stop the relative end wrapping at the integer boundary, refuse
at the engine boundary as well as the parser's, and tell MCP callers the form exists.

## Context this task exists for

The second Codex review (`gpt-5.6-sol`, read-only, xhigh) of `a57975b` returned REQUEST CHANGES with
two HIGH and three MEDIUM. Reproduced against a binary built from that head:

    mrw read 'f.txt:/two/,+9223372036854775807'
    ==> f.txt  3L  14B
    @@ 2--9223372036854775807
    EXIT=0

A read that served **nothing**, reporting success, with an inverted receipt — the failure this tool
exists to make visible, reached through `i + 1 + RelEnd` wrapping negative. The numeric branch
survived it only by accident, because its `end < start` check caught the wrapped value; the pattern
branch has no such check.

    mrw read 'f.txt:/a\/,/,+2'   → refused: "has both an end pattern and a relative end"

A legal single-pattern base, refused: `strings.Contains(base, "/,/")` reads an ESCAPED slash followed
by a comma as the two-pattern delimiter. The first cut searched where it had to scan.

    read : !! no match for /two/,+1,+2        (exit 1)
    plan : unexpected ",+1" after the pattern (exit 2)

Two suffixes: cutting the last one leaves `/two/,+1`, which each parser then read differently — the
cross-path divergence T4 was written to end, surviving in a shape T4's matrix did not cover.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/addr/addr.go` | edit | `singleStart` and `closingSlash` replace the substring checks: the base is scanned to its closing slash with `\/` honoured, and the remainder must be empty, so `/a\/,/` is accepted and `/a/,+2,+3` is refused identically on both paths. |
| `internal/addr/addr_test.go` | edit | The table gains the escaped slash, the double suffix, `5,+1,+2`, an unclosed pattern, and the integer boundary. |
| `internal/read/read.go` | edit | Both resolve branches compare before they add — `RelEnd > total-start` — so a large count clamps instead of wrapping. The pattern branch had no `end < start` net beneath it. |
| `internal/apply/apply.go` | edit | The same comparison on the write path, plus an op check at the `Apply` boundary: `create`, `insert-after` and `insert-before` refuse a relative end there too, not only in `plan.validate`. `Apply`'s doc comment says it validates every hunk, and this package's own tests build Inputs directly. |
| `internal/apply/apply_test.go` | edit | The engine-boundary test and the integer-boundary test. |
| `AGENTS.md` | edit | It still told agents a relative end "clamps at the last line" after the write path began refusing one — advice that makes a caller build a plan the tool rejects. |
| `internal/mcp/instructions.go`, `internal/mcp/mcp.go` | edit | The four wire surfaces an MCP caller actually reads: the initialize instructions (twice) and both tool descriptions. All four omitted the form while the server accepted it. The instructions carry a 4096-byte bound, so room was made by compressing prose rather than by raising it — the bound's own test says to. |
| `scripts/contract.sh` | edit | §64 gains the AGENTS asymmetry gate and an MCP handshake gate driven through the built server. |

## Ordered Steps

1. [S1] Extend `internal/addr`'s table with the escaped slash (`/a\\/,/,+2`), the double suffix (`/a/,+2,+3`), `5,+1,+2`, an unclosed pattern and the integer boundary, and confirm they are RED against the substring implementation — the S1 mutant below restores that implementation and the fence goes red, which is the same measurement. Then replace the substring checks with a scan that honours `\\/` and requires the base to be exactly one start. [proof: mutation]
2. [S2] Compare before adding at all three arithmetic sites — the two read branches and the write path — so a large count clamps or refuses instead of wrapping. [proof: mutation]
3. [S3] Refuse a relative end on `create`, `insert-after` and `insert-before` at the `Apply` boundary, keeping `delete` working. [proof: mutation]
4. [S4] Correct `AGENTS.md` to the read-clamps/write-refuses split and gate the ASYMMETRY rather than the form's presence. [proof: acceptance]
5. [S5] Teach the form on all four MCP wire surfaces, fitting the existing 4096-byte bound by compressing prose. [proof: mutation]
6. [S6] Gate the MCP wire text through the BUILT server: initialize instructions and both tool descriptions. [proof: acceptance]
7. [S7] Run every gate, including `go test -race` over the four packages this record touches. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/addr/ ./internal/apply/ -count=1 -v \
  -run 'TestARelativeEndIsCutOnlyFromASingleStart|TestTheEngineRefusesARelativeEndTheOpCannotHonour|TestARelativeEndAtTheIntegerBoundaryDoesNotWrap' 2>&1 | tee /tmp/adr026-t5.out \
  && grep -q '^--- PASS: TestARelativeEndIsCutOnlyFromASingleStart' /tmp/adr026-t5.out \
  && grep -q '^--- PASS: TestTheEngineRefusesARelativeEndTheOpCannotHonour' /tmp/adr026-t5.out \
  && grep -q '^--- PASS: TestARelativeEndAtTheIntegerBoundaryDoesNotWrap' /tmp/adr026-t5.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr026-t5.out \
  && go test ./... -count=1 \
  && go test -race ./internal/addr/ ./internal/read/ ./internal/plan/ ./internal/apply/ -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr026-t5c.out \
  && grep -q '^contract holds$' /tmp/adr026-t5c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARelativeEndIsCutOnlyFromASingleStart` | `internal/addr/addr_test.go` | `/a\/,/,+2` is accepted, `/a/,+2,+3` and `5,+1,+2` are refused, an unclosed pattern is named, and a base too large for an int is left to the caller's parser | — | S1 |
| `TestARelativeEndAtTheIntegerBoundaryDoesNotWrap` | `internal/apply/apply_test.go` | `RelEnd` at `math.MaxInt` is refused as out of range and changes nothing | — | S2 |
| `TestTheEngineRefusesARelativeEndTheOpCannotHonour` | `internal/apply/apply_test.go` | `create`, `insert-after` and `insert-before` are refused at the `Apply` boundary; `delete` still removes three lines | — | S3 |
| `§64` | `scripts/contract.sh` | The AGENTS asymmetry, and the built MCP server teaching `A,+N` plus the clamp/refuse split in its instructions and in both tool descriptions | — | S4, S5, S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestARelativeEndIsCutOnlyFromASingleStart` |
| 2 — something selects it | `addr.CutRelative` is on the only path either parser takes to a relative end, and the arithmetic sites are the only places `RelEnd` becomes a span; §64 drives all of it through the built binary and the built MCP server |
| 3 — the caller can discover it | `README.md`, `AGENTS.md`, `mrw read --help`, the MCP initialize instructions and both MCP tool descriptions — every one of them gated separately in §64 |
| 4 — it is used | §64 runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · a57975b* · mutant killed · exit 1 · `internal/apply/apply.go` · the write path adds before it compares again, so a count at math.MaxInt wraps negative, reads as in range, and the hunk applies against a span nobody asked for · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · covers:a relative end that cannot overflow
- 2026-09-06 · a57975b* · mutant killed · exit 1 · `internal/addr/addr.go` · the base is found by search rather than scanned, so an escaped slash closes the pattern early and the legal base /a\/,/ is refused again · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · covers:a base scanned with escapes honoured
- 2026-09-06 · a57975b* · mutant killed · exit 1 · `internal/apply/apply.go` · the engine stops refusing a relative end on an op that would ignore it, so a direct Apply caller creates a file or inserts a line while the address said otherwise · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · covers:the engine refusing what an op would ignore
- 2026-09-06 · a57975b* · mutant survived · exit 0 · `internal/mcp/instructions.go` · the MCP initialize instructions stop naming the form, so a model reaching mrw over MCP — which never reads README or --help — cannot know it exists; the handshake gate in §64 is what notices · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · covers:the MCP wire teaching the form
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-06 · a57975b* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the READING half of the MCP instructions stops naming the form while the WRITING half still does — the shape that survived a check for the form anywhere in the text, and the reason the handshake gate now asserts both passages · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · covers:the MCP wire teaching the form

## Invariants

- The instructions stay within 4096 bytes: room is made by compressing prose, never by raising the bound. Its own test says so.
- Every phrase another contract section asserts about the wire survives — the one-spelling rule was compressed away once and §57 caught it in the same run.
- `delete` still takes a relative end at both the parser and the engine boundary.
- A read still clamps; only the write path refuses.
- `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- Compressing the MCP instructions can silently drop a rule another section asserts. That happened once here and the contract caught it in the same run; the invariant above records it rather than trusting the next editor to remember.
- The scan is byte-oriented and assumes `\` escapes only the delimiter. A pattern ending in a literal backslash before its closing slash would read as escaped — the same assumption `internal/plan.parsePattern` already makes, so the two agree, and disagreeing with it would be the worse bug.

## Stop Condition

Stop and ask if fitting the form into the MCP instructions requires cutting a rule another contract
section asserts. That is the budget telling you the wire is full, and the answer is a decision about
what an MCP caller most needs, not a quieter gate.

## Out of Scope

- Raising the 4096-byte instruction bound (permanent: boundary: it is paid once per session by every caller, and the bound's own test says to shorten what is there instead)
- A backwards relative address (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-06 · a57975b* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · ms:28215
- 2026-09-06 · a57975b* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · ms:31927
- 2026-09-06 · a57975b* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · ms:25122
- 2026-09-06 · a57975b* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · ms:24167
- 2026-09-06 · a57975b* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · ms:27092
- 2026-09-06 · a57975b* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b15d4fec9941028a71ab85c534374208ec0fa92cb50952974d901d1a5926be7 · ms:25316
