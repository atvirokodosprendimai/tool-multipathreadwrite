# Task ADR-014-T2: Teach the continuation, once it exists

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (prose and one contract row)
**Owner:** unassigned
**Produces:** the taught form of the continuation on the MCP surface
**Consumes:** the first-page result and its continuation field (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the taught behaviour is the shipped behaviour`

## Goal

A caller that has only the MCP surface learns that a large read comes back as a page, that the
result says how to continue, and that stopping after page one means it has part of a file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go` | edit | the READING paragraph gains the page-and-continue behaviour |
| `internal/mcp/mcp.go` | edit | `mrw_read`'s description says a large read pages rather than failing |
| `internal/mcp/mcp_test.go` | edit | the taught behaviour is asserted against the shipped one |
| `README.md` | edit | the "ask for a narrower range" paragraph is now wrong — it pages |
| `scripts/contract.sh` | edit | §48 — what the wire teaches is what the binary does |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): the shipped `instructions` say a large read returns
   a page and how to continue; `mrw_read`'s description says it too, for a host that ignores
   `instructions`. [proof: acceptance]
2. [S2] Say the consequence, not only the mechanism. A caller told "you get a page" and not told
   "stopping here means you have part of the file" will stop and believe otherwise — the failure
   this ADR's Consequences admits is not fully solvable from the server side, which is exactly why
   the prose has to carry it. [proof: mutation]
3. [S3] Fix README. Its ADR-011 paragraph tells a reader an oversized read is refused and to ask for
   a narrower range. After T1 that is stale in the direction that matters: it describes a dead end
   the tool no longer has. [proof: acceptance]
4. [S4] Add contract §48: assert the shipped `instructions` describe paging, AND that a real
   oversized read against the built binary does what they describe — in one row. ADR-012 taught an
   enum the engine never sent and ADR-013 taught two examples that could not match; both were prose
   checked against nothing. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr014-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr014-t2.out \
  && [ "$(grep -cE '^--- PASS: (TestTheInstructionsTeachTheContinuation)\b' /tmp/adr014-t2.out)" = "1" ] \
  && grep -q '^# 48\.' scripts/contract.sh \
  && grep -q 'continue' README.md \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits** except
`grep -q 'continue' README.md`, which was counted and returns **1 hit today** — the word appears in
an unrelated sentence. It is kept anyway because it is the weakest clause in the fence and saying so
is the point: it would be green with this task undone. The clause that actually binds is §48, which
drives the taught text and the real binary together.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheInstructionsTeachTheContinuation` | `internal/mcp/mcp_test.go` | S1, S2 — the wire teaches paging AND the consequence of stopping | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `initializeResult()` and `tools()` are read by every host |
| 3 — the caller can discover it | this task IS the discovery surface |
| 4 — it is used | not measured and will not be — telemetry, refused by ADR-009; see ADR-012's Context |

## Mutation Log

<!-- filled during execution -->

## Invariants

- The paging behaviour described on the wire is the behaviour the binary has.
- README describes paging, not a dead-end refusal.
- `instructions` stays under `maxInstructionsChars`.

## Risks

- The prose drifts from the behaviour. Mitigated by §48 driving both in one row, which is the shape
  ADR-012 lacked when it taught `fail` instead of `failed`.
- Teaching paging pushes `instructions` over its bound. Then something already there must be
  shortened; raising the bound quietly is the wrong answer, since every session pays it.

## Stop Condition

Stop if teaching this needs `instructions` to grow past `maxInstructionsChars`. Shorten what is
there or cut the example, and say which — do not raise the constant to fit.

## Out of Scope

- Any behaviour change (T1 owns it)
- Changing `MaxResultChars` (deferred: parent ADR, Decision 4)

## Verification Log
<!-- filled during execution -->
