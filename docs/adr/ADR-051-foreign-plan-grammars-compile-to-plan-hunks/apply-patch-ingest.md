# Spec: Compile apply_patch to plan hunks

> **Date:** 2026-09-12 · **Status:** Ready-for-ADR
> **Owner:** M · **Becomes:** amendment of ADR-051 (`docs/adr/ADR-051-foreign-plan-grammars-compile-to-plan-hunks.md`) — not a competing Decision
> **Gate:** Status may become Ready-for-ADR only after `spec-verify --spec <this file>` exits 0.
> **Cross-references:** ADR-001, ADR-002, ADR-019, ADR-027, ADR-030, ADR-035, ADR-037, ADR-044, ADR-048, ADR-049, docs/adr/BACKLOG.md

Child of Accepted ADR-051. The Decision stands: a foreign grammar compiles to native `@@` text; `plan.Parse` and `apply.Apply` stay the only apply path. This file is the compile-rule detail a later `--format` (Aider SEARCH/REPLACE, MCP `format`, Delete/Move, git) must not re-derive.

## Problem

ADR-051 shipped the Decision and the first grammar, then left the compile rules in `internal/ingest/applypatch.go` without a spec. A later implementer would re-derive the subset, the 0/N match rule, CRLF, empty add, and the MCP cargo boundary from code.

## Goal

One child spec names the ingested `apply_patch` subset, the compile rules, what still goes through Parse/Apply/ledger/`--check`, why `--format` is explicit, and the named follow-ups — so a second grammar cannot silently widen this slice.

## Actors

| Actor | Kind | Goal |
|-------|------|------|
| CLI caller | human role | Feed a Codex `apply_patch` document to `mrw write` and get a per-hunk verdict |
| Compiler | system | Turn that document into native plan text, or refuse before anything is written |
| Apply | system | Apply the compiled plan under ADR-001–004 unchanged |
| MCP caller | external service | Pass `format` on existing `mrw_write` (`plan` default, `apply_patch`, `search_replace`) |

## Use Cases

### UC-1: CLI caller applies a Codex apply_patch

- **Trigger:** `mrw write --format=apply_patch [PLAN|-]` · **Preconditions:** document is an `apply_patch`; target paths sit under `--root`
- **Main flow:**
  1. Compile emits `@@` text. Parse and Apply run unchanged. `--check` / `--force` / the ledger mean what they meant on a native plan.
- **Failure paths:** a. unread compiled line → exit 1, FAIL+skip, tree unchanged. b. compile cannot locate or parse → exit 2, nothing written.
- **Postconditions:** every hunk has a verdict; a failure writes nothing.

### UC-2: Compiler locates an update old side

- **Trigger:** `*** Update File:` hunk with context / minus / plus lines · **Preconditions:** the file exists under root
- **Main flow:**
  1. Old side (context + minus, prefixes stripped) matches exactly one contiguous run. Compiled hunk is `replace` of that run with the new side (context + plus). Multi-line replace carries `anchor=` from the first old-side line.
- **Failure paths:** a. 0 matches → compile refuse. b. N>1 matches → compile refuse. c. no old side → compile refuse (use Add File, or give context).
- **Postconditions:** compile writes nothing; location is not a license.

### UC-3: CLI caller names the grammar

- **Trigger:** `write --format` · **Preconditions:** none
- **Main flow:**
  1. Default `plan` parses stdin/file as native `@@`. `apply_patch` selects the compiler. Help names both and that a git patch is not an `apply_patch`.
- **Failure paths:** a. `apply_patch` document with no flag → exit 2 (bad native plan). b. `--format=git` or unknown value → usage exit 2. c. git-shaped body under `--format=apply_patch` → compile refuse exit 2.
- **Postconditions:** no auto-detect from `*** Begin Patch`.

### UC-4: MCP caller writes through the existing tool

- **Trigger:** `mrw_write` · **Preconditions:** ADR-044 cargo stays two tools; 4096 stays
- **Main flow:**
  1. `plan` remains the required native `@@` document. Optional `format` (`plan` default, `apply_patch`) selects the same compile path as `write --format`. No third tool.
