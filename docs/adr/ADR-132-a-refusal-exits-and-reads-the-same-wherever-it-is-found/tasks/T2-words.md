# Task ADR-132-T2: a receipt says it once, in the caller's words

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `apply.Slashed`, `exitTwoLine`, the INTERRUPTED / TIMED OUT headline
**Consumes:** `refuseStage` (T1)
**Data dependency:** hermetic — the converter takes the separator, so `\` is driven on every platform; the served CLI receipt is asserted on the `windows-latest` shard
**Proof map:** v1
**Rests-on:** `a receipt says it once, in the caller's words`

## Goal

Decisions 3–6 of the record, each with a test that fails before it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/wirepath.go` | add | `Slashed(res, sep)`, moved from `internal/mcp/wirepath.go` |
| `internal/mcp/wirepath.go`, `internal/mcp/tools.go` | edit | `mrw_write` calls `apply.Slashed`; the headline |
| `cmd/mrw/main.go` | edit | the receipt, refusals, text, `drift` and `created d/` through `Slashed`; the exit-2 line; the headline |
| `internal/apply/apply.go` | edit | the read advice cut by `NoForce` |
| `cmd/mrw/words132_test.go`, `internal/mcp/words132_test.go` | add | the tests |
| `internal/mcp/testdata/legacy_golden.jsonl`, `cmd/mrw/testdata/outcomes113.golden` | edit | the read advice; the timeout headline |
| `scripts/contract.sh` | edit | §232 |
| `AGENTS.md` | edit | the sentences that quote these words |

## Ordered Steps

1. [S1] Write `TestTheCLIReceiptSpellsPathsWithSlash` (a `\`-separated result through the CLI's receipt builder gives `/` in `--json`, the text, `drift` and `created d/`; on Windows a real `d/f.txt` write's served receipt says `d/f.txt`), `TestAnExitTwoLineNamesTheFailLineInsteadOfRepeatingIt` (and keeps a ledger failure the error also carries), `TestAnMCPRefusalNamesMrwRead`, `TestAStoppedCheckIsHeadedByWhatStoppedIt` (interrupted and timed out, before it started and while it ran, CLI and MCP). Confirm RED. [proof: mutation]
2. [S2] Move `slashResult` to `apply.Slashed`; the CLI builds every receipt through it; the exit-2 line; the read advice under `NoForce`; the headline. Mutants: `Slashed` not called by the CLI's write; the exit-2 line printing the error whole; the headline reading `verdict` for a stopped check. [proof: mutation]
3. [S3] The two goldens; contract §232 — a check stopped by its timeout is headed TIMED OUT, and the pair, a failing check still headed FAIL; AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 600s -run 'TestTheCLIReceiptSpellsPathsWithSlash|TestTheCLIWriteBuildsItsReceiptThroughTheConverter|TestAnExitTwoLineNamesTheFailLineInsteadOfRepeatingIt|TestAnMCPRefusalNamesMrwRead|TestAStoppedCheckIsHeadedByWhatStoppedIt' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCLIReceiptSpellsPathsWithSlash \(' "$out" \
  && grep -qE '^--- PASS: TestAnExitTwoLineNamesTheFailLineInsteadOfRepeatingIt \(' "$out" \
  && grep -qE '^--- PASS: TestAnMCPRefusalNamesMrwRead \(' "$out" \
  && grep -qE '^--- PASS: TestAStoppedCheckIsHeadedByWhatStoppedIt \(' "$out" \
  && grep -qE '^--- PASS: TestAStoppedCheckIsHeadedByWhatStoppedItOverMCP \(' "$out" \
  && grep -qE '^--- PASS: TestTheCLIWriteBuildsItsReceiptThroughTheConverter \(' "$out" \
  && go test ./internal/apply/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 232\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCLIWriteBuildsItsReceiptThroughTheConverter` | `cmd/mrw/words132_test.go` | the CLI write's served `--json` receipt is built through the converter, driven with `_` as the separator (the survived mutant: on unix `/` is already the separator) | none | S2 |
| `TestTheCLIReceiptSpellsPathsWithSlash` | `cmd/mrw/words132_test.go` | the CLI's receipt, refusal, text, drift and created lines spell paths with `/` | none | S1, S2 |
| `TestAnExitTwoLineNamesTheFailLineInsteadOfRepeatingIt` | `cmd/mrw/words132_test.go` | the exit-2 line names the FAIL line, does not repeat its reason, and keeps a ledger failure | none | S1, S2 |
| `TestAnMCPRefusalNamesMrwRead` | `internal/mcp/words132_test.go` | an mrw_write on an unread file names mrw_read and not `mrw read` | none | S1, S2 |
| `TestAStoppedCheckIsHeadedByWhatStoppedIt` | `cmd/mrw/words132_test.go` | interrupted and timed-out checks, before and while running, are headed so on the CLI; a failed check is still FAIL | none | S1, S2 |
| `TestAStoppedCheckIsHeadedByWhatStoppedItOverMCP` | `internal/mcp/words132_test.go` | the same on mrw_write; a failed check is still FAILED | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four behaviours |
| 2 — something selects it | the write command's receipt, error path and report; `writeTool`; `reportCheck`, `checkReport` |
| 3 — the caller can discover it | the receipts and lines; AGENTS.md |
| 4 — it is used | the Windows peers' field tests of 2026-10-06; no telemetry (ADR-009) |

