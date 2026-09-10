# ADR-019: Desktop reach is one named root per run

**Status:** Accepted
**Accepted:** 2026-09-10 by M — *"A"*
**Date:** 2026-09-10
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-004-mrw-leaves-nothing-in-the-working-tree.md`, `docs/adr/ADR-010-mrw-speaks-mcp-over-the-same-engine.md`, `docs/adr/ADR-011-the-mcp-server-tells-a-host-what-it-is-and-what-it-will-return.md`, `docs/adr/ADR-016-the-mcp-surface-says-what-it-is-not.md`, `docs/adr/ADR-017-the-mcp-surface-can-find-what-it-serves.md`, `docs/adr/ADR-018-a-root-nobody-named-is-not-a-root.md`, `docs/adr/ADR-031-a-page-licenses-only-what-came-back.md`, `docs/adr/ADR-038-a-ledger-write-is-one-writer.md`, `docs/adr/ADR-039-a-fitting-read-licenses-only-what-came-back.md`, `docs/adr/BACKLOG.md`
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/root_test.go::TestAWriteCannotSpendAnotherRootsLedger`
**Invalidates:** none — checked. ADR-016's clause that MCP's narrower reach is *not a defect to widen* still holds: pick A keeps one `mrw mcp` process on one tree. ADR-011's `ResolveRoot` precedence, ADR-018's explicit-versus-accidental guard, and ADR-010's two-tool shape are reaffirmed, not touched.
**Served-path change:** a Desktop (or other no-shell MCP) host names the tree with launch `--root` (`args: ["--root", "/abs/path", "mcp"]`). One process, one root, one ledger. N folders are N named servers. A write is licensed only by that root's ledger.

## Context

**The class this record governs.** Every way an MCP server, or an MCP host that is not a checkout
shell, decides which tree mrw will serve and which ledger licenses a later write. Enumerated
2026-09-10 with

```
rg -n 'ResolveRoot|CheckRoot|func Serve\(' --glob '*.go'
```

Four live production sites, one root each: `mcp.ResolveRoot`, `mcp.CheckRoot`, `mcpCmd` handing
that root to `mcp.Serve`, and `Serve` threading it into every tool handler. `hold` / `promote` and
`seen.Record` / `seen.Load` already key their files by that root (`state.Path(root, …)` /
`state.Dir(root)`). Members left out, and why:

- CLI `--root` — a shell user already points mrw at any checkout. ADR-018 left it unguarded
  because a typed root is stated intent.
- `apply.Apply` / `internal/plan` — a run takes one root today. This record does not change that
  (Decision 1). Touching them is the per-hunk ledger question, and it is refused here.
- MCP `check` / `iter` / `seen` / `stats` — cargo for this population (Decision 5).

**Why this number, and what M accepted.** ADR-017's Out of Scope, T1 and T2 said reach was
`ADR-018`. That was true when they were written. ADR-018 took the root *guard* (issue #81, explicit
versus accidental). **Reach is 019**, reserved in `docs/adr/BACKLOG.md` while 018–039 shipped. No
`docs/adr/ADR-019*` file existed before this one; 040 was not taken. M, 2026-09-10, on ranking
this as the leftover to start: *"ok, accepted, work on it"*. That is Accept of the ranking, not of
a drafted ADR. This text did not exist then. M named Fork 2 as A on 2026-09-10 (*"A"*); the ranking quote is not that pick.

**The population.** M, 2026-09-04: *"we need wider MCP coverage, which basically calls local mrw
under the hood … the potential here IS HUGE, for analysts reading and writing mega documents into csv
files … humans aren't structured in their daily work … this is the single biggest feature that we
ship only for coders but not desktop users."* M then split the work: *"Both, as two records"* —
capability in ADR-017 (`grep` / `exclude`, shipped), reach here.

**What is already true, and what is not.** Capability finds files inside the root the server
already had. Reach is how a host that is not a checkout shell *names* that tree, and whether one
process may hold more than one, without a write being licensed by a ledger that belongs to a
different tree.

A BACKLOG measurement of ~40 plans, all single-root, concluded "N registrations wins". Filed
2026-09-04, then corrected the same day: the count is accurate and the population is a coder inside
one git checkout — the population that already has `--root`. M named the Desktop analyst, whose
documents are not in one repository. For them, "one fixed checkout chosen at startup" is the
thing that stops the tool working. Issue #75 is the Desktop symptom (no `CLAUDE_PROJECT_DIR`,
fallback cwd, sometimes `/`); ADR-018 closed the accidental-`/` half, not the reach half.

