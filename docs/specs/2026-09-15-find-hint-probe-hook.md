# Spec: Find, hint, probe, and hook leftovers

> **Date:** 2026-09-15 · **Status:** Implemented
> **Owner:** M · **Becomes:** four destinations — ADR-015 amendment (UC-1), reserved ADR-058 (UC-2), ADR-023 Verification Log plus BACKLOG receipt (UC-3), ADR-022 amendment (UC-4)
> **Gate:** Status may become Ready-for-ADR only after `spec-verify --spec <this file>` exits 0.
> **Cross-references:** `docs/adr/ADR-015-a-refusal-names-the-fix-for-the-two-mistakes-the-syntax-invites.md`, `docs/adr/BACKLOG.md` (ADR-058 reserved; ADR-023 other hosts), `docs/adr/ADR-023-a-reads-answer-is-the-served-text.md`, `docs/adr/ADR-022-a-path-scoped-rule-arrives-on-an-mrw-read-too.md`, `docs/adr/ADR-007-mrw-finds-the-files-it-serves.md`, `docs/adr/ADR-016-the-mcp-surface-says-what-it-is-not.md`, `docs/adr/ADR-048-mrw-models-no-target-syntax.md`, `docs/curve/reading-12-void.md`

## Problem

Four independently shippable leftovers sit after the 2026-09-15 inbox sweep, and none of them is an engine rewrite. An unquoted regex address that the shell splits is reported as N missing files (`hintUnexpandedGlob` fires only on `*?[`). Structural find is reserved as ADR-058 and not executed. ADR-023's envelope is measured on Claude Code 2.1.261 only. `.claude/hooks/rules-on-read.py` has no subprocess wall-clock timeout even though its matcher was rewritten after a 2 s alarm, and `main` always exits 0.

## Goal

One spec, four use cases: a split-regex UNREADABLE receipt names quoting; `mrw read --ast-grep` finds by structure through existing `read`; a Desktop envelope probe recipe is filed on ADR-023 and an unrun probe is not a miss rate; the PostToolUse hook returns within 2 s even if matching hangs, still at exit 0.

## Actors

| Actor | Kind | Goal |
|-------|------|------|
| CLI/MCP reader | human role | A refusal names the fix; find by structure without a write-time parser |
| Operator | human role | Know whether Desktop drops `content[]` when `structuredContent` is present |
| Scheduled hook | scheduled job | A pathological PostToolUse child cannot run for hours after the turn |

## Use Cases

### UC-1: CLI/MCP reader gets a quoting hint on shell-split UNREADABLE paths

- **Trigger:** one `mrw read` (CLI or MCP `specs`) reports two or more UNREADABLE paths · **Preconditions:** those paths contain no `*?[` and each matches `/^[A-Za-z][A-Za-z0-9_-]*$/`
- **Main flow:**
  1. Reader supplies several consecutive English-word fragments (the shell-split class).
  2. Each fragment is reported UNREADABLE as today.
  3. One extra hint names a quoted regex address and `--grep`.
- **Failure paths:** a. one missing `nope.go` → no extra hint. b. a path holding `*?[` → existing glob hint, not the English-word one.
- **Postconditions:** glob hint and ordinary missing-file silence stay; the new hint is additive text on paths that already failed.

### UC-2: CLI/MCP reader finds by structure without a write-time parser

- **Trigger:** `mrw read --ast-grep PATTERN` (MCP `ast_grep`) with optional paths · **Preconditions:** none; the binary may be absent
- **Main flow:**
  1. Reader asks for structural find on the read path only.
  2. mrw shells out to `ast-grep` on PATH, maps hits to line ranges, serves through existing `read.Run`.
  3. The ledger records served lines only, not AST nodes.
- **Failure paths:** a. `ast-grep` not on PATH → exit 2, reason names `ast-grep`. b. `--grep` and `--ast-grep` together → usage (two sources). c. present binary, zero hits → not the missing-binary path (names the pattern, nonzero, like `--grep`).
- **Postconditions:** `--grep` stays regex; apply/plan/seen/check/state are untouched; disagreement between the two finders is the feature.

### UC-3: Operator knows the Desktop envelope probe recipe and that unrun is not a miss

- **Trigger:** a Desktop or third-party session is at hand, or a reader asks whether Desktop was measured · **Preconditions:** ADR-023 Decision 1 already ships the bare envelope
- **Main flow:**
  1. Operator follows the recipe on ADR-023's Verification Log: one Desktop `mrw_read` of a two-line fixture.
  2. Log whether the model quoted the served line or only the receipt.
