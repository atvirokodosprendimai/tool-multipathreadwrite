# Task ADR-026-T4: One lexer, and an address no op can half-ignore

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** L (a new package, both parsers, apply, two adapters, the contract and the docs)
**Owner:** Zy
**Produces:** `internal/addr.CutRelative`, the op refusals, the receipt that keeps the caller's address, the write-side out-of-range refusal
**Consumes:** the `A,+N` read grammar (T1), `plan.ParseAddr` accepting `A,+N` (T2), contract §64 (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `one lexer for both paths`, `the refusal of a base that is not a single start`, `the refusal of a relative end on an op that would ignore it`, `the receipt echoing the caller's address`, `the write-side refusal past the last line`

## Goal

Make the two paths recognise and refuse the same RELATIVE-END strings **by construction** rather
assertion, and stop every remaining way a relative end could be parsed and then quietly not honoured.

## Context this task exists for

The Codex review of PR #125 (`gpt-5.6-sol`, read-only, xhigh) returned REQUEST CHANGES with three
HIGH findings, all three reproduced against a binary built from `b621b21` before any fix:

    mrw read 'f.txt:,+3'       → exit 0, @@ 1-4   (the plan path refused the same string)
    mrw read 'f.txt:5-7,+3'    → exit 0, @@ 5-6   (the declared end 7 silently overwritten)
    @@ f.txt 2,+3 insert-after → ok  f.txt 2 insert-after  -0 +1   (suffix ignored)
    @@ f.txt 2,+1 replace      → ok  f.txt 2 replace       (receipt drops the caller's address)
    @@ f.txt 5,+99 replace     → ok, replaced 2 lines      (a write doing less than it said)

⚠ **These tests were written with the fix already understood, not red-first**, because the finding
arrived as a review rather than as a plan. The Mutation Log is therefore doing the work the red-first
order usually does: every mechanism below is broken on purpose and the fence is watched to go red.
Said plainly here rather than left for a reader to infer from the dates.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/addr/addr.go` | add | The lexical half of the grammar, shared: suffix recognition, digit validation, the single-start check, and the wording of every refusal. One function cannot drift; two copies had already drifted. |
| `internal/addr/addr_test.go` | add | The base matrix — what is cut, what is left alone, what is refused and in whose words. |
| `internal/read/read.go` | edit | `parseRange` calls `addr.CutRelative`; the local `isDigits` and the duplicated refusals are deleted. |
| `internal/plan/plan.go` | edit | `ParseAddr` calls the same function; `validate` refuses a relative end on `insert-after`, `insert-before` and `create`; `Addr.String` renders a pattern base instead of printing `0,+N`. |
| `internal/apply/apply.go` | edit | The write path REFUSES a relative end that runs past the last line instead of clamping, and `srcAddrOf` keeps `,+N` so the receipt echoes the address the caller wrote. |
| `internal/curve/score.go` | edit | The THIRD `plan.Hunk → apply.Input` adapter, whose own comment claims to be "the mapping cmd/mrw performs". It dropped `RelEnd`, so the benchmark scored relative plans under different edit semantics from the tool it measures. T2 named two adapters; there were three. |
| `internal/mcp/tools_test.go` | edit | The parity helper `cliWrite` also dropped `RelEnd` — and `StartPat`/`EndPat` before that — while claiming to apply a plan "the way cmd/mrw does". A parity helper that is not parity makes the comparison it exists for meaningless. |
| `scripts/contract.sh` | edit | §64 gains a cross-path base matrix comparing the refusal MESSAGE rather than a fragment of it, the op refusals, the receipt assertions, and the clamp/refuse pairing. |
| `README.md`, `cmd/mrw/main.go` | edit | The Read grammar and `mrw read --help` omitted the form entirely, and the Write passage claimed a relative end clamps "exactly as `12-9999` does" — false, since a plan's `12-9999` is refused. |

## Ordered Steps

1. [S1] Write `internal/addr` with `CutRelative` and its table test, covering what is cut, what is left alone (`/a,+3/`, `5-7`, `-`, `0`), and every refusal. [proof: mutation]
2. [S2] Rewire `read.parseRange` and `plan.ParseAddr` onto it, deleting both local copies of the suffix logic and both copies of the refusal text. [proof: mutation]
3. [S3] Refuse a base that is not a single start — empty, `0`, `-`, or one that already carries an end (`5-7`, `5-`, `$-5`) — because a relative end REPLACES the end rather than joining it. [proof: mutation]
4. [S4] Refuse a relative end on `insert-after`, `insert-before` and `create` in `validate`, and confirm `delete` still takes one: the rule is "an op that cannot honour it must refuse it", not "relative ends are suspicious". [proof: mutation]
5. [S5] Make `srcAddrOf` and `Addr.String` keep the relative end, and render a pattern base as the pattern rather than as line 0. [proof: mutation]
6. [S6] Refuse, on the write path only, a relative end running past the last line — matching `5-9999`'s existing refusal — while the read path keeps clamping to match `2-99`. [proof: mutation]
7. [S7] Propagate `RelEnd` through the curve adapter and the MCP parity helper. [proof: acceptance]
8. [S8] Extend §64: the cross-path matrix comparing whole messages, the op refusals, the receipt, and the clamp/refuse pair. [proof: acceptance]
9. [S9] Correct `README.md` (both passages) and `mrw read --help`. [proof: human: read the Read passage, the Write passage and `mrw read --help` side by side and confirm all three describe the same grammar and the same clamp/refuse split; the claim they must not repeat is that a relative end clamps on a write]
10. [S10] Run every gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/addr/ ./internal/plan/ ./internal/apply/ -count=1 -v \
  -run 'TestARelativeEndIsCutOnlyFromASingleStart|TestARelativeEndIsRefusedWhereItWouldBeIgnored|TestAnAddressRendersBackAsTheCallerWroteIt|TestAReceiptEchoesTheRelativeEndTheCallerWrote|TestARelativeEndPastTheLastLineIsRefusedOnThePlanPath' 2>&1 | tee /tmp/adr026-t4.out \
  && grep -q '^--- PASS: TestARelativeEndIsCutOnlyFromASingleStart' /tmp/adr026-t4.out \
  && grep -q '^--- PASS: TestARelativeEndIsRefusedWhereItWouldBeIgnored' /tmp/adr026-t4.out \
  && grep -q '^--- PASS: TestAnAddressRendersBackAsTheCallerWroteIt' /tmp/adr026-t4.out \
  && grep -q '^--- PASS: TestAReceiptEchoesTheRelativeEndTheCallerWrote' /tmp/adr026-t4.out \
  && grep -q '^--- PASS: TestARelativeEndPastTheLastLineIsRefusedOnThePlanPath' /tmp/adr026-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr026-t4.out \
  && go test ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr026-t4c.out \
  && grep -q '^contract holds$' /tmp/adr026-t4c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARelativeEndIsCutOnlyFromASingleStart` | `internal/addr/addr_test.go` | The base matrix: what is cut, what is left alone, and that every refusal quotes what the caller wrote | — | S1, S2, S3 |
| `TestARelativeEndIsRefusedWhereItWouldBeIgnored` | `internal/plan/plan_test.go` | `insert-after`, `insert-before` and `create` refuse a relative end; `create -` still parses | — | S4 |
| `TestAnAddressRendersBackAsTheCallerWroteIt` | `internal/plan/plan_test.go` | `5,+3`, `/two/,+1`, `/a/,/b/`, `5-7` and `$` all render back as written | — | S5 |
| `TestAReceiptEchoesTheRelativeEndTheCallerWrote` | `internal/apply/apply_test.go` | The verdict line reports `6,+2`, not `6` | — | S5 |
| `TestARelativeEndPastTheLastLineIsRefusedOnThePlanPath` | `internal/apply/apply_test.go` | A write past the last line fails, names the caller's address and "out of range", and changes nothing | — | S6 |
| `§64` | `scripts/contract.sh` | Both paths refuse the same nine bases in the SAME WORDS through the built binary; the ops refuse; `delete` still accepts; the receipt keeps the suffix; read clamps where write refuses | — | S3, S4, S5, S6, S8 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestARelativeEndIsCutOnlyFromASingleStart` |
| 2 — something selects it | `addr.CutRelative` is the only path either parser takes to a relative end; §64 drives nine bases through both surfaces of the built binary and compares the messages, so re-splitting the lexer fails the row rather than waiting for a caller to find it |
| 3 — the caller can discover it | `README.md`'s Read and Write passages, `AGENTS.md` §1 and §2, and `mrw read --help` — S9 reads all of them against each other |
| 4 — it is used | §64 runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · b621b21* · mutant killed · exit 1 · `internal/addr/addr.go` · the lexer stops refusing a base that already carries an end, so `5-7,+3` silently discards the 7 the caller wrote and the two paths part company again · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · covers:the refusal of a base that is not a single start
- 2026-09-06 · b621b21* · mutant killed · exit 1 · `internal/addr/addr.go` · an empty base is accepted, which is the exact divergence the review found: `,+3` served lines 1-4 on the read path while the plan path refused it · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · covers:the refusal of a base that is not a single start
- 2026-09-06 · b621b21* · mutant killed · exit 1 · `internal/plan/plan.go` · insert-after and insert-before stop refusing a relative end, so `2,+3 insert-after` applies at line 2 and reports ok while the caller wrote a four-line address · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · covers:the refusal of a relative end on an op that would ignore it
- 2026-09-06 · b621b21* · mutant killed · exit 1 · `internal/apply/apply.go` · the write path clamps a relative end past the last line instead of refusing it, so a plan asking for 99 lines silently replaces two and reports ok · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · covers:the write-side refusal past the last line
- 2026-09-06 · b621b21* · mutant killed · exit 1 · `internal/read/read.go` · the read path stops HONOURING the shared lexer while still calling it, so the two paths disagree again on every refused base — which is what the cross-path matrix in §64 exists to catch · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · covers:one lexer for both paths
- 2026-09-06 · b621b21* · mutant killed · exit 1 · `internal/apply/apply.go` · the receipt drops the relative end again, reporting 3,+1 as 3 and hiding the span the hunk consumed from the one line a caller reads · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · covers:the receipt echoing the caller's address

## Invariants

- The two paths refuse the same string in the same words. §64 compares the message, not a fragment.
- `delete` still takes a relative end: the rule is that an op which cannot honour one must refuse it.
- A read still clamps at the last line; only the write path refuses.
- ADR-001, ADR-002 and ADR-013 are untouched: addresses still resolve against the original file, the ledger is still per line, and a pattern still has to match exactly once.
- `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- `internal/addr` is a new package on the engine's critical path. Mitigated by it holding no state, no I/O and no resolution — it is a pure string function with a table test, and everything that needs a file stays where it was.
- The op refusals are a breaking change for a plan that wrote `2,+3 insert-after` and relied on it being ignored. Judged safe: that behaviour shipped only on this unmerged branch.
- Rendering a pattern in `Addr.String()` changes existing diagnostic text. Mitigated by the whole suite plus §64.

## Stop Condition

Stop and ask if sharing the lexer forces resolution into the shared package too — that would put
file length into a string function and is the boundary ADR-013 draws.

## Out of Scope

- Unifying the two parsers wholesale (permanent: boundary: read's grammar is strictly richer, and folding it into `ParseAddr` grows a second grammar inside it — `docs/adr/BACKLOG.md`'s own argument, which this task deliberately does not overturn)
- A backwards relative address (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:25284
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:33461
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:28289
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:31456
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:29248
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:22696
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:22163
- 2026-09-06 · b621b21* · exit 0 · `set -o pipefail …` · acceptance-sha256:a191761f7910eee6a3c6d9deccec3d2ec22853a49457574704e0e977a46cefc1 · ms:29344