**What a run is today.** `mcp.Serve(in, out, root string)` takes one root. `apply.Apply(root, …)`
takes one root. `seen.Load(root)` loads one ledger. `state.Dir(root)` hashes the resolved absolute
root, so N namespaces already exist. Pending checkpoints (`pending.json`, ADR-031 / ADR-039)
sit beside that ledger via `state.Path(root, pendingName)`. A cross-repo plan would need the
ledger resolved *per hunk*. That is not a small generalisation; it changes what a run is, and
neither tree's history would reflect the other.

**The handshake already claims one tree.** `internal/mcp/instructions.go` and
`TestTheSurfaceSaysTheCLIIsRicher` require that this surface serves *"ONE fixed checkout"* while
the CLI can be pointed anywhere. `maxInstructionsChars` is 4096. Any pick that makes one process
serve more than one tree must retarget that sentence in the same commit, without raising the bound.

## Existing Primitives Audit

- **`mcp.ResolveRoot` / `mcp.Source` (ADR-011):** `--root`, else `CLAUDE_PROJECT_DIR`, else cwd.
  **Reused.** A naming pick that stays single-root keeps this precedence. A pick that adds an
  allow-list or `roots/list` extends `Source`; it does not quietly drop `SourceFlag`.
- **`mcp.CheckRoot` (ADR-018):** a fallback onto a filesystem root or the home directory is refused;
  anything explicit is honoured, `/` and `$HOME` included. **Reused, not widened into a path
  blacklist.** Desktop documents really can live under the home directory; naming it stays legal.
- **`state.Dir` / `state.Path` (ADR-004, ADR-034):** one state namespace per resolved absolute root,
  already. Pending and the ledger already follow. **Reused.** N roots do not need a new keying
  scheme; they need a rule for which root a *call* uses.
- **`seen.Record` lock (ADR-038):** exclusive `seen.lock` per state directory. Two MCP servers on
  one checkout already serialize. **Untouched.**
- **Checkpoints and `hold` / `promote` (ADR-031, ADR-039):** pending is per-root. **Reused.** A
  process that may switch roots between calls must pass the *call's* root into `hold`/`promote`,
  not the process's launch root, or an ack from tree A licenses a write in tree B. T2's fixture
  after a multi-root pick is exactly that.
- **MCP `roots/list` (ADR-011 Alternatives):** the protocol's own answer. Rejected in ADR-011
  because it needs the server to *request* the client, a direction the transport does not go, and
  `CLAUDE_PROJECT_DIR` closed the same gap for Claude Code. The follow-up was *"if a second host
  needs it, or if `CLAUDE_PROJECT_DIR` proves host-specific"*. Both are now true of Desktop. **Not
  taken here.** It is Fork 2 option C, not a silent default.
- **A per-call `root` argument with no allow-list:** audited and **rejected** in Alternatives
  unless M overrides. The BACKLOG sketch this record exists to honour says the boundary is set by
  the launcher, not by a path mrw (or a model) can widen.
- **Per-hunk ledger / `apply.Apply` taking many roots:** audited and **not taken**. Decision 1.
- **MCP tools for `check`, `iter`, `seen`, `stats`:** audited and **not taken**. Decision 5.

## Decision

**A run has one named root, and the ledger that licenses a write is that root's ledger and no
other.** Reach does not change what a run is. It decides how a no-shell host *names* the tree,
and whether one `mrw mcp` process may be allowed more than one such tree.

Naming pick: A — launch --root only

**What Accept settles.** M, 2026-09-10, on Fork 2: *"A"*. That is launch `--root` only. N folders
are N named servers. `mrw mcp` stays single-root (Fork 3: yes). Do not measure Desktop
`roots/list`. Do not implement B or C. The ranking quote *"ok, accepted, work on it"* authorised
drafting; *"A"* is the naming pick.

**What this record settles**, from records already Accepted plus that pick:

1. **One root per run, one ledger per that root.** `apply.Apply` and `seen.Load` stay one-root.
   `state.Dir` already namespaces per resolved absolute root; pending already follows. A plan does
   not span two trees. The engine directories `internal/read`, `internal/apply`, `internal/plan`,
   `internal/seen`, `internal/check`, `internal/state` stay byte-identical against the merge-base
   unless a later Accepted amendment of *this* record names them. A cross-root atomic plan is a
   different record.

2. **ALLOW-ONLY if a process is ever allowed more than one root.** Pick A does not allow that.
   A deny list needs symlink resolution and reintroduces path-handling bugs this corpus already
   closed (BACKLOG's reach sketch, 2026-09-04). B/C would have named the list at launch (or by
   the host); they are not this pick.

3. **The boundary is not a model-invented path.** An unconstrained per-call `root` on
   `mrw_read` / `mrw_write` is how a caller points mrw at `/` after ADR-018 refused the fallback.
   Rejected unless M overrides.

4. **ADR-018's explicit-versus-accidental rule stays.** `--root /` still starts. A fallback onto
   `/` or `$HOME` still does not. `ResolveRoot`'s precedence is unchanged. Pick A does not
   extend `Source`.