- **Failure paths:** a. probe not run → BACKLOG "ADR-023: other hosts" stays open; must not be treated as a miss rate.
- **Postconditions:** no engine ADR unless the probe finds a defect; Reading 12 stays VOID.

### UC-4: Scheduled hook returns within a wall-clock bound

- **Trigger:** PostToolUse child of `.claude/hooks/rules-on-read.py` · **Preconditions:** hook is wired as today (`main` always exits 0)
- **Main flow:**
  1. Hook runs matching as today.
  2. Even if matching is replaced with a catastrophic regex, `main` returns within 2 s.
- **Failure paths:** a. the bound fires → still exit 0 (a hook must never take the turn down).
- **Postconditions:** matcher `seg_match` stays; 2 s is the bound already named in that comment as the alarm that motivated it.

## Scenarios

### UC1-S1 [happy] Several English-word UNREADABLE paths name quoting [@implemented] → `internal/read/read_test.go::TestSeveralEnglishWordUnreadablePathsNameQuoting` cmd:`go test ./internal/read/ -count=1 -run '^TestSeveralEnglishWordUnreadablePathsNameQuoting$'`

```gherkin
Given a root with no files named rules, that, or will
When mrw read is invoked with those three paths in one call
Then each is UNREADABLE
And the report names quoting or --grep once as the fix
```

### UC1-S2 [failure] One missing nope.go stays silent [@implemented] → `internal/read/read_test.go::TestAnOrdinaryMissingFileGetsNoEnglishWordHint` cmd:`go test ./internal/read/ -count=1 -run '^TestAnOrdinaryMissingFileGetsNoEnglishWordHint$'`

```gherkin
Given a root with no nope.go
When mrw read is invoked with only nope.go
Then the report is UNREADABLE
And it does not mention quoting, glob, or --grep as a hint
```

### UC1-S3 [failure] A glob path keeps the glob hint [@implemented] → `internal/read/read_test.go::TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint` cmd:`go test ./internal/read/ -count=1 -run '^TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint$'`

```gherkin
Given a path holding * ? or [
When that path is UNREADABLE in a read that also has English-word misses, or alone
Then the glob hint still names an unexpanded glob and --grep
And it is not replaced by the English-word wording
```

### UC2-S1 [happy] ast-grep hits serve through read.Run [@implemented] → `cmd/mrw/astgrep_test.go::TestAstGrepServesRangesThroughRead` cmd:`go test ./cmd/mrw/ -count=1 -run '^TestAstGrepServesRangesThroughRead$'`

```gherkin
Given ast-grep on PATH (or a fake that prints one hit)
When mrw read --ast-grep PATTERN walks like --grep
Then matching ranges are served through read.Run
And the ledger records those served lines only
```

### UC2-S2 [failure] Missing ast-grep is exit 2 and names the binary [@implemented] → `cmd/mrw/astgrep_test.go::TestMissingAstGrepIsUsageAndNamesTheBinary` cmd:`go test ./cmd/mrw/ -count=1 -run '^TestMissingAstGrepIsUsageAndNamesTheBinary$'`

```gherkin
Given ast-grep is not on PATH
When mrw read --ast-grep PATTERN
Then the process exits 2
And the reason names ast-grep
```

### UC2-S3 [failure] grep and ast-grep together are usage [@implemented] → `cmd/mrw/astgrep_test.go::TestGrepAndAstGrepTogetherAreUsage` cmd:`go test ./cmd/mrw/ -count=1 -run '^TestGrepAndAstGrepTogetherAreUsage$'`

```gherkin
Given both --grep and --ast-grep are supplied
When mrw read runs
Then the process exits 2
And nothing is walked or served
```

### UC3-S1 [happy] ADR-023 Verification Log carries the Desktop probe recipe [@implemented] → `internal/adversarial/desktop_probe_test.go::TestTheDesktopEnvelopeProbeRecipeIsFiled` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestTheDesktopEnvelopeProbeRecipeIsFiled$'`

```gherkin
Given ADR-023 as the envelope record
When a reader looks for how to probe Desktop
Then a Verification Log recipe names one Desktop mrw_read of a two-line fixture
And it says to record whether the model quoted the served line or only the receipt
```

