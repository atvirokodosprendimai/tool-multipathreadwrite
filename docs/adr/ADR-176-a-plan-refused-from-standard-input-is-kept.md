# ADR-176: a plan refused from standard input is kept

**Status:** Accepted
**Accepted:** 2026-10-24 by Zy — asked which drafted records to accept, chose *"ADR-171 guide cost, ADR-172 + ADR-119 D5, ADR-173 + ADR-174, ADR-175 forget"*, and for the breaking pair *"v1.76.0, loud note (Recommended)"*.
**Renumbered:** 2026-10-27, from ADR-172 before it merged: PR #528 (brew tap) had taken 172 in public first. Zy chose to move this record (*"Ours → ADR-176"*). The acceptance above quotes the number as it was said. Code citations below are at `8278fcda`, the base the record was written on.
**Amended:** 2026-10-24, before acceptance, from a Codex xhigh challenge (1 P1, 6 P2, 1 P3), each finding checked against the code: the kept plan lives BESIDE the state directory, not in it, so ADR-077 keeps its rule whole and the plan is editable with mrw itself (P1 hard link into the exception, P2 no mrw edit path, P2 state accounting); the kept bytes are the ones the parser read, after the CRLF trim, so a re-run parses the same plan (P2); what counts as "nothing written" is `writer.MutationOf` on any exit after the plan was read, not a list of exits (P2 class audit); the stderr line tells the caller to rerun its own command with the path in place of `-` instead of reconstructing one (P2 retry command); names are made exclusively (P3); the proof matrix covers every Decision (P2).
**Date:** 2026-10-24
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-034, ADR-066, ADR-077, ADR-110, ADR-111, ADR-132, ADR-139, ADR-161, ADR-166, docs/receipts.txt
**Governs:** `cmd/mrw/main.go`, `docs/receipts.txt`, `README.md`, `AGENTS.md`, `internal/guide/guide.go`, `internal/state/lock.go`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/refused176_test.go::TestAPlanRefusedFromStandardInputIsKept`
**Invalidates:** None — additive; ADR-077 is untouched (the kept plans are not inside mrw's state directory)
**Served-path change:** a `mrw write` whose plan came from standard input and wrote nothing keeps the plan beside mrw's state directory and names its path on stderr (and as `refused_plan` under `--json`), so a caller fixes one header and reruns its command with that path in place of `-`, instead of sending the whole plan again.

## Context

An outside test of mrw v1.74.0 on a large Rails application (the "Rubie" tech summary, 24 October 2026, issue MRW-04; 17 paired Claude Code runs, Claude Opus 5.5) measured 36K characters of `mrw write` input that was refused or blocked and had to be sent again; single plans ran to 13K characters. 7 of 50 plans were refused for a `body=` count (MRW-01, ADR-171's subject), and each fix re-sent the whole plan: the refusal names the plan line, but the plan existed only in the shell command that piped it. A plan read from a file does not have this cost — the caller edits the file and runs the same command — and ADR-171 makes a plan file the recipe agents are taught; this record covers the callers who still pipe one (`printf … | mrw write -`, a heredoc, a host with no file tool).

Reproduced on v1.75.0 (2026-10-24): `printf '%s\n' '@@ a.rb 2-4 replace anchor="  def x" body=2' '  def x' '    2' '  end' | mrw write -` exits 2 with `line 4: body=2 is satisfied; 1 extra line(s) …`, and nothing of the plan is left anywhere.

**Audit of the class** — *a write whose plan was read from standard input and that ends with nothing in the tree*: the plan source is one site, `cmd/mrw/main.go:1721` (`src = os.Stdin`, name `<stdin>`), wrapped for the CRLF trim at `:1784`; "nothing in the tree" is one predicate, `writer.MutationOf(res) == writer.None` (`internal/writer/writer.go:35`), true for a refusal, a dry run, and a commit that failed before its first file landed or whose undo put everything back. The exits it covers, found with `mrw read --grep 'return refuse|refuseErr|refuseWith' cmd/mrw/main.go` after `:1799`: the parse refusal, `plan.LoadBodyFiles` (`:1860`), the `--json` UTF-8 refusal (`:1868`), the working-set load and pointer resolution (`:1893`), the writer's gates (harness, steps, depth, ledger, lock timeout: `internal/writer/flow.go:93-105`), validation and staging refusals (ADR-001, ADR-132), and a commit failure with nothing landed (ADR-066). A refusal BEFORE the plan is read (a flag error, `askedStepsError` at `:1718`) has nothing to keep. Left out: `--create` (its standard input is a file's content, not a plan, ADR-139), `--sub`/`--rewrite` (mrw made the plan; a rerun remakes it, ADR-161), a plan from a file (already on disk), and `mrw_write` (the host holds the plan it sent).

## Existing Primitives Audit

- `state.Base()` (`internal/state/state.go:48`): `<state home>/mrw`. The kept plans go in its sibling `<state home>/mrw-refused/`, computed from it in `cmd/mrw`; nothing inside `Base()` changes, so `mrw seen --prune` and its size accounting are untouched. `state.HoldWithin` (`internal/state/lock.go`) locks a file under a state directory; the lock here must live in the kept directory itself, so `internal/state` gains `HoldFileWithin`, the same lock at a path, and `HoldWithin` calls it with unchanged behaviour.
- `writer.MutationOf` (`internal/writer/writer.go:35`): the one predicate for "nothing reached the tree". Reuse.
- `ingest.NewPipedCRLFReader` (`main.go:1785`): the stream the parser reads. The tee sits after it.
- `maxPlanLine` (`internal/ingest/create.go:13`, 16 MiB): the kept copy's cap reuses the number.
- Keeping the plan inside the state directory: the first draft; rejected (Alternatives).

## Decision

**1. What is kept.** When `mrw write` reads its plan from standard input (`mrw write -`, or no PLAN argument) and the run ends, after the plan was read, with `writer.MutationOf(res) == None` — any refusal, `--dry-run`, or a commit with nothing landed — mrw keeps the bytes the parser read (after ADR-161's CRLF trim, so a rerun parses the same plan) as a new file in `<state home>/mrw-refused/`, mode 0600 in a directory of mode 0700, named `<UTC 20060102T150405.000000000Z>-<pid>-<8 hex>.plan` and created exclusively (`O_EXCL`), so no two runs, in one process or many, write the same name. A run with anything in the tree (exit 0, exit 3, `PARTIALLY APPLIED`) keeps nothing. The tee never changes what the parser reads or when: it copies up to 16 MiB and, past that, drops its copy and keeps nothing.

**2. What it says.** One stderr line after the refusal: `mrw: the plan from standard input is kept at <path> (<N> bytes, sha <8 hex>): fix it there and run the same command with <path> in place of -`. Under `--json` the refusal document gains `refused_plan` (the path), appended to `docs/receipts.txt` for `write` (ADR-111). When nothing can be kept — over 16 MiB, or the directory cannot be written (a read-only sandbox, ADR-166) — the line says so instead and names the reason; the refusal, its exit and its other keys are unchanged.

**3. Running it again and editing it.** The kept file is an ordinary file outside any checkout and outside mrw's state directory: `mrw write <path>` reads it as any plan file, and the caller edits it with `mrw --root <state home>/mrw-refused read|write` or the host's file tool. ADR-077 does not change. Rerunning it does not remove it.

**4. How long it stays.** The directory holds plans from every checkout; after keeping one, mrw removes all but the newest 16 by name (the time prefix), best effort. Creating a plan and trimming run under one lock, a `.lock` file in the kept directory itself (`state.HoldFileWithin`, waited for up to 10 seconds; a lock error means the plan is not kept, and stderr says why), so no other run trims between a plan's creation and the trim that spares it. The lock is a file in the kept directory, not in mrw's state directory, so every process that trims that directory shares it, whatever its state home: two homes whose `mrw-refused` is one physical directory (one a link to the other) take one lock. The trim removes only `*.plan`, so `.lock` and its `.lock.holder` stay. As a backstop for a lock file removed while held (the directory deleted under a run), the trim never removes a plan modified in the last minute, which another run may still be saving; a plan whose modification time is in the future (a clock rolled back) stays protected until that time plus a minute. So the directory can hold more than 16 until then, and an older run trimming at once may leave one more, until the next refusal. Nothing else removes them; `mrw seen --prune` does not reach the directory, and deleting it loses nothing mrw needs.

**What would make this decision wrong:** callers who never rerun the kept path, so the line is noise on every refusal. Measurable on the next field run by counting writes of a `mrw-refused/` path after a refusal; nothing measures it in this repository (ADR-009 refuses telemetry).

## Alternatives Considered

- **Inside the state directory, with an ADR-077 exception.** The first draft; rejected after the challenge: a hard link to the ack store placed in the exception's directory passes a check by location and name, the file cannot be edited with mrw (every path in the state directory is refused), and the nested directory escapes `mrw seen --prune`'s size accounting.
- **One file, overwritten each time.** Rejected: sessions sharing a checkout overwrite each other between a refusal and its retry.
- **`$TMPDIR`.** Rejected: on Linux `/tmp` is shared between users and its lifetime is the OS's; the state home is per user and is where mrw keeps its files (ADR-004).
- **Reconstruct the retry command.** Rejected: `--root`, `--dry-run`, `--strict-balance` and `--format` would each need carrying; the caller already has its command.
- **Keep it on the MCP surface too.** Rejected: `mrw_write`'s plan is an argument the host already holds.
- **Do nothing; ADR-171 teaches the plan file.** Rejected as the only answer: hosts with no file tool, and callers who keep piping, still pay the re-send.

## Component / Boundary Impact

`cmd/mrw` only: the write command tees its standard-input plan and keeps it on a nothing-written exit. No engine package changes. A new directory beside `<state home>/mrw`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| stderr of a refused `mrw write -` | one line naming the kept path, or why nothing was kept | `cmd/mrw` | callers |
| `--json` refusal of `mrw write` | `refused_plan` (string, absent when nothing kept) | `cmd/mrw` | adapters; `docs/receipts.txt` |
| `<state home>/mrw-refused/*.plan` | new files, newest 16, 0600 | `cmd/mrw` | `mrw write PATH`, callers |
| `scripts/contract.sh` | a row: a refused piped plan is kept and its rerun applies; a landed one keeps nothing | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the stderr line and `refused_plan` | T1 | T2 | no — a new line and a new key |

## Implementation

`tasks/README.md`: T1 (tee, keep, name, trim, contract row), T2 (README, AGENTS.md, `mrw instructions`). One pull request.

## Consequences

- **Positive:** a refused piped plan is fixed in place; the 36K-character re-send the field run measured becomes an edit of one line.
- **Negative:** plans, which hold code, now persist in the per-user state home, up to 16 at up to 16 MiB each (256 MiB worst case), until a later refusal trims them or someone deletes the directory.
- **Neutral:** a plan from a file, `mrw_write` and every write that lands behave as before; ADR-077 is unchanged.

## Out of Scope

- Keeping `--create`'s standard input when it is refused (deferred: docs/adr/BACKLOG.md "From ADR-176")
- Keeping a plan for `mrw_write` (permanent: boundary: the host holds the plan it sent)
- A setting for how many plans are kept (permanent: boundary: no configurability without a measured need)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a kept plan holds a secret the caller piped | Low | Med | 0600 in a 0700 directory in the per-user state home; the newest 16 only; the stderr line names the path |
| the tee changes what the parser reads | Low | High | the tee copies after the CRLF reader and never reads ahead; a test pipes the PowerShell CRLF fixture and an over-cap plan and asserts the receipt is byte-identical to today's |
| the stderr line breaks a caller that parses stderr | Low | Low | stderr is not a contract; `--json` carries the key |

## Rollback

Revert the tasks; delete `<state home>/mrw-refused/` (nothing reads it once reverted). `refused_plan` disappears from the receipt, which ADR-111 calls a removal: revert before a release ships it.

## Follow-ups

- [ ] none at authoring
