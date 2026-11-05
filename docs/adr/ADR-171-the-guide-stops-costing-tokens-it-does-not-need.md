# ADR-171: The guide stops costing a caller tokens it does not need

**Status:** Accepted
**Accepted:** 2026-10-24 by Zy — asked which drafted records to accept, chose *"ADR-171 guide cost, ADR-172 + ADR-119 D5, ADR-173 + ADR-174, ADR-175 forget"*, and for the breaking pair *"v1.76.0, loud note (Recommended)"*.
**Amended:** 2026-10-24, before acceptance, from a Codex xhigh challenge (9 P2/P3, each checked against the code): the count rule names every case the parser needs a count for (Decision 1); a file avoids shell quoting, not incomplete authoring (Decision 3); after a compaction the ledger still licenses lines the model no longer holds, so the guide says to re-read what will be edited (Decision 2); the core stays under 300 words and ADR-141's test is updated, not "kept"; the temporary location and the host-file-tool exception are stated (Decision 3); every surface that teaches the count is dispositioned (Decision 5); the fences gain an engine-diff gate, the skill-equality test and positive checks; the BACKLOG receipt is a T2 step.
**Date:** 2026-10-24
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-037, ADR-054, ADR-060, ADR-070, ADR-141, ADR-161, ADR-168, ADR-170, ADR-175, docs/agents.md, README.md, AGENTS.md
**Governs:** `internal/guide/guide.go`, `docs/agents.md`, `.claude/skills/mrw/SKILL.md`
**Enforced-by:** None — until T1 lands, which adds `internal/guide/cost171_test.go::TestTheCoreTeachesTheBodyCountAsOptional`
**Invalidates:** ADR-141 — core rule 4's clause "body= is a line count and goes on the header", which agents read as "write one", and the core's header, which pointed at the full contract (`core141_test.go:44` pins that prefix and changes with it); ADR-161 Decision 4's recipe 5 as the default way to send a plan. ADR-070's rule that a count, when written, goes on the header is kept.
**Served-path change:** `mrw instructions --core`, `mrw instructions`, `mrw write --help` and `mrw skill` tell a caller that `body=` is optional for an ordinary body, that a plan is best sent in a file, and that the rules are read once a session while the lines to be edited are re-read after a compaction; no parser, exit code or receipt changes.

## Context

An outside team ran mrw v1.74.0 against Claude Code alone on their Rails monolith: 17 paired headless runs, Claude Opus 5.5, mrw set up as `docs/agents.md` §1 says (external tech summary v1, 2026-10-24; external report v3, 2026-10-23/24). Same code quality in blind reviews; with mrw, cost +18% ($61.43 vs $51.89), output tokens +35%, model time 161 vs 126 minutes. Their tech summary traced most of the extra to three things the guide tells an agent to do:

1. **`body=` counts (MRW-01).** 7 of 50 plans (14%) were refused for a miscount, 13 count errors, 9 of them off by one, 7 of those leaving out a closing `end`; in 4 of 17 runs the agent wrote a Python script only to count lines. For an ordinary inline body the count is optional — the same plan without `body=` applies (reproduced 2026-10-24 on v1.75.0) — but core rule 4 says "body= is a line count and goes on the header" (`internal/guide/guide.go:129`), which an agent reads as an instruction to write one.
2. **The rules text, re-read (MRW-02).** 45 reads of `mrw instructions`, 279K characters: 8% of all tool output with mrw and 72% of the extra tool output against Claude Code alone. 15 of 17 runs read `--core` and then the full text at the start (1.6K + 12.9K characters), and 13 more reads came after compactions (65K). `docs/agents.md:51` says to run `--core` at session start, "and `mrw instructions` adds the recipes"; the core's header says "(mrw instructions prints the whole contract)" (`guide.go:125`).
3. **Plans through the shell (MRW-03).** Recipe 5 sends a plan as `printf '%s\n' "$plan" | mrw write -` (`guide.go:53`, `AGENTS.md:152`), and agents use heredocs. A host guard that splits each command with shell quoting rules (theirs used Python `shlex.split`) refuses a plan holding an odd apostrophe — a Ruby comment's "today's" — and all 8 of their `cat > /tmp/… <<` attempts were blocked; the agent then re-sent each plan with the host's Write tool. A plan in a file never passes through the guard: `mrw write /tmp/x.plan` applies a plan holding `'` and `$` (checked 2026-10-24 on v1.75.0).