5. **Cargo stays parked.** `--check`, `iter`, `seen`, `stats` are not MCP tools. `--files-from`
   stays off this surface (ADR-017 Decision 4). This record does not reopen them.

6. **`maxInstructionsChars` stays the literal 4096.** Do not raise it. Do not generate
   `AGENTS.md` from `guide.Shared()`. Do not retag v1.11.0. Do not promote torn `Load`. Do not
   lock the target file.

**Forks, settled 2026-09-10.**

**Fork 1 — one MCP server / one ledger, versus one ledger per root.** Settled as engine: **one
ledger per root**, via `state.Dir`. A server bound to one root therefore has one ledger.

**Fork 2 — how a Desktop session names the root.** **A — launch `--root` only.** M: *"A"*.
Desktop config: `"args": ["--root", "/abs/path", "mcp"]`. N folders = N named servers. B and C
are not taken. Do not measure Desktop `roots/list`.

**Fork 3 — whether `mrw mcp` stays single-root.** **Yes.** `Serve(in, out, root string)` stays.
No allow-list. No per-call `root`.

**Rejected:** a per-call `root` argument with no allow-list; a plan that applies across two trees;
MCP `check` / `iter` / `seen` / `stats`.

**Go/no-go, checked during execution of T2 once a pick exists. If any fails, the task is withdrawn
rather than shipped:**

- **`internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`,
  `internal/state` stay byte-identical** against the merge-base.
- **`go.mod` still declares exactly one requirement.**
- **`maxInstructionsChars` is still the literal 4096.**
- **ADR-018 still honours an explicit `--root /` and still refuses a fallback `/`.**
- **A CLI `mrw --root DIR read` still points at any checkout.** This record does not guard the CLI.
- **Contract section 78 is the next free number** — highest on `origin/main` (`c444fc5`) is
  `# 77.` (ADR-039). Do not collide with 040; this record is 019.

**What would falsify Decision 1:** a write that applies to a path under root B using a pending
checkpoint or `seen` entry that `hold` recorded under root A. That is the lying-ledger failure this
number exists to prevent. T2's two-root fixture is the data that can produce it, once a pick lets
one process see two roots. Under pick **A** the fixture is two `mrw mcp` processes, which already
isolate via `state.Dir`; the Enforced-by then asserts the isolation rather than a new switch.

## Alternatives Considered

- **Leave Desktop on N registrations and `--root` (pick A) without a record.** Today's README.
  Rejected as *silence*: M named this the largest open product question, ADR-017 deferred it here,
  and the coder-population measurement was already filed as the wrong population. A record that
  ratifies A is a valid *pick*; it is not the default this draft may encode.
- **Per-hunk ledger, one plan across two git trees.** The BACKLOG cost. Rejected here: it changes
  what a run is, it touches `apply.Apply`, and an all-or-nothing plan spanning two histories is a
  different trust question. Revisit only when a caller routinely changes a contract and its
  consumer in one atomic write.
- **Unconstrained per-call `root`.** The model names any directory. Rejected: it is ADR-018's
  fallback `/` wearing a parameter, and it contradicts "boundary set by the launcher".
- **Deny list.** Rejected by the BACKLOG sketch this record carries: symlink resolution, and every
  path bug already closed.
- **MCP `check` / `iter` / `seen` / `stats` as the Desktop answer.** Rejected: BACKLOG and
  ADR-017. `--check` is a Go-test runner; the others are introspection. Finding is ADR-017;
  reach is the root.
- **`roots/list` as the silent default.** Tempting, because ADR-011's own follow-up is now due.
  Rejected as a *silent* default: it needs a client request the transport does not make, and Desktop
  has not been measured to send roots. It remains option C.
- **Wait for a Desktop-population measurement before drafting.** Rejected: M said work on it, and
  the record can hold the settled constraints plus the forks. Guessing a Desktop in code is what
  T1 was waiting on; Status is now Accepted on *"A"*.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | Which tree the server serves, and now how a no-shell host names it | Yes — this is reach |
| `cmd/mrw` | `mcpCmd` is the call site for `ResolveRoot` / `CheckRoot` / `Serve` | Wiring only, after a pick |
| `internal/seen` / `internal/state` | Unchanged keying | No — already per-root |
| `internal/apply` / `internal/plan` | Unchanged | No — Decision 1 |

