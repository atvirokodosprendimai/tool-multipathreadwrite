# ADR-088: The codebase is analysed on every commit, and production holds nothing only a test reaches

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — "no dead code, no features in tests only", then "create claude rule to scan for dead code after commits. use available tools for that, if no tool available - install them so that our codebase would be analysed for static code, linting, dead code, linting, etc."
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-012, ADR-019, ADR-022, ADR-023, ADR-037, ADR-039, ADR-082, docs/adr/BACKLOG.md
**Governs:** `.golangci.yml`, `scripts/static.sh`, `.claude/hooks/static-after-commit.py`, `.claude/rules/static-analysis.md`, `.claude/settings.json`, `.github/workflows/ci.yml`, `internal/check/shell.go`, `internal/check/check.go`, `internal/mcp/schema.go`, `internal/mcp/instructions.go`
**Enforced-by:** `cmd/mrw/statichook_test.go::TestTheStaticHookReportsAfterACommit`
**Served-path change:** one — `mrw read` whose answer cannot be written (a full disk, `/dev/full`) now exits 2 and records nothing, where it recorded the lines and exited 0; everything else a caller sees is unchanged: `%w` formats as `%v` did, and the moved identifiers are unexported.

## Context

**What prompted it.** A dead-code pass on 2026-09-27 (`deadcode ./...` at `c304581`) found no dead
code but five production identifiers that only tests reach, and nothing ran it except by hand. The
repository gated on `gofmt`, `go vet`, `go test` and the contract; no linter, no dead-code check and
no vulnerability scan ran anywhere.

**The class**, enumerated 2026-09-27 at `d352ba7`:
- *production reachable only from tests*: `deadcode ./...` plus
  `staticcheck -tests=false -checks U1000 ./...` — `checkSignals`, `Shell`, `readSchema`,
  `readDescriptions`, `maxInstructionsChars`. Five.
- *lint findings*: `golangci-lint run` (v2.14.0) with `.golangci.yml` — 19 once repeated findings are
  no longer collapsed: errorlint 7 (production), nilerr 8, unparam 2, wastedassign 1, copyloopvar 1,
  staticcheck 3 (SA4000 in `seen_test.go`, ST1008 twice in test helpers).
- *vulnerabilities*: `govulncheck ./...` (v1.8.0) — none.

## Existing Primitives Audit

- `.claude/settings.json` already registers one PostToolUse hook (ADR-022, `rules-on-read.py`); the
  new hook sits beside it and follows its rules — exit 0 always, bounded time.
- `staticcheck` and `deadcode` were installed; `golangci-lint` 2.14.0 (Homebrew) and `govulncheck`
  v1.8.0 were installed for this record. `scripts/static.sh` runs the Go tools with `go run
  <module>@<version>`, which leaves `go.mod` untouched (verified), so local and CI use one version.

## Decision

1. **Nothing in production exists only for a test.**
   - `Shell()` returns `(sh, pathDir, ok)` and `Run` calls it, as ADR-082 wrote it; the wrapper goes.
   - `checkSignals` moves into `group_unix_test.go`; `Run` never used it.
   - `readSchema`/`readDescriptions` move into `schema_test.go` (ADR-023 forbids publishing them),
     and `TestTheReadReceiptMatchesItsSchema` compares a real `mrw_read` receipt with that table in
     both directions, so the table checks production rather than itself.
   - `maxInstructionsChars` moves into `instructions_test.go`; the 4096 bound was always an assertion
     (contract rows on the built binary, ADR-037), never runtime behaviour. The two done fences that
     grep for its literal (ADR-019 T3, ADR-039 T3) are re-fenced to the new file.
2. **`scripts/static.sh` is the static gate**: `gofmt -l`, `go vet`, `golangci-lint run`, `deadcode`
   (output must be empty), staticcheck U1000 with `-tests=false`, and `govulncheck`. Any finding exits
   1; a missing golangci-lint names how to install it.
