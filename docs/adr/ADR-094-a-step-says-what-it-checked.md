# ADR-094: A step says what it checked

**Status:** Accepted
**Accepted:** 2026-09-29 by Zy — "ADR-094 step checked", selected under "Which records do you accept as written, so I can execute them?", on the record as drafted after a Codex review, including its narrowing of ADR-092 Decision 8 for the two template tokens
**Date:** 2026-09-29
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-009, ADR-048, ADR-054, ADR-061, ADR-072, ADR-080, ADR-083, ADR-092
**Governs:** `internal/check/check.go`, `cmd/mrw/main.go`, `internal/guide/guide.go`
**Enforced-by:** `cmd/mrw/then094_test.go::TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten`
**Invalidates:** ADR-092 — five clauses, each narrowed rather than reversed: Decision 1's "`--then-sh 'CMD'` runs `CMD` as given" (a command holding `{files}` or `{packages}` is refused, not run); Decision 2's list of what a `steps` block is refused for (it gains that token); Decision 3's "Each step runs as the check does" (as the check's process runs, never as its command is chosen: a step is never scoped); Decision 5's "The human receipt prints one line per step" (a passing step with output prints a second); and Decision 8's "Nothing filters the command's words", with its Out of Scope twin at `:193` (true of every word but mrw's own two template tokens, which Decision 2 matches; no shell word is parsed). ADR-092's tasks stay done: no test or contract row pins a passing step to one line — `— PASS` occurs in `cmd/mrw/main.go` alone across `cmd/`, `internal/` and `scripts/contract.sh`, checked 2026-09-29 at 8cbb89e.
**Served-path change:** `mrw write` and `mrw check` refuse, exit 2 and before anything is written, a `--then` step or `--then-sh` command holding `{files}` or `{packages}`, naming the step and the token; and a passing step's line in the human receipt is followed by the last non-empty line of its output.

## Context

The mrw gap survey of 2026-09-29 (workflow `wf_d4444a2d-6cc`, agentsmemory drawer `349dbd2b`, wing
`wing_tool-multipathreadwrite`, room `findings`; run on v1.31.0 at 2ea8bd5, a docs-only diff to
8cbb89e) ranked two step findings second and third of fifty. Both were re-verified here on
2026-09-29 against main 8cbb89e and the installed `mrw v1.31.0 (2ea8bd5)` on macOS, in a scratch root
holding an unformatted `a.go` (`gofmt -l a.go` lists it) and `"check": "true"`:

- **C4 = lint-L1 — a placeholder in a step is text.** `command()` substitutes `{packages}` and
  `{files}` into `scoped_check` (`internal/check/check.go:511-525`); `RunSteps` hands a step's command
  to `run` as written (`internal/check/check.go:792`). A declared step
  `"fmt": "test -z \"$(gofmt -l {files})\""` ran `gofmt` on a file literally named `{files}`, whose
  error went to stderr, so the substitution was empty and `test -z` passed: `mrw check a.go --then fmt`
  printed `then 1/3 fmt: … — PASS`, exit 0, having checked nothing. `mrw check PATH --then X` scopes
  the check to `PATH` and never `X`. `--then-sh 'gofmt -l {files}'` is the same text reaching `sh`
  (there it failed loudly, exit 3, because nothing masked `gofmt`'s status).
- **lint-L2 + X1 — a passing step shows none of its output.** `reportSteps` prints a passing step as
  its head line alone (`cmd/mrw/main.go:1724-1731`), while a passing CHECK prints its whole tail
  above its verdict (`cmd/mrw/main.go:2005-2007`). In the same probe `mask: echo FAIL-visible-line;
  false || true` printed `— PASS` with nothing under it, and the `fmt` step above hid its
  `lstat {files}: no such file or directory`. `--json` already carried both lines: a passing step's
  `then.steps[].tail` is filled (`internal/check/check.go:376`, copied at `:797`). quality-harness
  3.1.4 refuses a `.quality-harness.json` whose check is a constant success
  (`plugin/scripts/lifecycle.mjs:602`, `constantSuccessCheck`, read 2026-09-29 in the local
  checkout); mrw reports the same `|| true` shape as a pass.
- **C14 — `--quiet` does not quiet step lines.** Traced: `--quiet` exists on `write` only (`mrw check
  --quiet` is a usage error) and reaches `report` alone (`cmd/mrw/main.go:1352`); `reportCheck` and
  `reportSteps` take no such argument (`:1411-1412`), so a passing check and every step line print
  under it — while its usage says "print only failures and the summary line" (`:1002`).

The survey's own recommendation bounds the fix: refuse the placeholder rather than expand it, and make
a pass visible rather than detect a vacuous one, because mrw judges the process and never its words
(ADR-003 rule 1, ADR-048, ADR-092:193).

## Existing Primitives Audit

- `command()` (`internal/check/check.go:511`) is the one place mrw gives `{packages}` and `{files}` a
  meaning. **Keep** it untouched; the new `check.Placeholder` names the same two tokens beside it, and
  a test binds them (T1).
- `checkSteps` (`internal/check/check.go:137`), reached through `Config.StepCommands()` only when a
  step is asked for by name (ADR-092 T5), already refuses a step with no name or no command. **Reuse**:
  it gains the placeholder case, and inherits its whole-block, sorted-order judgement.
- `askedStepsError` (`cmd/mrw/main.go:1599`) refuses an empty `--then-sh` and the depth guard before
  the plan is read — called on `write` at `:1114` and on `check` at `:1929`. **Reuse** for the ad-hoc
  case. `resolveSteps` (`:1616`, called at `:1256` and `:1932`) carries `StepCommands`' refusal on both.
- `reportSteps` (`cmd/mrw/main.go:1715`) and `lastLines` (`internal/check/check.go:715`). **Reuse**:
  the tail a passing step needs is already in `StepResult.Tail`; only the printing changes.
- **The class, enumerated 2026-09-29** — every command string mrw hands to a shell, and whether a
  placeholder in it is substituted: `mrw read --grep 'run\(ctx, root, cfg,' internal/check/check.go`
  → 2 call sites, `:242` (the check, whose command comes from `command()`) and `:792` (a step); and
  `mrw read --grep 'return cfg\.Check|r\.Replace\(cfg\.ScopedCheck\)' internal/check/check.go` → 4
  returns, `:518`/`:522` substituting `scoped_check` and `:513`/`:524` returning `check` as written.
  So four sources, three of them run as written: a declared step, a `--then-sh` command, and the
  `check` field. This record covers the two step members. **`check` is left out on purpose** (Out of
  Scope): it is not a step, a passing check already prints its tail — the same probe with
  `"check": "test -z \"$(gofmt -l {files})\""` printed `| lstat {files}: no such file or directory`
  above `check PASS` — and refusing it would mean either refusing at `Load`, which every write reads
  (ADR-072) including writes that never run `check`, or refusing once it is chosen, after the write
  has landed. That is a different weighing, on a surface ADR-054 and ADR-061 own.

## Decision

1. **A step runs as written and is never scoped.** Only `scoped_check` has placeholders. A step —
   declared or ad hoc, on `write` or on `check`, given paths or not — runs its command in the root as
   written; `mrw check PATH --then X` scopes the check to `PATH` and never `X`. ADR-092 Decision 3's
   "each step runs as the check does" means as the check's process runs (shell, log, timeout, group,
   verdict), not as its command is chosen.
2. **A step whose command holds `{files}` or `{packages}` is refused, exit 2, before anything is
   written, naming the step and the token.** A declared one is refused by `checkSteps` with the rest of
   ADR-092 Decision 2's list — `.quality-harness.json: step "fmt": its command holds {files}, which
   mrw expands only in scoped_check; a step runs as written`. As that list already is since ADR-092 T5,
   the block is judged whole, and only when a step is asked for by name: a token in `lint` refuses
   `--then vet` and the message names `lint`, while a write asking for no step never reads the block.
   A `--then-sh` holding one is refused by `askedStepsError`, beside the empty-command rule, before the
   plan is read. Both refusals leave through the path an unknown `--then` or an empty `--then-sh` takes
   today, on `write` and on `check`, so `--json` gives ADR-072's single refusal document, and the tally
   counts them as ADR-083 counts those: on `write` a declared step's refusal comes after the plan
   parsed and is one `refused_apply` (rule 1), a `--then-sh`'s comes before it and is not counted
   (rule 4); `check` counts nothing. Only the two exact tokens are matched: `{file}`,
   `{ files }` and `{FILES}` run as written.

   **This narrows ADR-092's rule against filtering a step's words; it does not reverse it** (Decision 8;
   Out of Scope, ADR-092:193). That Out of Scope bullet declines to filter shell as a safety boundary, because a caller routes around a safety filter.
   These two tokens are mrw's own template grammar, and the refusal says "mrw will not do what this
   text asks of it". Routing around it — naming the paths, or writing jq's shorthand `{files}` as
   `{files: .files}` — is the outcome wanted, not a bypass. No shell word is parsed.
3. **A passing step shows its last line.** Under a passing step's head the human receipt prints the
   last non-empty line of its tail as `  | <line>`, raw, as the check's tail lines are printed. A step
   whose tail holds no non-empty line — no output, or only blank lines — adds no line: its head prints
   alone, and a kept log's pointer (below) prints as today. `then last:` stays failure-only, as ADR-092
   Decision 5 and AGENTS.md teach `check last:`. When a passing step's output ran past `tail_lines` and
   its log was kept (ADR-092 T4), the pointer above the shown line counts every line above it — the
   lines before the tail plus the tail lines before the shown one, so blank lines after the shown one
   add nothing — and `... N earlier line(s) in FILE` stays exact. A pass that did not run past
   `tail_lines` deletes its log (`internal/check/check.go:386-390`): the rest of its tail is then in the
   JSON `tail` and nowhere else.
4. **`--json` does not change.** `then.steps[].tail` already carries a passing step's tail; T2 pins that
   the human line is the last non-empty entry of that list.
5. **`--quiet` hides no verdict.** It drops ok and skip hunk rows and the file lines, as it always has;
   the check's report and every step line stay, the new line included — it is the line a terse caller
   most needs. Its usage is rewritten to say so: "print only failures and the summary line" has
   overstated it since before ADR-092, because a passing check prints under it too.

**What would falsify it:** `mrw write --no-check --then fmt` with a declared
`"fmt": "test -z \"$(gofmt -l {files})\""` exiting 0, or writing its file, or running any step; or
`--then-sh 'echo LASTLINE; false || true'` exiting 0 with no `  | LASTLINE` under its `— PASS` line.
Contract §178 and §179 build exactly those, each beside the case that must still run and print. Valid
for the CLI on Linux (contract.sh) and on every CI platform through the Go tests; the MCP surface runs
no step (ADR-092:192). §180, also allocated to this record, is not used.

**What it cannot see**, and says so: a masked failure that prints nothing (`false || true`), or whose
last line reads like success. Detecting a constant-success command means reading its words
(ADR-003 rule 1, ADR-048, ADR-092:193); the head line already prints the command, `|| true` included.

## Alternatives Considered

- **Expand `{files}` and `{packages}` in a step (the survey's option a).** Rejected: a step has no
  whole-project form to fall back to. ADR-061:37 makes the check fall back for an empty path list
  (`mrw check --full`, an empty working set), for `{packages}` on a path it cannot map, and for a mixed
  template with an empty map; for a step each case would need its own rule and row — refuse after the
  write landed, or run a command naming no file, the silent omission ADR-003 exists to prevent.
  Refusing is one rule, and it reserves the tokens: a later record can give them a meaning without
  breaking any caller, where expanding now would fix a meaning nobody asked for.
- **Refuse declared steps only, and run `--then-sh` as given.** Rejected: it is the same mistake —
  a caller who learned `scoped_check`'s grammar, which `mrw instructions` teaches the same agent who
  types `--then-sh`. The cost lands on a caller who meant the literal text (jq's `{files}`), and the
  refusal names the rewrite.
- **Refuse when the step runs.** Rejected: that is after the write landed; ADR-092 Decision 2 puts a
  step refusal before anything is written, in ADR-072's order.
- **Print the whole tail under a passing step, as a passing check prints its own.** Rejected: every
  green write would print up to `tail_lines` (default 30) lines per step, and the caller is usually an
  agent paying for every line (`cmd/mrw/main.go:2039-2040`). The last line is where most runners
  summarise (`ok`/`FAIL` for `go test`, the error for `gofmt` or `vet`), and the window is in the JSON.
- **Print a passing step's last line as `then last:`.** Rejected: `then last:` and `check last:` mark
  a failure's last line, and a reader or hook looking for them would find passes.
- **Report a pass as vacuous when the command ends in `|| true`**, as quality-harness 3.1.4 does.
  Rejected: mrw judges the process, never its words (ADR-003, ADR-092:193); a shell parses shell.
- **Hide the step lines under `--quiet`.** Rejected: Decision 5.

## Component / Boundary Impact

`internal/check` owns which commands run and what may be in them: `Placeholder` sits beside
`command()`, and `checkSteps` refuses. `cmd/mrw` owns when a refusal happens and what the receipt
prints. `internal/guide` teaches it. `internal/check` is an engine package; this record owns T1's change
there (`Placeholder`, one `checkSteps` case — `command()` untouched), and `internal/read`, `apply`,
`plan`, `seen` and `state` stay byte-identical.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `.quality-harness.json` `steps` | a command holding `{files}` or `{packages}` is refused when a step is asked for by name | project | `Config.StepCommands` → `resolveSteps` |
| `--then-sh` on `write` and `check` | a command holding either token is refused, exit 2, before the plan is read | caller | `askedStepsError` |
| `check.Placeholder(cmdline string) string` | new export: the first token a command holds, or "" | T1 | `checkSteps`, `askedStepsError` |
| human receipt | a passing step with output prints its last non-empty tail line, with the check's tail prefix, under its head; a kept-log pointer counts every line above the shown one | T2 | callers |
| `write --quiet` usage | names what it drops and that the check and step verdicts still print | T2 | callers |
| `write --json` / `check --json` | none — `then.steps[].tail` already carries a passing step's tail | — | hooks, agents |
| `mrw instructions`, AGENTS.md "Using mrw", README | teach both rules | T3 | callers; the centralised `mrw` skill mirrors AGENTS.md at the next release |
| exit codes | none new — the refusal is exit 2 as an unknown `--then` is | — | callers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `check.Placeholder`, the placeholder refusal | T1 | T3 | Yes for a step holding a token — refused where it ran; no for any other |
| the passing-step line, the `--quiet` usage | T2 | T3 | No — one line added under a pass |

## Implementation

See `tasks/README.md`.

## Consequences

- **Positive:** a step that would have checked a file named `{files}` is refused by name before the
  write; a `go test ./... || true` step shows `FAIL` under its `PASS`; `--quiet` says what it does.
- **Negative:** a project whose step holds a literal `{files}` or `{packages}` (jq's shorthand) is
  refused on upgrade, and — the block being judged whole — so is every `--then` in that tree until the
  step is rewritten. One line more per passing step that printed anything.
- **Neutral:** without `--then`/`--then-sh` nothing changes; `--json`, exit codes and the tally are as
  today; `scoped_check` and `check` run exactly as before.

## Out of Scope

- Expanding `{files}` or `{packages}` in a step (permanent: boundary: a step has no whole-project form to fall back to, so ADR-061's empty-list and unmapped rules have no safe reading for it; refusing reserves the tokens, and a record that gives them a meaning later breaks no caller)
- Refusing a placeholder in the `check` field, which also runs as written (deferred: docs/adr/BACKLOG.md — "From ADR-094")
- Detecting a constant-success command such as one ending in `|| true` (permanent: fact: mrw judges the process and never its words; citation: file `docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict.md:193`)
- A masked failure that prints nothing, which stays indistinguishable from a silent real pass (permanent: boundary: without parsing the command there is no output to show; the head line prints the command)
- The `then` head line printing a command Go-quoted when it holds `"` or `\`, not only control bytes (deferred: docs/adr/BACKLOG.md — "From ADR-094")
- Steps on the MCP surface (permanent: boundary: this record changes the CLI only; steps on MCP remain ADR-092's deferral, BACKLOG "From ADR-092")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A step legitimately holds the literal text `{files}` (jq object shorthand) and every `--then` in that tree is refused on upgrade | Low | Med | the refusal names the step, the token and why; `{files: .files}` is the rewrite; Rollback below |
| A third token is added to `command()` and not to `Placeholder` | Low | Med | the two sit together with doc comments pointing at each other; T1's binding test fails if `Placeholder` names a token `command()` does not substitute (the reverse needs the reader) |
| The shown line reads like success while an earlier line carries the failure | Med | Low | Decision 3 says so; the whole window is in the JSON, and a kept log is named |
| Another receipt consumer assumes one line per passing step | Low | Low | none found: no test, contract row or shipped plugin parses `then` lines (checked 2026-09-29) |
| Refusing in `checkSteps` makes a write that asks for no step read the block again | Low | Med | T1 keeps `TestAPlainWriteIgnoresAMalformedStepsBlock` in its fence |

## Rollback

Revert the tasks. Nothing persists: a harness whose step holds a token is refused under this record
and runs as written again without it; the receipt loses one line per passing step.

## Follow-ups

- [ ] Update the centralised `mrw` skill from AGENTS.md at the release that ships this record.
