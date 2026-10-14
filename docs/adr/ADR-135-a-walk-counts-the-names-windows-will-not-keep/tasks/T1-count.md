# Task ADR-135-T1: the walk counts a discovered name Windows will not keep and says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `rooted.UnkeepableName`, `rooted.ErrWin32Alias`; `WalkSkipped.Unkeepable`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a discovered name Windows will not keep is counted on the skipped line and the rest is served`

## Goal

Decisions 1–4 of the record, with a test that fails before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | edit | `ErrWin32Alias`, `UnkeepableName`; `aliasRefusal` carries the sentinel |
| `internal/read/walk.go` | edit | the set, the count, the `SkipNote` clause |
| `internal/read/unkeepable135_test.go`, `internal/rooted/unkeepable135_test.go` | add | the tests |
| `docs/receipts.txt`, `internal/mcp/schema_test.go` | edit | `mcp_read skipped.unkeepable`, and its place in the read schema the receipt test holds to a real answer |
| `scripts/contract.sh` | edit | the §216 row that holds `skipped` to an exact object lists the new key |
| `AGENTS.md`, `README.md` | edit | the `-- skipped:` paragraph names it |

## Ordered Steps

1. [S1] Write `TestAWalkCountsTheNamesWindowsWillNotKeep` (Windows only, names made through `\\?\` paths): a tree holding `a.txt`, `aux.txt` and `trail.`, all matching; the walk serves `a.txt`, `WalkSkipped.Unkeepable` is 2 and `SkipNote` says so. Write `TestAnEscapeIsNotCountedAsAName` and `TestOnlyANameRefusalIsUnkeepable` (every platform). Confirm RED: on Windows the first fails on windows-latest, the second does not compile.
2. [S2] `UnkeepableName`, `ErrWin32Alias`, the set and the count in the walker, the `SkipNote` clause and tail. Mutant: `UnkeepableName` answers true for every refusal (the escape is counted). The walk-level mutant, the walk dropping in silence again, can only be run where a name is refused, on Windows: it is the red run on windows-latest. [proof: mutation]
3. [S3] `docs/receipts.txt`, AGENTS.md, README; the Windows red and green runs on windows-latest are recorded in the Verification Log. No contract row: only a Windows build refuses a name, and the contract runs on Linux. [proof: human: the Windows runs come from a CI runner, not from a test the contract can drive]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAnEscapeIsNotCountedAsAName' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnEscapeIsNotCountedAsAName \(' "$out" \
  && go test ./internal/rooted/ -count=1 -timeout 300s -run 'TestOnlyANameRefusalIsUnkeepable|TestADeviceNameReachedThroughALinkIsNotUnkeepable' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestOnlyANameRefusalIsUnkeepable \(' "$out" && grep -qE '^--- PASS: TestADeviceNameReachedThroughALinkIsNotUnkeepable \(' "$out" \
  && GOOS=windows go vet ./internal/read/ ./internal/rooted/ \
  && go test ./internal/rooted/ ./internal/read/ ./internal/mcp/ -count=1 -timeout 900s \
  && grep -q '^mcp_read skipped.unkeepable$' docs/receipts.txt \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWalkCountsTheNamesWindowsWillNotKeep` | `internal/read/unkeepable135_windows_test.go` | on a Windows build the rest is served, the names are counted and the note says so | none | S1, S2 |
| `TestAnEscapeIsNotCountedAsAName` | `internal/read/unkeepable135_test.go` | a refusal that is not about a name stays uncounted | none | S1, S2 |
| `TestOnlyANameRefusalIsUnkeepable` | `internal/rooted/unkeepable135_test.go` | `UnkeepableName` accepts the device-name refusal and rejects an escape and a state refusal | none | S2 |
| `TestADeviceNameReachedThroughALinkIsNotUnkeepable` | `internal/rooted/unkeepable135_link_test.go` | a device name reached through a link stays an `ErrDeviceName` and is not counted as the file's own name | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `UnkeepableName` and the count |
| 2 — something selects it | the walker's silent-drop site, on every discovered path |
| 3 — the caller can discover it | the `-- skipped:` line and `skipped.unkeepable`; AGENTS.md; README |
| 4 — it is used | the 2026-10-09 Windows chaos round (four sessions); no telemetry (ADR-009) |

## Invariants

- What a walk serves and every exit code are unchanged.
- A path the caller names is refused by name as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a refusal other than a device name or a Win32 alias must be counted for the grep to be explained: the oracle argument then needs a record of its own.

## Out of Scope

- `--ast-grep` hits (deferred: docs/adr/BACKLOG.md)

## Mutation Log
- 2026-10-09 · e1ee41b · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: UnkeepableName answers true for every refusal — an escape is counted as a name · acceptance-sha256:a7b7f697d1c917fbe3fbe62f8adc6d9f5550318847ed6255c106c0af0c532554 · covers:a discovered name Windows will not keep is counted on the skipped line and the rest is served
- 2026-10-09 · aa5727c · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: UnkeepableName answers true for every refusal — an escape is counted as a name · acceptance-sha256:a23e2ff6bee101ff081b7e4edd410e89760bf614e90704122b893a5c5dc74b1b · covers:a discovered name Windows will not keep is counted on the skipped line and the rest is served
- 2026-10-09 · aa5727c* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: a device name reached through a link is counted — the count says where a link leads · acceptance-sha256:a23e2ff6bee101ff081b7e4edd410e89760bf614e90704122b893a5c5dc74b1b · covers:a discovered name Windows will not keep is counted on the skipped line and the rest is served

## Verification Log
- 2026-10-09 · e1ee41b · exit 0 · `set -o pipefail …` · acceptance-sha256:a7b7f697d1c917fbe3fbe62f8adc6d9f5550318847ed6255c106c0af0c532554 · ms:33334
- 2026-10-09 · aa5727c · exit 0 · `set -o pipefail …` · acceptance-sha256:a23e2ff6bee101ff081b7e4edd410e89760bf614e90704122b893a5c5dc74b1b · ms:38639
- 2026-10-09 · aa5727c* · exit 0 · `set -o pipefail …` · acceptance-sha256:a23e2ff6bee101ff081b7e4edd410e89760bf614e90704122b893a5c5dc74b1b · ms:37275
- 2026-10-09 · aa5727c* · exit 1 · `set -o pipefail …` · acceptance-sha256:a23e2ff6bee101ff081b7e4edd410e89760bf614e90704122b893a5c5dc74b1b · ms:1614 · test-lock-sha256:609c9427e7467b17ad99dfc9a1c491ea3324c2bef276d2695687a24495b5bf6d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC91bmtlZXBhYmxlMTM1X3Rlc3QuZ28JVGVzdEFuRXNjYXBlSXNOb3RDb3VudGVkQXNBTmFtZQk0YjMzNzRlYjYzMGZlMGNiMzViYWVjODljYzY3MzZjZWMxMDAzZTc5MzE3ZGI2NmMyYmEwNTk4MDRhMzgwZDVjCmJvZHkJaW50ZXJuYWwvcmVhZC91bmtlZXBhYmxlMTM1X3dpbmRvd3NfdGVzdC5nbwlUZXN0QVdhbGtDb3VudHNUaGVOYW1lc1dpbmRvd3NXaWxsTm90S2VlcAkzNTVmNDlkMjVlNjAwNzM2OThkODRhYjI1NjMyNTIwZmQ5ZDVkZjViODI4MWI1YWE5YWFmZDBiMTYwYmQ3MTQwCmJvZHkJaW50ZXJuYWwvcm9vdGVkL3Vua2VlcGFibGUxMzVfbGlua190ZXN0LmdvCVRlc3RBRGV2aWNlTmFtZVJlYWNoZWRUaHJvdWdoQUxpbmtJc05vdFVua2VlcGFibGUJNDJiNTBhZDU5ZTA4MTE0YjM3NWM3YTliYTM1ZmVjMGJlMmZmYjM5OWZiOWZjYWMzNGFkY2FmZDNjOTk0YWM5Ywpib2R5CWludGVybmFsL3Jvb3RlZC91bmtlZXBhYmxlMTM1X3Rlc3QuZ28JVGVzdE9ubHlBTmFtZVJlZnVzYWxJc1Vua2VlcGFibGUJZjkyYjg2ZjIwMWQyZmRhOWYzZWFkZjMzODQxNzIyNTg4ZjllMjQ4NjRmMmE1ODAwOWNkNzE5YmY0ZDNiMTlkZg
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  === RUN   TestADeviceNameReachedThroughALinkIsNotUnkeepable
      unkeepable135_link_test.go:23: a direct device name was not counted: aux.txt leads to "aux.txt", which is a Windows device name: some Windows APIs still open it as a device on every build, so mrw neither creates, reads nor edits it
  --- FAIL: TestADeviceNameReachedThroughALinkIsNotUnkeepable (0.00s)
  === RUN   TestOnlyANameRefusalIsUnkeepable
      unkeepable135_test.go:16: deviceName(aux.txt) = aux.txt leads to "aux.txt", which is a Windows device name: some Windows APIs still open it as a device on every build, so mrw neither creates, reads nor edits it, want a name refusal
      unkeepable135_test.go:19: an alias refusal is not a name refusal
  --- FAIL: TestOnlyANameRefusalIsUnkeepable (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.179s
  FAIL
  ```
- 2026-10-09 · aa5727c* · exit 0 · `set -o pipefail …` · acceptance-sha256:a23e2ff6bee101ff081b7e4edd410e89760bf614e90704122b893a5c5dc74b1b · ms:36230
- 2026-10-10 · human-observed · Zy's session read windows-latest runs for the Windows-only walk test TestAWalkCountsTheNamesWindowsWillNotKeep: red 37988152676 on 4264d6f (windows-shard 4: Unkeepable = 0, want 2) and green 37989334974 on e1ee41b (14 pass, 2 skipping); the final head's run is recorded on the PR
