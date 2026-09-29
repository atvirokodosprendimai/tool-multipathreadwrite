# Task ADR-097-T1: The usage error names the subcommand a flag was spelled as

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the named refusal (`subcommandForFlag` in `usageError`); contract §188
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the named sentence`, `the unchanged sentence`, `the command's own children`, `exact-name lookup`, `the help pointer`, `the padded refusal runs first`, `a contract row drives the binary`, `the tree is gofmt-clean`, `no engine package changes`, `go.mod declares one requirement`

## Goal

`mrw --instructions` exits 2 with ``mrw: --instructions is not a flag; the subcommand is
`mrw instructions` (see: mrw --help)`` on stderr and nothing on stdout, for every spelling the parser
reads as that name and for every exact subcommand name at every level, while every other usage error
keeps its v1.31.0 wording.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `usageError` (`:192-194`) asks `subcommandForFlag(cmd, err)` first; `subcommandForFlag` (new, beside it) cuts the exact prefix `flag provided but not defined: -` from `err.Error()` and returns that name as reported together with `cmd.Command(name).Name`, or two empty strings — the flag is echoed as the caller's name, never the canonical one, so an alias is not reported as a different flag |
| `cmd/mrw/usage097_test.go` | add | the two tests below |
| `scripts/contract.sh` | edit | §188, placed after §187 |

**What selects it:** `installUsageErrors` (`cmd/mrw/main.go:185-190`) already sets `usageError` as
every command's `OnUsageError`, and urfave calls it on the command whose parse failed
(`command_run.go:189-193`). No new hook, flag or caller: deleting the one call in `usageError` returns
today's message, which the named test and §188 both catch.

## Ordered Steps

1. [S1] Write the two tests below in `cmd/mrw/usage097_test.go` (reusing `runSplit` from
   `usage078_test.go`) and confirm `TestAFlagNamedForASubcommandNamesTheSubcommand` RED at `25acdc1`
   (it gets `flag provided but not defined: -instructions`); `TestAnUnknownFlagNamingNoSubcommandIsRefusedAsBefore`
   passes there and must stay green. [proof: mutation]