## Invariants

- No receipt key is added or removed; exit codes are T1's.
- On unix the CLI's `--json` receipt is byte-identical.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a receipt path turns out to be read back by mrw itself in its `/` spelling on Windows.

## Out of Scope

- The CLI `read --json` receipt's paths (permanent: boundary: the record's)

## Mutation Log
- 2026-10-06 · d493df7* · mutant survived · exit 0 · `cmd/mrw/main.go` · S2: the CLI write does not convert its receipt — paths keep the separator · acceptance-sha256:efb51a247c65debb4b2600ea953d91e3b8ac881dda72709fbbdba09025ae094a
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the exit-2 line prints the error whole — the FAIL reason twice · acceptance-sha256:efb51a247c65debb4b2600ea953d91e3b8ac881dda72709fbbdba09025ae094a
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the CLI headline reads verdict for a stopped check — TIMED OUT headed FAIL · acceptance-sha256:efb51a247c65debb4b2600ea953d91e3b8ac881dda72709fbbdba09025ae094a
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the CLI write does not convert its receipt — paths keep the separator · acceptance-sha256:d2606d15f63e5ac00f77edc12b987e04c76ed092dffbaa3d1381a33fddd7f93d
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the exit-2 line prints the error whole — the FAIL reason twice · acceptance-sha256:d2606d15f63e5ac00f77edc12b987e04c76ed092dffbaa3d1381a33fddd7f93d
- 2026-10-06 · d493df7* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the CLI headline reads verdict for a stopped check — TIMED OUT headed FAIL · acceptance-sha256:d2606d15f63e5ac00f77edc12b987e04c76ed092dffbaa3d1381a33fddd7f93d

