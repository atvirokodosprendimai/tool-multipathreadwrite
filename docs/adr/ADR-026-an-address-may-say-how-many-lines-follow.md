# ADR-026: An address may say how many lines follow

**Status:** Accepted
**Date:** 2026-09-06
**Owner:** M
**Accepted:** M, 2026-09-06, choosing "Implement `,+N` as relative" over "Refuse `+N`" when the two were put side by side with the measured receipt: *"$ mrw read 'f.txt:/alpha/,+3' → @@ 1-4"*, which is the preview M selected and is therefore the specification of what the form means.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-013-a-plan-addresses-what-it-can-find.md`, `docs/adr/ADR-015-a-refusal-names-the-fix-for-the-two-mistakes-the-syntax-invites.md`, `docs/adr/ADR-006-the-root-confines-reads-too-and-a-replace-must-replace-something.md`
**Governs:** `internal/addr/**`, `internal/read/read.go`, `internal/plan/plan.go`, `internal/apply/apply.go`, `internal/curve/score.go`
**Enforced-by:** `internal/read/read_test.go::TestARelativeEndServesTheLinesAfterTheStart`
**Invalidates:** none — checked
**Served-path change:** `mrw read 'f.go:/func Start/,+20'` serves the matching line and the twenty lines after it; today it serves the matching line and line 20.

## Context

Measured 2026-09-06 against `main` `bab2128`, on a five-line fixture:

    $ mrw read 'f.txt:/alpha/,+3'
    ==> f.txt  5L  31B  sha 31d0cdeb
    @@ 1-1
        1| alpha
    @@ 3-3
        3| gamma
    EXIT=0

The caller asked for the match plus three lines. They were served the match and line 3, at exit 0.
Nothing in that receipt is a lie — mrw prints every range it served — but nothing in it announces
that the address did something other than what the caller wrote, and the `,+N` form is the one a
caller arrives with from `sed` and `ed`.

The cause is two correct mechanisms meeting. A comma already separates several ranges of one file
(`internal/read/read.go:149`, `splitRanges`), with `/a/,/b/` deliberately kept together as one
range. And `strconv.Atoi` (`internal/read/read.go:219`) accepts a leading sign, so `+3` parses as
the absolute line 3 rather than as a bad line number. `f.txt:/alpha/,+3` is therefore a spec with
two addresses in it, and both resolve.

The plan side refuses the same string today — `strconv.Atoi` there is reached through
`ParseAddr` with no comma splitting (`internal/plan/plan.go:496-505`), so `@@ f.txt 5,+3 replace`
fails with `bad line number "5,+3"`. One path silently serves something else; the other refuses.

⚠ **But the BARE form is silently wrong on the plan side too, and that was not known when this
record was drafted.** Measured 2026-09-06 by running §64 against a worktree at `bab2128`:
`@@ a.go +3 replace` is ACCEPTED there and applies at line 3, exit 0 — `ParseAddr` never splits on
a comma, so `+3` reaches `strconv.Atoi` whole and parses as the absolute 3. So the two paths did
not differ in kind, only in which spelling reached the sign-accepting parser. Both are refused now,
in the same words, and §64 asserts that on both.

Reported from a quality-harness session on 2026-09-04 as an open question — *"unclear whether that
form is supported; the docs I had did not say"* — and the docs still do not: `grep -rn ',+[0-9N]'`
over `README.md`, `AGENTS.md`, `docs/adr/` and `internal/read/` returns nothing as of 2026-09-06.

## Existing Primitives Audit

- **`internal/addr` — CREATED BY THIS RECORD, and it is the one new component.** The lexical half
  of the address grammar: recognising `,+N`, validating its digits, checking the base is a single
  start, and the exact wording of every refusal. It did not exist when this record was drafted; the
  first cut duplicated that logic in both parsers with contract §64 as the anti-drift gate, and the
  Codex review of PR #125 found the copies had ALREADY drifted before the branch merged —
  `f.txt:,+3` served lines 1-4 at exit 0 on the read path while the plan path refused the identical
  string. `docs/adr/BACKLOG.md`'s "One address parser, not two" pre-registered the trigger — *worth
  a record if it happens twice* — and this was the second time. Resolution stays split, as ADR-013
  requires: only `internal/apply` knows how long a file is.
- **`splitRanges` (`internal/read/read.go:149`)** — already carries the one precedent for a comma
  that does not separate: `/a/,/b/` is held together by a lookahead for `,/`. A relative end is the
  same shape, one branch over. **Reshaped**, not replaced.
- **`parseRange` (`internal/read/read.go:186`)** — already returns a `Range` with a `Start`/`End`
  pair and already resolves `$` late, at the point the file length is known. A relative end resolves
  in the same place for the same reason. **Reused.**
- **`plan.ParseAddr` (`internal/plan/plan.go:481`)** — already scans a pattern before splitting on
  `-` or `,`, precisely so those characters can appear inside a regex. The relative end is parsed
  after that scan. **Reused.**
- **Clamping** — `2-99` on a five-line file serves `@@ 2-5` at exit 0 (measured 2026-09-06), so a
  numeric end past EOF already clamps rather than refusing. A relative end inherits that rule rather
  than inventing a second one.

## Decision

An address in a read spec or a plan hunk may carry a **relative end**: `A,+N` means the range that
begins at `A` and ends `N` lines after it — `N+1` lines in total, matching `sed`'s `addr,+N`. `A`
may be a line number, `$`, or a `/regexp/`; the relative end is resolved after `A` is, so
`/func Start/,+20` is the match and the twenty lines below it.

Four edges, each following a rule this tree already has rather than a new one:

1. **An end past the last line CLAMPS ON A READ and is REFUSED ON A WRITE.** Not two rules but each
   path's own existing one: `mrw read f.txt:2-99` serves what exists and exits 0, while a plan
   addressed `5-9999` is already refused as out of range. The first cut clamped on both, so
   `@@ f.txt 5,+99 replace` quietly replaced two lines and reported `ok` — a write doing less than
   its address said, which is the failure this tool exists to make visible. Measured and corrected
   after the Codex review of PR #125.
2. **`+0` is refused**, naming the fix, because `0` is already refused as a line number
   (`f.txt:0` → `bad line number "0"`, exit 2) and `,+0` says the same thing as writing `A` alone.
3. **A relative end with nothing before it is refused** — `f.txt:+3` names no start to be relative
   to. The refusal says so and names the two things the caller probably meant (`3`, or `A,+3`),
   which is ADR-015's rule.
4. **`/a/,/b/` is unchanged**, and so is every other comma: a comma not followed by `+` or `/` still
   separates two addresses of the same file.

**What would falsify this:** a caller who wrote `,+N` meaning "and also line N". That reading is
available today and this ADR takes it away. It is judged safe because `+N` is undocumented, is
byte-identical in meaning to `N`, which is what a caller wanting an absolute line writes, and
produces a receipt (`@@ 1-1 @@ 3-3`) that no caller has ever reported as intended — the one report
in the corpus (2026-09-04) is from a caller who wanted the relative reading. If a caller is found
relying on the absolute reading, this decision is wrong and the form should be refused instead,
which was the alternative considered below. That evidence does not exist today and this ADR does not
create a way to gather it; ADR-009 refuses telemetry.

## Alternatives Considered

- **Refuse `+N` as a bad line number.** `f.txt:/alpha/,+3` would exit 2 naming the fix, which is
  ADR-015's own shape and the smaller diff. Rejected by M on 2026-09-06 in favour of implementing
  the form: a refusal teaches the caller that mrw has no relative addressing, and the caller's
  instinct — borrowed from `sed` — is a good one that costs a few lines to honour.
- **Implement it in the read path only.** Half the diff, and the read path is where the silent case
  is. Rejected because the two grammars are documented as one in `AGENTS.md` ("A plan address may be
  a line number, an `N-M` range, `$`, or a pattern"), and a form that works in a read and fails in
  the plan built from that read is a worse trap than the one being fixed.
- **Two copies of the lexer, held together by contract §64.** What the first cut shipped, and it is
  recorded as rejected rather than deleted because the reasoning was not silly: the two grammars are
  parsed in different packages against different types, and a row driving the built binary does check
  both. It failed for a reason worth keeping: **a contract row can only notice a drift after someone
  writes the case that exposes it**, and three such cases (`,+3`, `5-7,+3`, `569BEJNRXghkl5,+3`) were already
  divergent when the row was green. One function cannot drift. The review that found this also
  named the boundary — share the lexing, keep the resolution split — which is what `internal/addr`
  does.
- **`A,~N` and the rest of `sed`'s address arithmetic.** Rejected as unrequested scope; `,+N` is the
  form that was actually reached for.

## Component / Boundary Impact

None — internal to the address parsers. `internal/read` and `internal/plan` each keep one reason to
change (how a read spec is addressed; how a plan hunk is addressed), and neither gains a dependency
on the other. The MCP surface parses the same `specs` and `plan` strings through the same two
functions, so it inherits the form without a change of its own.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read <path>:<addr>` CLI grammar | `A,+N` becomes one range instead of two addresses | `internal/read.ParseSpec` | CLI callers, `mrw_read` MCP tool |
| `@@ <path> <addr> <op>` plan grammar | `A,+N` is accepted where it was `bad line number` | `internal/plan.ParseAddr` | CLI callers, `mrw_write` MCP tool |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `read.parseRange` accepting a `,+N` suffix, and its refusal wording | T1 | T3 | No — additive; the form was previously two addresses |
| `plan.ParseAddr` accepting a `,+N` suffix | T2 | T3 | No — additive; the form was previously an error |

## Implementation

See `docs/adr/ADR-026-an-address-may-say-how-many-lines-follow/tasks/README.md`.

## Consequences

- **Positive:** the form a caller arrives with from `sed` does what it looks like it does, on both
  paths, instead of silently serving a different line on one and failing on the other.
- **Positive:** `mrw read 'f.go:/func Start/,+20'` removes the read-then-count step that a caller
  otherwise pays to turn a match into a range.
- **Negative:** one more address form to document and to keep working in two parsers; the grammar
  gets larger, and every future address change has one more case to consider.
- **Negative:** the absolute reading of `+N` is withdrawn. It was undocumented and nobody is known to
  use it, but the withdrawal is real and cannot be detected from a caller's side except by the
  changed receipt.
- **Neutral:** the MCP surface changes with no code of its own, because it shares both parsers.

## Out of Scope

- A backwards relative start (`A,-N`, or `-N,A`) (deferred: `docs/adr/BACKLOG.md`)
- The rest of `sed`'s address arithmetic — `addr,~N`, `first~step`, `addr1,addr2!` (permanent: boundary: `,+N` is the form that was reached for in the field; a grammar grows one requested form at a time or it becomes a language nobody asked for)
- Relative addressing across files (permanent: boundary: an address is resolved against one file, which is what makes a plan's addresses independent of each other under ADR-001)
- Changing what a bare comma means between two addresses of one file (permanent: fact: `splitRanges` keeps `/a/,/b/` together and separates everything else; citation: file `internal/read/read.go:167`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller relied on `+N` meaning the absolute line N | Low | Low | The form is undocumented and identical in meaning to `N`; the one field report of it (2026-09-04) wanted the relative reading. Named in the Decision as what would falsify this |
| The two parsers drift — one accepts a form the other refuses, which is the trap being fixed | Med | Med | T3's contract row drives the BUILT binary through both paths in one section, so a divergence fails the contract rather than being found by a caller |
| A relative end past EOF is read as an error rather than a clamp, diverging from `2-99` | Low | Med | The clamp is the decided behaviour, is asserted by a test in T1 and by the contract row in T3, and is stated in the Decision beside the precedent it inherits |

## Rollback

Revert the three commits. mrw keeps no persistent state about address forms — the read-before-modify
ledger records resolved line numbers, never the address that produced them (`internal/seen`), so a
ledger written under this ADR stays valid after a revert. A caller's scripts that adopted `,+N` break
on revert with `bad line number`, which is loud.

## Follow-ups

- [ ] If a caller is found relying on `+N` as an absolute line number, this decision is wrong: refuse the form as the rejected alternative describes, rather than adding a compatibility mode.
