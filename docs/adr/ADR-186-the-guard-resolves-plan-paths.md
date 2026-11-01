# ADR-186: the benchmark's write guard resolves a plan file's path from `$TMPDIR`, `mktemp` and a literal

**Status:** Accepted
**Accepted:** 2026-11-01 by Zy — answered *"Accept and merge (Recommended)"* to "Accept and merge once CI is green?", after five Codex passes on #582
**Date:** 2026-11-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-160, docs/bench/observability/CORE-RESULTS.md
**Governs:** `internal/bench/hook.go`, `internal/bench/hook_test.go`
**Enforced-by:** `internal/bench/hook_test.go::TestAPlanFileAtAResolvablePathOutsideTheTreePassesTheHook`
**Invalidates:** ADR-156 Decision 1's clause that a redirection target "with an expansion" is judged inside, for exactly the forms Decision 1 below names; `TestAWriteOutsideTheTaskTreePassesTheHook`'s refusal of `echo x > $TMPDIR/plan.txt`, which moves to the new test as a pass. `TestARedirectionToAPathTheHookCannotResolveNamesTheTwoFormsThatPass` (#576) pinned `$(mktemp -d)` and `"$TMPDIR/…"` as refused; its cases move to `$HOME`, which nothing here resolves. Checked and unchanged: a relative target, a quoted literal path, a glob, a dot-dot, a link into the tree or a dangling one, a hard link to a tree file, the tree recognised by identity, and every other refusal; ADR-156's boundary that the hook is a measurement aid, never a security boundary, with `slipped_writes` the backstop.
**Served-path change:** none for mrw; `bench hook` (configuration B's PreToolUse guard) passes a redirection whose target is `$TMPDIR`, `${TMPDIR}`, or a variable assigned once in the same command from `$TMPDIR`, `$(mktemp)`, `$(mktemp -d)` or a literal absolute path, plus an optional literal suffix, when the resolved path lands outside the attempt's tree; it still refuses every other expansion.

## Context

ADR-156 lets a write outside the task tree pass the guard (Zy, 2026-10-14) so the MRW arm can write a plan file for `mrw write PLAN`, and judges any target with an expansion as inside, since the hook cannot see the shell's variables. The agents write plans exactly that way. Across runs 2 to 5 of the core-first ablation (`docs/bench/observability/CORE-RESULTS.md`), the guard refused a plan-file redirection in about one MRW cell in three, both arms alike. Those refusals had four shapes:

- `P=$TMPDIR/plan.txt; : > $P` and `cat > $TMPDIR/plan.txt`
- `cat > "$TMPDIR/plan.txt"`
- `p=$(mktemp -d); … > $p/plan.txt`, `P=$(mktemp); cat >> $P`, `P=$(mktemp -d)/plan.txt`
- `p=/tmp/plan.txt; : > $p`

The refusal hint added in #576 names the forms that pass, and each cell recovered, but each refusal is a round the native arm, which has no guard, never pays. That biases any MRW-versus-native comparison the benchmark runs.

**Audit of the class** — *a redirection target whose value the hook can know from the command and its own environment*. Enumerated from the refused commands in the four runs' transcripts and from what a shell can bind without running anything:

| Form | Value known without running? | After |
|---|---|---|
| `$TMPDIR`, `${TMPDIR}`, unquoted or in double quotes | yes: the hook inherits the attempt's `TMPDIR` | resolved |
| `NAME=$TMPDIR…`, assigned once | yes | resolved |
| `NAME=$(mktemp)` or `NAME=$(mktemp -d)`, no arguments, assigned once | a fresh name directly under `$TMPDIR` | resolved as a path under `$TMPDIR` |
| `NAME=/literal/absolute/path`, assigned once | yes | resolved |
| a variable assigned twice, appended (`+=`), or named by `read`, `for`, `declare`, `local`, `typeset`, `unset` or `readonly`; any variable in a command holding `eval` or `source` | no | inside, as now |
| `mktemp` with arguments (`-p`, a template) | not without running it | inside, as now |
| any other variable, a relative path, a quoted literal, a glob, a dot-dot | no, or already refused | inside, as now |

## Existing Primitives Audit

- `outsideTree` (`internal/bench/hook.go`): judges a resolved absolute path, following links, refusing a dot-dot and a hard link into the tree. Reused unchanged; every resolved target goes through it.
- `classify`, `splitHeredocs`, `scanShell`: unchanged; resolution rewrites the command's non-heredoc text before them.
- `SlippedWrites`: the backstop after each attempt, unchanged.

## Decision

1. **Resolution, before classification.** In the command text outside heredoc bodies, an expansion `$TMPDIR`, `${TMPDIR}`, `$NAME` or `${NAME}`, unquoted or the whole of a double-quoted word, with an optional literal suffix of `[A-Za-z0-9._/-]`, is replaced by its value when that value is known:
   - `TMPDIR` is the hook's own environment variable, used exactly as spelled (a trailing slash kept), only when it is an absolute path of those characters.
   - `NAME` is known from a statement of its own, `NAME=VALUE` or `export NAME=VALUE`, standing at the top level (outside every quote and parenthesis, so not inside `$(…)`, a subshell or quoted plan text), as the command's first statement or after a `;`, with no control keyword (`if`, `then`, `do`, `case` and the rest) before it (never after `&&`, `||` or `|`, even across a newline, where it may not run or runs in a subshell) and followed by nothing, a `;`, a newline or `&&` (so not a prefix to a command, as in `P=x cat > $P`, and not a pipeline or background job), where `VALUE` is `$TMPDIR…`, `$(mktemp)`, `$(mktemp -d)` (optionally followed by a literal suffix), or a literal absolute path, in those characters. `$TMPDIR` followed by an identifier character is another variable (`$TMPDIR_TREE`) and is not a base. Only an expansion after that statement reads the value.
   - `$(mktemp)` and `$(mktemp -d)` resolve to `$TMPDIR/tmp.mktemp`: a name directly under `$TMPDIR`, which is where `mktemp` makes one.
2. **A name the command could rebind is not resolved.** A command holding `eval` or `source` as a word, or a backslash or backquote outside single quotes, resolves no variable. A name mentioned at all in a segment holding `read`, `for`, `declare`, `local`, `typeset`, `unset`, `readonly`, `mapfile`, `readarray`, `getopts` or `printf -v` (so `IFS= read -r P`, `(read P)`, `read -r "P"`, `read P<f` and `printf -vP` count), or that is assigned more than once or appended to anywhere in the command, is not resolved; a `for` loop over another name does not stop it, since the plan builders in the ablation loop over `i`. `TMPDIR` itself is resolved only when the command neither assigns nor rebinds it, and a command substitution inside such a command, or inside one that resolves nothing (`eval`, `source`, a backslash or a backquote), is read with no `TMPDIR` at all, since bash hands the substitution whatever the outer command bound. The Codex reviews of this record found each of these limits as a way a resolved path could differ from what bash writes.
3. **The resolved target is judged as any absolute target is.** `outsideTree` decides; a target that lands in the tree, through a link, a dot-dot or a hard link, is refused as before, and the refusal hint is unchanged.
4. **Every other expansion is judged inside, as ADR-156 decided.**

**What would make this decision wrong:** a resolved path that differs from what the shell writes. The rebinding words of Decision 2 and the single-assignment rule are the guard against it; `slipped_writes` counts any that gets past.

## Alternatives Considered

- **Leave it; the refusal hint already names what passes.** Rejected: each refusal still costs the MRW arm a round, and in the ablation it happened in about one cell in three.
- **Give configuration B a file-writing tool for plans.** Rejected: it changes the treatment the benchmark measures.
- **Expand every variable by running the command in a sandbox.** Rejected: the hook reads shell by heuristic and runs nothing (ADR-156); running the command would be a second shell.
- **Resolve a quoted literal path too.** Rejected here: none of the refused commands used one, and ADR-156 judged it inside on purpose.

## Component / Boundary Impact

`internal/bench` only: the hook's classification of a redirection target. No mrw package, flag, receipt or exit code. Configuration B's guard becomes more permissive for the forms above; it remains a measurement aid.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench hook` verdict | the forms of Decision 1 pass when they resolve outside the tree | T1 | configuration B's agent |
| `TestAWriteOutsideTheTaskTreePassesTheHook` | its `$TMPDIR/plan.txt` refusal moves to the new test | T1 | CI |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** the MRW arm stops paying a round for the plan files agents actually write.
- **Negative:** the guard trusts a variable's single assignment; a shell construct outside Decision 2's list that rebinds a name would be misjudged. `slipped_writes` counts it.
- **Neutral:** mrw itself is unchanged.

## Out of Scope

- A quoted literal target, a relative target, `mktemp` with arguments (permanent: boundary: ADR-156 judged them inside, and none of the refused commands in the ablation used them)
- Shell the hook does not model (permanent: boundary: ADR-156's list of known ways past the hook stands; this record adds none it knows of)
- Re-running the benchmark to measure the saved rounds (permanent: boundary: this record changes the guard and claims no measured effect)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A resolved path is not what the shell writes | Low | Low | Decision 2's rebinding words, single assignment, `outsideTree` on the result, and `slipped_writes` |
| `TMPDIR` points into the tree | Low | Low | `outsideTree` judges the resolved path, so a target in the tree is refused |

## Rollback

Revert T1.

## Follow-ups

- [ ] none at authoring