## Verification Log
- 2026-10-06 · d493df7* · exit 1 · `set -o pipefail …` · acceptance-sha256:efb51a247c65debb4b2600ea953d91e3b8ac881dda72709fbbdba09025ae094a · ms:732 · test-lock-sha256:e352cfedc64407d6fae2282b5522f25bff053929c36c898c595bb47e0a645a79 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy93b3JkczEzMl90ZXN0LmdvCVRlc3RBU3RvcHBlZENoZWNrSXNIZWFkZWRCeVdoYXRTdG9wcGVkSXQJMmY4MjMwZTg1ZWQxMjk1OGIzNzA5MDQzMTc2NTU2MmY0ZTkxZmI5ODNjOTQ1NWJkODRhMzBmZDhkMDJkNGJhMwpib2R5CWNtZC9tcncvd29yZHMxMzJfdGVzdC5nbwlUZXN0QW5FeGl0VHdvTGluZU5hbWVzVGhlRmFpbExpbmVJbnN0ZWFkT2ZSZXBlYXRpbmdJdAkxZjc5Y2JkNDJiODgyNDA5NmUyMWZmMGMzMjdhMDMzMzcxMWFlZTkzMzYwZWEyMzIwMGQ2YTRjOTM4NTQxMTUyCmJvZHkJY21kL21ydy93b3JkczEzMl90ZXN0LmdvCVRlc3RUaGVDTElSZWNlaXB0U3BlbGxzUGF0aHNXaXRoU2xhc2gJZTgwMDE3ODhiMDY4MjBlYWM2M2M0MDQ4ODExZDAzYjM1NmEyMjU0MWYxZGY0M2RhNWZmNzA1OTkzZTdkNTY1MApib2R5CWludGVybmFsL21jcC93b3JkczEzMl90ZXN0LmdvCVRlc3RBU3RvcHBlZENoZWNrSXNIZWFkZWRCeVdoYXRTdG9wcGVkSXRPdmVyTUNQCTZkYzljYjYyNDQyNmVmZTc5OGVjNTdkNjg5OGUzM2Y0MTM0ZjllMjRjNGJjNDU3NmQwODg3NWM1MWY4MWIwMDYKYm9keQlpbnRlcm5hbC9tY3Avd29yZHMxMzJfdGVzdC5nbwlUZXN0QW5NQ1BSZWZ1c2FsTmFtZXNNcndSZWFkCWIzMDEzZTY3MDgzZDQ0MzFkOTdiOTU4MTNmMDVkNjg5YjA5NWNkYTVmMzA5MWY1M2ViNzY4MDZhNjNhNzBkOGY
  ```
  --- last 10 line(s) of stdout (of 23 after folding 23 raw)
          1 hunk(s), 1 file(s), 1 failed
  --- FAIL: TestAnMCPRefusalNamesMrwRead (0.01s)
  === RUN   TestAStoppedCheckIsHeadedByWhatStoppedItOverMCP
      words132_test.go:38: "interrupted": want "check INTERRUPTED", got "check FAILED (exit -1): sleep 30 — interrupted — the write applied; the tree is changed and unverified. Do not re-send the plan: it applied.\n"
      words132_test.go:38: "timed out after 5s": want "check TIMED OUT", got "check FAILED (exit -1): sleep 30 — timed out after 5s — the write applied; the tree is changed and unverified. Do not re-send the plan: it applied.\n"
      words132_test.go:38: "timed out before it started": want "check TIMED OUT", got "check DID NOT RUN: timed out before it started — the write applied; the tree is changed and unverified\n"
  --- FAIL: TestAStoppedCheckIsHeadedByWhatStoppedItOverMCP (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.186s
  FAIL
  ```
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:efb51a247c65debb4b2600ea953d91e3b8ac881dda72709fbbdba09025ae094a · ms:38385
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:efb51a247c65debb4b2600ea953d91e3b8ac881dda72709fbbdba09025ae094a · ms:52090
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:efb51a247c65debb4b2600ea953d91e3b8ac881dda72709fbbdba09025ae094a · ms:47951
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2606d15f63e5ac00f77edc12b987e04c76ed092dffbaa3d1381a33fddd7f93d · ms:38140
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2606d15f63e5ac00f77edc12b987e04c76ed092dffbaa3d1381a33fddd7f93d · ms:37808
- 2026-10-06 · d493df7* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2606d15f63e5ac00f77edc12b987e04c76ed092dffbaa3d1381a33fddd7f93d · ms:37603
- 2026-10-06 · d493df7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d2606d15f63e5ac00f77edc12b987e04c76ed092dffbaa3d1381a33fddd7f93d · ms:0 · test-lock-sha256:b9c1be9f246241dc07051c410746c4b1d093410b0977962c470fe12169912597 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy93b3JkczEzMl90ZXN0LmdvCVRlc3RBU3RvcHBlZENoZWNrSXNIZWFkZWRCeVdoYXRTdG9wcGVkSXQJMmY4MjMwZTg1ZWQxMjk1OGIzNzA5MDQzMTc2NTU2MmY0ZTkxZmI5ODNjOTQ1NWJkODRhMzBmZDhkMDJkNGJhMwpib2R5CWNtZC9tcncvd29yZHMxMzJfdGVzdC5nbwlUZXN0QW5FeGl0VHdvTGluZU5hbWVzVGhlRmFpbExpbmVJbnN0ZWFkT2ZSZXBlYXRpbmdJdAk4MzdjZGJjN2ZkNzQ3OGVlMThiM2Y2ZWU0MWQwOTA2YTk0MWFmNzQ3MzdmNzY4N2E4Njc0M2ViZTY5ZTA0YTA1CmJvZHkJY21kL21ydy93b3JkczEzMl90ZXN0LmdvCVRlc3RUaGVDTElSZWNlaXB0U3BlbGxzUGF0aHNXaXRoU2xhc2gJZTgwMDE3ODhiMDY4MjBlYWM2M2M0MDQ4ODExZDAzYjM1NmEyMjU0MWYxZGY0M2RhNWZmNzA1OTkzZTdkNTY1MApib2R5CWNtZC9tcncvd29yZHMxMzJfdGVzdC5nbwlUZXN0VGhlQ0xJV3JpdGVCdWlsZHNJdHNSZWNlaXB0VGhyb3VnaFRoZUNvbnZlcnRlcgk3YzY1OTRmY2U3NzA3MTliZDM2YWM4MTNlYzU0YzllOWRiMTY3NjM1ZTc0NGE1NTgxNDcwMmYxNTIwMDRlOTE1CmJvZHkJaW50ZXJuYWwvbWNwL3dvcmRzMTMyX3Rlc3QuZ28JVGVzdEFTdG9wcGVkQ2hlY2tJc0hlYWRlZEJ5V2hhdFN0b3BwZWRJdE92ZXJNQ1AJNmRjOWNiNjI0NDI2ZWZlNzk4ZWM1N2Q2ODk4ZTMzZjQxMzRmOWUyNGM0YmM0NTc2ZDA4ODc1YzUxZjgxYjAwNgpib2R5CWludGVybmFsL21jcC93b3JkczEzMl90ZXN0LmdvCVRlc3RBbk1DUFJlZnVzYWxOYW1lc01yd1JlYWQJYjMwMTNlNjcwODNkNDQzMWQ5N2I5NTgxM2YwNWQ2ODliMDk1Y2RhNWYzMDkxZjUzZWI3NjgwNmE2M2E3MGQ4Zg · test-lock-kind:replace
