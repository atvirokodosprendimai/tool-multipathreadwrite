# Task ADR-133-T1: a CLI read whose stdout is the null device records nothing and says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `toNullDevice` in `cmd/mrw/main.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a CLI read whose stdout is the null device records nothing and says so`

## Goal

Decisions 1–3 of the record, with a test that fails before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the read action skips `seen.Record` and says so |
| `cmd/mrw/nulldevice_other.go`, `cmd/mrw/nulldevice_windows.go` | add | `toNullDevice`; on Windows a console is told apart from NUL |
| `cmd/mrw/nullread133_test.go`, `cmd/mrw/nulldevice133_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §234 |
| `AGENTS.md`, `README.md` | edit | the read-before-write rule names it |

## Ordered Steps

1. [S1] Write `TestAReadSentToTheNullDeviceLicensesNothing`: with stdout opened on `os.DevNull`, a read exits 0, says on stderr that nothing was recorded, and leaves the file out of the ledger; with stdout a regular file, the same read records it. Confirm RED.
2. [S2] `toNullDevice(stdout)` compares stdout's file info with `os.Stat(os.DevNull)`; on Windows, where a character device has no file identity and so compares equal to NUL, a handle `GetConsoleMode` accepts answers false first (the reviews of #350). The read action skips `seen.Record` and writes the stderr line, when it served anything, if it answers true. Mutant: the check answering false (the null-device read records). The Windows console branch has no CI fixture, since no runner puts a console on stdout. [proof: mutation]
3. [S3] Contract §234 — a read to `/dev/null` then a write to that line exits 1, naming it unread; the pair, a read to a file, then the same write exits 0. AGENTS.md and README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAReadSentToTheNullDeviceLicensesNothing|TestAReadWhoseAnswerCannotBeWrittenRecordsNothing' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAReadSentToTheNullDeviceLicensesNothing \(' "$out" \
  && grep -qE '^--- PASS: TestAReadWhoseAnswerCannotBeWrittenRecordsNothing \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 234\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadSentToTheNullDeviceLicensesNothing` | `cmd/mrw/nullread133_test.go` | a read to the null device records nothing and says so; a read to a file records | none | S1, S2 |
| `TestOnlyTheNullDeviceCountsAsNull` | `cmd/mrw/nulldevice133_test.go` | the null device counts; a pipe and a regular file do not | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `toNullDevice` and the skipped record |
| 2 — something selects it | the read action, on every CLI read |
| 3 — the caller can discover it | the stderr line; the write refusal that follows; AGENTS.md; README |
| 4 — it is used | the 2026-10-08 quality-harness report; no telemetry (ADR-009) |

## Invariants

- A read to anything but the null device records exactly as before.
- Stdout and the exit code of a read are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if `os.DevNull` does not stat as the same file as a handle opened on it on a CI platform.

## Out of Scope

- Pipes, files and terminals (permanent: boundary: the record's Decision 3)

## Mutation Log
- 2026-10-08 · 18b1a4a* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the null-device check answers false — a read to /dev/null records its lines · acceptance-sha256:504b54dc77537645a98467c0a383ddc6b8170d77e4900b385cc776fec2f75b04
- 2026-10-08 · 1cefff1* · mutant killed · exit 1 · `cmd/mrw/nulldevice_other.go` · S2: the null-device check answers false — a read to /dev/null records its lines (after the platform split, #350) · acceptance-sha256:504b54dc77537645a98467c0a383ddc6b8170d77e4900b385cc776fec2f75b04

## Verification Log
- 2026-10-08 · 18b1a4a* · exit 1 · `set -o pipefail …` · acceptance-sha256:504b54dc77537645a98467c0a383ddc6b8170d77e4900b385cc776fec2f75b04 · ms:1648 · test-lock-sha256:0baa6174f201edefd40298dbd8ff51ed6c9814b1dac452056790caabcf0aad24 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9udWxscmVhZDEzM190ZXN0LmdvCVRlc3RBUmVhZFNlbnRUb1RoZU51bGxEZXZpY2VMaWNlbnNlc05vdGhpbmcJMDBlNmY1NzM3NTE1YzE3ODFhNmU2N2Q2ZTFhZmRhYzU0ZDE2YzY2NGYwZTMwOWQ3NDBkMWIzNDg1MmEwZDczMw
  ```
  --- last 8 line(s) of stdout
  === RUN   TestAReadSentToTheNullDeviceLicensesNothing
      nullread133_test.go:65: a read whose answer went to the null device licensed its lines
  --- FAIL: TestAReadSentToTheNullDeviceLicensesNothing (0.01s)
  === RUN   TestAReadWhoseAnswerCannotBeWrittenRecordsNothing
  --- PASS: TestAReadWhoseAnswerCannotBeWrittenRecordsNothing (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.313s
  FAIL
  ```
- 2026-10-08 · 18b1a4a* · exit 1 · `set -o pipefail …` · acceptance-sha256:504b54dc77537645a98467c0a383ddc6b8170d77e4900b385cc776fec2f75b04 · ms:41402
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAReadSentToTheNullDeviceLicensesNothing
  --- PASS: TestAReadSentToTheNullDeviceLicensesNothing (0.01s)
  === RUN   TestAReadWhoseAnswerCannotBeWrittenRecordsNothing
  --- PASS: TestAReadWhoseAnswerCannotBeWrittenRecordsNothing (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.237s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	39.402s
  ```
- 2026-10-08 · 18b1a4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:504b54dc77537645a98467c0a383ddc6b8170d77e4900b385cc776fec2f75b04 · ms:40426
- 2026-10-08 · 1cefff1* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:504b54dc77537645a98467c0a383ddc6b8170d77e4900b385cc776fec2f75b04 · ms:0 · test-lock-sha256:2e2e5e8912e0eb03904b5457cd19246632a75ce2ea7d11136c113a04b4295589 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9udWxsZGV2aWNlMTMzX3Rlc3QuZ28JVGVzdE9ubHlUaGVOdWxsRGV2aWNlQ291bnRzQXNOdWxsCWI5Njc4NTZhOTExZjE1OGJhZGE3MTRiMzJlYzljOTJmNzA5NTZjYWZhNzU1MTI3YTVhNGE3ODdjNzBiMjk5MWYKYm9keQljbWQvbXJ3L251bGxyZWFkMTMzX3Rlc3QuZ28JVGVzdEFSZWFkU2VudFRvVGhlTnVsbERldmljZUxpY2Vuc2VzTm90aGluZwkwMGU2ZjU3Mzc1MTVjMTc4MWE2ZTY3ZDZlMWFmZGFjNTRkMTZjNjY0ZjBlMzA5ZDc0MGQxYjM0ODUyYTBkNzMz · test-lock-kind:replace
- 2026-10-08 · 1cefff1* · exit 0 · `set -o pipefail …` · acceptance-sha256:504b54dc77537645a98467c0a383ddc6b8170d77e4900b385cc776fec2f75b04 · ms:38945
