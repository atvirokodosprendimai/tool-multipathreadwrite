# ADR-109: A loader opens only a regular file, and asks the descriptor

**Status:** Accepted
**Accepted:** 2026-10-01 by Zy — "work on the deffered ones", on the ADR-108 deferrals in `docs/adr/BACKLOG.md` "From ADR-108", of which the FIFO race is this record's scope. The record's text was drafted after that instruction and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-073, ADR-074, ADR-104, ADR-106, ADR-107, ADR-108
**Governs:** `internal/regular/regular.go`, `internal/read/read.go`, `internal/read/astgrep.go`, `internal/apply/apply.go`, `internal/apply/tree.go`, `internal/ingest/applypatch.go`, `internal/plan/plan.go`, `internal/check/check.go`, `internal/mcp/ack.go`, `internal/mcp/tools.go`, `internal/state/state.go`, `internal/seen/seen.go`, `internal/iter/iter.go`, `scripts/contract.sh`
**Enforced-by:** `internal/regular/regular_unix_test.go::TestOpenRefusesANonRegularFileAtOnce`
**Served-path change:** (1) `mrw check`, and a write whose check is due, no longer hang when `.quality-harness.json` is a FIFO: the config is refused as not a regular file, exit 2. (2) A file swapped for a FIFO, socket or device between a loader's path check and its open is refused at the open instead of blocking the call: `read`, `--grep`, `apply`, the foreign-format compilers, `body=@`, and the ast-grep path probe; a socket, which cannot be opened at all, gets the same refusal rather than the open's own error. (3) A FIFO at the legacy `.mrw/seen` or `.mrw/iteration` in the checkout no longer hangs every command: it is not migrated and loads as nothing. Refusal texts are unchanged. Exit codes keep their meanings.

## Context

