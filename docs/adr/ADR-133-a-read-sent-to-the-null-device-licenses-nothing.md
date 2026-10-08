# ADR-133: a read sent to the null device licenses nothing

**Status:** Accepted
**Accepted:** 2026-10-08 by Zy — "Write a record for it (Recommended)", the option that read: a CLI read whose stdout is the null device counts as read for no lines, and stderr says so. That stops the read being used to edit those lines. Same idea as ADR-088 T4. It catches /dev/null only; it can't catch piping to head or grep.
**Date:** 2026-10-08
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-031, ADR-039, ADR-088
**Invalidates:** None — it narrows what a CLI read records; no accepted clause promises that a discarded read licenses
**Governs:** `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `cmd/mrw/nullread133_test.go::TestAReadSentToTheNullDeviceLicensesNothing`
**Served-path change:** `mrw read` whose standard output is the null device (`/dev/null`, `NUL`) still serves and exits as before, but records nothing in the read ledger and says so on stderr; a later write to those lines is refused as unread, as after a `--stat`.

## Context

mrw will not edit a line it has not served (ADR-002, narrowed to lines by ADR-005). On the CLI the ledger records a range once its answer is written to stdout (ADR-088 T4 moved the record after a checked flush, so an answer that could not be written licenses nothing). A caller who sends that answer to `/dev/null` has been served nothing it can see, yet the ledger records every line.

Reported 2026-10-08 by a quality-harness session, from its own work: it ran `mrw read path:A-B >/dev/null` several times and then edited those lines, each write accepted. Reproduced the same day with v1.50.0 in a scratch root: `mrw read f.txt:2 >/dev/null` exits 0 and `@@ f.txt 2 replace` then applies, exit 0. That is the failure the guard exists for — discipline that holds while the work is slow and drops once it gets fast — met by the tool rather than the caller.

The MCP surface does not have this gap: a served read there licenses nothing until it is acknowledged (ADR-031, ADR-039).

**Audit of the class** — *a path that records a CLI serve in the read ledger*: `mrw read --grep 'seen\.Record\(' cmd/ internal/` names one call on the CLI, the read action in `cmd/mrw/main.go`; the others are the MCP surface's ack path, which this record leaves alone. One site.

## Existing Primitives Audit

- **The checked flush before `seen.Record`** (ADR-088 T4) — the one place a CLI read decides whether its answer reached the caller; the null-device test sits beside it.
- **`--stat`** (ADR-005) — a read that serves no content records no span; the refusal a later write meets is the existing "has not been read: mrw served no lines".
- **`os.SameFile` and `os.DevNull`** — the standard library names the null device on each platform and compares two file infos; no new dependency.

## Decision

1. **A CLI read whose standard output is the null device records nothing.** Before `seen.Record`, the read action asks whether stdout is the null device: on unix, whether its file info is the same file as `os.Stat(os.DevNull)`; on Windows, whether the handle's device type is `FILE_DEVICE_NULL` (`NtQueryVolumeInformationFile`), since a Windows character device carries no file identity and `os.SameFile` would match a console too (the reviews of #350). When it is, the ledger is left as it was.
2. **It says so on stderr**, once per such read that served any lines: the answer went to the null device, so nothing was recorded and a write to these lines needs a read whose answer the caller sees. A read that served nothing says nothing more than it did. Stdout is untouched and the exit code is the read's own, so nothing that parses either changes.
3. **Nothing else is guessed.** A pipe, a file, a terminal and every other stdout record as before: mrw cannot see what a reader downstream of a pipe or a file does with the answer.

## Alternatives Considered

- **Warn and record anyway** — rejected by Zy, 2026-10-08: the next write still goes through, and a warning is exactly what the fast path stops reading.
- **Refuse the read (non-zero exit)** — rejected: the read served what it was asked; exit codes are a contract, and a script that probes existence with a discarded read would start failing for a write it never makes.
- **Leave it (the ledger records what was served, not what was looked at)** — rejected: on the CLI the null device is the one discard mrw can see, and seeing it costs one stat.

## Component / Boundary Impact

`cmd/mrw` only. `internal/seen` and every engine package stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` to the null device | records nothing; one stderr line | T1 | CLI callers |
| `scripts/contract.sh` | §234 | T1 | CI Linux |
| `AGENTS.md`, `README.md` | the read-before-write rule names it | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** `mrw read … >/dev/null` can no longer stand in for reading.
- **Negative:** a caller who discarded a read on purpose before writing must read the lines where it sees them; the refusal names the lines.
- **Neutral:** `| head -1`, `| grep`, `> file` and a caller who did not look still license, as before; this narrows the class and does not close it.

## Out of Scope

- A pipe, a file or a terminal the caller does not read (permanent: boundary: mrw cannot see past its own stdout; Decision 3)
- The MCP surface (permanent: boundary: ADR-031 and ADR-039 already license nothing until a read is acknowledged)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the null device does not stat as the same file on some platform | Low | the guard does nothing there, as before this record | the test opens `os.DevNull` on every CI platform, Windows included |
| a Windows console counted as NUL | Low after the fix | a read the caller sees licenses nothing | the device type is asked of the handle, which names NUL whatever access the handle has; no CI runner has a console on stdout, so the console case is traced, not run |

## Rollback

Revert T1: a read to the null device records again. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up.
