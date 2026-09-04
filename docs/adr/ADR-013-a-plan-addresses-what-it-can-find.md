# ADR-013: A plan addresses what it can find, and refuses what it finds twice

**Status:** Accepted
**Accepted:** 2026-09-04 by M — *"so you listed 6 defects, I think it is wise now to adress these first, then focus on paper and scientific benchmarking. those 6 defects are real blockers."* This record is the first of them. Quoted rather than inferred, because ADR-011 had to add its acceptance sentence retroactively when a reviewer asked whose decision the status was.
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (owns the plan format this extends, and the original-file rule this must not break), ADR-002 (owns the read-before-write ledger this must not bypass), ADR-007 (owns the regex address `read` already has), ADR-006 (the root boundary, untouched)
**Governs:** `internal/plan/**`, `internal/apply/**`
**Enforced-by:** `internal/apply/apply_test.go::TestAnAmbiguousRegexAddressIsRefused`
**Invalidates:** ADR-012-T1-S2 in one clause, named rather than glossed. Its `instructions` text taught *"a regexp works in a read, never here"*, and its Mutation Log records a killed mutant that gave the published example a regexp address — *"the exact authoring mistake its own description warns about."* T3 rewrites that prose and **that mutant would now survive**; the ADR-012 entry is left standing beside this note rather than edited, because it was true when it was written. Nothing else: ADR-001 says a plan's addresses resolve against the ORIGINAL file and that is preserved exactly; it does not say addresses must be line numbers. ADR-002's guard is strengthened by construction rather than weakened — see Decision 3.
**Served-path change:** a plan can say `@@ internal/store/store.go /^func \(s \*Store\) Get/,/^\}/ replace` instead of naming line numbers a caller had to read the file to learn. The escaping is not decoration — RE2 reads an unescaped `(` as a group and an unescaped `*` as a quantifier, so the unescaped form matches nothing. Both taught examples shipped unescaped and were caught in review of PR #74.

## Context

**`read` can find a site; a plan cannot.** `mrw read 'f.go:/^func Apply/'` locates its own line. A
plan takes `N`, `N-M`, `N-`, or `$` and nothing else, so every edit whose site is not already known
becomes two phases: read to harvest line numbers, then emit a plan quoting them. AGENTS.md documents
that pipeline as the lever — *"take the numbers from the read you just did"* — and it is real, but it
is a workaround for an asymmetry, not a feature.

**The cost, measured 2026-09-04 against this repository at `14d28b3`.** The task is the one AGENTS.md
advertises: one mechanical edit at every site, where the new content does not depend on what is
there. 43 Go files, insert a line before each `^package`:

| Path | Calls | Bytes |
|---|---|---|
| Harvest read (`f.go:/^package/` × 43) then a line-addressed plan | 2 | 3,806 + 2,635 = **6,441** |
| A regex-addressed plan, no harvest read | 1 | **3,010** |

One round trip instead of two, and 53% of the bytes. Note what the table also shows: **the plan
itself gets bigger** — 3,010 against 2,635 — because a regex is longer than a line number. The saving
is entirely the read that no longer happens, and any claim that this makes plans cheaper is false.

**And the saving does not generalise, which is the part worth writing down.** For an edit whose new
content depends on the current text — the ordinary case, and every edit this ADR's own author made
today — the caller must see the lines regardless, so the harvest read happens anyway and regex
addressing saves the arithmetic and nothing else. An earlier framing in this session claimed the
result-size problem was "mostly self-inflicted" by this one; the measurement above does not support
that, and the result-size problem is left where it is, as its own defect with its own record.

**What line numbers buy, and must not be given up.** A line number is unambiguous. A regex is not:
it can match zero times, or four. That is precisely the silent-wrongness class this project exists to
refuse, and it is the reason a plan takes numbers today. Any regex address that resolves a
multi-match by picking one has made mrw into the tool ADR-001 was written against.

## Existing Primitives Audit

- **`internal/read`'s address parser (ADR-007):** the regex address already exists, is tested, and
  its syntax is what a caller has learned. **Reused as the SYNTAX, not as the code** — `read`
  resolves an address against a file it is about to render, `plan`/`apply` resolve against a file
  about to be edited under a ledger check, and the ambiguity rule below is deliberately stricter than
  `read`'s. Sharing the resolver would force one of the two to accept the other's rule.
- **`plan.ParseAddr` (`internal/plan/plan.go:377`):** parses `N`, `N-M`, `N-`, `$`. **Extended** with
  the two pattern forms; every existing form parses to exactly what it parses to today.