No new module. No architecture-doc delta (this repository has no `docs/architecture.md`).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| launch `--root` | pick A: one named root per `mrw mcp` process; already the flag | T2 | Desktop / MCP hosts |
| `scripts/contract.sh` | `# 78.` — two `--root`s, shared state home; A's ack must not license B | T2 | CI |
| handshake `instructions` | *"ONE fixed checkout"* stands; name launch `--root` as how this surface is pointed | T3 | MCP hosts |
| `TestTheSurfaceSaysTheCLIIsRicher` | still true; T3 adds `TestTheSurfaceNamesTheRootThePickChose` | T3 | the old contract's own test |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| Naming pick encoded in this Decision as `Naming pick:` | T1 | T2, T3 | No — docs |
| Binary behaves as the pick | T2 | T3 | ⚠ Yes if B or C: one process may see more than one tree; a write still cannot spend another root's ledger |
| Teaching matches the pick | T3 | — | Yes for handshake if B or C |

## Implementation

See `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run/tasks/README.md`.

T1 encoded M's pick A. T2 pins two-root isolation through the built binary (`# 78.`). T3 teaches
that this surface is still one `--root` at launch.

## Consequences

- **Positive, once a pick ships:** a no-shell caller can work in the tree they meant, and a write
  cannot be licensed by a ledger from a tree they did not name.
- **Positive now:** the forks are named, so the next session cannot re-derive "N registrations
  wins" from the coder measurement, and cannot sneak cargo or a per-hunk ledger in as reach.
- **Negative, if B or C:** the handshake's guarded *"ONE fixed checkout"* sentence moves, and
  every existing MCP caller that assumed one tree for the process's lifetime has to re-read.
  Fail-safe if isolation holds; breaking if teaching lags.
- **Negative, if A:** Desktop still needs N config entries (or one `--root` at a documents
  folder). That may be the right product. It is not proven by the coder-plan count.
- **Neutral:** CLI `--root` is unchanged. ADR-017's find is unchanged. ADR-018's guard is
  unchanged.

## Out of Scope

- MCP tools for `check`, `iter`, `seen`, `stats` (permanent: boundary: BACKLOG's Desktop entry and ADR-017 Decision 5 — a Go-test runner and introspection are not this population's work)
- `--files-from` over MCP (permanent: boundary: ADR-017 Decision 4 — it exists to undo shell word-splitting and MCP has no shell)
- A plan that applies across two roots / a per-hunk ledger (permanent: boundary: Decision 1 — that changes what a run is and owns `internal/apply`)
- An unconstrained per-call `root` argument (permanent: boundary: Decision 3 — the launcher or host names the allow-list; a model-invented path is ADR-018's fallback wearing a parameter)
- Guarding the CLI's `--root` (permanent: boundary: ADR-018 — a typed root is stated intent)
- Raising `maxInstructionsChars` / 4096 (permanent: boundary: M's standing instruction and ADR-037's go/no-go)
- Generating `AGENTS.md` from `guide.Shared()` (permanent: boundary: ADR-037 deferred that)
- Locking the target file a plan writes (permanent: boundary: ADR-002 parked that)
- Promoting torn `Load` / atomic save via rename (deferred: `docs/adr/BACKLOG.md` — From ADR-038)
- Syntax awareness, streaming apply, Windows `%LOCALAPPDATA%` (permanent: boundary: parked by the ranking this record was accepted to start)
- A Desktop-population measurement of how many trees one session actually needs (deferred: `docs/adr/BACKLOG.md` — this record's Follow-ups; the coder count is not that measurement)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A session encodes pick A, B, or C because it is convenient, not because M named it | Low | High — a guessed Desktop | Closed: M named *"A"*. T1's Accepted quote is that word |
| Pick B/C shares one `pending.json` across roots | **High** | **Critical** — the lying ledger | `hold`/`promote` already take `root`. T2's two-root fixture acks A and writes B; a single-root test is green against today's `Serve` |
| Handshake overflows 4096 when *"ONE fixed checkout"* is retargeted | Med | High | Go/no-go: shorten MCP-only prose; Stop Condition if the bound is the proposed fix |
| `roots/list` is picked and Desktop sends nothing | Med | High | Option C needs a measured Desktop `roots/list` or a fallback that is still explicit (`--root`), never `/` |
| ADR-016's reach clause is left standing after B or C ships | Med | High | Invalidates names the clause; T3 moves `TestTheSurfaceSaysTheCLIIsRicher` in the same commit |
| Contract `# 78.` collides | Low | High | Highest on `c444fc5` is `# 77.`; grep before writing the row |

## Rollback

Revert the commits. Pick A is additive config. Picks B and C are additive allow-lists; removing
them restores one-root `Serve`. No ledger format change. Pending files already written per-root
stay valid. CLI unaffected.

## Follow-ups

- [x] M picks Fork 2. 2026-09-10: *"A"* — `Naming pick: A — launch --root only`. Fork 3 stays
      single-root. T2.
- [x] `roots/list` measurement — not taken. Pick A does not wire a client request.
- [ ] Measure the walk on a real document tree once a root model exists (ADR-017 Follow-up).
- [ ] If a caller routinely needs one atomic plan across two git trees, that is a new record, not
      an amendment that quietly grows `Apply`.
