# ADR-011 Tasks

Implementation tasks for ADR-011: the MCP server tells a host what it is, and what it will return.
See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers. This
README is a derived index — when it disagrees with a task file, the task file wins and the README
must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |

Forced rather than chosen: T3 enforces the limit T2 advertises, and a limit enforced before it is
declared is a refusal the caller had no way to anticipate. T1 is first because it is the only defect
here that a user meets without doing anything unusual — every other gap needs a host to look for
something we do not say.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Bind the server to the checkout the host meant | pending | — | 4 named `--- PASS:` lines, `# 40.` in `contract.sh`, `CLAUDE_PROJECT_DIR` in README, both engine clauses, `./scripts/contract.sh` |
| T2 | Declare what each tool is, and what it returns | pending | — | 5 named `--- PASS:` lines, `# 41.` in `contract.sh`, both engine clauses, `./scripts/contract.sh` |
| T1 | Bind the server to the checkout the host meant | done | — | 4 named `--- PASS:` lines, `# 40.` in `contract.sh`, `CLAUDE_PROJECT_DIR` in README, both engine clauses, `./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `mcp.ResolveRoot` | T2 | T1 before T2 — T2's conformance tests call tools against a resolved root |
| T2 | the tool descriptor set, `mcp.MaxResultChars` | T3 | T2 before T3 — T3 enforces the constant T2 declares |

## Notes

- **Run every clause of every fence separately before writing a line of its task**, and record the
  counts. Running the fence as a whole is necessary and not sufficient: a fence that is red for an
  honest reason hides a clause that was green from the day it was written, and this repository has
  produced four of those — the last inside a task file whose own prose claimed the check had been
  done. Each task file here records the zero-hit counts rather than asserting them.
- **`outputSchema` was not usable as a fence token.** It already appears once, in a `tools.go`
  comment saying we do not declare one. T2 uses the Go identifier `OutputSchema` instead. That near
  miss is why the tokens are counted rather than chosen by eye.
- **Confirm §40, §41 and §42 are unused** before relying on those clauses. The highest section is 39
  as of `4431d15`.
- **The go/no-go conditions are in every fence, in both forms.** `git status --porcelain
  --untracked-files=all` sees an untracked new engine file; a merge-base `git diff` sees a committed
  change; neither sees what the other does. If either fails, the task is withdrawn rather than
  shipped — that is a real outcome, and T3 is the task most likely to reach it.