- **Failure paths:** a. an `apply_patch` blob with no `format` (or `format=plan`) is a bad native plan. b. `format=git` or unknown is usage. c. unread compiled sibling → FAIL+skip, tree unchanged.
- **Postconditions:** same compile → `plan.Parse` → `apply.Apply` → ledger → ack. 4096 stays.

### UC-5: CLI/MCP caller names SEARCH/REPLACE

- **Trigger:** `write --format=search_replace` or `mrw_write` `format=search_replace` · **Preconditions:** document is Aider SEARCH/REPLACE fences
- **Main flow:**
  1. Compile emits `@@` text from unique exact SEARCH matches. Parse and Apply run unchanged.
- **Failure paths:** a. unread compiled line → exit 1, FAIL+skip. b. 0/N SEARCH matches → compile refuse exit 2. c. near-miss SEARCH → compile refuse, not fuzzy apply. d. no flag → bad native plan.
- **Postconditions:** explicit flag; no auto-detect; location is not a license.

## Scenarios

### UC1-S1 [happy] A served two-hunk apply_patch writes both replacements [@implemented] → `cmd/mrw/writeformat_test.go::TestWriteFormatApplyPatchServedWritesBoth`

```gherkin
Given a two-hunk Update whose old sides were both served
When the caller runs write --format=apply_patch
Then both replacements land, exit 0, and --check if asked runs on the touched files
```

Contract §82 already drives the served half on the built binary.

### UC1-S2 [failure] An unread compiled sibling writes nothing [@implemented] → `internal/ingest/applypatch_test.go::TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing`

```gherkin
Given a two-hunk Update whose second old side was never served
When compile succeeds and Apply runs
Then one hunk FAILs with has not been read, siblings skip, the file is unchanged
```

CLI twin: `cmd/mrw/writeformat_test.go::TestWriteFormatApplyPatchUnreadWritesNothing` (exit 1). Contract §82 pairs the same unread half.

### UC2-S1 [happy] A unique old side compiles to a replace plan.Parse accepts [@implemented] → `internal/ingest/applypatch_test.go::TestCompileApplyPatchGoesThroughParse`

```gherkin
Given an Update whose old side matches exactly one run
When CompileApplyPatch emits plan text
Then plan.Parse accepts it and the replace address is that run
```

### UC2-S2 [failure] An old side matching several times is a compile refusal [@implemented] → `internal/ingest/applypatch_test.go::TestAnAmbiguousOldSideIsACompileRefusal`

```gherkin
Given an old side that occurs three times
When CompileApplyPatch runs
Then it refuses naming the match count, writes nothing, and the CLI exit is 2
```

### UC3-S1 [happy] write --help names apply_patch and that a git patch is not one [@implemented] → `cmd/mrw/writehelp_test.go::TestWriteHelpNamesApplyPatchFormat`

```gherkin
Given a PATH caller who can only read write --help
When they read the description
Then it names --format=apply_patch and that a git patch is not an apply_patch
```

### UC3-S2 [failure] A git patch is not an apply_patch [@implemented] → `internal/ingest/applypatch_test.go::TestAGitPatchIsNotAnApplyPatch`

```gherkin
Given a document that starts with diff --git or --- a/
When it is compiled as apply_patch, or the caller passes --format=git
Then compile or usage refuses at exit 2 and nothing is written
```

`--format=git` is contract §82. A document without `--format` is the same section's no-flag row.

### UC4-S1 [happy] mrw_write still takes a native plan [@implemented] → `internal/mcp/tools_test.go::TestTheWriteToolReturnsTheSameResultAsTheCLI`

```gherkin
Given the shipped MCP tool list
When a caller invokes mrw_write without format
Then the required field is plan (native @@), format defaults to plan, and there is no third tool
```

### UC4-S2 [failure] An apply_patch blob on mrw_write plan is a bad native plan [@implemented] → `internal/mcp/writeformat_test.go::TestAnApplyPatchBlobOnWritePlanIsABadNativePlan`

```gherkin
Given an apply_patch document in mrw_write.plan and no format
When the tool runs
Then Parse refuses it as a native plan; nothing is written; no new tool appears
```

### UC4-S3 [failure] MCP format apply_patch with one unread hunk writes nothing [@implemented] → `internal/mcp/writeformat_test.go::TestWriteFormatApplyPatchUnreadWritesNothing`

