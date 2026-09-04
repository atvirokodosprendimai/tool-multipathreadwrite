# ADR-014: A read too large is a first page, not a dead end

**Status:** Proposed
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-011 (owns `MaxResultChars` and the refusal this changes), ADR-007 (owns the read this bounds), ADR-002 (owns the ledger a partial read must record honestly), ADR-009 (the tally whose attribution limit is why the cap's VALUE stays unmeasured here)
**Governs:** `internal/mcp/**`
**Enforced-by:** to be named by T1 — the test that a continuation actually continues
**Invalidates:** none — checked. ADR-011 decided that an oversized read is refused rather than truncated, and that stands unchanged: nothing here truncates. What changes is what the refusal HANDS BACK.
**Served-path change:** a caller whose read exceeds the limit gets the first page and the spec for the next one, instead of an error it must translate into a range by hand.

## Context

**The cap works and the refusal is a dead end.** ADR-011-T3 bounded the MCP result and the bound is
real. Measured 2026-09-04 against `14d28b3`, same file served both ways:

| Served | CLI out | CLI peak RSS | MCP out | MCP peak RSS |
|---|---|---|---|---|
| 1 KB | 1,032 | 6.1 MB | 1,436 | 6.4 MB |
| 10 KB | 9,757 | 6.0 MB | 10,316 | 6.6 MB |
| 50 KB | 48,689 | 6.0 MB | 49,931 | 6.9 MB |
| 100 KB | 97,313 | 6.1 MB | 99,410 | 7.5 MB |
| 190 KB | 184,866 | 6.3 MB | 188,499 | 8.4 MB |
| 400 KB | 389,154 | 6.9 MB | **294** | 7.9 MB |

The MCP path costs about 12× the served bytes in peak RSS while the CLI streams flat at ~6 MB, and
at 400 KB the transport returns 294 bytes — a refusal — at no memory or time penalty. So the bound is
a guard rather than an assertion after the cost is paid, which is the property ADR-011 claimed and
this is the first measurement of it.

**What the caller then has to do is the defect.** The refusal says *"Ask for a range instead — for
example f400.go:1-2678"*. That is one suggested range and nothing else: no page two, no count of what
remains, no way to say "continue". A caller wanting the whole file must compute the next range
itself, and a caller that stops after the suggested range has silently read a third of a file it
believes it has read. Reading `cmd/mrw/main.go` already returns ~13,000 tokens, so this is not an
exotic path.

**The cap's VALUE is unjustified, and this record does not fix that.** 200,000 was chosen because it
fits under Claude Code's per-tool ceiling. Nothing has measured whether a model's next edit gets
*worse* as more context is served — the widely-reported degradation with long context is about
retrieval and QA, and edit authoring is a narrower, more mechanical task nobody has measured it on.
Until that curve exists the honest position is that the constant bounds a RESOURCE and is silent
about quality, and the record says so rather than implying the number is principled. Changing the
number belongs to whatever measures it.

## Existing Primitives Audit

- **`MaxResultChars` and the capped writer (ADR-011-T3):** the bound. **Reused unchanged** — this
  record changes the refusal's payload, not when it fires.
- **The refusal's suggested range (`tools.go:372`):** already computes a line count that would fit.
  **Extended**: the same arithmetic becomes the first page's span rather than prose in an error.
- **`internal/read`'s span support and `seen.Record`'s span merge:** the ledger already accumulates
  spans per sha, so N sequential page reads license the union of what they served — verified reading
  `seen.merge` for ADR-012. **Reused unchanged, and it is what makes paging safe**: page 2 does not
  invalidate page 1's licence, and a write across both is licensed exactly when both were served.
- **MCP pagination (`nextCursor`):** the protocol's own idiom, and it is for `*/list` methods, not
  tool results. **Not taken as a mechanism**, but its SHAPE is worth copying: an opaque token the
  caller hands back. Here the token can be an ordinary mrw spec, which is better because the caller
  can read it, edit it, or ignore it.
- **Streaming the result across messages:** still unavailable — MCP is one call, one message. Noted
  because it is the obvious wish and it is not on the table.

## Decision

**1. An oversized read returns its FIRST PAGE plus the spec for the next**, rather than only an
error. The result carries the lines that fit, and a field naming the exact spec to send to continue —
`f400.go:2679-5357`. The caller can send it verbatim, narrow it, or stop.

**2. It is still `isError: true` when nothing was asked for narrowly enough**, so a caller that
ignores the field is not silently handed a third of a file as if it were the whole. ADR-011's refusal
principle is untouched: what the model receives must never look like the complete answer when it is
not. **The page is labelled, in the prose block and in the structured content, with what it is and
what remains.**

**3. What is served IS recorded in the ledger**, per ADR-002, so a page-one read licenses editing
page-one lines and nothing else. A partial read that licensed the whole file would be the
read-before-write bypass in a new costume; a partial read that licensed nothing would make paging
useless. The span mechanism already does exactly this and needs no change.

**4. The cap's value does not move.** No number changes in this record. Whether 200,000 is the right
place is a question for a measurement that does not exist yet, and inventing a different constant
would only relocate the same unjustified choice.

**Go/no-go, checked during execution:**

- **No engine change.** The five directories other than `internal/mcp` stay byte-identical in both
  `git status --porcelain --untracked-files=all` and a merge-base `git diff`.
- **No new dependency**; `go.mod` still declares exactly one requirement.
- **Paging is proved by paging.** The Enforced-by test must read a file to completion by following
  the continuation spec, and assert the concatenation equals the whole file. A test that only checks
  the field is present would pass over a spec that points at the wrong lines.
- **`gofmt -l .` empty and `go vet ./...` clean**, in the fence.

## Alternatives Considered

- **Leave it: refuse and suggest a range.** Today's behaviour. Rejected because the suggestion is not
  actionable past the first page, and because the failure mode of a caller that follows it once is a
  confident partial read.
- **Truncate and mark it.** Rejected, restating ADR-011: a truncated result that reaches a model
  looking like the file is the silent wrong answer this project exists to refuse. Paging differs
  precisely in that the caller must ASK for the rest.
- **Raise the cap.** Rejected: it moves the boundary without removing it, and the amplification
  measured above means a bigger constant costs proportionally more memory for the same defect.
- **An opaque `nextCursor` token.** Rejected in favour of an ordinary spec string. A cursor the
  caller cannot read is a cursor it cannot narrow, and mrw's specs are already the language the
  caller speaks.
- **Auto-page: serve the whole file across several tool results.** Not available (one call, one
  message), and undesirable anyway — the point of the bound is that the caller chooses to spend the
  context.
- **Decide the cap's value here, from the RSS curve above.** Tempting and wrong: that curve measures
  what the SERVER spends, and the question worth answering is what the CALLER's accuracy does. Using
  a memory measurement to justify a context-budget number would be the same category error as
  ADR-012's rejected criterion, one level down.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | The transport bound and now the continuation it offers | Yes |
| `internal/read` | Unchanged — it already serves spans | Untouched |
| `internal/seen` | Unchanged — span accumulation is what makes paging safe | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| An oversized `mrw_read` result carries the first page and a continuation spec | Public contract, additive | MCP callers reading large files |
| The structured content gains a field naming the next spec and what remains | Public contract, additive | Any host reading `structuredContent` |
| `isError` stays true on an oversized read | Unchanged | — |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the first-page result and its continuation field | T1 | T2 | No — new |
| the taught form of the continuation on the MCP surface | T2 | — | No — additive |

## Implementation

Two tasks: T1 makes the refusal a first page and proves it by paging to completion; T2 teaches it,
after it exists, for the reason ADR-012 and ADR-013 both learned the hard way.

## Consequences

- **Positive:** a large file becomes readable over MCP without the caller doing arithmetic, and each
  page is ledger-licensed for exactly the lines it served.
- **Positive:** the caller keeps the choice about spending context, which is the property that makes
  this different from truncation.
- **Negative:** more round trips for a whole large file, by design.
- **Negative:** a caller can still stop after page one believing it has the file. Mitigated by
  labelling and by `isError`, not eliminated — and worth stating plainly rather than claiming solved.
- **Neutral:** the cap's value stays where it is, unjustified, and now says so in its own docs.

## Out of Scope

- Changing `MaxResultChars` (deferred: needs the quality curve, which needs the benchmark harness)
- Measuring whether served size degrades edit accuracy (deferred: docs/adr/BACKLOG.md — it is the harness's question, and the most paper-worthy one this project has)
- Paging the WRITE path (permanent: boundary: a plan is authored by the caller and bounded by what it chose to say)
- Streaming a result across several messages (permanent: boundary: MCP is one call, one message)
- Any CLI behaviour change (permanent: boundary: ADR-010's go/no-go — the CLI streams and stays unbounded)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A continuation spec points at the wrong lines and pages silently skip content | Med | **High** | The Enforced-by pages a file to completion and asserts the concatenation equals the whole file — a presence check would not catch it |
| A page licenses more than it served | Low | **Critical** | Spans are recorded per page by the existing `seen` mechanism; a test drives page one then attempts a page-two write and expects refusal |
| A caller reads page one and believes it has the file | Med | Med | `isError` stays true and the page says what remains; not fully solvable from this side |
| The cap's value is read as endorsed because this record touches the area | Med | Low | Decision 4 and the Consequences say explicitly that it is unmeasured |

## Rollback

Revert the commits. The refusal returns to prose, which every current caller already handles; the
added field is additive and nothing depends on it.

## Follow-ups

- [ ] When the benchmark harness exists, measure edit accuracy against served bytes and give
      `MaxResultChars` a justified value — or record that the curve is flat and the constant may stay
      arbitrary, which is also an answer