- **`apply`'s resolution loop (`internal/apply/apply.go:513-700`):** already resolves EOF sentinels
  into concrete lines before anything else happens, and already reports a failure per hunk with the
  caller's own wording echoed back. **Reused unchanged in shape** — a pattern is one more thing to
  resolve in the place `$` is resolved.
- **`covered()` (`internal/apply/apply.go:499-511`):** the per-line ledger check. **Reused unchanged,
  and it is what makes this safe** — see Decision 3.
- **A Go AST / tree-sitter / language server:** audited and **NOT taken.** See Alternatives; this is
  the obvious "better" answer and it is refused on a dependency and a scope argument, not overlooked.

## Decision

**1. A plan address may be a pattern.** Two forms, matching what `read` already accepts:

    @@ f.go /^func Apply/ insert-before
    @@ f.go /^func Apply/,/^}/ replace

A pattern is a Go regular expression matched against a whole line, unanchored unless the caller
anchors it. `N`, `N-M`, `N-` and `$` keep their current meanings exactly.

**2. The START is exactly-once; the END is a delimiter.** A start pattern must match **exactly one**
line in the file. Zero matches fails the hunk naming the pattern; two or more fails it naming the
pattern **and the line numbers it matched**, so the caller can narrow it or address by number. There
is no "first match" for a start, no "nearest to a hint", no `occurrence=2` selector — every one of
those turns a refusal into a wrong edit, and a plan that edits the wrong function is the failure this
project's whole ledger exists to prevent.

