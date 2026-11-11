# Task ADR-117-T2: A spec list drops a leading byte order mark and says so

**Status:** pending
**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Repository owner
**Produces:** `speclist.Parse(r, label, notes)`: a leading byte order mark dropped from the first line, one note naming the list
**Consumes:** `specList` (`cmd/mrw/main.go`), `files_from` (`internal/mcp/tools.go`)
**Data dependency:** none; hermetic, no state home.
**Proof map:** v1
**Pilot:** the qh-core pilot task (Zy, 2026-11-11: *"ADR-117 task, strip and note on stderr, exempt?"*). Written by hand: qh has no authoring tool. Exempt from `adr-lint` and `adr-debt`; its evidence is qh's, not `adr-verify`'s.

## Goal

A `--files-from` list that starts with a UTF-8 byte order mark serves its first spec. Measured on
v1.86.5 (babbf03), 2026-11-11: `printf '\xef\xbb\xbfgo.mod:1\n' | mrw read --files-from -` printed
`==> ﻿go.mod  UNREADABLE  open …/﻿go.mod: no such file or directory`, exit 1, and a list whose first
line was `﻿# c` read the comment as a path. Notepad, `Out-File -Encoding utf8` in Windows PowerShell 5.1,
and a pipe under a UTF-8 `$OutputEncoding` with a preamble write that mark (the same source as ADR-142).

## Affected Files

| File | Change | Why |
|---|---|---|
| `internal/speclist/speclist_test.go` | add | `TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo` |
| `internal/speclist/speclist.go` | edit | drop the mark from the first line; write the note to `notes` |
| `cmd/mrw/main.go` | edit | `specList` passes `os.Stderr` |
| `internal/mcp/tools.go` | edit | `files_from` passes `nil`: the MCP caller has no stderr, and the server's own log is not the caller's |

## Ordered Steps

1. [S1] Add the test; run the fence: red (`Parse` takes two arguments).
2. [S2] Drop the mark from the first line in `Parse`, write `mrw: <label>: dropped the byte order mark at the start of the list` to `notes` when it is not nil; update both callers. Run the fence green; kill the planned mutants with `qh mutate-go`.

## Acceptance

```bash
go test -json -count=1 -p=1 ./internal/speclist
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|---|---|---|---|---|
| `TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo` | `internal/speclist/speclist_test.go` | the mark leaves the first spec and the first comment; no note without it; a mark past the first line is kept; a list of only the mark is empty | — | S1, S2 |

Planned mutants: the mark is dropped from every line, not the first (killed by "a mark past the first line is the spec's"); the note is not written (killed by "first spec").

## Invariants

A list with no leading mark parses exactly as before: same specs, no note. Edge spaces are kept (ADR-069).

## Stop Condition

A spec that legitimately starts with U+FEFF on its first line would be renamed; none is known, and a
path cannot be typed with one by accident.

## Out of Scope

- A note on `mrw_read`'s answer for a `files_from` list with the mark (deferred: Zy, if an MCP caller meets it).
- A contract row driving the built binary (external: this pilot is proven by qh's native Go evidence only).

## Reachability

| Rung | Evidence |
|---|---|
| 1 | `TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo` on `Parse` |
| 2 | `specList` and `files_from` call `Parse` (read, not tested by this fence) |

## Mutation Log

## Verification Log