### UC3-S2 [failure] An unrun Desktop probe is not a miss rate [@implemented] → `internal/adversarial/desktop_probe_test.go::TestAnUnrunDesktopProbeIsNotAMissRate` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunDesktopProbeIsNotAMissRate$'`

```gherkin
Given the probe has not been run
When a reader scores hosts
Then BACKLOG still says the other-hosts entry is not blocking
And nothing treats the absent observation as a miss rate
```

### UC4-S1 [happy] The hook returns within two seconds under a hanging matcher [@implemented] → `cmd/mrw/ruleshook_test.go::TestTheRulesHookReturnsWithinTheWallClockBound` cmd:`go test ./cmd/mrw/ -count=1 -run '^TestTheRulesHookReturnsWithinTheWallClockBound$'`

```gherkin
Given the wired rules-on-read.py
When matching is replaced with a catastrophic regex
Then main returns within 2 seconds
```

### UC4-S2 [failure] A fired bound still exits 0 [@implemented] → `cmd/mrw/ruleshook_test.go::TestTheRulesHookStillExitsZeroAfterTimeout` cmd:`go test ./cmd/mrw/ -count=1 -run '^TestTheRulesHookStillExitsZeroAfterTimeout$'`

```gherkin
Given the wall-clock bound fires
When the hook process ends
Then the exit status is 0
```

## Facts

| ID | Assertion (invariant / behavior) | Test (`path::name`) | Tag | Cmd (optional) |
|----|----------------------------------|---------------------|-----|----------------|
| F-1 | An UNREADABLE path that holds `*?[` names glob / `--grep`; a path with no metacharacter gets no glob hint | `internal/read/read_test.go::TestAGlobUnreadablePathNamesTheGlobEscape` | @implemented | go test ./internal/read/ -count=1 -run '^TestAGlobUnreadablePathNamesTheGlobEscape$' |
| F-2 | The ungoverned class is several UNREADABLE fragments that look like English words, produced when the shell splits an unquoted regex address; mrw sees arguments, not the shell (ADR-015 D4 ethos) | `internal/read/read_test.go::TestSeveralEnglishWordUnreadablePathsNameQuoting` | @implemented | go test ./internal/read/ -count=1 -run '^TestSeveralEnglishWordUnreadablePathsNameQuoting$' |
| F-4 | `--grep` is regex `read.Walk` then `read.Run`; MCP `grep` calls the same `Walk` via `grepSpecs`; apply does not grep | `internal/read/walk_test.go::TestWalkReturnsOneSpecPerMatchingFile` | @implemented | go test ./internal/read/ -count=1 -run '^TestWalkReturnsOneSpecPerMatchingFile$' |
| F-5 | Structural find is a flag beside `--grep` named `--ast-grep` (MCP `ast_grep`), READ path only; it shells out to the `ast-grep` CLI if present, maps hits to line ranges, and serves through existing `read`; `--grep` stays regex | `cmd/mrw/astgrep_test.go::TestAstGrepServesRangesThroughRead` | @implemented | go test ./cmd/mrw/ -count=1 -run '^TestAstGrepServesRangesThroughRead$' |
| F-6 | ADR-023 envelope is measured on Claude Code 2.1.261 only; Desktop / other hosts are one probe when a session is at hand, recorded on ADR-023's Verification Log; not blocking; Reading 12 is VOID | `internal/adversarial/desktop_probe_test.go::TestTheDesktopEnvelopeProbeRecipeIsFiled` | @implemented | go test ./internal/adversarial/ -count=1 -run '^TestTheDesktopEnvelopeProbeRecipeIsFiled$' |
| F-7 | `.claude/hooks/rules-on-read.py` `main` returns within 2 s even if matching hangs, and still exits 0 | `cmd/mrw/ruleshook_test.go::TestTheRulesHookReturnsWithinTheWallClockBound` | @implemented | go test ./cmd/mrw/ -count=1 -run '^TestTheRulesHookReturnsWithinTheWallClockBound$' |
| F-10 | The English-word hint fires only when one `read` invocation reports two or more UNREADABLE paths that contain no `*?[` and match `/^[A-Za-z][A-Za-z0-9_-]*$/`; a single missing `nope.go` stays silent | `internal/read/read_test.go::TestAnOrdinaryMissingFileGetsNoEnglishWordHint` | @implemented | go test ./internal/read/ -count=1 -run '^TestAnOrdinaryMissingFileGetsNoEnglishWordHint$' |
| F-11 | MCP `ast_grep` is the same primitive as CLI `--ast-grep` (ADR-016: surfaces must not disagree) | `internal/mcp/astgrep_test.go::TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli` | @implemented | go test ./internal/mcp/ -count=1 -run '^TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli$' |
| F-12 | Missing `ast-grep` binary is exit 2 and names `ast-grep`; a present binary with zero hits is not that path | `cmd/mrw/astgrep_test.go::TestMissingAstGrepIsUsageAndNamesTheBinary` | @implemented | go test ./cmd/mrw/ -count=1 -run '^TestMissingAstGrepIsUsageAndNamesTheBinary$' |
| F-13 | `--grep` and `--ast-grep` together (MCP `grep` and `ast_grep` together) are usage, two sources of specs | `cmd/mrw/astgrep_test.go::TestGrepAndAstGrepTogetherAreUsage` | @implemented | go test ./cmd/mrw/ -count=1 -run '^TestGrepAndAstGrepTogetherAreUsage$' |
| F-14 | A glob `*?[` UNREADABLE path keeps the existing glob hint; it is not rewritten as the English-word hint | `internal/read/read_test.go::TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint` | @implemented | go test ./internal/read/ -count=1 -run '^TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint$' |
| F-15 | License for an `--ast-grep` serve is served lines, not AST nodes; the walk observes nothing (ADR-007) | `cmd/mrw/astgrep_test.go::TestAstGrepObservesOnlyServedLines` | @implemented | go test ./cmd/mrw/ -count=1 -run '^TestAstGrepObservesOnlyServedLines$' |
| F-16 | An unrun Desktop probe must not be treated as a miss rate; BACKLOG "ADR-023: other hosts" stays open until observed | `internal/adversarial/desktop_probe_test.go::TestAnUnrunDesktopProbeIsNotAMissRate` | @implemented | go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunDesktopProbeIsNotAMissRate$' |

