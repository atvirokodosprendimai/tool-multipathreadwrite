# ADR-139: `mrw write --create PATH` makes one file from its standard input

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — "/loop continue delivering, end to end, no dead code", on the survey's second look-mode item ("a throwaway file costs a create plan and a hand-counted `body=`", 6 of 14 sessions as the author's 2026-10-09 synthesis recorded the replies)
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-070, ADR-075, ADR-054, docs/adr/BACKLOG.md
**Invalidates:** None — a new flag that compiles to an ordinary create plan; the plan format, the engine and every guard are unchanged
**Governs:** `internal/ingest/create.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/ingest/create139_test.go::TestCompileCreateMakesACreatePlanOfStdin`
**Served-path change:** `mrw write --create PATH`, with the file's content on standard input, creates PATH exactly as the plan `@@ PATH 0 create` with those lines as its body would: all-or-nothing, refused if PATH exists, the project's check run by the same rule, the same receipt and exit codes. No hand-counted `body=` and no plan. Without the flag nothing changes.

## Context

Making a new file through mrw takes a plan: a header line, `body=N` when the body is empty or holds a line beginning `@@`, and the body. Six of the 14 survey sessions said they avoid it for a throwaway file (a fixture, a scratch script) and write it with a shell redirect, which bypasses the ledger, the lock and the check. The failure the plan format guards against, a body miscounted or cut at a line starting `@@`, is the one a caller meets most often when the content is itself a document.

**The mechanism already exists.** `--format=apply_patch` and `--format=search_replace` compile a foreign document into a native plan before `plan.Parse` and `Apply` run unchanged (`internal/ingest`). A file's content is the simplest document there is.

**What a create makes, so the flag promises no more.** Measured 2026-10-10 on this tree: a `create` plan writes each body line followed by `\n`, so content with no final newline gains one and CRLF lines come out LF; an empty body makes an empty file (`body=0`). `--create` makes what a create plan makes of the same lines, and says so; it is not a byte-exact copy of standard input.

**Audit of the class** — *a path that turns a caller's document into a plan*: `mrw read --grep 'plan.Parse\(' --exclude '*_test.go' cmd internal` names the three parse sites in the write action (`cmd/mrw/main.go`: the native plan, `apply_patch`, `search_replace`) and the plan loaders of `check`/`iter`, which are other commands. The write action's switch is the one place; `--create` is a fourth case of it.

## Existing Primitives Audit

- **`ingest.emit`** — writes a hunk header and body, with `body=N` for an empty create, `raw=true` for a body line beginning `@@`, and the quoting a path with a space needs (ADR-070); reused, not copied.
- **`lines.Split` / `lines.Unsplittable`** — split text by its own terminator and name content mrw cannot treat as lines (UTF-16, a NUL), which a create plan cannot carry.
- **The `--format` switch in the write action** — where a document becomes hunks; `--create` is selected there.
- **`rooted.IsRooted`** — a path given to a plan is relative to the root; `--create` holds to it.

## Decision

1. **`mrw write --create PATH`** reads all of standard input and creates `PATH`, relative to the root, through the same all-or-nothing apply, lock, default check and receipt as a plan. It takes no PLAN argument and no `--format`: either is a usage error, exit 2, and so is an empty PATH or a PATH that is not relative to the root.
2. **The content is compiled to one create hunk** by `ingest.CompileCreate(path, content)`: the content split into lines by `lines.Split`, emitted by `emit`. A line beginning `@@`, an empty content and a path with a space are carried by the existing rules. Content mrw cannot split into lines (`lines.Unsplittable`: a UTF-16 or UTF-32 mark, a NUL early on) is refused, exit 2, saying so, since a create plan cannot carry it either.
3. **The file is what a create plan makes of those lines**: each line ends in `\n`, so a missing final newline is added and CRLF becomes LF; empty input makes an empty file. A PATH that exists, or is a link out of the root, is refused by the apply as for any create.
4. **`mrw_write` is unchanged**: it takes a plan, and its callers have a file tool.

## Alternatives Considered

- **A new `mrw create` subcommand** — rejected: `write` is the one writer path that holds the checkout's lock, runs the check and keeps the receipt; a second command would copy all three.
- **`body=@file` from outside the root** — rejected here: reading a file the caller names outside the root is a boundary decision of its own (ADR-007); the content a caller has is on standard input.
- **Byte-exact copy of standard input** — rejected: it needs a create that does not normalise line endings, which is an engine change to a promise ADR-005 §3 settled for edits; the flag stays what a plan is.
- **Leave it (a heredoc plan works)** — rejected: six sessions said it costs enough that they bypass mrw, which is the failure the tool exists to prevent.

## Component / Boundary Impact

`internal/ingest` (one new file with one function) and `cmd/mrw` (one flag, one switch case). `apply`, `plan`, `seen`, `rooted`, `state`, `check`, `writer` and `mcp` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write --create PATH` | new flag | T1 | CLI callers |
| `ingest.CompileCreate` | new function | T1 | `cmd/mrw` |
| `scripts/contract.sh` | §240 | T1 | CI Linux |
| `AGENTS.md`, `README.md` | the write section names the flag and what it makes | T1 | every agent |
| `mrw instructions` | names `--create` among the write flags a contract row requires | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a new file is one command, through the ledger, the lock and the check; a document that contains `@@` lines is carried without a hand-set `raw=true`.
- **Negative:** the file is not a byte copy of standard input (final newline, CRLF); a caller who needs exact bytes writes with their own tool and says so.
- **Neutral:** no engine change; the receipt and exit codes are a plan's.

## Out of Scope

- Byte-exact content (permanent: boundary: the create plan's rule, ADR-005 §3; an engine change)
- `--create` on `mrw_write` (permanent: boundary: MCP hosts have a file tool and the tool takes a plan)
- Creating several files in one call (permanent: boundary: that is a plan, which already does it)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a document line beginning `@@` ends the body early | Low | a truncated file | `emit` declares `body=N raw=true`; `TestCompileCreateMakesACreatePlanOfStdin` carries one |
| a caller reads the flag as byte-exact | Medium | a missing final newline surprises | the help, AGENTS.md and the receipt-free refusal text say "what a create plan makes of these lines" |
| stdin is a terminal and blocks | Low | an interactive hang | the same as `mrw write -`; the help says the content is read from stdin |

## Rollback

Revert T1: the flag is gone and a create takes a plan. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up.