3. **`.golangci.yml`** enables the standard set plus linters that find defects (errorlint, nilerr,
   unparam, wastedassign, copyloopvar, misspell, unconvert, usestdlibvars, durationcheck) and shows
   every finding (no per-linter cap). errcheck stays whole: the one function exclusion is `fmt.Fprint*`,
   whose error reaches a checked `Flush`, cannot occur on an in-memory buffer, or has nowhere to go on
   stderr. Every other discarded error is written `_ =` at its site, so the discard is a visible choice,
   and a deliberate swallow of a non-nil error is `//nolint:nilerr // <why>`. Tests are exempt from
   errorlint, unparam, errcheck and ST1008 (fixed test arguments, unwrapped errors, cleanup that cannot
   change a verdict), and staticcheck's QF refactoring hints are off.
4. **After every commit, the hook runs it.** `.claude/hooks/static-after-commit.py` (PostToolUse on
   Bash) runs `scripts/static.sh` when the command ran `git commit` (read quote-aware) and HEAD's
   newest reflog entry is a commit made in the last 15 minutes that no session has analysed yet — a
   claim file per commit makes that atomic, so another session's command cannot consume it. The
   script runs in its own process group, killed whole at 280 s. `.claude/rules/static-analysis.md` says
   what to do with the verdict, and says it for an agent without hooks too.
5. **CI runs `scripts/static.sh`** in the `test` job, with golangci-lint installed at the same version.
6. **A read whose answer did not reach the caller records nothing.** Found while removing the blanket
   errcheck exclusion (the Codex review of #259): `mrw read` recorded the served lines, then flushed its
   buffered answer on return, unchecked, so a failed write licensed lines nobody saw — ADR-002 inverted.
   The answer is flushed and checked before `seen.Record`; a failure exits 2 and names it.

## Alternatives Considered

- **Enable gosec, revive and gocritic too.** Rejected: 531 gosec findings are file permissions and
  file inclusion (G306, G304, G301) in a tool whose job is reading and writing files the caller names;
  revive's are `redefines-builtin-id` style; gocritic's `mapKey` flags deliberate padded-path test
  data. Hundreds of suppressions would bury the findings that matter.
- **Run the analysers in `scripts/contract.sh`.** Rejected: a contract row drives the built binary
  (`contract.md`); static analysis of the source is not a promise the binary makes.
- **A git `post-commit` hook.** Rejected: it would not reach the model, which is who acts on a finding,
  and the quality-harness already owns this checkout's git hooks.

## Component / Boundary Impact

`internal/check`, `internal/mcp`, `internal/read`, `internal/state`, `internal/rooted`,
`internal/authoring`, test files; repository tooling (`scripts/`, `.claude/`, CI).

## Wiring & Contract Changes

One exit code: a read whose answer cannot be written exits 2 (contract §171). CI gains a step;
`CONTRIBUTING.md` and `AGENTS.md` list the new gate.

## Inter-task Contracts

T2's `scripts/static.sh` runs the deadcode and U1000 checks T1 makes clean; T3's hook runs T2's script.

## Implementation

See `tasks/`.

## Consequences

- A finding reaches the model after the commit that introduced it, not at the next manual pass.
- Every commit in a Claude Code session pays one analysis: 8 s for `scripts/static.sh` with a warm
  module cache, measured 2026-09-27 at load 28 on 10 cores. The first run also downloads the tools.
  A 280 s bound caps it.

## Out of Scope

- gosec, revive, gocritic (permanent: boundary: the Alternatives say why; revisit when a finding class they alone catch bites)
- Windows-only files in the analysis (permanent: boundary: CI analyses on Linux; a `GOOS=windows` deadcode run matched on 2026-09-27)
- Exported identifiers used only by tests that are not functions (permanent: boundary: neither deadcode nor U1000 reports them; none exist on 2026-09-27)
- A contract row for the analysis itself (permanent: boundary: static analysis is not a promise the binary makes; T4's read promise has §171)
- A failed receipt write after a write has landed (permanent: boundary: the tree changed either way and the exit code carries the verdict; the receipt is written `_ =` and says so)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A tool release changes findings under a pinned config | Medium | Low | versions pinned in `scripts/static.sh` and CI |
| The hook slows a commit turn | Medium | Low | runs only when HEAD moved on a commit; 280 s bound |
| `govulncheck` needs the network | Low | Low | a failure is reported as the tool's, not as a vulnerability |

## Rollback

Revert the tasks; the hook entry is one block in `.claude/settings.json`.

## Follow-ups

None.