2. [S2] Add `subcommandForFlag` and the branch in `usageError`: when it returns a match, the error is
   `--<flag> is not a flag; the subcommand is `<cmd.FullName()> <child.Name>` (see: <cmd.FullName()> --help)`
   at `exitUsage`, `<flag>` being the name as urfave reported it (an alias stays the alias);
   otherwise today's `fmt.Sprintf("%v (see: %s --help)", …)` unchanged. Doc comments
   say why the text is the handle (urfave v3.11.0 returns plain `fmt.Errorf`, `command_parse.go:208`,
   `:219`, and reads it back the same way in `flagFromError`, `:14-23`). [proof: mutation]
   Mutants to kill: the branch removed (today's line); a prefix match over `cmd.Commands`; the lookup
   on `cmd.Root()` instead of `cmd`; the `(see: … --help)` pointer dropped from the named sentence;
   the name taken from any usage error rather than only the not-defined prefix; the child's canonical
   name echoed as the flag (`--i` reported as `--inner`).
3. [S3] Contract §188, driving `$MRW` under `fixture`: `mrw --instructions` exits 2, empty stdout,
   stderr holds the exact named sentence; paired, `mrw --bogus188` exits 2 with
   `flag provided but not defined: -bogus188 (see: mrw --help)` and not `is not a flag`;
   `mrw check --stats` exits 2 with today's `(see: mrw check --help)` line; `mrw '--instructions= '`
   exits 2 with ADR-069's `ends in whitespace` refusal (ordering); and `mrw instructions` exits 0
   with a non-empty stdout, so the named command is the one that works. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAFlagNamedForASubcommandNamesTheSubcommand|TestAnUnknownFlagNamingNoSubcommandIsRefusedAsBefore|TestAUsageErrorWritesNothingToStdout|TestMcpTakesNoArguments|TestAnAttachedFlagValueWithTrailingSpaceIsRefused|TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator|TestEverySubcommandReachesTheAgentFacingGuide' -v 2>&1 | tee "$out" \
  && missing=$(for t in TestAFlagNamedForASubcommandNamesTheSubcommand TestAnUnknownFlagNamingNoSubcommandIsRefusedAsBefore TestAUsageErrorWritesNothingToStdout TestMcpTakesNoArguments TestAnAttachedFlagValueWithTrailingSpaceIsRefused TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator TestEverySubcommandReachesTheAgentFacingGuide; do grep -qE "^--- PASS: $t \(" "$out" || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 188\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

The named test carries the verdict: it is red on `25acdc1`, and the regression tests beside it cannot
satisfy it. The PASS loop names every test in the run, so a regression test renamed away — which
`-run` skips as "no tests to run", exit 0 — fails the fence instead of passing it. The log is a fresh
`mktemp` file, so a concurrent run of this fence in another checkout cannot supply its PASS lines.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFlagNamedForASubcommandNamesTheSubcommand` | `cmd/mrw/usage097_test.go` | through `rootCommand()`: `--instructions`, `-instructions`, `--instructions=x`, `'--instructions '` and `--instructions read` each exit 2, write nothing to stdout, and return exactly ``--instructions is not a flag; the subcommand is `mrw instructions` (see: mrw --help)``; `--seen` names `mrw seen`; on a test-built tree `mrw outer inner` with `installUsageErrors` applied and `ExitErrHandler: func(context.Context, *cli.Command, error) {}` set on its root as `rootCommand` does (without it urfave's `HandleExitCoder` calls `os.Exit` and takes the test binary down, `cmd/mrw/main.go:129-134`), `mrw outer --inner` names `mrw outer inner` with `(see: mrw outer --help)` — a child of the command given the flag, below the root; `inner` also carries the alias `i`, and `mrw outer --i` returns exactly ``--i is not a flag; the subcommand is `mrw outer inner` (see: mrw outer --help)`` — the flag as typed, the child by its name; `mrw instructions` still exits 0 | — | S1, S2 |
| `TestAnUnknownFlagNamingNoSubcommandIsRefusedAsBefore` | `cmd/mrw/usage097_test.go` | `--bogus`, `--instruction`, `--instructionsx`, `--Instructions`, `check --stats`, `read --instructions` and `iter --add` each exit 2 with nothing on stdout and exactly `flag provided but not defined: -<name> (see: <command path> --help)`, never `is not a flag`; `--help=x` keeps its `invalid value` wording; `--help`, `-h`, `--version` and `-v` exit 0; `refusePaddedFlagValues(rootCommand(), []string{"--instructions= "})` is ADR-069's `ends in whitespace` refusal | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `subcommandForFlag` and the branch in `usageError` |
| 2 — something selects it | `installUsageErrors` sets `usageError` on every command; both tests drive it through `rootCommand()` and §188 through the built binary |
| 3 — the caller can discover it | the refusal itself names the subcommand; T2 adds the clause to README.md and AGENTS.md |
| 4 — it is used | telemetry is refused (ADR-009); the evidence for building it is the 2026-09-29 observation in the record's Context |

## Mutation Log
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · the named branch removed: every usage error takes the v1.31.0 wording · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:the named sentence
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · a prefix match over cmd.Commands: --instruction would name mrw instructions · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:exact-name lookup
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · the lookup on the root, not the command given the flag: read --instructions would name a root subcommand · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:the command's own children
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · the (see: … --help) pointer dropped from the named sentence · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:the help pointer
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · the name taken from any usage error text, not only the not-defined prefix · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:the unchanged sentence
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · the child canonical name echoed as the flag: --i reported as --inner · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:the named sentence
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · the ADR-069 padded-value guard disabled: --instructions= would reach the parser · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:the padded refusal runs first
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · cmd/mrw not gofmt-clean · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:the tree is gofmt-clean
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package (internal/lines) changed against the merge-base · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · covers:no engine package changes

## Invariants

- Exit 2 for every usage error; nothing on stdout (ADR-078, `TestAUsageErrorWritesNothingToStdout`).
- Every usage error still names `(see: <command path> --help)`.
- Nothing refused before is accepted: the named subcommand is not run.
- ADR-069's padded-value refusal still comes first (`TestAnAttachedFlagValueWithTrailingSpaceIsRefused`).
- `internal/*` unchanged; `go.mod` one requirement.

## Risks

- urfave's error text is the only handle; a reworded message on upgrade falls through to today's
  wording, and the named test goes red on that upgrade PR.
- Case: urfave's `Command(name)` is exact (`HasName` is `slices.Contains`, `command.go:246-248`), and
  `mrw Instructions` answers `unknown command`, exit 2 (observed 2026-09-29, v1.31.0); the unchanged
  test pins `--Instructions` so a case-folding lookup added later goes red.

## Stop Condition

Stop and ask if an existing test must change to pass, if `TestAFlagNamedForASubcommandNamesTheSubcommand`
is not red at `25acdc1`, or if M reverses Decision 5 (the `(see: … --help)` pointer) at acceptance —
then the named test's expected sentence and §188's grep change together.

## Out of Scope

- README.md and AGENTS.md — T2.
- Siblings, `iter` verbs, prefixes and near-misses — the record's Out of Scope.

## Verification Log
- 2026-09-29 · 3582232* · exit 1 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:380 · test-lock-sha256:d3ef4eea61c5da364e007e7fe3c8ba496f8a34dcd1024656222361d603db5dcc · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdXNhZ2UwOTdfdGVzdC5nbwlUZXN0QUZsYWdOYW1lZEZvckFTdWJjb21tYW5kTmFtZXNUaGVTdWJjb21tYW5kCThkODdjOGViM2I1MTJmMjk1ZmRiYzNlYWI5MjMwOTIxMzg2YmRiMDg2YzdlM2IyZDJlN2NmZWMyMGI1MjM2NjgKYm9keQljbWQvbXJ3L3VzYWdlMDk3X3Rlc3QuZ28JVGVzdEFuVW5rbm93bkZsYWdOYW1pbmdOb1N1YmNvbW1hbmRJc1JlZnVzZWRBc0JlZm9yZQkyNjBlZGNlYmE3ZjIxOWE4MjhjMjkxOGIzMWQ3ZmE4ZGY4OGFlOTY3M2EyNDZjYWNjNjhiYmIzODFlODZlODI0
  ```
  --- last 10 line(s) of stdout (of 25 after folding 25 raw)
      usage097_test.go:23: ["--instructions" "read"]: err flag provided but not defined: -instructions (see: mrw --help) (exit 2), stdout ""; want exit 2 and "--instructions is not a flag; the subcommand is `mrw instructions` (see: mrw --help)"
      usage097_test.go:27: --seen: flag provided but not defined: -seen (see: mrw --help)
      usage097_test.go:37: outer --inner: err flag provided but not defined: -inner (see: mrw outer --help), want "--inner is not a flag; the subcommand is `mrw outer inner` (see: mrw outer --help)"
      usage097_test.go:37: outer --i: err flag provided but not defined: -i (see: mrw outer --help), want "--i is not a flag; the subcommand is `mrw outer inner` (see: mrw outer --help)"
  --- FAIL: TestAFlagNamedForASubcommandNamesTheSubcommand (0.00s)
  === RUN   TestAnUnknownFlagNamingNoSubcommandIsRefusedAsBefore
  --- PASS: TestAnUnknownFlagNamingNoSubcommandIsRefusedAsBefore (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.117s
  FAIL
  ```
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:336
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:349
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:359
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:354
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:352
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:353
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:520
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:369
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:373
- 2026-09-29 · 3582232* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:59b27e8c92e8e3a5d6a518c93eb57a65367af4aa525ca833926591d99ef54cb5 · ms:0 · test-lock-sha256:da4b5d1f00fab13666829c3c340f45faa6eb8addf116142d2d619516eee37972 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdXNhZ2UwOTdfdGVzdC5nbwlUZXN0QUZsYWdOYW1lZEZvckFTdWJjb21tYW5kTmFtZXNUaGVTdWJjb21tYW5kCThkODdjOGViM2I1MTJmMjk1ZmRiYzNlYWI5MjMwOTIxMzg2YmRiMDg2YzdlM2IyZDJlN2NmZWMyMGI1MjM2NjgKYm9keQljbWQvbXJ3L3VzYWdlMDk3X3Rlc3QuZ28JVGVzdEFuVW5rbm93bkZsYWdOYW1pbmdOb1N1YmNvbW1hbmRJc1JlZnVzZWRBc0JlZm9yZQljNmJjMDk1MjQ3MTk3ZjFkOWIyZTI3OTI5ZDc3NzExNDE2NmUyODdlNTI3ZDhhZjNkNTcyNmViNzgzNmVkNjhk · test-lock-kind:replace
