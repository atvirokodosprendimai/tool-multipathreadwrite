# Task ADR-011-T1: Bind the server to the checkout the host meant

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `mcp.ResolveRoot(explicit string, lookup func(string) (string, bool)) (string, string)`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the CLAUDE_PROJECT_DIR lookup`, `the precedence of an explicit --root`, `the stderr announcement`

## Goal

`mrw mcp` launched by a host with no `--root` serves the project the host is working in, not whatever
working directory it inherited, and says on stderr which tree it chose.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/root.go` | add | `ResolveRoot` and its rules, so the precedence is one testable function rather than an `if` in a command |
| `internal/mcp/root_test.go` | add | its tests |
| `cmd/mrw/main.go` | edit | **THE CALL SITE.** `mcpCmd` calls `ResolveRoot` instead of taking `--root` verbatim |
| `README.md` | edit | the config block loses the `--root` advice it currently gives and gains what actually happens |
| `scripts/contract.sh` | edit | §40 — launch the real binary from a foreign cwd with `CLAUDE_PROJECT_DIR` set, and assert which tree it served |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): an explicit root wins over the environment; the
   environment is used when no root is given; a `CLAUDE_PROJECT_DIR` naming a non-directory or an
   empty string falls back to the working directory rather than failing; the function reports WHICH
   source it used so the caller can say so. [proof: acceptance]
2. [S2] Implement `ResolveRoot`, returning the root AND the source that produced it. Returning the
   source is not decoration: the announcement in S4 must not claim an environment origin for a
   fallback, and a boolean would make that mistake representable. [proof: mutation]
3. [S3] Call it from `mcpCmd`. `cmd/mrw` decides precedence for every other flag, and this is the one
   place a subcommand's root is not simply the global flag — so it is called here, not inside
   `Serve`, which must stay startable with any root. [proof: mutation]
4. [S4] Announce the resolved root on **stderr** at startup, naming the source. The spec forbids
   anything on stdout that is not an MCP message, and a server that silently binds somewhere
   unexpected is the failure this task exists to prevent — a host's log is where that gets noticed.
   §40 asserts the line is on stderr and names the chosen tree. [proof: acceptance]
5. [S5] Correct the README config block. It currently tells a reader to pass `--root` for a specific
   checkout; that advice is what the environment variable removes the need for, and a block a reader
   must adapt is a block that gets adapted wrongly. [proof: acceptance]
6. [S6] Add contract §40: run the built binary with cwd set to a DIFFERENT directory from
   `CLAUDE_PROJECT_DIR`, ask it to read a file that exists in only one of them, and assert which one
   answered. Then repeat with an explicit `--root` and assert it wins. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr011-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr011-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestAnExplicitRootWinsOverTheEnvironment|TestTheProjectDirEnvironmentIsUsedWhenNoRootIsGiven|TestANonDirectoryProjectDirFallsBackToTheWorkingDirectory|TestResolveRootReportsWhichSourceItUsed)\b' /tmp/adr011-t1.out)" = "4" ] \
  && grep -q '^# 40\.' scripts/contract.sh \
  && grep -q 'CLAUDE_PROJECT_DIR' README.md \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was grepped for BEFORE this fence was written and returned **zero hits**: `ResolveRoot`
and the four test names across `internal/` and `cmd/`, `# 40.` in `contract.sh`, `CLAUDE_PROJECT_DIR`
in `README.md`. That check is the one whose absence produced four vacuous fences in this repository,
the last of them inside a task file that claimed to have performed it — so the counts are recorded
here rather than asserted.

