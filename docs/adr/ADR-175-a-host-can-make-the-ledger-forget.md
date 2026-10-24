# ADR-175: A host can make the read ledger forget, after a compaction

**Status:** Accepted
**Accepted:** 2026-10-24 by Zy — asked which drafted records to accept, chose *"ADR-171 guide cost, ADR-172 + ADR-119 D5, ADR-173 + ADR-174, ADR-175 forget"*, and for the breaking pair *"v1.76.0, loud note (Recommended)"*.
**Date:** 2026-10-24
**Amended:** 2026-10-24, before acceptance, from a Codex xhigh challenge (7 findings, each checked against the code): `--forget` also empties this root's pending MCP acknowledgements, which `internal/mcp/ack.go:487` promotes into the ledger without an existing observation (Decision 1); it clears what is persisted at its serialization point and does not revoke a command already in flight (Decision 2); `--sub`/`--rewrite` plans are exempt from the re-read rule (Decision 3); the hook takes the checkout from its input's `cwd`, since `CLAUDE_PROJECT_DIR` stays on the starting project after a worktree switch (Decision 4); T2's fence checks the hook as one block and both exclusions; rollback distinguishes before and after release; the BACKLOG receipt is filed with this record.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-034, ADR-038, ADR-118, ADR-133, ADR-150, ADR-161, ADR-169, docs/agents.md, README.md, AGENTS.md
**Governs:** `cmd/mrw/main.go`, `internal/seen/seen.go`, `internal/mcp/ack.go`, `docs/agents.md`, `scripts/contract.sh`
**Enforced-by:** `internal/seen/forget175_test.go::TestForgetDropsEveryLicenceInTheCheckout`
**Invalidates:** None — opt-in; the ledger, its format and every licence rule are unchanged until something calls the new flag.
**Served-path change:** `mrw seen --forget` empties this checkout's read ledger and says how many files it held; after it, a write needs a fresh read of the lines it touches. Nothing calls it unless a person or a host hook does.

## Context

The read ledger is on disk, outside the tree, per checkout (`internal/seen/seen.go:1-24`, ADR-004, ADR-161 Decision 14). When a host compacts its context, the model keeps a summary, not the lines it was served, and the ledger still licenses them: a write to a line read before the compaction is allowed although the model no longer holds that line. An outside A/B filed it (Rubie tech summary v1, 2026-10-24, mrw v1.74.0, issue MRW-12): with a 100K window, 2 writes without a fresh read after a compaction with mrw (8 with the host's Edit tool, which has the same blind spot), none broke a test. mrw cannot see a compaction; a host hook can (Claude Code's `SessionStart` hook with matcher `compact`, or `PreCompact`). The report's ask: something a host hook can bump so that old licences need a fresh read.

Zy chose this record on 2026-10-24 — *"A. Guide cost (Recommended), B. Bounded fixes, C. Contract ADRs"*.

## Existing Primitives Audit

- `seen.Drop(root, paths)` (`internal/seen/seen.go:651-673`): load, delete, save under `seen.lock`, never over a newer mrw's ledger (ADR-150). Forgetting is the same with every path; T1 adds `Forget` beside it, sharing `withLock`, `loadAll` and `save`. Reuse.
- `mrw seen` (`cmd/mrw/main.go:707-760`): the command that already owns the ledger's other operations (`--prune`, `--dry-run`). The flag goes there.
- An epoch counter stored in the ledger: rejected (Alternatives) — it changes the format (ADR-150's versioning) to do what emptying the ledger does.

## Decision

**1. `mrw seen --forget`.** Under `seen.lock`, it removes every observation in this checkout's ledger — read and written alike (ADR-118's `Written` files included: after a compaction the model holds a summary of what it wrote too) — and saves the empty ledger with its header (ADR-150). Before that, under the pending store's own lock (taken before any ledger lock, `internal/mcp/ack.go:79-83`), it empties this root's `pending-v3.json`: a served-but-unacknowledged MCP span is promoted into the ledger by a later `ack` when the file's sha still matches, with no existing observation needed (`ack.go:487`), so leaving it would let a pre-compaction serve license a post-compaction write. It prints `forgot N file(s): a write now needs a fresh read of the lines it touches`, exit 0; an empty ledger is `forgot 0 file(s)`, exit 0. A ledger written by a newer mrw is not touched: exit 2, the message being ADR-150's notice. `--forget` with `--prune` or `--dry-run` is a usage error, exit 2. It does not touch the tally (`mrw stats`), the working set or the state directory itself. Pending stores of other roots below the same checkout are not reached (they are per root); MCP is off by default (ADR-170).

