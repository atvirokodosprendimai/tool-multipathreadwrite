# Task ADR-144-T2: Only the commands that read the ledger tell its damage

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `tellsLedgerDamage`, the scan that does not hold the ledger open
**Consumes:** `seen.DamageNotice` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `only read, write and seen tell a damaged ledger, and the scan reads the ledger into memory and closes it before parsing`

## Goal

Three findings of the Codex review of PR #390, confirmed against the code: the notice repeated for ever on `check` and `iter`, which never save the ledger; `mcp` scanned the launch directory's ledger before it resolves its own root; and the scan held the ledger open for the whole parse, where a Windows writer's rename retries for 100 ms (`state.write`) and an open handle without delete sharing blocks it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `tellsLedgerDamage`, and `Before` asks it |
| `internal/seen/seen.go` | edit | `DamageNotice` reads the ledger whole and closes it, then parses |
| `cmd/mrw/damage144_test.go` | edit | the test |
| `docs/adr/ADR-144-a-damaged-read-ledger-is-told-once.md` | edit | Decision 4 and the boundary |

## Ordered Steps

1. [S1] Write `TestOnlyTheCommandsThatReadTheLedgerTellItsDamage`. Confirm RED.
2. [S2] `tellsLedgerDamage` and its call in `Before`. Mutant: it answers true for every verb. [proof: mutation]
3. [S3] `DamageNotice` reads the ledger into memory and closes the file before it parses. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestOnlyTheCommandsThatReadTheLedgerTellItsDamage' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestOnlyTheCommandsThatReadTheLedgerTellItsDamage \(' "$out" \
  && go test ./internal/seen/ ./cmd/mrw/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestOnlyTheCommandsThatReadTheLedgerTellItsDamage` | `cmd/mrw/damage144_test.go` | `read`, `write` and `seen` tell; `check`, `iter`, `mcp` and `version` do not, and `iter` prints nothing against a damaged ledger | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `tellsLedgerDamage` |
| 2 — something selects it | `Before` in `cmd/mrw/main.go` |
| 3 — the caller can discover it | the sentence appears where a command reads the ledger, and nowhere it could not heal |
| 4 — it is used | contract §248 drives `read`; no telemetry (ADR-009). S3 is not covered by a test: the Windows sharing mode it avoids cannot be reproduced on a unix host, and CI's Windows shards run the scan only against a ledger no writer holds |

## Invariants

- A ledger mrw wrote reports nothing, and `Load` is unchanged.

## Risks

- The ledger is read whole into memory: it is bounded by the files read in one checkout, and a hostile one is as large as a hostile `Load` input already is.

## Stop Condition

Stop and ask if a command that saves the ledger is found outside `read` and `write`.

## Out of Scope

- A dry-run or refused `write` that does not save, so the sentence repeats (permanent: boundary: it is accurate each time, and the remedy it names is a read)

## Mutation Log

## Verification Log
- 2026-10-10 · 951f712* · exit 1 · `set -o pipefail …` · acceptance-sha256:523b461efd9b614dcf88c8e1bfec32a90cafc2eb7597cf438c540be80dc69e5f · ms:251 · test-lock-sha256:b29296470e4a482646f504fd477cb6c0f1d6b71c3359e546924245495a2d509c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9kYW1hZ2UxNDRfdGVzdC5nbwlUZXN0QURhbWFnZWRMZWRnZXJJc1RvbGRPbmNlQnlUaGVDTEkJYTQ1YjcxMGM1NTBlNGVlNGQ1MjU3ZWFlNzRjZjdiNzRhNDBmODY4N2I3Yjk2ZDI3N2RkYTU2OGJiOTY5OWQ5Ngpib2R5CWNtZC9tcncvZGFtYWdlMTQ0X3Rlc3QuZ28JVGVzdE9ubHlUaGVDb21tYW5kc1RoYXRSZWFkVGhlTGVkZ2VyVGVsbEl0c0RhbWFnZQljYWExMzc3ZGY0OTdjNjc2MTkwNDZjZDRiZmI1YzQxMjgwMTg3NmVkYTcxMWY2Yzg4YWRjNDE2M2QzM2ZhYWIz
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test]
  cmd/mrw/damage144_test.go:58:13: undefined: tellsLedgerDamage
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [build failed]
  FAIL
  ```
