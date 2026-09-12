# ADR-051: Foreign plan grammars compile to plan hunks

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"steal their ideas and apply to us where it matters and makes us ahead."*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-030, ADR-035, ADR-048, ADR-049, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `scripts/contract.sh`
**Enforced-by:** `internal/ingest/applypatch_test.go::TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing`
**Invalidates:** none — checked. ADR-001 rejected unified diff as the *native* plan; this record adds a compiler in front of that plan, it does not replace it.
**Served-path change:** `mrw write --format=apply_patch [PLAN|-]` compiles a Codex `apply_patch` document into `@@` hunks, then `plan.Parse` and `apply.Apply` run unchanged. Default `write` is still the native plan. A git patch is not an `apply_patch`.
**Notes:** M's quote is Accept of this steal — foreign grammars that compile to the atomic engine — not of Morph/streaming (ADR-049), syntax-aware write (ADR-048), cargo MCP tools (ADR-044), LOCALAPPDATA (ADR-050), or generating AGENTS.md (ADR-045). 4096 stays. 019 A stands.

## Context

**The class this record governs.** Every foreign edit grammar that is compiled into mrw plan hunks before Apply. Enumerated 2026-09-12 with

```
git ls-files cmd/mrw/main.go internal/plan/plan.go internal/apply/apply.go scripts/contract.sh
```

Four tracked files on the write path (1+1+1+1). The compiler package is new and is named by the tasks. Members left out: the MCP `plan` argument (deferred, receipt in BACKLOG), Aider SEARCH/REPLACE (deferred), git unified diffs (permanent: they are not this grammar).

A 2026-09-12 competitor scan (v1.13.0, `d7e39bd`) found Codex `apply_patch` closest on the job: one multi-file document, no line numbers, trained into the models that already emit it. OpenAI leaves atomicity to the harness. Codex-rs applies sequentially; a failed write can already have mutated the target (`delta.exact=false`). That sequential leak is the thing we do not copy.

ADR-001 already rejected unified diff as the native plan: context matching makes the failure mode fuzzy, and hunks are offset-relative. This record does not reopen that. The native document stays `@@`. A foreign document is compiled into that document, or it is refused.

## Existing Primitives Audit

- **`plan.Parse` + `apply.Apply`.** Reused unchanged. ADR-030's reachability paragraph says a fourth `apply.Input` construction site that is not fed by `plan.Parse` reopens that record. The compiler emits plan *text*; Parse is the only door.
- **`anchor=` (ADR-035).** Reused. A compiled multi-line `replace` carries `anchor=` taken from the first old-side line.
- **Per-line ledger (ADR-002).** Reused. Compile locates lines by matching the old side against the file on disk. That is an address, not a license. Apply still refuses an unread line.
- **`--force`.** Unchanged. It remains the escape hatch for an unread compile, not a new default.
- **Auto-detect from `*** Begin Patch`.** Audited and rejected: a git patch and an `apply_patch` share `@@` and `+/-` lines. Pretending they are the same is the fork this record closes.

## Decision

A foreign grammar compiles to the existing plan. Apply does not change. A failed hunk still writes nothing. An unread compiled line still refuses.

The first grammar is Codex `apply_patch` (`*** Begin Patch` / `*** Update File:` / `*** Add File:` / context ` ` / `-` / `+` / `*** End Patch`). The caller selects it with an explicit flag:

```
mrw write --format=apply_patch [PLAN|-]
```

`--format` defaults to `plan`, the native document. Unknown values are usage (exit 2). `--format=git` is usage and names that a git patch is not an `apply_patch`.

Compile-time context matching is location, not fuzzy apply:

1. The old side of each update hunk (context + minus lines, prefixes stripped) must match **exactly one** run of lines in the file. None or several is a compile refusal (exit 2), and nothing is written.
2. The compiled hunk is a `replace` of that run with the new side (context + plus), or `create` for `*** Add File:`.
3. Apply then runs. If those lines were never served, the hunk fails, siblings report `skip`, and the tree is unchanged (exit 1). That is the property a two-hunk document with one unread line must keep.

`*** Delete File:` and `*** Move to:` are refused this slice: mrw has no unlink op, and emptying a file is not a delete.

## Alternatives Considered

- **Auto-detect `*** Begin Patch` on stdin.** Rejected: a git patch can carry the same `@@` / `+/-` body. An explicit flag is the safer fork. M's brief named this fork and picked the flag.
- **Compile to `[]apply.Input` and skip Parse.** Rejected under ADR-030: Apply is a public entry point; the engine refuses what the parser refuses because every production site is fed by Parse. A fourth construction site reopens that record.
- **Make apply_patch the native plan.** Rejected by ADR-001's existing alternative: context matching is fuzzy as a *plan*, and hunks become offset-relative. We steal the grammar models already emit, not the sequential engine.
- **Fuzzy apply of a near-miss old side.** Rejected: that is applying lines nobody served, which is the thing ADR-002 exists to refuse.
- **File-level license (Claude Edit / Cursor StrReplace).** Rejected: we are already stricter, per line.

