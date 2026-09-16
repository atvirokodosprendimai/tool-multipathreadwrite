# Reading: under-ceiling host-cut probe

**Collected 2026-09-16** while executing `docs/specs/2026-09-16-dangling-high-impact-plan.md` Task 3.
PATH `mrw` v1.22.0 (`6896442`). Default MCP ceiling 200000 (flag omitted). **This is not ADR-039.**
ADR-039 is the licensing consistency close; this file is the measurement that record's Follow-ups
still asked for.

## Host

Cursor (this session). A namespace search for `mrw` MCP tools returned zero matches, so `mrw_read`
was never a host tool result here. The model half of the plan did not run. Do not quote this as a
Claude Code or Desktop observation.

## Wire (JSON-RPC client, no model)

Subprocess `mrw --root <throwaway> mcp` on stdio, isolated `XDG_STATE_HOME`. The payloads below are
what the server sent, not what a host delivered to a model.

### 1. Fitting (not a page)

- Spec: `small.txt:1-20` (20 lines, 851 bytes on disk).
- Encoded `tools/call` result: 1640 characters (under 200000).
- Served text: 1529 characters. No `-- PARTIAL:`. `isError` key **absent**.
- Numbered lines 1–20, continuous. No `characters truncated` (that phrase is not in this repository).
- Checkpoints: 1 open / 1 close (ADR-039 fitting markers).

### 2. Paged, `isError` absent (post-024; size near reading 18's page)

- Spec: bare `large.txt` (3799 lines, 326714 bytes). Throwaway fixture, not reading-18's
  `tmp/curve/200000-early-1`.
- Encoded `tools/call` result: 155013 characters.
- `-- PARTIAL: lines 1-1630 of 3799. 2169 line(s) remain.`
- `isError` key **absent** (ADR-024). Numbered lines 1–1630 continuous. No `characters truncated`.
- Checkpoints: 9 open / 9 close.

Raw JSON-RPC lines were not committed.

## Model

Not run on this host.

## Score

| Server sent | Model received | Verdict |
|-------------|----------------|---------|
| Continuous, under ceiling (1640 encoded chars) | not observed | Class **not closed**. A wire dump is not a host-cut. |
| Page with `-- PARTIAL:`, `isError` absent, 155013 encoded chars | not observed | Same. Compare to reading 18 only when a truncating host is in the path. |

## What this does not establish

- Not a miss rate for other hosts.
- Not "ADR-039 closed it".
- Not an engine defect: nothing here showed a host cutting an under-ceiling result.
- Reading 12 remains VOID. Reading 18 remains the paged Claude Code 2.1.261 cut.

BACKLOG stays **Deferred** until a host that truncates is measured.