```gherkin
Given a two-hunk Update whose second old side was never served
When mrw_write runs with format=apply_patch
Then one hunk FAILs with has not been read, siblings skip, the file is unchanged
```

### UC5-S1 [failure] An unread SEARCH/REPLACE sibling writes nothing [@implemented] → `internal/ingest/searchreplace_test.go::TestATwoHunkSearchReplaceWithOneUnreadLineWritesNothing`

```gherkin
Given a two-hunk SEARCH/REPLACE whose second SEARCH was never served
When compile succeeds and Apply runs
Then one hunk FAILs with has not been read, siblings skip, the file is unchanged
```

CLI: `cmd/mrw/writeformat_test.go::TestWriteFormatSearchReplaceUnreadWritesNothing`. MCP: `internal/mcp/writeformat_test.go::TestWriteFormatSearchReplaceUnreadWritesNothing`.

### UC5-S2 [failure] A near-miss SEARCH is a compile refusal [@implemented] → `internal/ingest/searchreplace_test.go::TestANearMissSearchIsACompileRefusal`

```gherkin
Given a SEARCH that differs from the file by one extra space
When CompileSearchReplace runs
Then it refuses (matched no lines), writes nothing, and does not fuzzy-apply
```

### UC5-S3 [failure] SEARCH/REPLACE without format is a bad native plan [@implemented] → `internal/mcp/writeformat_test.go::TestASearchReplaceBlobOnWritePlanIsABadNativePlan`

```gherkin
Given a SEARCH/REPLACE document and no format
When write / mrw_write runs
Then Parse refuses it as a native plan; nothing is written
```

## Facts

Class notes sit in the assertion. Members that behave differently are their own row.

