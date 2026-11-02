# ADR-191: a host's own Read licenses the lines it showed

**Status:** Proposed
**Date:** 2026-11-02
**Amended:** 2026-11-02, before acceptance, from a Codex high challenge (4 findings, each checked against the code): the licence comes from the content the Read returned, compared with the file on disk, never from the requested range or the current version alone; a payload without that content records nothing; `.git` is refused by `rooted.GitDir`, not `Resolve`; the ledger lock is tried with a deadline; the hook skips the startup migration; S1 pairs each payload with what the model received (second challenge).
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-133, ADR-161, ADR-175, ADR-190, docs/agents.md
**Governs:** `cmd/mrw/main.go`, `internal/seen/seen.go`, `docs/agents.md`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/hookread191_test.go::TestAHostReadLicensesOnlyTheLinesItShowedAsTheyAre`
**Invalidates:** None — opt-in. Nothing changes until a host runs the new command from its PostToolUse hook.
**Served-path change:** `mrw hook read` reads a host's PostToolUse payload for its own Read tool on standard input. It records in the ledger the lines whose text in the payload's returned content equals the file on disk now, as a `mrw read` of them would; any other line, and any payload without returned content, records nothing. It prints nothing and always exits 0, so a hook never fails the host's tool. `docs/agents.md` gives the Claude Code hook block.

## Context

The ledger licenses what mrw served (ADR-002). An agent with mrw also reads with its host's own tool, and those lines never reach the ledger, so a write to them is refused until mrw serves them again.

In an external, pre-registered A/B on a private Ruby repository (2026-10-31, mrw v1.79.0, 17 pairs), the arm with mrw made 86 host `Read` calls. 26 times it then read the same file again through `mrw read`, per-file, about 1,000 lines. The model had the lines; mrw did not know it.

Owner direction, 2026-11-02: "hide as much from agent and do all that we can with static tools". A hook is invisible to the agent: it runs after the host's tool, and the agent sees no call.

**What this record does not know yet.** The fields Claude Code puts in a PostToolUse payload for `Read`, beyond `tool_name` and `tool_input` (`file_path`, `offset`, `limit`), are not measured here. T1 captures a real payload first, and the Decision rests only on `tool_input` until it has. Read serves at most 2,000 lines from `offset` by default (its own tool description); that it cuts a line over 2,000 characters is an assumption T1 checks. A line the host may have cut is never licensed.

**Audit of the class**: *every host tool that shows file lines to the model*.
- Read: decided here.
- Grep in content mode: it shows matching lines with context. Left out: what it showed depends on flags the payload must be parsed for.
- Bash `cat`, `sed` and the like: left out, since what reached the model is unknowable (ADR-133's argument).

## Existing Primitives Audit

- The ledger write a `mrw read` makes (`internal/seen`, `Record`), and its lock. Reused: the hook records spans exactly as a read of the same range would.
- `mrw seen --forget` (ADR-175): runs on compaction. A licence the hook records is forgotten with the rest.
- `rooted.Resolve` refuses a path outside the root or in mrw's state; `rooted.GitDir` (`internal/rooted/gitdir.go`) refuses `.git`. The payload's `file_path` meets both, as a read's path does.
- `seen.Record` takes the ledger lock through `state.Hold`, which blocks. The hook needs a bounded try, which T1 adds beside it.
- `.claude/hooks/rules-on-read.py`: this repository's precedent for a PostToolUse hook on reads.

## Decision

1. **`mrw hook read` licenses what the Read returned, as it is on disk.** It reads one PostToolUse JSON object on standard input, and acts only when `tool_name` is `Read`, the response is not an error, and the file resolves inside the root and outside `.git` and mrw's state (`cwd` from the payload, or `--root`).
   - From the payload's returned content it takes each numbered line, in the shape S1 captures.
   - It records as served only the lines whose returned text equals that line of the file on disk now, read once under one sha.
   - A line that differs, because the host cut it or because the file changed after the read, is not licensed. A response shape the command does not recognise records nothing.
2. **It never fails, and never hangs, the host.** It exits 0 and prints nothing on standard output. It tries the ledger lock with a deadline of one second and records nothing when it cannot take it. At most one line goes to standard error, for a reason a person would want (a lock it could not take, a newer ledger, ADR-150).
3. **The hook block is documented, not installed.** `docs/agents.md` gains the Claude Code PostToolUse block, matcher `Read`, beside the compaction block, and says what it costs: every host read writes the ledger.
4. **T1 proves the payload is what the model received, before it codes.** S1 pairs each captured PostToolUse payload with the same call's tool result in the session transcript, which is what the model was given. Cases: a whole file, an offset and limit, a line over 2,000 characters, a file past Read's size limit, a non-text file. The command takes lines only from a representation S1 shows is identical in both. If none is, the record stops: requested ranges do not prove delivered lines.
5. **Nothing blocks before the deadline.** `mrw hook read` skips the startup migration of a legacy `./.mrw` (ADR-004), which takes a lock of its own. The one-second deadline covers the whole command.

**What would make this decision wrong:** a host whose PostToolUse payload carries content other than what the model received: shown differently, or cut after the hook saw it. Then mrw would license lines the model never saw, the failure ADR-133 exists to prevent. S1's capture is the check, and a shape the command does not recognise records nothing.

## Alternatives Considered

- **ADR-190 alone (quote the old text).** It covers a SEARCH/REPLACE edit after a host read, but not a native `@@` plan, which the guide teaches first.
- **A shell or Python hook script.** Rejected: the ledger's format and lock are mrw's (ADR-150). A Go subcommand keeps them in one place, under its tests.
- **License Grep and Bash output too.** Left out: what the model saw is not determinable from the payload.

## Component / Boundary Impact

`cmd/mrw`, the subcommand, and `internal/seen`, reusing its record path (engine, owned here). `docs/agents.md`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw hook read` | new subcommand | T1 | host hooks |
| `docs/agents.md` | the PostToolUse block | T1 | agent setups |
| contract §329 | new rows | T1 | CI |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** a native plan after a host read lands without a second read: the shape of the 26 re-reads above.
- **Negative:** the ledger now also holds what another tool showed. Its truth rests on the payload's.
- **Neutral:** without the hook block, nothing changes.

## Out of Scope

- Grep content-mode output (permanent: boundary: what it showed depends on flags, and is a later record if measured)
- Bash `cat` and `sed` output (permanent: boundary: what reached the model is unknowable, ADR-133)
- The measurement against native (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The payload over-claims what was shown | Low | High | only the payload's returned content licenses, compared with disk; S1 captures it first |
| A hook adds latency to every host read | Med | Low | one mrw process per read; the lock is tried for one second at most; opt-in |

## Rollback

Remove the hook block; the subcommand then records nothing anyone calls.

## Follow-ups

- [ ] The pre-registered paid run shared with ADR-190 (BACKLOG.md)
