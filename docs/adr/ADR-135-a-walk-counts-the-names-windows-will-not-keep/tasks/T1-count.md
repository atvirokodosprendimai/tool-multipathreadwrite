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

## Verification Log
- 2026-10-09 · e1ee41b · exit 0 · `set -o pipefail …` · acceptance-sha256:a7b7f697d1c917fbe3fbe62f8adc6d9f5550318847ed6255c106c0af0c532554 · ms:33334