ADR-073 and ADR-074 refuse a FIFO, a socket or a device before mrw opens it, because opening a FIFO for reading
blocks until something writes to it. They decide from a path `Stat`, and the open that follows is blocking, so a
file replaced by a FIFO in between hangs the call (the Codex re-review of #304; filed in BACKLOG "From ADR-108").
ADR-108's A6 and A8 closed two sites by opening without blocking and asking the descriptor.

**Audit of the class** — *an open that reads a caller- or checkout-named file*: `mrw read --grep
'os\.(Open|OpenFile|ReadFile)\(|\.OpenFile\(|\.Open\(' --exclude '*_test.go' internal/read internal/apply
internal/ingest internal/plan internal/check` — **9** sites. Six read a checkout file after a path check:
`read.readCapped` (`internal/read/read.go:63`, serving reads and the `--grep` walk), the ast-grep probe
(`astgrep.go:95`), `apply.readLines` (`apply.go:1764`), `tree.readDir` (`tree.go:151`), `ingest.targetBytes`
(`applypatch.go:314`) and `plan.readBounded` (`plan.go:917`). One reads a checkout file with no check at all:
`check.Load` (`check.go:92`) reads `.quality-harness.json` with `os.ReadFile` — a FIFO there hung `mrw check`,
measured 2026-10-01 on v1.37.2 (killed at 5 s). Two are not members: `check.lastLines` reads mrw's own log
under the state directory, and `check.go:114`/`:211` only stat. Outside these packages, ADR-108 closed
`mcp.currentSHA` and `mcp.countFileLines`; `authoring` reads mrw's own state; and `cmd/mrw` opens a plan file or a
`--files-from` list, where a pipe is a legitimate input. The Codex review of #306 found two more members: the
legacy pre-ADR-004 state in the CHECKOUT — `state.Migrate` (`internal/state/state.go:114`, `os.ReadFile` at every
CLI start), and `seen.Load`, `seen.IsStale` and `iter.load` when `ReadPath` picks `.mrw/seen` or `.mrw/iteration`
(a FIFO at `.mrw/seen` hung `mrw read` on v1.37.2, killed at 5 s) — and sockets, whose open fails before any
descriptor exists (EOPNOTSUPP on macOS, ENXIO on Linux), so a descriptor check alone never saw them.

## Existing Primitives Audit

- **`lines.NotRegular`** (`internal/lines/lines.go:52`) — the one refusal text, used by every site today.
- **ADR-108's open discipline** (`mcp.currentSHA`, `mcp.countFileLines`) — `O_NONBLOCK`, then `f.Stat()`; copied
  twice already, so it becomes one function rather than a seventh copy.

## Decision

1. A leaf package `internal/regular` has one function, `Open(path)`: it opens with `O_RDONLY|O_NONBLOCK`, takes
   the descriptor's `FileInfo`, and closes and refuses anything that is neither a regular file nor a directory with
   `ErrNotRegular`, whose text is `lines.NotRegular`. A directory is returned, so each caller keeps the error it
   gave for one. `O_NONBLOCK` makes the open of a FIFO return at once; on a regular file it has no effect, and on
   Windows Go ignores it.
2. `read.readCapped`, the ast-grep probe, `apply.readLines`, `ingest.targetBytes`, `plan.readBounded`,
   `check.Load` and the two ADR-108 MCP sites open through it. `tree.readDir` opens its directory through the root
   with `O_NONBLOCK` added.
3. The path checks before them stay where they give a message or a decision a caller relies on (ADR-073's
   refusal, ADR-106's identity, ADR-107's size); where a path `Stat` decided "regular" and nothing else read it
   (`targetBytes`, `LoadBodyFiles`), the descriptor now decides.
4. `apply` refuses a file the load finds is no longer regular on its hunk, as ADR-107 refuses one that grew, so
5. `regular.Open` classifies a failed open by the path: one that exists and is neither a regular file nor a
   directory is `ErrNotRegular`, so a socket is refused in the same words as a FIFO.
6. `state.Migrate`, `seen.Load`, `seen.IsStale` and `iter.load` open through `regular.Open`; a legacy file that is
   not regular is not migrated, and loads as nothing, which licenses nothing.
   the receipt keeps every verdict.

## Alternatives Considered

- **One shared size limit beside the helper** — rejected here: each loader's limit has its own test seam
  (`maxFileBytes`, `maxLoadBytes`, `maxTargetBytes`, `maxBodyBytes`, `maxCountBytes`) locked by ADR-104, ADR-107
  and ADR-108 tasks; merging them is churn this decision does not need.
- **`os.Root` for every read** — rejected: ADR-106 deferred the read path to its own trigger, and `os.Root` follows
  in-root links, so it would not by itself refuse a FIFO.

## Component / Boundary Impact

New leaf package `internal/regular` (imports `internal/lines` only). Engine packages owned: `internal/read`
(T1), `internal/apply` (T2), `internal/plan` (T3), `internal/check` (T4), `internal/state` and `internal/seen` (T5);
`internal/ingest`, `internal/mcp` (T3) and `internal/iter` (T5) are not engine packages. `internal/lines` and
`internal/rooted` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `internal/regular.Open` | the one non-blocking, descriptor-checked open | T1 | T1–T4 |
| `read`, `--grep`, ast-grep probe | a FIFO swapped in is refused at the open | T1 | callers |
| `apply` | a FIFO swapped in is refused on its hunk | T2 | callers |
| foreign formats, `body=@` | the descriptor decides | T3 | plan authors |
| `.quality-harness.json` | a FIFO is refused, exit 2 | T4 | `mrw check`, writes |
| legacy `.mrw/seen`, `.mrw/iteration` | a FIFO is not migrated and loads as nothing | T5 | every command |
| `scripts/contract.sh` | §207 (T4), §208 (T5) | T4, T5 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `regular.Open`, `regular.ErrNotRegular` | T1 | T2, T3, T4, T5 | no |

## Implementation

See `tasks/README.md`: T1 (helper, read), then T2–T5.

## Consequences

- **Positive:** no loader blocks on a FIFO, however the path came to name one; `mrw check` survives a FIFO config.
- **Negative:** none observed; a regular file opens as before.
- **Neutral:** refusal texts and exit codes are unchanged.

## Out of Scope

- The B1–B5 robustness items (deferred: `docs/adr/BACKLOG.md` "From ADR-108")
- Contract rows for the race sites (permanent: boundary: a swap between a path check and an open cannot be arranged from the shell without a seam; each loader's unit test drives it with a FIFO directly)
- `check.lastLines` and `authoring`'s loaders (permanent: boundary: they read mrw's own files under its state directory, which ADR-077 serves to nobody; the legacy files in the checkout are T5's)
- `cmd/mrw`'s plan file and `--files-from` list (permanent: boundary: a pipe is a legitimate input there, `mrw write -` and process substitution among them)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a filesystem where `O_NONBLOCK` changes a regular read | Low | Medium | POSIX gives it no effect on regular files; Linux and macOS measured; Go ignores it on Windows; Linux may fail EWOULDBLOCK on a file under an incompatible lease, where a blocking open would wait |

## Rollback

Revert T1–T5. No receipt or format change.

## Follow-ups

- None — the record carries no open follow-up.
