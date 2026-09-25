# ADR-068: The read ledger keeps a path exactly as it was served

**Status:** Accepted
**Accepted:** 2026-09-25 by M — *"accept"*
**Date:** 2026-09-24
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-039, ADR-067, docs/adr/BACKLOG.md
**Governs:** `internal/seen/seen.go`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/ledgerpath_test.go::TestAReadOfATrailingSpacePathDoesNotLicenseItsTrimmedSibling`
**Invalidates:** none — checked
**Served-path change:** A read of a file whose name ends in a space or a carriage return no longer licenses a write to the file whose name is that one trimmed. Reproduced on v1.24.0, 2026-09-24: after `mrw read -- "x "`, a write to `x` applied, exit 0. Now it is refused as unread, exit 1, and a write to `x ` still applies. A ledger with CRLF line endings, which mrw never writes, is now discarded as stale.

## Context

**What was observed.** Codex, reviewing #214 on 2026-09-24, traced it from the source; it was then
reproduced end to end on the installed v1.24.0 (`7da61c1`):

```
printf 'same\n' > x; printf 'same\n' > 'x '
mrw read -- 'x '                                # serves only "x "
printf '@@ x 1 replace\nWROTE\n' | mrw write -   # applies: x now holds WROTE
```

The `--` matters. Without it, urfave/cli (v3.11.0, `command_parse.go:81`) trims every positional
argument before the first `--`, so `mrw read 'x '` reads `x` itself and its header says so. The first
repro in this record omitted `--` and so read the wrong file. The ledger defect needs the read to
reach `x `, which `--` does on the CLI and a spec does over MCP. The trimming is a separate finding,
deferred below.

**Why.**
- The ledger writes each observation as `<sha>  <spans>  <path>` with the path verbatim
  (`internal/seen/seen.go:399`).
- `parseLine` reads it back with `strings.TrimSpace(text)` (`seen.go:203`), which removes the
  path's trailing spaces. Leading spaces survive: they sit after the SHA and the spans.
- So the observation of `x ` is loaded under the key `x`. `apply` finds a licence for `x`, and its
  SHA guard passes because the two files hold the same bytes.
- ADR-002's guarantee, that mrw will not edit a file it has not read, fails for any pair of names
  that differ only in trailing whitespace. Its SHA check is the only thing standing in the way, and it
  stands only when the contents differ.
- **Amended 2026-09-25, from the Codex review of #214.** A trailing `\r` was trimmed one layer
  lower: `Load` scanned with `bufio.ScanLines`, which drops a `\r` before the `\n`. A read of a
  file named `x\r` licensed `x`. The first draft of this record kept that strip on purpose and
  claimed such a file could not be named in a plan; a quoted plan path carries the `\r` mid-line,
  so it can. Executed RED before the fix.

**Why now.** Until #214, an MCP read of a spaced name that fit on one page failed with `-32603`,
which hid the MCP half of this. #214 makes those reads work, so the ledger must key them exactly first.

**The class this record governs:** a path that round-trips through a state file and can come back
different. Enumerated 2026-09-24 with
`git grep -nE 'TrimSpace|Fields|bufio.NewScanner' -- internal/seen internal/iter internal/state internal/mcp/ack.go`:
- `internal/seen/seen.go:203`: the ledger. **Governed here.**
- `internal/mcp/ack.go` (pending checkpoints): stored as JSON (`:393`, `:409`), exact. Not a member.
- `internal/state/prune.go:407`: the root marker, which already says "never TrimSpace". Not a member.
- `internal/iter/iter.go:60`, `:109`, `:125`: the working set trims its lines. An `@N` pointer
  can then resolve `x ` to `x`, but a write there still needs a ledger licence for `x`. With this
  record, that is a refusal, not a silent write. Deferred (below).

## Existing Primitives Audit

- **`parseLine`** (`seen.go:221`) and the ledger scanner. Reshaped: `Load` and `IsStale` split on
  `\n` alone (`scanLF`), and `parseLine` trims nothing.
- **The ledger header** (`seen.go:152`). Unchanged. No format change, so every LF-terminated v2
  ledger mrw wrote is still accepted; one converted to CRLF by outside tooling now fails the header
  check and is discarded as stale. A line the writer emitted is read back exactly. One residue: an
  older mrw that LOADED `x ` as `x`
  and then saved the ledger wrote it back under `x`. That entry is kept, and it licenses `x` only
  while `x` still holds the bytes it was hashed from (Risks).

## Decision

The ledger reader keeps the path byte for byte as it was written: `Load` and `IsStale` split on
`\n` alone, the terminator `save` writes, and `parseLine` trims nothing. A read of `x ` licenses
`x ` and nothing else, and a read of `x\r` licenses `x\r`. No ledger format change, no migration,
no exit code change.

**What would make this decision fail:** a ledger mrw did not write, with CRLF endings. Its header
no longer matches, so it is discarded as stale and every write needs a fresh read: a refusal, never a
wrong licence.

## Alternatives Considered

- **Refuse edge-whitespace paths outright.** Rejected: they are legal names, the CLI already serves
  them, and a refusal is a regression for anyone who has one.
- **Bump the ledger header, discarding every existing ledger at upgrade** (the v0.0.11 precedent).
  Rejected: the only entry it would clear needs an edge-space name, a sibling with identical bytes,
  and a read by an older mrw. Every user would pay a re-read to clear it.
- **Change the ledger format (quote or escape the path).** Rejected: the writer already emits the
  path verbatim and the path is the last field, so reading it verbatim is enough. A format change
  would discard every existing ledger at upgrade for no gain.

## Component / Boundary Impact

`internal/seen` only, which is an engine package, so this record owns that change. Byte-identical:
`internal/read`, `internal/apply`, `internal/plan`, `internal/check`, `internal/state`,
`internal/lines`, `internal/iter`, `cmd/mrw` (tests aside).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| read ledger parsing | a path keeps its edge spaces | T1 | `mrw write`, MCP `mrw_write` |
| contract §127 | `read -- "x "` then a write to `x` exits 1, nothing written; to `x ` applies; the same for `x\r` | T1 | CI, `adr-verify` |

## Inter-task Contracts

None — one task.

## Implementation

See `docs/adr/ADR-068-the-read-ledger-keeps-a-path-exactly/tasks/README.md`.

## Consequences

- **Positive:** ADR-002 holds for names that differ only in a trailing space or carriage return.
- **Negative:** none known.
- **Neutral:** no format change; a ledger mrw wrote reads back correctly. One with CRLF endings, which mrw never writes, is discarded as stale.

## Out of Scope

- `internal/iter` trimming a working-set line, which can resolve an `@N` pointer to the trimmed name (deferred: docs/adr/BACKLOG.md)
- urfave/cli trimming a positional argument before `--`, so `mrw read 'x '` serves `x`; its header names `x`, so this is visible rather than silent, but the caller did not get the file they named (deferred: docs/adr/BACKLOG.md)
- Filenames containing a newline (permanent: boundary: every mrw document — plan, ledger, served header — is line-delimited, and ADR-005 models lines)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A ledger line carries a `\r` the writer did not put there | Low | Low | only mrw writes the ledger (`seen.go`, `save`), with a bare `\n`; a CRLF ledger fails the header check and is discarded, so the failure is a refusal |
| A ledger saved by an older mrw already holds a trimmed key (`x` for a read of `x `) | Low | Med | it licenses `x` only while `x` holds the hashed bytes; the SHA guard refuses once `x` changes, and a fresh read of `x ` records the exact key |

## Rollback

Revert `parseLine`, `scanLF`, the tests and §127. Nothing persistent moves.

## Follow-ups

- [ ] Release with #214 as v1.24.1.