A fourth item is a gap in what the docs show: a write to a `.rb` file runs no check in a tree that declares none (MRW-09; `internal/writer/flow.go:30`, ADR-054), and README shows no `scoped_check` for a stack other than Go and Rust.

`docs/agents.md:59-62` chose "run `--core` at session start" after a Windows agent found the read recipes only at the end of a day (2026-10-22). That choice stands: what this record removes is the full text read after it, and the re-reads of the rules after compaction.

## Existing Primitives Audit

- `guide.Core()` and `guide.Recipes()` (`internal/guide/guide.go:124`, `:42`): the texts `mrw instructions` and `mrw skill` print. Edit in place.
- `guide.Skill()` (`:151`) builds the skill from them; `.claude/skills/mrw/SKILL.md` is regenerated by `mrw skill --dir .claude/skills`, and `cmd/mrw/skill168_test.go::TestTheRepositorySkillIsTheBinarysSkill` holds the two equal (ADR-168).
- `internal/guide/core141_test.go`: holds each core rule to the full text (`:24-25` pin rule 4's phrases), the core under 300 words and its header prefix (`:44`). The core measured 299 words on 2026-10-24, so the new wording replaces words, never adds them; the test's pinned phrases change with it.
- Contract §242 (`scripts/contract.sh:8835`) checks the core's size and flags; a new row checks the new guidance through the built binary.
- `mrw write PLAN` already reads a plan from any path. Reuse; no new flag.
- `.quality-harness.json` `scoped_check` with `{files}` (ADR-061, ADR-124): the Ruby check is a documented use of it, not new code.

## Decision

**1. `body=` is optional for an ordinary body.** The parser needs a count in four cases and refuses one in two (`internal/plan/plan.go`): an empty `create` is `body=0`; a body holding a line that begins `@@ ` needs `body=N` — and `raw=true` too when that line is a valid hunk header, since an uncounted one starts a new hunk and an uncounted malformed one is refused as a bad header (`plan.go:323`); a body whose first line begins `body=` needs its count (ADR-070); `from=` and `body=@path` take no inline count. Every other inline body runs to the next `@@` header or the end of the plan, and needs no count. Core rule 4 becomes "A multi-line replace needs anchor= copied from the line your read printed; omit body= unless mrw instructions names your case." The full text's `body=` sentences (`guide.go:92-95`) open with that list, keep ADR-070's worked header, and say what a count still buys: it catches a body shorter than its declared count, lost in a channel or in the authoring. It cannot catch a hunk left out whole. Plans mrw compiles from foreign formats keep their counts.

**2. The rules are read once a session; after a compaction, what will be edited is re-read.** The core's header becomes "The eight rules that matter most; read them once a session, and after a compaction only if lost:" (the core measures 296 words with Decisions 1-3, against ADR-141's 300). The ledger survives a compaction and still licenses lines the model no longer holds; this record does not change that (ADR-175 proposes an opt-in epoch), so the guide tells the model to re-read the ranges it will edit, which the ledger does not require, and to read the rules again only when they are no longer in its context. `docs/agents.md`'s trigger becomes "Run `mrw instructions --core` once, at the start of a session, before the first read. After a compaction, re-read the lines you will edit; read the rules again only if they are gone from your context. Run `mrw instructions` when a refusal names something the core does not." The skill carries the core and follows.

**3. A plan is best sent in a file.** Recipe 5 becomes "different changes at several sites: one read of all of them, then the plan in a file — written with the host's file tool, in a temporary directory you may write — so no shell parser sees its text:" with the command `mrw read 'a.go:/func A/,+12' b.go:40-60 && mrw write "$TMPDIR/plan.txt"`. Rule 1 names `mrw write PLANFILE`. A plan file is the caller's input, not a repository edit: writing it with the host's file tool is allowed, and the `docs/agents.md` trigger and subagent brief say so in one sentence. Where no temporary directory outside the checkout is writable, a plan file inside it is fine and is removed after the write; where the host has no file tool, `mrw write - <<'PLAN'` with a quoted delimiter stays the form. The file avoids shell quoting; it does not make the plan complete — a count catches a body shorter than declared, and nothing catches a hunk left out.