| ID | Assertion (invariant / behavior) | Test (`path::name`) | Tag | Cmd (optional) |
|----|----------------------------------|---------------------|-----|----------------|
| F-1 | The ingested envelope is `*** Begin Patch` … `*** End Patch`. Non-blank text before or after is a compile refusal. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-2 | This slice ingests `*** Update File:` and `*** Add File:`. `*** Delete File:` and hunk-less `*** Move to:` compile as ADR-057. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-3 | `*** Delete File: p` compiles to `@@ p - unlink`. `*** Move to: q` after `*** Update File: p` with an empty hunk compiles to `@@ p - rename`. Extra `@@` hunks with Move to are a compile refusal naming hunks. Emptying a file is still not a delete. | `internal/ingest/applypatch_unlink_test.go::TestCompileDeleteFileIsUnlink` | @implemented | |
| F-4 | `*** End of File` is not a marker. A line so named is unexpected or an illegal hunk line. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-5 | A line beginning `@@` is a hunk delimiter only. Any ChangeContext after `@@` is discarded. Location is the unique old-side run, never that trailer. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-6 | Old side = context (` `) plus minus lines, prefixes stripped. New side = context plus plus lines. Add File accepts plus lines only. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-7 | The old side must match exactly one contiguous run of on-disk lines (equal after the file's one trailing LF is stripped). Several matches refuse, naming the count. | `internal/ingest/applypatch_test.go::TestAnAmbiguousOldSideIsACompileRefusal` | @implemented | |
| F-8 | Zero matches refuse (`old side matched no lines`). An Update with no old side refuses (`use Add File, or give context`). | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-9 | The document's CR LF and bare CR become LF before parse. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-10 | ~~The on-disk file is not CR LF-normalised; an LF old side matches zero times against a CR LF file and compile refuses.~~ **Superseded by ADR-065 (2026-09-24):** the target's lines are read as the write engine numbers them (`lines.Split`), so an LF old side matches a CR LF or CR-only file; the write engine keeps the file's line endings when it writes. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-11 | `\ No newline at end of file` and other `\`-prefixed hunk lines are skipped, not part of either side. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-12 | Emitted body lines are always LF-terminated. Patch no-newline markers do not change that. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-13 | An Add File with no plus lines compiles to `create` with `body=0` (ADR-027 deliberate empty file). | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-14 | A compiled multi-line `replace` carries `anchor=` from the first old-side line (ADR-035). A single-line replace carries none. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-15 | A rooted path is refused (`rooted.IsRooted`, not `filepath.IsAbs`). Compile reads the file only to locate; it writes nothing. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-16 | One `*** Update File:` may carry several `@@`-separated hunks. A blank line flushes the current hunk. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchRules` | @implemented | |
| F-17 | Compile emits plan text only. `plan.Parse` is the only door into `apply.Apply`. ingest never builds `apply.Input`. | `internal/ingest/applypatch_test.go::TestCompileApplyPatchGoesThroughParse` | @implemented | |
| F-18 | Unique old-side match is location, not license. An unread compiled line FAILs, siblings `skip`, tree unchanged, CLI exit 1. | `internal/ingest/applypatch_test.go::TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing` | @implemented | |
| F-19 | After a successful compile, `--check`, `--force`, and the per-line ledger mean what they mean on a native plan. | `cmd/mrw/writeformat_test.go::TestWriteFormatApplyPatchForceWritesUnread` | @implemented | |
| F-20 | Compile refusal is exit 2 and counts as a parse refusal (the document did not become a plan). An unread compiled hunk is exit 1. | `cmd/mrw/writeformat_test.go::TestWriteFormatApplyPatchCompileRefusalIsUsage` | @implemented | |
| F-21 | `--format` defaults to `plan`. An `apply_patch` document without the flag is a bad native plan (exit 2). No auto-detect from `*** Begin Patch`. | `cmd/mrw/writeformat_test.go::TestWriteFormatApplyPatchNoFlagIsUsage` | @implemented | |
| F-22 | `--format=git` is usage (exit 2) and names that a git patch is not an `apply_patch`. Unknown values are usage. | `cmd/mrw/writeformat_test.go::TestWriteFormatGitIsUsage` | @implemented | |
| F-23 | A body that looks like git (`diff --git` prefix, `--- a/` prefix, or a `--- a/` line) is refused under `--format=apply_patch`. | `internal/ingest/applypatch_test.go::TestAGitPatchIsNotAnApplyPatch` | @implemented | |
| F-24 | This slice adds no MCP tool (ADR-044 / ADR-019 cargo). 4096 stays. 019 A stands. | `internal/mcp/writeformat_test.go::TestWriteDeclaresFormatOnTheExistingTool` | @implemented | |
| F-25 | MCP ingest of `apply_patch` is a `format` string on existing `mrw_write` (`plan` default, `apply_patch`), not a new tool. | `internal/mcp/writeformat_test.go::TestWriteDeclaresFormatOnTheExistingTool` | @implemented | |
| F-26 | Aider SEARCH/REPLACE is `--format=search_replace` / MCP `format=search_replace`. Compiles to `@@`. Unread writes nothing. Explicit flag. Exact unique SEARCH only — no fuzzy. | `internal/ingest/searchreplace_test.go::TestATwoHunkSearchReplaceWithOneUnreadLineWritesNothing` | @implemented | |
| F-27 | `mrw_write` accepts optional `format` now (`plan` default, `apply_patch` same compile path as CLI, `git` refuse). M 2026-09-12: *"YES, we have to be competitive"*. | `internal/mcp/writeformat_test.go::TestWriteFormatApplyPatchUnreadWritesNothing` | @implemented | |

## Domain

An **apply_patch document** is a Codex envelope (`*** Begin Patch` … `*** End Patch`) of file ops. A **compiler** turns that into **plan text** (`@@` hunks). **Parse** and **Apply** are unchanged. **Location** (unique old-side run) is not a **license** (per-line ledger). A **format flag** selects the grammar; it is never inferred.

## Contracts Touched

| Surface | Change | Consumers |
|---------|--------|-----------|
| `mrw write --format` | `plan` (default), `apply_patch`, or `search_replace` | CLI; contract §82, §84; `write --help` |
| `mrw_write` MCP | optional `format` (`plan` default, `apply_patch`, `search_replace`); `git` refuse; no third tool | MCP hosts; contract §83, §84 |

ADR-051 Wiring inherits this. No exit-code change: compile refuse = 2; unread compiled hunk = 1.

## Non-Goals

- Sequential apply that leaves a half-written file (permanent: boundary: that is the leak we do not copy; ADR-001 stands)
- File-level-only license (permanent: boundary: ADR-002 is per line)
- Fuzzy apply of a near-miss old side (permanent: boundary: unread still refuses)
- Syntax-aware write / wrap-tail as a parser (permanent: fact: ADR-048 models no target syntax; citation: file `docs/adr/ADR-048-mrw-models-no-target-syntax.md:31`)
- Morph / Relace / Instant Apply streaming (permanent: fact: ADR-049 waits for a size that hurts; citation: file `docs/adr/ADR-049-streaming-apply-waits-for-a-size-that-hurts.md:31`)
- Auto-detect vs explicit flag (permanent: boundary: ADR-051 picked the flag)
- Treating a git patch as an `apply_patch` (permanent: boundary: they share `@@` and they are not the same grammar)
- Treating SEARCH/REPLACE as apply_patch compile (permanent: boundary: F-26 is a second `--format`)
- Native unlink/rename for Delete File / hunk-less Move to (permanent: fact: ADR-057; citation: file `docs/adr/ADR-057-unlink-and-rename.md:31`)
- Move to with in-file hunks (deferred: docs/adr/BACKLOG.md "Move to with hunks")
- A third MCP tool named apply_patch (permanent: fact: ADR-044; citation: file `docs/adr/ADR-044-mcp-cargo-stays-two-tools.md:7`)
- ast-grep-shaped `--grep` (deferred: docs/adr/BACKLOG.md)
- Raising `maxInstructionsChars` / 4096 (permanent: boundary: ADR-037; 4096 stays)
- Reopening ADR-019 B/C (permanent: fact: pick A stands; citation: file `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md:106`)
- Cargo MCP tools (permanent: fact: ADR-044; citation: file `docs/adr/ADR-044-mcp-cargo-stays-two-tools.md:7`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later grammar treats unique context match as a license | High | High — drops ADR-002 | F-18; unread two-hunk fixture |
| On-disk CR LF looks like a missing old side | Med | Med — exit 2 instead of apply | Superseded by ADR-065: compile against the write engine's lines; the file's CR LF is kept on write, not stripped |
| MCP grows a third tool named apply_patch | Med | High — 044/019 cargo | F-24, F-25, F-27 |
| SEARCH/REPLACE is implemented inside apply_patch compile | Med | High — silent scope | F-26; second `--format` or a new ADR |

ADR-051 Risks still apply (unread fixture passing on exit 2, sequential compile, first-match apply). This spec adds the three above.

## Open Questions

## Verify

```bash
spec-verify --spec docs/adr/ADR-051-foreign-plan-grammars-compile-to-plan-hunks/apply-patch-ingest.md
```

F-1…F-27 are bound. SEARCH/REPLACE is `--format=search_replace` (F-26).

## Grill Log (appendix)

Scouted from ADR-051, `internal/ingest/applypatch.go`, `cmd/mrw/main.go`, contract §82, and BACKLOG. Not a live interview — M asked for the spec after the Decision shipped.

| # | Question | Fact | Decision |
|---|----------|------|----------|
| 1 | Which apply_patch ops do we ingest? | F-2 F-3 F-4 | Update + Add this slice; Delete/Move refuse; End of File is not a marker |
| 2 | How does context become a line range? | F-5 F-6 F-7 F-8 F-16 | Unique exact old-side run; @@ trailer discarded; 0/N/empty-old refuse |
| 3 | CRLF, trailing newline, empty add? | F-9 F-10 F-11 F-12 F-13 | Document normalised; file read as the write engine numbers it (F-10, ADR-065); `\` markers skipped; emit LF; empty add is body=0 |
| 4 | What still goes through Parse / Apply / ledger / --check? | F-17 F-18 F-19 F-20 | Compile emits text; unread is Apply exit 1; compile refuse is exit 2 |
| 5 | --format vs auto-detect vs git? | F-21 F-22 F-23 | Explicit flag; default plan; git is usage; git-shaped body refused |
| 6 | Does mrw_write grow format? | F-24 F-25 F-27 | Yes now (M 2026-09-12 competitive). Existing tool only; no cargo |
| 7 | Next grammar? | F-26 | SEARCH/REPLACE is a named second --format, not silent scope |
| 8 | Path of this record? | non-behavioral | Child spec under ADR-051; Decision unchanged |