## Component / Boundary Impact

New package `internal/ingest` compiles a foreign document to plan text. `cmd/mrw` write selects it by `--format`. `internal/plan` and `internal/apply` do not change ownership. No architecture doc exists in this repository; this is a front door on the existing write path, not a new bounded context.

C4: ingest is a compiler in front of the existing write container. It reads files only to locate an old side. It writes nothing. Apply remains the single writer.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write --format` | new flag; `plan` (default) or `apply_patch` | `cmd/mrw/main.go` `writeCmd` | CLI callers; contract §82 |
| `write --help` | names `--format=apply_patch` and that a git patch is not one | `writeCmd` Usage / Description | PATH callers |
| contract §82 | two-hunk apply_patch, one unread line writes nothing; paired with the served case and the no-flag case | `scripts/contract.sh` | `adr-verify`, CI |
| `internal/ingest.CompileApplyPatch` | new | T1 | T2 (`writeCmd`) |

No MCP `format` argument this slice. No exit-code change: compile refusal is 2 (the document did not become a plan); an unread compiled hunk is 1 (the plan failed and nothing was written).

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `ingest.CompileApplyPatch` (T1) | T1 | T2 | No — T2 is the first caller |

## Implementation

See `docs/adr/ADR-051-foreign-plan-grammars-compile-to-plan-hunks/tasks/README.md`.

## Consequences

- **Positive:** a model that already emits `apply_patch` can feed mrw without dropping ADR-001–004. The sequential leak stays theirs.
- **Negative:** callers must pass `--format=apply_patch`. A document without the flag is still a bad native plan (exit 2).
- **Neutral:** compile reads the file to locate the old side. That is not a license and is not a target-syntax parse (ADR-048). Wrap-tail stays a read.

## Out of Scope

- Sequential apply that leaves a half-written file (permanent: boundary: that is the leak we do not copy; ADR-001 stands)
- File-level-only license (permanent: boundary: ADR-002 is per line)
- Morph / Relace / Instant Apply streaming (permanent: fact: ADR-049 waits for a size that hurts; citation: file `docs/adr/ADR-049-streaming-apply-waits-for-a-size-that-hurts.md:31`)
- Fuzzy apply of a near-miss old side (permanent: boundary: unread still refuses)
- Syntax-aware write / wrap-tail as a parser (permanent: fact: ADR-048 models no target syntax; citation: file `docs/adr/ADR-048-mrw-models-no-target-syntax.md:31`)
- Auto-detect vs explicit flag (permanent: boundary: this record picked the flag)
- Treating a git patch as an `apply_patch` (permanent: boundary: they share `@@` and they are not the same grammar)
- Aider SEARCH/REPLACE (deferred: docs/adr/BACKLOG.md)
- `*** Delete File:` / `*** Move to:` (deferred: docs/adr/BACKLOG.md)
- MCP `format` on `mrw_write` (deferred: docs/adr/BACKLOG.md)
- ast-grep-shaped `--grep` (deferred: docs/adr/BACKLOG.md)
- Raising `maxInstructionsChars` / 4096 (permanent: boundary: ADR-037's go/no-go; 4096 stays)
- Reopening ADR-019 B/C (permanent: fact: pick A stands; citation: file `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md:106`)
- Cargo MCP tools, LOCALAPPDATA, generate AGENTS.md (permanent: fact: ADR-044 / ADR-050 / ADR-045 already refused them; citation: file `docs/adr/BACKLOG.md:41`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A two-hunk unread fixture passes on parse-fail (exit 2) rather than hunk-fail (exit 1) | High | High — the mutant that ignores `--format` writes nothing via the wrong door | §82 asserts exit 1, `FAIL`+`skip`, and `has not been read` |
| Compile applies the first hunk before the second is checked | Low | High — that is Codex-rs's leak | Compile emits text only; Apply is unchanged |
| An ambiguous old side is applied at the first match | Med | High — fuzzy apply | None-or-several is a compile refusal |
| Teaching `--format` on MCP instructions overflows 4096 | Med | High | Teach on `write --help` / CLI Description, not Shared() |

## Rollback

Delete `--format`, delete `internal/ingest`, delete contract §82. Nothing persists the flag; no on-disk format changes. Native `write` is unchanged.

## Follow-ups

- [ ] Aider SEARCH/REPLACE as a second `--format` if models keep emitting it
- [ ] MCP `format` once the CLI flag has been in use
- [ ] `*** Delete File:` only if mrw gains an unlink op
