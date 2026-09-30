# ADR-102: A write that landed is reported and counted as landed

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — approved the plan "resolve the seven findings of the 2026-09-30 Codex design review" (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`), whose ADR-102 is this record's scope, and set the goal "deliver open tasks end to end". The record's text was drafted after that approval and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-005, ADR-023, ADR-032, ADR-054, ADR-066, ADR-072, ADR-075, ADR-083
**Governs:** `internal/writer/writer.go`, `internal/authoring/authoring.go`, `internal/mcp/tools.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/writer/mutation102_test.go::TestMutationSaysHowMuchOfAPlanLanded`
**Invalidates:** ADR-083 Decision 2, narrowly — "a filesystem error from apply is one `refused_apply` … whatever reached disk before it" becomes: one `partially_applied` when a file reached disk, one `refused_apply` when none did. ADR-066's deferral "Recording the ledger after a partial commit" is taken up. ADR-054's closed vocabulary gains one name, printed at zero like the rest. Otherwise none — checked.
**Served-path change:** (1) `mrw_write` answers a write that landed but whose ledger could not be saved with its bounded receipt, `isError` true and an `error` field, where it answered a bare JSON-RPC error; the receipt of every write that failed carries `error` (structured and in text). (2) The MCP message for a write too large to report says not to re-run the plan, where it suggested re-running with a larger ceiling. (3) `mrw stats` and `stats --json` carry a sixth outcome, `partially_applied`, counted in `landed`; a partial commit is counted there instead of `refused_apply`, on both surfaces. (4) After a partial commit the files that were written are licensed for the next write without a re-read, as after a complete one. Exit codes are unchanged.

## Context

**What was observed** (2026-09-30, the Codex design review of `f1d5996`, each point checked against source):

1. `internal/mcp/tools.go:702-709`: when `writer.Apply` returns `LedgerError` — the plan landed and the ledger
   save failed — `mrw_write` records `applied` and returns `callToolResult{}` with a JSON-RPC internal error. No
   receipt. The CLI prints the landed receipt (`cmd/mrw/main.go:1368-1386`, then `refuseWith`), and
   `writer.go:21-23` says a caller reports it. An MCP client cannot tell this from a write that did nothing,
   and may retry a non-idempotent plan on a changed tree.
2. `tools.go:914-917`: the message for a write whose receipt does not fit says "the write HAPPENED. Read the
   files, or re-run with a larger --max-result-chars". Re-running applies an insert or a rename twice.
3. A partial commit — `Applied` false, some `files[].written` true (ADR-066) — is tallied `refused_apply`
   (`main.go:1324-1328`, `tools.go:722-735`), so `Tally.Landed` (`authoring.go:115-124`, "how many plans WROTE
   the tree") misses a changed tree.
4. `writer.go:51` returns before recording the ledger unless the plan applied whole, so the files a partial
   commit wrote keep their old ledger entry: the next write to them is refused as changed since read, though
   mrw itself wrote them (ADR-005: a file mrw wrote is wholly known).
5. "Did anything land" is recomputed in three places (`tools.go:872-878`, `tools.go:890` `writtenFiles`,
   `apply.go:844` `writtenSoFar`) and asked as `res.Applied` in the tallies, which is false for a partial commit.

**Audit of the class.** The class is *a surface that reports or counts a write by `Applied` alone, or drops the
receipt of one that landed*. Enumerated 2026-09-30 by `mrw read --grep 'res\.Applied|\.Applied &&|LedgerError'
cmd/mrw/main.go internal/mcp/ internal/writer/ --exclude '*_test.go'`: the MCP ledger branch and tally switch,
the CLI tally at `:1324`, `refuseWith`, the normal-path tally `:1420-1429`, `writer.Apply` — all in scope. The
MCP bounded receipt already asks "did any file change" (`tools.go:872`) and keeps written file records
(ADR-032); it moves onto the shared classification. **Left out:** `writtenFiles` and `writtenSoFar` filter or
name records rather than classify, and stay.

## Existing Primitives Audit

- **`boundedReceipt`** (`tools.go:807`) — reused for the ledger-failure answer; it already never elides a
  failed hunk or a written file.
- **`seen.Record`/`seen.Drop`** — reused by `writer.Apply` for a partial commit, as for a complete one.
- **`receipt.Error`** (CLI, ADR-072) — the MCP receipt gains the same field name.

## Decision

1. `writer.MutationOf(res)` says how much of a plan reached the tree: `writer.None` (a dry run, or nothing
   written), `writer.Partial` (not applied, at least one file written), `writer.Complete` (applied). Every
   tally and the MCP unreportable branch ask it.
2. A sixth outcome, `partially_applied`, joins the closed vocabulary between `refused_apply` and
   `check_not_run`, and `Landed()` counts it. A plan whose commit failed after a file landed is one
   `partially_applied`; one whose commit failed before any file landed stays `refused_apply`.
3. `writer.Apply` records a partial commit like a complete one: written files wholly known at their new sha,
   removed ones dropped. A ledger failure there is added to the commit error; it is not a `LedgerError`, which
   callers read as "the plan landed".
4. `mrw_write` answers a `LedgerError` with `boundedReceipt`, `isError` true, after counting and recording the
   landed write (`applied`, the recent window). `writeReceipt` carries `error` whenever the write returned one.
5. The unreportable message says the write happened and not to re-send the plan, names the files as where to
   look, and keeps "restart with a larger `--max-result-chars`" as advice for later writes.
6. A partial commit joins the recent-write ring (ADR-055) as a landed write, on both surfaces.

## Alternatives Considered

- **Relabel `Landed` as "applied whole"** — rejected: the one number that claims to count changed trees would
  keep missing one.
- **A `partial` hunk status** — ADR-066 rejected it: `files[].written` already records the fact; this record
  counts it, and changes no hunk status.
- **Return a `LedgerError` for a partial commit's ledger failure** — rejected: callers read `LedgerError` as
  "the plan landed whole".

## Component / Boundary Impact

`internal/writer`, `internal/authoring`, `internal/mcp`, `cmd/mrw` — no engine package: the classification lives in
`writer`, which already imports `apply` and works from `res.Files`, so every engine package stays byte-identical
(the 2026-09-30 design critique moved it out of `apply`, and out of `authoring`, which must not import the engine);
`go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `stats`, `stats --json` | sixth outcome `partially_applied`, counted in `landed` | T1 | CLI callers, scripts |
| the ledger | a partial commit's written files recorded | T2 | the next write |
| `mrw_write` result | receipt on a ledger failure; `error` field on any failed write | T3 | MCP hosts |
| `mrw_write` unreportable text | "do not re-run" | T4 | MCP hosts |
| `scripts/contract.sh` | §196 (T1, T2), §197 (T3) | T1–T3 | CI Linux |
| AGENTS, README, opencode plugin | the stats vocabulary and `Landed` | T1 | readers, the `mrw` skill |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `writer.MutationOf(res)` | T1 | T2, T3 | no |

## Implementation

See `tasks/README.md`: T1 (classification and the sixth outcome), T2 (the ledger after a partial commit), T3
(the MCP receipt on a ledger failure), T4 (the unreportable text).

## Consequences

- **Positive:** no surface counts or answers a changed tree as unchanged.
- **Negative:** a script summing the five old names now misses `partially_applied`; `stats --json` carries it
  at zero, so a script that reads `landed` is unaffected.
- **Neutral:** exit codes and hunk statuses are unchanged.

## Out of Scope

- The ignored cleanup errors and a partial commit's leftovers (deferred: `docs/adr/BACKLOG.md` "From ADR-102")
- Sharing the rest of the CLI/MCP write orchestration (deferred: `docs/adr/BACKLOG.md` "From ADR-102")
- A contract row for the unreportable text (permanent: boundary: the floor guard sizes the ceiling so the message is reached only by a write whose written-file records alone exceed it; T4's unit test drives `unreportableAt` directly)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| licensing a partial commit's files lets a re-sent plan land again on them | Low | Med | the licence is what a complete write already gives (ADR-005); the receipt names what landed |
| the `error` field changes the MCP output schema | Low | Low | additive and omitempty; the legacy golden is regenerated and logged |
| an older binary still running drops the unknown `partially_applied` key when it next saves the tally (`authoring.go:185-187`, `:212-213`) | Med | Low | accepted: the count restarts once the old process is gone; documented in the release notes |

## Rollback

Revert T1–T4. The tally file keeps any `partially_applied` count; an older binary ignores an unknown name.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `internal/writer/mutation102_test.go::TestMutationSaysHowMuchOfAPlanLanded` in T1's commit.