**4. A stack without a check gets an example.** README's check section shows one non-Go `scoped_check`, `"scoped_check": "bundle exec rubocop --force-exclusion {files}"`, and says a tree that declares no check gets none, except a Go module, where mrw infers `go test` from `go.mod` (`internal/check/check.go:139`). mrw does not infer a Ruby check (ADR-054). Amended 2026-10-24 from the Codex review of #526, which found the first wording ("whatever the language") contradicted the Go inference.

**5. Every surface that teaches the count is dispositioned.** `mrw instructions --core`, `mrw instructions`, `mrw skill`: Decisions 1-3. `mrw write --help` (`cmd/mrw/main.go`, its `body=` lines): opens with "body= is optional for an ordinary body" and keeps the line-count sentence. The MCP instructions (`internal/mcp/instructions.go`): unchanged — the tools are off unless enabled (ADR-170), a plan there is a JSON string no shell parses, and the counted example shows ADR-070's header form within the 4096-byte budget. ADR-037's shared sentences hold no count rule and are unchanged.

**What would make this decision wrong:** agents that, told to omit `body=`, write bodies missing lines that a count would have caught. A body that goes missing entirely is already refused ("replace with an empty body would delete 2-4"); a body short by a line applies. The external runs show the opposite error, a miscount refusing a correct body, at 14% of plans.

## Alternatives Considered

- **Apply a plan whose count is off by one and warn.** Rejected: guessing which line the caller meant is the write-that-changes-the-wrong-thing this tool exists to prevent.
- **Put the eight rules in the trigger text instead of a tool call.** Rejected: `docs/agents.md:33-37` measured that copies go stale with the next release; the skill is the copy that updates itself (ADR-168). The once-a-session call costs 1.6K characters.
- **Print one line on a repeat `mrw instructions` call in the same state directory.** Rejected for now: it makes a read-only command depend on state, and Decision 2 removes the reason for the repeat (deferred: docs/adr/BACKLOG.md).
- **Run `ruby -c` on `.rb` writes when ruby is on PATH.** Rejected: a check per language inside mrw is the inference ADR-054 declined.
- **Recommend the MCP server, whose plan never touches a shell.** Rejected: ADR-170 turned it off for hosts with a shell, for measured reasons.

## Component / Boundary Impact

`internal/guide` and `cmd/mrw`'s write help (text); docs. No package changes ownership; the engine packages do not change.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw instructions --core` | header, rule 1's plan form, rule 4 | `internal/guide` | agents, standing triggers |
| `mrw instructions` | recipe 5; the `body=` sentences | `internal/guide` | agents |
| `mrw write --help` | `body=` is optional for an ordinary body | `cmd/mrw` | agents |
| `mrw skill` | follows Core and Recipes | `internal/guide` | skill hosts |
| `docs/agents.md` trigger, README, AGENTS.md | the same words | docs | adopters |
| contract | a new row for the guidance, beside §242 | `scripts/contract.sh` | CI |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the new rule 4, header and recipe 5 text | T1 | T2 | No |

## Implementation

`tasks/README.md`: T1 (the binary's text, its tests and contract row), T2 (the docs, the regenerated skill, the BACKLOG receipt). One pull request.

## Consequences

- **Positive:** the three costs the external runs traced to the guide stop being taught.
- **Negative:** every standing trigger that copied the old `docs/agents.md` sentence (this owner's three host files among them) needs the new one; that is the owner's to change.
- **Neutral:** the parser, exit codes and receipts are unchanged.

## Out of Scope

- MRW-04, a refused plan from standard input is kept (permanent: boundary: decided by its own record, ADR-176, accepted the same day as ADR-172 and renumbered before it merged, since PR #528 had taken 172)
- MRW-12, an epoch that makes old licences need a fresh read (permanent: boundary: decided by its own record, ADR-175, accepted the same day)
- A one-line answer to a repeated `mrw instructions` (deferred: docs/adr/BACKLOG.md)
- Re-measuring the external A/B after this lands (external: the external team: their headless A/B harness; this corpus has none on their repository)
- The owner's own host triggers (`~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, Cursor rules) (permanent: boundary: they live outside this repository; the owner changes them)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A body missing a line applies without a count | Low | Med | the full text names when a count is worth writing; a wholly missing body is refused |
| Agents keep reading the full text out of habit | Med | Low | the core header and the trigger both say when to read it |
| A plan file left behind | Med | Low | outside the checkout where possible; inside it, removed after the write |

## Rollback

Revert the text; nothing persistent changes.

## Follow-ups

- [ ] none at authoring