The named-test count is deliberately not `-run`: a `-run` regex silently drops any name it does not
match. Both engine clauses are present because each sees what the other misses (ADR-010).

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnExplicitRootWinsOverTheEnvironment` | `internal/mcp/root_test.go` | S2 — precedence, the rule a user relies on to override a host | — | S1, S2 |
| `TestTheProjectDirEnvironmentIsUsedWhenNoRootIsGiven` | `internal/mcp/root_test.go` | S2 — the bug this task fixes | — | S1, S2 |
| `TestANonDirectoryProjectDirFallsBackToTheWorkingDirectory` | `internal/mcp/root_test.go` | S2 — a hostile or stale environment must not make the server unusable | — | S1, S2 |
| `TestResolveRootReportsWhichSourceItUsed` | `internal/mcp/root_test.go` | S2, S4 — the announcement cannot claim an origin the resolution did not have | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | `mcpCmd` calls `ResolveRoot` (S3); replacing the call with the bare flag makes §40 red, and §40 is inside the fence |
| 3 — the caller can discover it | the README config block, which is how anyone installs this — and §40 drives the binary the block describes |
| 4 — it is used | `mrw stats` counts plans per checkout, so a wrong root shows up as a tally that stays empty while work happens; nothing measures the resolution directly and this task does not add telemetry (ADR-009's premise) |

## Mutation Log

- 2026-09-03 · 7cfe742 · mutant killed · exit 1 · `internal/mcp/root.go` · S2: ignore the host environment, restoring the bug — the server binds to its inherited working directory and §40 must go red · acceptance-sha256:0fb76ff0f239719fbbdb7704f6047d06f82649f17fcf15ce9e9f026ce6a2ea65
- 2026-09-03 · dfcc2d8 · mutant killed · exit 1 · `cmd/mrw/main.go` · rung 2: bypass the call site so ResolveRoot is unreachable — the package would be built, tested and unused, which is how rooted.Descendable shipped dead · acceptance-sha256:0fb76ff0f239719fbbdb7704f6047d06f82649f17fcf15ce9e9f026ce6a2ea65

## Invariants

- An explicit `--root` is never overridden by the environment.
- Nothing is written to stdout by the resolution; the announcement is stderr only.
- ADR-006 confinement applies unchanged to whichever root is chosen — this task changes which tree is
  served, never what "inside the root" means.
- `go.mod` declares exactly one requirement, and the six engine directories are untouched.

## Risks

- An environment variable that names somewhere unexpected binds the server there silently. Mitigated
  by the stderr announcement and by explicit `--root` precedence; not by refusing, because a host
  that sets the variable is the authority on its own project.
- A test that sets a real environment variable leaks into siblings. Mitigated by `ResolveRoot` taking
  a lookup function rather than calling `os.LookupEnv` itself, which is also why it is testable
  without `t.Setenv`.

## Stop Condition

Stop if resolving the root requires the transport to make a REQUEST of the client — that is
`roots/list`, it is recorded in Out of Scope, and it is a different shape of change. An environment
lookup that fails should fall back, never negotiate.

## Out of Scope

- Reading `roots/list` from the client (deferred: docs/adr/BACKLOG.md)
- Any change to what confinement means inside a root (permanent: boundary: ADR-006 owns that, and this task must not have an opinion about it)

## Verification Log
- 2026-09-03 · 4e8f9ea* · exit 1 · `set -o pipefail …` · acceptance-sha256:0fb76ff0f239719fbbdb7704f6047d06f82649f17fcf15ce9e9f026ce6a2ea65 · ms:222
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/mcp/root_test.go:26:12: undefined: SourceFlag
  internal/mcp/root_test.go:27:41: undefined: SourceFlag
  internal/mcp/root_test.go:35:14: undefined: ResolveRoot
  internal/mcp/root_test.go:35:55: undefined: projectDirEnv
  internal/mcp/root_test.go:37:52: undefined: projectDirEnv
  internal/mcp/root_test.go:39:12: undefined: SourceProjectDir
  internal/mcp/root_test.go:40:41: undefined: SourceProjectDir
  internal/mcp/root_test.go:40:41: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-09-03 · 78d7c83 · exit 0 · `set -o pipefail …` · acceptance-sha256:0fb76ff0f239719fbbdb7704f6047d06f82649f17fcf15ce9e9f026ce6a2ea65 · ms:20209
- 2026-09-03 · 7cfe742 · exit 0 · `set -o pipefail …` · acceptance-sha256:0fb76ff0f239719fbbdb7704f6047d06f82649f17fcf15ce9e9f026ce6a2ea65 · ms:11602
- 2026-09-03 · dfcc2d8 · exit 0 · `set -o pipefail …` · acceptance-sha256:0fb76ff0f239719fbbdb7704f6047d06f82649f17fcf15ce9e9f026ce6a2ea65 · ms:12216