## Domain

One milestone spec fans to four destinations. UC-1 extends ADR-015's refusal-names-the-fix class (argument shape, not shell detection). UC-2 executes reserved ADR-058 on the read path only; ADR-048 still forbids a write-time parser. UC-3 is a human-observed probe recipe, not an engine change. UC-4 extends ADR-022's existing matcher bound to any hang. Contract rows start at §106. Highest shipped ADR file is 060; 058 stays reserved for structural find.

## Contracts Touched

| Surface | Change | Consumers |
|---------|--------|-----------|
| `mrw read` UNREADABLE report | Additive English-word hint when F-10 holds | CLI and MCP readers of UNREADABLE lines |
| `mrw read --ast-grep` / MCP `ast_grep` | New find flag beside `--grep` / `grep` | CLI and MCP readers; not apply |
| ADR-023 Verification Log | Desktop probe recipe; BACKLOG stays not-blocking until observed | Operators; no engine caller |
| `.claude/hooks/rules-on-read.py` | 2 s wall-clock bound, still exit 0 | Claude Code PostToolUse in this repo |
| `scripts/contract.sh` | Rows from §106 | CI Linux contract |

## Non-Goals

- Write-time / apply parser (ADR-048) — permanent: boundary: mrw models no target syntax
- Replacing regex `--grep`; bundling `ast-grep` — permanent: boundary: disagreement is the feature; missing binary is exit 2
- Neighbour licence on single-line; `--strict-balance` as default; `body=@` inside `Parse`; widen prose — permanent: boundary: already refused
- Retag `v1.20.0` / `v1.20.1` — permanent: boundary: release already shipped
- Delete `// */` above `TestReadHintsAMultiLineReplaceNeedsTheNextLine` — permanent: fact: first-red-lock hasher UNPROVEN workaround
- Redo Reading 12 (VOID) — permanent: fact: `docs/curve/reading-12-void.md`
- `,+N` after a regex — already shipped as ADR-026; out of this spec
- Engine go/no-go: UC-1 may touch `internal/read/read.go`; UC-2 may touch walk/CLI/MCP **read** only; apply/plan/seen/check/state stay byte-identical unless a later ADR owns them
- Taking ADR-058 for the hint or the hook — 058 is reserved for structural find
- An engine ADR for UC-3 unless the probe finds a defect

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| English-word hint fires on two genuinely missing files named like tokens | Med | Low | F-10 requires two or more in one invocation and the token regex; F-1 silence on `nope.go` stays |
| Fake or hostile `ast-grep` on PATH | Low | Med | Serve only line ranges through existing `read`; license is served lines; missing binary is explicit |
| Hook 2 s bound truncates a slow legitimate delivery | Low | Med | Matcher is already linear `seg_match`; 2 s is the alarm that motivated it; bound still exit 0 |
| Unrun Desktop probe scored as coverage | Med | High — a hole reads as measured | F-16; BACKLOG not-blocking; human-observed log row, not a Go miss-rate |

