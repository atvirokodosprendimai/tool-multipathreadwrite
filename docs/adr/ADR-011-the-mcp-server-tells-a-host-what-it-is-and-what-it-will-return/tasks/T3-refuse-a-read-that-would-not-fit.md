# Task ADR-011-T3: Refuse a read that would not fit, instead of buffering it

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** the bounded read path
**Consumes:** `mcp.MaxResultChars`, the tool descriptor set (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the declared limit being the enforced limit`, `the refusal naming what to ask for instead`

## Goal

A read whose output would exceed the declared limit is refused with a message naming the limit and
the narrower request to make, rather than buffered to gigabytes and then truncated by the host.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | bound the read and refuse over the limit, using `read.Options.MaxLines` rather than a second limiter |
| `internal/mcp/tools_test.go` | edit | its tests |
| `README.md` | edit | say the MCP read is bounded, what the limit is, and how to ask for less |
| `docs/adr/BACKLOG.md` | edit | replace the read-buffering entry, whose stated reason for deferring is what this ADR corrects |
| `scripts/contract.sh` | edit | §42 — drive the real binary at a file over the limit and assert the refusal, the exit status and the peak behaviour |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a read under the limit is served unchanged; a read
   over it comes back `isError: true` with a message naming the limit; the refusal names the range
   syntax to retry with; the CLI path for the same file is UNAFFECTED. [proof: acceptance]
2. [S2] Bound the read. Use `read.Options.MaxLines`, which ADR-007 already defines and which "is
   always reported: a silent truncation reads as 'that was the whole file'". The engine keeps its
   semantics; this task chooses a value for one transport. [proof: mutation]
3. [S3] Refuse rather than truncate when the bound is hit. ADR-007's cap reports itself, and that is
   right for a human reading a terminal; over MCP the consumer is a model, and a truncated file that
   arrives looking like the file is the silent wrong answer this project exists to refuse.
   [proof: mutation]
4. [S4] Make the refusal actionable: name the limit, the size the request would have produced, and
   the range form (`path:1-500`) that would fit. A refusal a caller cannot act on converts one bad
   call into a loop of them.
5. [S5] Correct the `BACKLOG.md` entry. It defers this on the ground that "any cap is a behaviour
   divergence from the CLI"; the host caps at 25,000 tokens regardless, so the real choice was
   between refusing legibly and being truncated by someone else after paying the memory. The
   correction belongs where the wrong reason is written. [proof: acceptance]
6. [S6] Add contract §42: read a file over the limit through the real binary and assert `isError`,
   the limit named in the message, and that the same file through `mrw read` on the CLI still
   succeeds — the go/no-go that this bounded one transport and not the engine. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr011-t3.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr011-t3.out \
  && [ "$(grep -cE '^--- PASS: (TestAReadOverTheLimitIsRefusedNotTruncated|TestTheRefusalNamesTheLimitAndARangeToRetry|TestAReadUnderTheLimitIsUnchanged|TestTheCLIReadIsUnaffectedByTheMCPLimit)\b' /tmp/adr011-t3.out)" = "4" ] \
  && grep -q '^# 42\.' scripts/contract.sh \
  && grep -q 'bounded' README.md \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was grepped for BEFORE this fence was written and returned **zero hits**: the four test
names, `# 42.` in `contract.sh`, and `bounded` in `README.md`. The engine clauses matter more here
than anywhere else in this ADR: this task is the one that could plausibly reach into `internal/read`,
and both forms of the check are present because each sees what the other misses.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadOverTheLimitIsRefusedNotTruncated` | `internal/mcp/tools_test.go` | S3 — the whole point: a truncation that reads as the file is the failure | — | S1, S3 |
| `TestTheRefusalNamesTheLimitAndARangeToRetry` | `internal/mcp/tools_test.go` | S4 — a refusal a caller can act on | — | S1, S4 |
| `TestAReadUnderTheLimitIsUnchanged` | `internal/mcp/tools_test.go` | S2 — the bound does not change the ordinary case | — | S1, S2 |
| `TestTheCLIReadIsUnaffectedByTheMCPLimit` | `internal/mcp/tools_test.go` | S2 — one transport is bounded, the engine is not | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | `readTool` applies the bound; removing it makes `TestAReadOverTheLimitIsRefusedNotTruncated` and §42 red, both inside the fence |
| 3 — the caller can discover it | `_meta["anthropic/maxResultSizeChars"]` declared in T2 is the same constant, so a host reads the limit before it hits it; the README says it too |
| 4 — it is used | `mrw stats` counts plans, not reads, so nothing counts refusals — and counting them would need the telemetry ADR-009 refused |

## Mutation Log

## Invariants

- The limit enforced here is the constant T2 advertises; there is one constant and two readers.
- A read under the limit returns exactly what it returned before this task.
- No file under the six engine directories changes: `read.Options` already carries the bound.
- A refusal writes nothing and observes nothing — a refused read must not leave a ledger entry
  claiming the caller saw a file they did not.

## Risks

- The limit is wrong for some caller. Mitigated by it being declared in `tools/list`, named in the
  refusal, and settable in one place; and by the CLI path being untouched for anyone who needs the
  whole file.
- A caller loops on the refusal. Mitigated by S4: the message names the range form that fits, so the
  retry is obvious rather than guessed.

## Stop Condition

Stop if bounding the read requires a change under `internal/read`. `read.Options.MaxLines` exists and
this task sets it; if it turns out not to be sufficient, that is an ADR-007 question and a different
record, not a quiet edit to the engine — and the fence's two engine clauses will say so before the
commit lands.

## Out of Scope

- Bounding the WRITE path (permanent: boundary: measured 2026-09-03 at 17 MB for 2000 hunks across 2000 files; the receipt is proportional to hunks, not to file content, so there is nothing to bound)
- Streaming a large read across several responses (permanent: fact: MCP is one call, one message; citation: url https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- Reducing the copy amplification itself (deferred: docs/adr/BACKLOG.md)

## Verification Log