**2. Whole checkout, not this session; what is persisted, not what is in flight.** The ledger has no sessions (ADR-038): every caller in the checkout shares it, so `--forget` costs a sibling session in the same checkout a re-read. That is the safe side — a refusal, never a wrong write — and it is why nothing calls `--forget` on its own. It clears what the ledger holds at the moment it takes `seen.lock`; a write already past its snapshot (`internal/writer/flow.go:152`, before the write lock at `internal/writer/writer.go:80`) or a read whose output was already printed may record after it. The documented hook runs at session start after a compaction, when that session has nothing in flight; a sibling's in-flight command is the stated limit.

**3. A caller-authored edit after it needs a read that shows the lines.** `mrw read --if-changed PATH=SHA` answers an unchanged file with one line and records nothing (ADR-169), and `mrw read --quiet` licenses lines without printing them (ADR-161 recipe 7); after a compaction neither puts the lines back in front of the model, so `docs/agents.md` says to re-read without them. The plan makers — `mrw write --grep … --sub`, `--ast-grep … --rewrite`, and their MCP forms (ADR-161 Decisions 1–3, 12) — compute each edit from the file's current bytes and license exactly those lines; they need no re-read and are not blocked.

**4. The consumer is a host hook, documented.** `docs/agents.md` §7's neighbour gains a short section with the Claude Code hook: on `SessionStart` with matcher `compact|clear`, a command that reads the hook input's `cwd` (Claude Code documents that `CLAUDE_PROJECT_DIR` stays on the starting project after a worktree switch, while `cwd` follows the session) and runs `mrw --root "$cwd" seen --forget` — written with `jq`, which the section names as a requirement. Other hosts need their own equivalent. mrw ships no hook and installs none.

**What would make this decision wrong:** sessions sharing a checkout re-reading so often after each other's compactions that the cost outweighs the two unlicensed writes measured. Nothing measures it yet; it is opt-in.

## Alternatives Considered

- **An epoch number in the ledger** (`mrw seen --new-epoch`; each observation stamped). Rejected: a format change under ADR-150's versioning to express "nothing is licensed", which an empty ledger already says, and older binaries would misread a newer epoch.
- **Expire licences by age.** Rejected: a compaction is an event, not a duration; any timeout is wrong for one host or another.
- **Per-session ledgers.** Rejected for this record: the ledger is the checkout's on purpose (ADR-161 Decision 14), and a session identity mrw does not have would be needed.
- **Do nothing; the report rated it Low.** Possible; Zy's choice of bundle C includes it.

## Component / Boundary Impact

- `internal/seen` (engine package, owned here): `Forget`.
- `internal/mcp`: `ForgetPending`, emptying this root's pending store under its lock.
- `cmd/mrw`: the flag on `mrw seen`.
No ownership change.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw seen --forget` | new flag; empties the ledger and this root's pending MCP acknowledgements | `cmd/mrw`, `internal/seen`, `internal/mcp` | host hooks, people |
| `docs/agents.md` | the compaction hook | docs | agent setups |
| ledger format | unchanged | — | — |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `mrw seen --forget` | T1 | T2 | No |

## Implementation

`tasks/README.md`: T1 (the flag), T2 (the hook recipe and the words). One pull request.

## Consequences

- **Positive:** a host that compacts can make every later write need a fresh read.
- **Negative:** sibling sessions in the same checkout lose their licences too.
- **Neutral:** without the hook nothing changes.

## Out of Scope

- Detecting a compaction inside mrw (permanent: boundary: mrw sees commands, not the host's context)
- Shipping or installing a hook (permanent: boundary: the host's configuration is the user's; the record documents it)
- Hooks for hosts other than Claude Code (deferred: docs/adr/BACKLOG.md, "ADR-175: compaction hooks for Codex and Cursor")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A sibling session is refused after another compacts | Med | Low | the refusal names the unread lines; Decision 2 says why |
| The documented hook's event or matcher differs in a Claude Code version | Med | Low | T2's human-observed step records the version it was run on |

## Rollback

Before a release ships it: remove the flag and the section; nothing persistent changes (an emptied ledger refills on the next read). After a release: the flag is a published command a documented hook calls, so it is deprecated — it keeps working and says it is deprecated for at least one minor release — before any removal, and the release notes name the hook to remove.

## Follow-ups

- [ ] none at authoring