## Open Questions

<!-- empty: implement-the-plan accepted remaining grill items from BACKLOG; leftover numbering is Grill Log non-behavioral -->

## Verify

```bash
spec-verify --implemented docs/specs/2026-09-15-find-hint-probe-hook.md
```

## Stress suite

Added after execute, 2026-09-15. Oracles from this spec (F-2, F-10, F-12, F-14, ADR-058 D1–D2, UC-4), never from the implementation. Named attacks are the shapes a shell, a hostile PATH binary, or a hanging matcher would use to make a green look like coverage.

- `internal/read/leftovers_stress_test.go` — F-10 token iff the spec regex; 400 random argv mixes against an independent English-count; JSON mapper never panics; 0-based line 0 serves line 1 and line 1 serves line 2 (the +1 drop survived the line-0 fixture because of the `< 1` clamp); `{}` is not zero hits; a `../` hit is a Problem; Walk's source still names no ast-grep.
- `internal/adversarial/leftovers_stress_test.go` — built-binary matrix over the two-source table (`rg 'two sources of specs|two answers|exclude without'`). A hanging `ast-grep` on PATH is not this record's promise.
- `cmd/mrw/leftovers_stress_test.go` — `--files-from`+`--ast-grep`; `--exclude` with `--ast-grep`; range+finder; exit 1 plus `[]`; alarm armed before `run`; hang in `seg_match` still exits 0.
- `internal/mcp/leftovers_stress_test.go` — MCP `specs` of English-word fragments; range+`ast_grep`; instructions name `ast_grep` not `--ast-grep`.

Arm 3 does not bound a hanging `ast-grep` subprocess. Encoding that gap as a pass would go red if anyone added a timeout.

## Grill Log (appendix)

| # | Question | Fact | Decision |
|---|----------|------|----------|
| 1 | One spec or four? Which path? | non-behavioral | One spec at `docs/specs/2026-09-15-find-hint-probe-hook.md` fans to four destinations; this repo had no `docs/specs/` yet |
| 2 | Scouted F-1 glob hint already shipped? | F-1 | Accept — `hintUnexpandedGlob` and the two existing tests stay |
| 3 | Ungoverned class is shell-split English-word UNREADABLE fragments? | F-2 | Accept — argument shape, not shell detection (ADR-015 D4) |
| 4 | Is `,+N` after regex in this spec? | non-behavioral | Reject as Fact — shipped ADR-026; Non-Goal |
| 5 | Is `--grep` regex Walk then Run, MCP same Walk, apply does not grep? | F-4 | Accept |
| 6 | Flag name `--ast-grep` vs a mode on `--grep`? | F-5 | Accept `--ast-grep` beside `--grep`; MCP `ast_grep`; `--grep` stays regex |
| 7 | Desktop probe in-spec Fact vs Non-Goal of the engine ADRs? | F-6 | In-spec human-observed recipe on ADR-023 Verification Log; no engine ADR unless a defect |
| 8 | Is UC-4 still in given `seg_match`? Bound seconds? | F-7 | Still in — remaining class is any hang; 2 s from the matcher comment |
| 9 | English-word hint trigger? | F-10 | Accept recommended: two or more UNREADABLE paths, no `*?[`, `/^[A-Za-z][A-Za-z0-9_-]*$/`; single `nope.go` silent |
| 10 | MCP argument shape? | F-11 | Same primitive as CLI; ADR-016 same sentence |
| 11 | Missing binary vs empty hits? | F-12 | Missing: exit 2 names `ast-grep`. Empty hits: not that path |
| 12 | Both find flags at once? | F-13 | Usage, two sources of specs |
| 13 | Glob path vs English-word hint? | F-14 | Glob hint kept; not rewritten |
| 14 | What does ast-grep license? | F-15 | Served lines only; walk observes nothing |
| 15 | Unrun Desktop probe as miss rate? | F-16 | Must not; BACKLOG stays open |
| 16 | Next contract section / ADR-058 reservation? | non-behavioral | Next free contract §106; 058 reserved for structural find; do not take 058 for hint or hook |
