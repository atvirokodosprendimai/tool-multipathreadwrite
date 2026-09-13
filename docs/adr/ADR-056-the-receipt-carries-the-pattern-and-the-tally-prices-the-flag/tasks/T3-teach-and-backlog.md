# Task ADR-056-T3: teach both; correct the pre-registration's data source

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** README, AGENTS.md, BACKLOG correction
**Consumes:** T1, T2
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `receipt has pattern`, `stats prices the flag`, `backlog reads stats`

## Goal

Name `pattern` on the receipts and the pricing block in README and AGENTS.md; in BACKLOG, replace the impossible "replay" sentence with the `mrw stats --json` source, criterion unchanged.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | Safety bullet gains the two sentences. |
| `AGENTS.md` | edit | Rules bullet and the `mrw stats` entry. |
| `docs/adr/BACKLOG.md` | edit | Pre-registration data source; ADR-055 BACKLOG row for the MCP pattern closed with a receipt. |
| `cmd/mrw/main.go` | edit | `write --help` names `pattern` in the JSON receipt. |
| `cmd/mrw/writehelp_test.go` | edit | Red: help names it. |

## Ordered Steps

1. [S1] `TestWriteHelpNamesThePatternField` — RED. [proof: mutation]
2. [S2] Teach in the four files. S1 GREEN. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v -run 'TestWriteHelpNamesThePatternField' 2>&1 | tee /tmp/adr056-t3.out \
  && grep -q '^--- PASS: TestWriteHelpNamesThePatternField' /tmp/adr056-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr056-t3.out \
  && grep -q 'mrw stats --json' docs/adr/BACKLOG.md \
  && grep -q 'pricing' AGENTS.md README.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWriteHelpNamesThePatternField` | `cmd/mrw/writehelp_test.go` | `--help` says the JSON receipt carries `pattern` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `write --help` |
| 3 — the caller can discover it | README, AGENTS.md |
| 4 — it is used | the BACKLOG campaign reads the block |

## Mutation Log
_(tool-written)_
- 2026-09-13 · a4020e7* · mutant killed · exit 1 · `cmd/mrw/main.go` · the help misspells the pricing block: a PATH caller cannot find it from the help, and TestWriteHelpNamesThePatternField must go red · acceptance-sha256:75ca63e300f4c59c84ab87849f07f2aaea97150895e0d2af58287229c1505459

## Invariants

- The pre-registration's CRITERION is not edited; only its data source.

## Risks

| Risk | Mitigation |
|------|------------|
| Teaching drifts from the binary | the help test |

## Stop Condition

None.

## Out of Scope

- The mrw skill update happens at release, outside the record.

## Verification Log
_(tool-written)_
- 2026-09-13 · a4020e7* · exit 1 · `set -o pipefail …` · acceptance-sha256:75ca63e300f4c59c84ab87849f07f2aaea97150895e0d2af58287229c1505459 · ms:597 · test-lock-sha256:1d99180c639c2f19c7358289a3fafb7b239bf18f3d9dfbd96d4ab95e1a9bbf6f · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvd3JpdGVoZWxwX3Rlc3QuZ28JVGVzdFdyaXRlSGVscE5hbWVzQWR2aXNvcmllc0FuZFN0cmljdEJhbGFuY2UJZjJjY2QzZTA5YjMxNGI2MjA1MGI0MGNlMGM3NzdiMTQ2MWFjYzQ3MmZjOTA1ZjFlZDJkOWVmMjExYzhiNmI0Mgpib2R5CWNtZC9tcncvd3JpdGVoZWxwX3Rlc3QuZ28JVGVzdFdyaXRlSGVscE5hbWVzQXBwbHlQYXRjaEZvcm1hdAljMzg2NTk3NjUyNjYxNWFjNDliYjk3MzkyODZjY2YyN2JhNzEzNmUyM2RiMjc1MGMxOTZmNDkzYzYzYWJjNzVlCmJvZHkJY21kL21ydy93cml0ZWhlbHBfdGVzdC5nbwlUZXN0V3JpdGVIZWxwTmFtZXNFY2hvUGFkCWI0OGMyMDkyMmNiMjJkMmUwYjY5NDg2OGNhNzM0YjIyODdhNzVhZDk5ZWRkMTVkOTZjMzIzNDBmYjRlYWYxODAKYm9keQljbWQvbXJ3L3dyaXRlaGVscF90ZXN0LmdvCVRlc3RXcml0ZUhlbHBOYW1lc0hvd1RvUXVvdGVBSGVhZGVyT3B0aW9uCTU3Njk4NjUwZDQ5NGJmNjgzNGIzMzgwYzQxYzg4MjliZTI3MDM4NWZjYTdlMzAzZjA4ZDcwNmMyZTJmOTZmMmUKYm9keQljbWQvbXJ3L3dyaXRlaGVscF90ZXN0LmdvCVRlc3RXcml0ZUhlbHBOYW1lc05vQ2hlY2sJODA5MzA0NzU1NGI3YTJkY2JjNjY4YTljOWQ2OWU3YTI0NDY0MzExYjA5MWQ5N2U3ZmZmN2Q1YTZjOGVhMWM5Ngpib2R5CWNtZC9tcncvd3JpdGVoZWxwX3Rlc3QuZ28JVGVzdFdyaXRlSGVscE5hbWVzVGhlUGF0dGVybkZpZWxkCTIyYjZmYjU0NmM0ZTA0ZDg1Yzc0MzI4Y2IwZjQyZTA5MjQ4NjdlM2NlZDFmZjg3NzY4YzBkMTdjMjBlODM2MWM
  ```
  --- last 10 line(s) of stdout (of 125 after folding 125 raw)
          "advisories". When three of your last ten landed writes carried one, the
          receipt prints a pattern line — read past the range before the next one.
          --strict-balance is opt-in: it refuses a single-line replace whose line's
          {} () [] do not balance and whose body does not match them (the wrap-tail
          shape) as a failed hunk — exit 1, nothing is written. Off by default; braces
          inside strings count, so drop it for that plan.
  --- FAIL: TestWriteHelpNamesThePatternField (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.201s
  FAIL
  ```
- 2026-09-13 · a4020e7* · exit 1 · `set -o pipefail …` · acceptance-sha256:75ca63e300f4c59c84ab87849f07f2aaea97150895e0d2af58287229c1505459 · ms:635
  ```
  --- last 10 line(s) of stdout (of 68 after folding 68 raw)
          {} () [] do not balance and whose body does not match them (the wrap-tail
          shape) as a failed hunk — exit 1, nothing is written. Off by default; braces
          inside strings count, so drop it for that plan. Both JSON receipts (--json
          and mrw_write) carry "pattern" {advisory_writes, window, fires} on every
          write, and mrw stats prices the flag as writes land — how many the flag would
          have refused, and whether those writes then broke, held or went unchecked.
  --- FAIL: TestWriteHelpNamesThePatternField (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.207s
  FAIL
  ```
- 2026-09-13 · a4020e7* · exit 0 · `set -o pipefail …` · acceptance-sha256:75ca63e300f4c59c84ab87849f07f2aaea97150895e0d2af58287229c1505459 · ms:446
