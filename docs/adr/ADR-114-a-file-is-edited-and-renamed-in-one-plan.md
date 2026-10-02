# ADR-114: A file is edited and renamed in one plan

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — "Engine: edit then rename (Recommended)", the answer to "Gap item 5 (apply_patch `*** Move to:` with hunks): the engine refuses an edit and a rename of the same file in one plan. Which way should ADR-114 go?", on the gap list he answered "all" on 2026-10-01. The record's text was drafted after that answer and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-02
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-051, ADR-066, ADR-102, ADR-106
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/ingest/applypatch.go`, `AGENTS.md`, `README.md`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/editrename114_test.go::TestAFileIsEditedAndRenamedInOnePlan`
**Served-path change:** A plan may carry line edits on a file and one `rename` of it: the edited content lands at the destination, in the same all-or-nothing plan, and the receipt names the source removed and the destination written. `--format=apply_patch` compiles `*** Update File:` + `*** Move to:` with hunks into such a plan instead of refusing it.

## Context

Codex's apply_patch grammar moves and edits a file in one section — `*** Update File: a`, `*** Move to: b`, then hunks — and models emit it. mrw refused it twice over: the compiler (`internal/ingest/applypatch.go:90,119,136`, "Move to with hunks is not compiled this slice"), and under it the engine, which refuses any path op beside another hunk on the same path (`internal/apply/apply.go:1025-1030`, "unlink/rename cannot mix with other hunks"). A caller had to send two plans, and the pair is not atomic: the first can land and the second be refused.

**Audit of the class** — *a plan shape refused only because two ops share a path, or because a grammar's move carries hunks*: the engine's one refusal at `apply.go:1027`, found with `mrw read --grep 'cannot mix' internal`, and the compiler's three at `internal/ingest/applypatch.go:90,119,136`, found with `mrw read --grep 'Move to with hunks' internal`. The engine's covers unlink beside edits (meaningless: the file is removed), two path ops on one file (ambiguous), and rename beside edits (meaningful: this record); no test pins it. The compiler's is pinned twice, by `TestCompileMoveToIsRename` (`internal/ingest/applypatch_unlink_test.go:81-86`) and by contract §99's second pair (`scripts/contract.sh`), and both change with T2.

## Existing Primitives Audit

- **The content commit** (`apply.go`, the staged-temp rename onto the target, with `changedSince` before it) — lands the edited source.
- **`commitPathOps`'s rename** (`pathop.go`) — moves it, with the undo that reverses completed renames (ADR-066).
- **`planPathOp`** — validates the rename, including that the whole file was read.

## Decision

1. **The engine accepts line edits plus exactly one `rename` on a file.** Unlink beside edits, and two path ops on one file, stay refused with the existing words. The rename is validated by `planPathOp` as today (the whole file read, the destination free); the line hunks are validated as any edit, against the original file.
2. **It commits as the edit, then the rename.** Content commits before path ops already, so the source's edit lands through the ordinary staged rename, and the rename then moves the edited file. The rename carries the source's mode with it, as a rename does.
3. **One record per path.** The rename rewrites the source's record in place — removed, renamed to the destination — and appends the destination's record with the edited content's sha and line count; a dry run lists the source once. If a later path op fails and the undo reverses this rename, the source's record is restored to the edit that landed, not dropped, so the receipt and the ledger say the source holds the edited content.
4. **apply_patch compiles it.** `*** Move to:` inside an `*** Update File:` section with hunks emits the update hunks against the source and one `rename` to the destination; a section with only `Move to` emits the rename alone, as now.

## Alternatives Considered

- **Compile to `create dest` + `unlink src` in ingest** — rejected by Zy's answer: the edited lines would skip the per-line read guard, the file's mode would not travel, and the receipt would read as two unrelated ops.
- **Stage the edited content at the destination and unlink the source** — rejected: a new commit path with its own undo, where the existing edit and rename each already have one.
- **Keep refusing** — rejected by Zy's answer.

## Component / Boundary Impact

This record owns `internal/apply` (an engine package) and `internal/ingest`. `internal/read`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` and `internal/rooted` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a native plan | line edits + one rename on a file apply | T1 | CLI, MCP |
| `--format=apply_patch` / `format: apply_patch` | Move to with hunks compiles | T2 | CLI, MCP |
| AGENTS.md, README | the ops paragraph says so | T2 | readers |
| `scripts/contract.sh` | §214 (T2) | T2 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the engine accepts edit + rename | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2, in one pull request.

## Consequences

- **Positive:** a move-and-edit is one atomic plan on both surfaces and in apply_patch.
- **Negative:** the engine's commit gains a record rewrite and an undo branch.
- **Neutral:** every plan the engine accepted before applies as before.

## Out of Scope

- Unlink beside edits, and two path ops on one file (permanent: boundary: an unlink removes what the edits change, and two path ops on one file have no single meaning)
- Aider SEARCH/REPLACE (permanent: boundary: its grammar has no move)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a duplicate record for the source reaches the ledger and licenses a path that is gone | Medium | High | the record is rewritten in place; a test asserts one record per path and the ledger after |
| a failure after the edit landed reports the source wrong | Low | Medium | the undo restores the edited record; a test injects a later rename failure |

## Rollback

Revert T2, then T1. No persistent state or receipt key changes.

## Follow-ups

- None — the record carries no open follow-up.