**Amended during execution, 2026-09-04 (review of PR #74).** This first said *each endpoint* must
match exactly once, and that is wrong in a way that made the record's own headline example fail. An
end pattern is not a second site: it is a delimiter relative to a start that has already been pinned
to one line, so the end resolves to the **first match at or after the start** — what `ed`, `sed` and
mrw's own `read` mean by `/a/,/b/`. Under the original rule `/^}/` — the natural end of any function
body, and the end this record's Served-path line teaches — was ambiguous in every file holding two
functions, including the fixture the record itself defines. The refusal principle is untouched
because it was always about *which site do you mean*, and the start alone asks that; once the start
is unique, "the first `^}` after it" is one deterministic line rather than a choice among
candidates. An end that matches only ABOVE the start is still refused, because it delimits nothing.

**3. A pattern resolves against the ORIGINAL file, and then faces the ledger unchanged.** ADR-001's
rule is preserved: resolution reads the file as it was before any hunk applied, so patterns and line
numbers in one plan cannot disagree about what they address. ADR-002's rule is preserved by
ORDERING, and this is the part that had to be checked rather than assumed: `covered()` is called with
the **resolved** span inside the resolution loop, so a pattern resolves to a line first and that line
then meets the same per-line coverage check a typed number would. A pattern naming a line the caller
never read is refused exactly as `40-58` would be. **A regex address is not a way to edit a file you
have not seen.**

**Go/no-go, checked during execution and recorded in the task verification logs:**

- **No new dependency.** `go.mod` still declares exactly one requirement; `regexp` is stdlib and
  `internal/read` already uses it.
- **Every existing plan still parses to the same hunks.** A corpus of the plans in `contract.sh` and
  in the ADR examples parses identically before and after.
- **The ambiguity refusal is proved on a fixture the plan did not build.** The two-match case needs a
  file authored independently of the plan that addresses it — ADR-012 shipped a mutant that survived
  because a fixture built from the plan cannot falsify the plan's guard, and this is the same trap.
- **`gofmt -l .` is empty and `go vet ./...` passes**, in the fence and not only in CI. ADR-012's
  fences omitted both and a formatting failure reached CI twice in one day.

## Alternatives Considered

- **Leave it: read, then quote line numbers.** Today's answer, and it works. Rejected on the measured
  round trip above for the content-independent class, and on the authoring burden for every class —
  the caller must parse rendered output to recover numbers, which is a step where a model can quietly
  be wrong and nothing checks it.
- **Pick the first match on ambiguity.** The obvious ergonomic choice, and it is the one thing this
  record most wants to refuse. A plan that silently edits the first of four matches is a silent wrong
  answer with a receipt saying `ok`.
- **An `occurrence=N` guard to disambiguate.** Rejected for now: it makes a plan depend on the ORDER
  of matches in a file, which changes when unrelated code moves. The refusal message names the
  matched line numbers, so a caller that hits ambiguity has the numbers to address by.
- **Symbol or AST addressing — `@@ f.go @func:Apply replace`.** The genuinely better answer for code,
  and **rejected here rather than deferred vaguely.** It needs a parser per language: Go's is in the
  standard library, nothing else's is, and mrw is language-agnostic by construction — it edits YAML,
  Markdown and Dockerfiles in this repository's own tests. Taking it means either a tree-sitter
  dependency (refused by ADR-004 and ADR-010's arithmetic, and it is cgo, which forfeits the single
  static binary and the Windows job) or a Go-only feature in a tool that does not otherwise know what
  Go is. A regex over lines is the language-agnostic 80%.
- **Make `read` and `apply` share one address resolver.** Attractive on DRY grounds and rejected on
  the rule: `read` serving two matches is helpful, `apply` editing two matches is a bug. One resolver
  would have to be parameterised by that difference, which is the whole of the logic.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/plan` | The plan grammar, now including two pattern address forms. Still knows text, not files | Yes — changes when the grammar changes |
| `internal/apply` | Resolution and application, now resolving a pattern where it resolves `$`. Still the only thing that knows how long a file is | Yes |
| `internal/read` | Untouched. Its regex address keeps its own, looser rule | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| `plan.Addr` gains a pattern form | Internal contract, additive | `internal/apply` |
| A plan may carry `/re/` and `/re/,/re/` addresses | Public contract, additive | Every caller; no existing plan changes meaning |
| A hunk may fail with an ambiguity reason | Public contract, additive | Anyone reading `hunks[].reason` |
| `mrw_write`'s `plan` description and the MCP `instructions` teach the new form | Public contract, content | Any MCP host |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| `plan.Addr` carrying a compiled pattern | T1 | T2 | No — new field |
| the resolution-and-refusal behaviour in `apply` | T2 | T3 | No — new |
| the taught form on the MCP surface and in AGENTS.md | T3 | — | No — additive |

## Implementation

Three tasks. T1 parses and does not resolve; T2 resolves and refuses; T3 teaches it. T2 after T1
because a grammar that parses into nothing is dead code, and T3 last because ADR-012's whole finding
is that a surface teaching a format it does not have is worse than a surface teaching nothing.

## Consequences

- **Positive:** one round trip instead of two for the content-independent multi-site edit, measured
  at 43 sites; and no line-number arithmetic in any plan, which is the step a caller does by hand.
- **Positive:** the failure mode of a moved line changes from "edits the wrong lines" to "matches
  nothing and is refused", because a pattern describes what it wants and a number does not.
- **Negative:** plans get bigger — 14% on the measured case. Bytes are not what this buys.
- **Negative:** a second way to say the same thing. Two callers reading one plan may now see an
  address they cannot compare by eye to another hunk's.
- **Negative, and the one to watch:** a caller may believe a pattern licenses editing an unread file.
  It does not, and the refusal says so, but the belief is the risk this record is most likely to be
  wrong about.
- **Neutral:** `read`'s address rule and a plan's now differ in strictness. Deliberate, documented in
  the audit, and asserted in T2.

## Out of Scope

- Symbol / AST addressing (permanent: boundary: needs a parser per language; see Alternatives)
- `occurrence=N` or any positional disambiguator (deferred: docs/adr/BACKLOG.md — revisit only if the ambiguity refusal proves common in practice, which nothing currently measures)
- Changing `read`'s address rule to match this one (permanent: boundary: serving two matches is useful, editing two is a bug)
- The MCP result-size ceiling (permanent: boundary: a separate defect with its own record; the measurement in Context is why they were separated)
- Any change to what a plan does once resolved (permanent: boundary: ADR-001 owns application; this record only changes how a hunk finds its lines)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A pattern becomes a ledger bypass | Low | **Critical** | Resolution happens before `covered()`, which takes the RESOLVED span; T2's Enforced-by drives a pattern against an unread file and asserts the refusal |
| Ambiguity resolved silently by a later change | Low | **Critical** | The refusal is the Enforced-by test, and its fixture is authored independently of the plan — ADR-012's surviving mutant is the precedent for why that matters |
| A catastrophic regex costs more than the edit | Low | Med | Go's `regexp` is RE2: linear time, no backtracking. Named because the reflex is to worry about it |
| Callers use a pattern where a number is clearer | Med | Low | Both stay legal; the guidance keeps numbers as the default for a site you already read |

## Rollback

Revert the commits. Every existing plan still parses and applies identically, because the pattern
forms are additive syntax that no current plan uses. No state format moves, no ledger entry changes
meaning.

## Follow-ups

- [ ] If the ambiguity refusal turns out to be common, revisit whether the message should suggest the
      narrowed pattern rather than only the matched line numbers
- [ ] **Publish a pattern-addressed example on the MCP surface.** T3 teaches the form in prose but
      every shipped example is still line-addressed, so ADR-012's Enforced-by never dry-runs a
      pattern. Blocked on `treeFor`, which builds its fixture from `Addr.Start`/`End` — zero until
      `apply` resolves a pattern — so the example cannot be executed by the existing harness as it
      stands. Named in review of PR #74.
