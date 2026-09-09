# Task ADR-037-T3: README and the skill name the command

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** none
**Consumes:** `mrw instructions` subcommand (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `README naming mrw instructions`, `the skill naming mrw instructions`, `the test going red when either mention is removed`

## Goal

A caller who reads README or the central `mrw` skill learns that the binary prints its own
contract, so those mirrors do not keep teaching "read AGENTS.md first" as the only path.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/agentsdoc_test.go` | edit | `TestReadmeAndSkillNameInstructions` — the failing test that makes S1 red. |
| `README.md` | edit | Name `` `mrw instructions` `` next to the other subcommands. |
| `.claude/skills/mrw/SKILL.md` | edit | Same invocation, so the centralised skill does not omit a command the binary grew — issues #51 and #73. |
| `docs/adr/BACKLOG.md` | edit | Receipts for this record's deferrals, already drafted with the ADR; confirm the "From ADR-037" heading is present. |

## Ordered Steps

1. [S1] Write `TestReadmeAndSkillNameInstructions` and confirm it is RED: README.md or the skill lacks `` `mrw instructions` ``. [proof: mutation]
2. [S2] Add `` `mrw instructions` `` to README.md in the subcommand list and to `.claude/skills/mrw/SKILL.md`. [proof: mutation]
3. [S3] Confirm `docs/adr/BACKLOG.md` has a `## From ADR-037` section naming each deferred Out of Scope entry. [proof: acceptance]
4. [S4] Confirm both documentation tests pass. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v \
  -run 'TestReadmeAndSkillNameInstructions|TestEverySubcommandReachesTheAgentFacingGuide' 2>&1 | tee /tmp/adr037-t3.out \
  && grep -q '^--- PASS: TestReadmeAndSkillNameInstructions' /tmp/adr037-t3.out \
  && grep -q '^--- PASS: TestEverySubcommandReachesTheAgentFacingGuide' /tmp/adr037-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr037-t3.out \
  && grep -q '## From ADR-037' docs/adr/BACKLOG.md \
  && [ -z "$(gofmt -l .)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestReadmeAndSkillNameInstructions` | `cmd/mrw/agentsdoc_test.go` | README.md and the skill both contain the backticked invocation | — | S1, S2 |
| `TestEverySubcommandReachesTheAgentFacingGuide` | `cmd/mrw/agentsdoc_test.go` | AGENTS.md still names every command including `instructions` | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestReadmeAndSkillNameInstructions` |
| 2 — something selects it | deleting either backticked mention fails that test |
| 3 — the caller can discover it | README and the skill name the command |
| 4 — it is used | nothing measures this yet |

## Mutation Log

- 2026-09-09 · 85b7455* · mutant killed · exit 1 · `README.md` · S2: README drops the backticked invocation, so a caller who reads only the README is not told the binary prints its contract · acceptance-sha256:789ed745a6f51c25d23a6f0dc8b566a064f641b2aeb0d7ca20825b47d1f67ba8 · covers:README naming mrw instructions
- 2026-09-09 · 85b7455* · mutant killed · exit 1 · `.claude/skills/mrw/SKILL.md` · S2: the skill drops the backticked invocation, so issue #73 recurs for the mirror · acceptance-sha256:789ed745a6f51c25d23a6f0dc8b566a064f641b2aeb0d7ca20825b47d1f67ba8 · covers:the skill naming mrw instructions

## Invariants

- No behaviour change. The command already works from T2.
- Deferrals in BACKLOG name ADR-037 so `adr-debt` can see the receipt.

## Risks

- A mention that is not the backticked invocation. Mitigation: the fence greps `` `mrw instructions` ``.

## Stop Condition

Stop if documenting the command seems to require restating the five shared sentences in full in
README — point at `mrw instructions` instead.

## Out of Scope

- Rewriting AGENTS.md's "Using mrw" section to be generated from `Shared()` (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:789ed745a6f51c25d23a6f0dc8b566a064f641b2aeb0d7ca20825b47d1f67ba8 · ms:406
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:789ed745a6f51c25d23a6f0dc8b566a064f641b2aeb0d7ca20825b47d1f67ba8 · ms:714
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:789ed745a6f51c25d23a6f0dc8b566a064f641b2aeb0d7ca20825b47d1f67ba8 · ms:379
