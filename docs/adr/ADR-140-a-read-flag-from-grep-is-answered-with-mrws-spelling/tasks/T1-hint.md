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
- 2026-10-10 · b5e00d3 · mutant killed · exit 1 · `cmd/mrw/foreignflag.go` · S2: the hint is given on every command, write included · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · covers:read given a grep flag it lacks exits 2 with mrw's spelling appended, and every other usage error is worded as before
- 2026-10-10 · b5e00d3* · mutant killed · exit 1 · `cmd/mrw/foreignflag.go` · S2: the flag lookup returns no hint · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · covers:read given a grep flag it lacks exits 2 with mrw's spelling appended, and every other usage error is worded as before
- 2026-10-10 · b5e00d3* · mutant survived · exit 0 · `cmd/mrw/foreignflag.go` · S2: any pattern beginning with a dash gets the hint · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · covers:read given a grep flag it lacks exits 2 with mrw's spelling appended, and every other usage error is worded as before
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-10 · b5e00d3* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the no-file-matched message carries no hint · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · covers:read given a grep flag it lacks exits 2 with mrw's spelling appended, and every other usage error is worded as before
- 2026-10-10 · d598450 · mutant killed · exit 1 · `cmd/mrw/foreignflag.go` · S2: any pattern beginning with a dash gets the hint · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · covers:read given a grep flag it lacks exits 2 with mrw's spelling appended, and every other usage error is worded as before

## Verification Log
- 2026-10-10 · b5e00d3 · exit 0 · `set -o pipefail …` · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · ms:2238
- 2026-10-10 · b5e00d3* · exit 0 · `set -o pipefail …` · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · ms:1763
- 2026-10-10 · b5e00d3* · exit 0 · `set -o pipefail …` · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · ms:1802
- 2026-10-10 · b5e00d3* · exit 0 · `set -o pipefail …` · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · ms:1667
- 2026-10-10 · d598450 · exit 0 · `set -o pipefail …` · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · ms:2246
- 2026-10-10 · d598450* · exit 1 · `set -o pipefail …` · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · ms:766 · test-lock-sha256:db3f46037cb53b635e38126d6d2143677e0586cb2dd80d3aed6f1227d9c8db06 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9mb3JlaWduZmxhZzE0MF90ZXN0LmdvCVRlc3RBRm9yZWlnblJlYWRGbGFnSXNBbnN3ZXJlZFdpdGhNcndzU3BlbGxpbmcJYjdjNDU3M2EwNDNiYzMzY2E0ZmU1MjIzMGVlZDI2OGYzODE3M2M0ODU1OWZhNjE2NGE5ZTY1MWI4M2Y4NWEwNgpib2R5CWNtZC9tcncvZm9yZWlnbmZsYWcxNDBfdGVzdC5nbwlUZXN0RXZlcnlGb3JlaWduRmxhZ0hpbnRJc0FUcnVlU3BlbGxpbmcJMTZjNThjYWRhNmU3Y2I2NzYxNzdiOWIxMTdkNzgwM2U1MjQ4MWMyM2RmY2Y3MmUzZjM3M2UxNjZlOTdiOWM2OA
  ```
  --- last 10 line(s) of stdout (of 71 after folding 71 raw)
      foreignflag140_test.go:35: mrw read --grep -F f.txt: exit 1, want 1 with the hint:
          no file matched /-F/
      foreignflag140_test.go:35: mrw read --grep -w f.txt: exit 1, want 1 with the hint:
          no file matched /-w/
  --- FAIL: TestAForeignReadFlagIsAnsweredWithMrwsSpelling (0.01s)
  === RUN   TestEveryForeignFlagHintIsATrueSpelling
  --- PASS: TestEveryForeignFlagHintIsATrueSpelling (0.04s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.296s
  FAIL
  ```
- 2026-10-10 · d598450* · exit 0 · `set -o pipefail …` · acceptance-sha256:c799bf8b58e0a48dfe63f7b2e75775c213734ef75af2d62757c89682a1b1367f · ms:880
