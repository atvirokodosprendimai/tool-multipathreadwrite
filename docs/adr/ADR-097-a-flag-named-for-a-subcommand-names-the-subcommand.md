# ADR-097: A flag named for a subcommand names the subcommand

**Status:** Accepted
**Accepted:** 2026-09-29 by Zy — "Accept, keep suffix (Recommended)", selected under "Next step is ADR-097. Accept it as drafted, and keep the `(see: mrw --help)` suffix on the new message?" (the option wording was the session's), on the record as drafted, with Decision 5 settled as written
**Date:** 2026-09-29
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-063, ADR-069, ADR-078
**Governs:** `cmd/mrw/main.go`, `scripts/contract.sh`, `README.md`, `AGENTS.md`
**Enforced-by:** `cmd/mrw/usage097_test.go::TestAFlagNamedForASubcommandNamesTheSubcommand`
**Invalidates:** none — checked. ADR-078 Decision 1 ("the error and the help to read … nothing on stdout") is kept: the new sentence carries the same `(see: <command> --help)` pointer, exit 2, stderr only. ADR-069's padded-argument guards run in `main` before the parser and are unchanged, so a padded attached value is still refused first. ADR-063's `mrw instructions` is unchanged; this record only points callers at it.
**Served-path change:** `mrw --instructions` (and `-instructions`, `--instructions=v`, `'--instructions '`) answers ``mrw: --instructions is not a flag; the subcommand is `mrw instructions` (see: mrw --help)`` on stderr, exit 2, nothing on stdout, where v1.31.0 answered `mrw: flag provided but not defined: -instructions (see: mrw --help)`; the same holds for `--read`, `--write`, `--check`, `--iter`, `--seen`, `--stats` and `--mcp` at the root, and for any subcommand a future command declares below it; every other usage error is worded exactly as before.

## Context

**What was observed.** On 2026-09-29, against the installed `mrw v1.31.0 (2ea8bd5)` on macOS, in a
scratch directory:

| Invocation | v1.31.0 answer |
|---|---|
| `mrw --instructions` | `mrw: flag provided but not defined: -instructions (see: mrw --help)`, exit 2 |
| `mrw -instructions`, `mrw --instructions=x`, `mrw '--instructions '` | the same line, exit 2 |
| `mrw --stats`, `mrw --read` | `flag provided but not defined: -stats` / `-read`, exit 2 |
| `mrw --bogus` | `flag provided but not defined: -bogus (see: mrw --help)`, exit 2 |
| `mrw check --stats` | `flag provided but not defined: -stats (see: mrw check --help)`, exit 2 |
| `mrw iter --add` | `flag provided but not defined: -add (see: mrw iter --help)`, exit 2 |
| `mrw -v`, `mrw --version`, `mrw version` | `mrw version v1.31.0 (2ea8bd5)`, exit 0 |

`version` is offered both as a subcommand and as `-v`/`--version` (`cmd/mrw/main.go:128`, the
framework's version flag; `versionCmd` at `:384`), so a caller who has met `mrw --version` reasonably
tries `mrw --instructions` for ADR-063's subcommand. The refusal is correct and exit 2 is correct, but
it names nothing the caller can do: the thing they wanted exists one character-pair away and the
message does not say so.

**Where the message comes from.** Every command's `OnUsageError` is `usageError`
(`cmd/mrw/main.go:185-194`, installed on the whole tree by `installUsageErrors`, ADR-078), which
returns `cli.Exit(fmt.Sprintf("%v (see: %s --help)", err, cmd.FullName()), exitUsage)`. urfave/cli
v3.11.0 builds the unknown-flag error as a plain `fmt.Errorf("%s%s", providedButNotDefinedErrMsg,
flagName)` with `providedButNotDefinedErrMsg = "flag provided but not defined: -"`
(`command_parse.go:10`, `:208`, `:219` in the module cache). `flagName` is the token with its dashes
and any `=value` already removed (`:143-151`), after the parser's `strings.TrimSpace` (`:81`) — which
is why `-x`, `--x`, `--x=v` and `'--x '` all produce the same error. There is no typed error; the text
is the only handle, and urfave reads its own message back the same way (`flagFromError`,
`command_parse.go:14-23`). `OnUsageError` is called on the command whose parse failed
(`command_run.go:189-193`), so the command in hand is the one the flag was given to.

**Audit of the class.** The class is: *an unknown flag whose name is the name of a subcommand of the
command it was given to.* Enumerated on 2026-09-29 against v1.31.0 (2ea8bd5) with commands, not
memory:

1. `mrw nosuch` → `unknown command "nosuch" (want read, write, check, iter, seen, stats, mcp,
   instructions, version, help)`: **10** children of the root — the 9 of `cmd/mrw/main.go:173` plus
   urfave's `help`.
2. `mrw read --grep 'Commands: ' --exclude '*_test.go' cmd/ internal/` → **1** site,
   `cmd/mrw/main.go:173`: mrw declares subcommands at the root only.
3. `for s in read write check iter seen stats mcp instructions version; do mrw $s help; done` → each
   prints its own help, exit 0: urfave appends its `help` command (alias `h`, `help.go:66-74`;
   `ensureHelp`, `command_setup.go:241-251`) to **every** command, so each of the **9** leaves has
   exactly one child.
4. Names that are also defined flags where they are children never reach `OnUsageError`: `help`/`h`
   (`--help`/`-h` on every command, `flag.go:39-45`; `mrw <sub> --help` and `-h` exit 0 for all 9) and
   `version` (`-v`/`--version` on the root).

So the members that trigger today are **8**, all at the root: `--read`, `--write`, `--check`,
`--iter`, `--seen`, `--stats`, `--mcp`, `--instructions`. Below the root the only child is `help`,
which is also a flag, so **0** trigger there today; the rule is written for every level so a
subcommand declared later under any command is covered without anyone remembering this record.

**Left out on purpose, and why** (each is also in Out of Scope or Alternatives):

- **A sibling** — `mrw check --stats`. `stats` is not a subcommand of `check`; naming it guesses that
  the caller meant a different command rather than a `check` flag they misspelt.
- **`iter`'s verbs** — `mrw iter --add`. `add|rm|clear|note` are positional arguments parsed by the
  `iter` action (`cmd/mrw/main.go:1774`, `:1797`), not subcommands; the parser has no list of them.
- **A prefix, a typo or another case** — `--instruction`, `--instructionsx`, `--Instructions`. Only an
  exact name is a fact; the dispatcher itself is exact (`mrw Instructions` answers `unknown command
  "Instructions"`, exit 2, observed 2026-09-29; urfave's `HasName` is `slices.Contains`,
  `command.go:246-248`).
- **Any other usage error** — `invalid value "x" for flag -help`, `flag needs an argument`: they do
  not carry the not-defined prefix and are worded as before.
- **MCP** — its tools take JSON arguments, not flags; nothing on that surface reaches `usageError`.

**Found during the audit, not governed here** (Follow-ups below): `mrw read help` and `mrw read h`
print `read`'s help instead of serving a file named `help` or `h`, even after `--`; only `./help`
reaches it (observed 2026-09-29 against v1.31.0). And the doc comment `// instructionsCmd prints…`
sits above `versionCmd` (`cmd/mrw/main.go:383-384`) while `instructionsCmd` at `:402` has none.

## Existing Primitives Audit

- **`usageError` / `installUsageErrors`** (`cmd/mrw/main.go:179-194`) — reused. The one place every
  flag the parser rejects arrives, on every command, with the command it was given to. The change is
  a branch inside it; nothing new is installed.
- **`cli.Command.Command(name)`** (urfave) — reused. It is how the parser itself finds a subcommand
  (`command_parse.go:112`), by name or alias, so "is a subcommand" means exactly what dispatch means.
- **`cmd.FullName()`** — reused, as the existing message already does, to spell the command path
  (`mrw`, or `mrw <parent>` below the root).
- **`refusePaddedFlagValues`** (`cmd/mrw/main.go:2493`, ADR-069) — unchanged, and it is why ordering
  is safe: it runs in `main` before `root.Run`, so a padded attached value is refused before the
  parser ever reports a flag as unknown.
- **`CommandNotFound`** (`cmd/mrw/main.go:165-172`) — not reused. It answers an unknown positional
  subcommand; this record is about a known subcommand spelled as a flag.

## Decision

1. When the parser refuses a flag as not defined, and the flag's name — as urfave reports it, dashes
   and `=value` removed — is exactly the name (or an alias) of a subcommand of the command it was
   given to, the usage error is:

   ``--<name> is not a flag; the subcommand is `<command path> <subcommand>` (see: <command path> --help)``

   where `<name>` is the name as the parser reported it — an alias stays the alias, so the sentence
   never names a flag the caller did not type — and `<subcommand>` is that child's canonical `Name`.
   `main` prefixes `mrw: ` as it does every error, e.g.
   ``mrw: --instructions is not a flag; the subcommand is `mrw instructions` (see: mrw --help)``.
   The flag is always spelled with two dashes, because urfave's error keeps no record of how many
   the caller typed.
2. Exit 2, the message on stderr, nothing on stdout — ADR-078 unchanged. Nothing that was refused
   before is accepted: the subcommand is named, not run.
3. The lookup is the command's own children, exact, case-sensitive, by `cmd.Command(name)`. Siblings,
   ancestors, `iter`'s verbs, prefixes and near-misses keep today's message word for word.
4. The flag name is taken from urfave's error text by the exact prefix `flag provided but not
   defined: -`. Any other error text, including a reworded one after an upgrade, falls through to
   today's message unchanged — the change can go quiet, never wrong.
5. The `(see: … --help)` pointer is kept on the new message. M's example of 2026-09-29 omitted it;
   ADR-078 Decision 1 requires every usage error to name the help to read, and dropping it would amend
   an Accepted record for no gain. Zy kept it at acceptance (2026-09-29).

This is falsifiable today: `mrw --instructions` on v1.31.0 prints the not-defined line, so T1's first
test is red before the change and the contract row §188 fails against the installed binary.

## Alternatives Considered

- **An `--instructions` flag that prints the contract and exits, beside the subcommand.** Rejected by M
  on 2026-09-29 (chose "1", this record, over it). It fixes one name and leaves the other seven, adds a
  second spelling to teach and test for one command, and mirrors `--version` only because urfave
  special-cases that one flag.
- **urfave's `Suggest: true`.** Rejected: `suggestFlagFromError` proposes the nearest FLAG, and it runs
  only when `OnUsageError` is nil (`command_run.go:194-199`), which ADR-078 made non-nil everywhere.
- **Match a prefix or a near-miss (edit distance).** Rejected: a guess. A wrong guess sends a caller to
  a different subcommand, and two of them (`write`, `check`) change the tree or run shell. An exact
  name is a statement of fact about the command tree.
- **Name a sibling as well** (`mrw check --stats` → `mrw stats`). Rejected: the caller gave the flag to
  `check`; the likelier mistake is a misspelt `check` flag, and the existing message plus `--help`
  already serves that.
- **Pre-scan argv in `main`, as `refusePaddedFlagValues` does.** Rejected: it would classify flags a
  third time beside the parser and `flagRole`, and could only guess which command a token belongs to;
  `OnUsageError` is called at the exact point the parser has decided, on the right command.
- **Drop the `(see: … --help)` suffix, as in the example.** Rejected for this record (Decision 5);
  M can reverse it at acceptance, and T1's test states the choice so a reversal is one visible edit.

## Component / Boundary Impact

None — internal to `cmd/mrw` (`usageError` and one helper beside it). The engine packages
(`internal/read`, `apply`, `plan`, `seen`, `check`, `state`, `iter`, `rooted`, `lines`, `subproc`)
are unchanged, and `go.mod` keeps its one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| CLI stderr on an unknown flag named like a subcommand | new sentence naming the subcommand; exit 2 and empty stdout unchanged | `usageError` (T1) | agents and humans reading stderr |
| CLI stderr on every other usage error | unchanged | `usageError` | unchanged |
| `scripts/contract.sh` | §188: the named case and the unchanged case, driving the built binary | T1 | CI Linux, `adr-verify` |
| `README.md`, `AGENTS.md` | one clause beside `mrw instructions` | T2 | readers of the install notes and the agent guide |

No exit code, no flag, no subcommand, no JSON and no MCP change.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the named refusal (`subcommandForFlag` in `usageError`) | T1 | T2 | No — T2 documents what T1 ships |

## Implementation

See `tasks/README.md`: T1 (red tests, the branch in `usageError`, contract §188), then T2 (the clause
in README.md and AGENTS.md).

## Consequences

- **Positive:** `mrw --instructions` tells the caller the command that does what they wanted, in one
  line, without a help dump; the same for the other seven root subcommands, and for any subcommand
  added under any command later.
- **Negative:** the wording depends on urfave's error text. An upgrade that rewords it silently
  returns today's message; T1's test is the canary that goes red on that upgrade.
- **Neutral:** the hint names the subcommand, not a command line to replay: root flags the caller
  gave (`-C dir`) are not repeated in it. The help pointer is unchanged.

## Out of Scope

- A sibling or ancestor subcommand named as a flag, e.g. `mrw check --stats` (permanent: boundary: the rule names a child of the command the flag was given to; anything else guesses intent)
- `iter`'s verbs spelled as flags, e.g. `mrw iter --add` (permanent: boundary: the verbs are positional arguments the `iter` action switches on, not subcommands, and a second list of them beside that switch would drift)
- Prefix, near-miss or case-insensitive matches (permanent: boundary: only an exact name is a fact about the command tree; a guess can send a caller to a command that writes or runs shell)
- A typed urfave error instead of its text (permanent: fact: urfave/cli v3.11.0 returns the unknown-flag error as plain `fmt.Errorf` text and parses it back by prefix itself; citation: version `github.com/urfave/cli/v3@v3.11.0`)
- The MCP surface (permanent: boundary: MCP tools take JSON arguments, not flags, and never reach `usageError`)
- `mrw instructions`' own text (`internal/guide`) (permanent: boundary: a caller reading it has already found the subcommand)
- The centralised `mrw` skill that mirrors AGENTS.md (external: agentsmemory: the `mrw` skill mirror, refreshed with `am_update_skill` by whoever merges T2)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A urfave upgrade rewords "flag provided but not defined: -" | Low | Low | falls through to today's message (never a false claim); T1's named test goes red on the upgrade PR |
| The hint omits root flags (`mrw -C d --seen` names `mrw seen`) | Med | Low | it names the subcommand, not a replayable line; `(see: mrw --help)` stays |
| `mrw --check` names `mrw check` when the caller meant `mrw write --check` | Low | Low | the statement is true; the help pointer names where `--check` lives |
| `-instructions` (one dash) is echoed as `--instructions` | Certain | Low | urfave's error drops the dash count; `--` is the spelling the docs teach |
| A script matches the old "flag provided but not defined: -instructions" line | Low | Low | only the 8 subcommand names change; exit 2 is unchanged and is the contract |

## Rollback

Revert T1 and T2. No state, no exit code and no format changes; the old message returns word for word.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `cmd/mrw/usage097_test.go::TestAFlagNamedForASubcommandNamesTheSubcommand` in T1's commit.
- [x] Add to docs/adr/BACKLOG.md: `mrw read help` / `mrw read h` (also after `--`) print `read`'s help instead of serving a file named `help`/`h`; urfave's per-command `help` subcommand wins over a positional (observed 2026-09-29, v1.31.0).
- [x] Add to docs/adr/BACKLOG.md: the doc comment `// instructionsCmd prints…` sits above `versionCmd` (`cmd/mrw/main.go:383-384`); `instructionsCmd` (`:402`) has none.
