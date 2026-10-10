# Task ADR-140-T1: the usage error for a grep flag names mrw's spelling

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `foreignFlagHint` in `cmd/mrw/foreignflag.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `read given a grep flag it lacks exits 2 with mrw's spelling appended, and every other usage error is worded as before`

## Goal

Decisions 1–3 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/foreignflag.go` | add | the table and `foreignFlagHint` |
| `cmd/mrw/main.go` | edit | `usageError` appends the hint |
| `cmd/mrw/foreignflag140_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §241 |
| `AGENTS.md` | edit | the read section |

## Ordered Steps

1. [S1] Write `TestAForeignReadFlagIsAnsweredWithMrwsSpelling` (each table flag on `read` exits 2, keeps the old prefix and ends in its sentence; an unlisted flag, `write -i`, and `--grep` valid use are untouched) and `TestEveryForeignFlagHintIsATrueSpelling` (`(?i)`, `\Q…\E` and `\b…\b` do what their sentences say against a fixture). Confirm RED.
2. [S2] The table and the call in `usageError`. Mutants: the hint gated on any command (it appears on `write`); the table lookup returning "" for every flag. [proof: mutation]
3. [S3] Contract §241 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAForeignReadFlagIsAnsweredWithMrwsSpelling|TestEveryForeignFlagHintIsATrueSpelling' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAForeignReadFlagIsAnsweredWithMrwsSpelling \(' "$out" \
  && grep -qE '^--- PASS: TestEveryForeignFlagHintIsATrueSpelling \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 900s -run 'Usage|Subcommand|Flag' \
  && grep -q '^# 241\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAForeignReadFlagIsAnsweredWithMrwsSpelling` | `cmd/mrw/foreignflag140_test.go` | each table flag on `read` is exit 2 with the old prefix and its sentence; others untouched | none | S1, S2 |
| `TestEveryForeignFlagHintIsATrueSpelling` | `cmd/mrw/foreignflag140_test.go` | the runnable equivalents work against a fixture | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `foreignFlagHint` |
| 2 — something selects it | `usageError`, on every parser rejection |
| 3 — the caller can discover it | the error text itself; AGENTS.md |
| 4 — it is used | the survey (4 sessions) and the author's session; no telemetry (ADR-009) |

## Invariants

- Exit code 2 and the old message prefix are unchanged.
- Only `mrw read` gets a hint.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a table sentence cannot be run against the binary: a hint nobody checked is a guess.

## Out of Scope

- Other commands (deferred: docs/adr/BACKLOG.md)

## Mutation Log

## Verification Log
