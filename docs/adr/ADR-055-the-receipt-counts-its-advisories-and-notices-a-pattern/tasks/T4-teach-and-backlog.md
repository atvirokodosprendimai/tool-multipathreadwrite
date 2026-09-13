# Task ADR-055-T4: Teach the three; BACKLOG pre-registration

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** none
**Consumes:** `apply.Result.Advisories` (T1), `authoring.Recent` + pattern line (T2), `Options.StrictBalance` / `--strict-balance` (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `help names the three`, `README one line`, `BACKLOG pre-registers the default`, `AGENTS.md and skill moved`

## Goal

`write --help` names the advisory count, the pattern line and `--strict-balance` (opt-in; what it refuses; that it is off). README Safety gets one line. AGENTS.md's "rules that will bite you" bullet on the balance row gains the flag; the centralised `mrw` skill's provenance pin moves with the release that ships this. BACKLOG: inventory rows flip to ADR-055; the From-ADR-054 section gets its receipt; a pre-registration for a default `--strict-balance` states the criterion BEFORE the campaign exists — false-positive rate of the signature on plans that did not break the tree, over Zeus, this repository and playtrix.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `writeCmd` Description. |
| `cmd/mrw/writehelp_test.go` | edit | `TestWriteHelpNamesAdvisoriesAndStrictBalance`. |
| `README.md` | edit | One Safety line. AckRule untouched. |
| `AGENTS.md` | edit | The balance bullet gains `--strict-balance`; the summary-line clause named. |
| `docs/adr/BACKLOG.md` | edit | Rows to ADR-055; receipt; pre-registration. |

## Ordered Steps

1. [S1] Write `TestWriteHelpNamesAdvisoriesAndStrictBalance` and confirm it is RED. [proof: mutation]
2. [S2] Teach on `write --help`, README, AGENTS.md. Confirm GREEN. Deleting `--strict-balance` from the Description must fail S1. [proof: mutation]
3. [S3] BACKLOG rows, receipt and the pre-registration. `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v -run 'TestWriteHelpNamesAdvisoriesAndStrictBalance|TestWriteHelpNamesNoCheck' 2>&1 | tee /tmp/adr055-t4.out \
  && grep -q '^--- PASS: TestWriteHelpNamesAdvisoriesAndStrictBalance' /tmp/adr055-t4.out \
  && grep -q '^--- PASS: TestWriteHelpNamesNoCheck' /tmp/adr055-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr055-t4.out \
  && grep -q 'ADR-055' docs/adr/BACKLOG.md \
  && grep -q -- '--strict-balance' README.md AGENTS.md \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWriteHelpNamesAdvisoriesAndStrictBalance` | `cmd/mrw/writehelp_test.go` | `--help` names the advisory count, the pattern line, and `--strict-balance` as an opt-in refusal | — | S1, S2 |
| `TestWriteHelpNamesNoCheck` | `cmd/mrw/writehelp_test.go` | ADR-054's teaching still holds | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the help test and the BACKLOG greps |
| 2 — something selects it | `writeCmd` Description is `write --help` |
| 3 — the caller can discover it | this task |
| 4 — it is used | PATH callers; the skill mirror |

## Mutation Log
(empty until execute)
- 2026-09-13 · a0ec9cc* · mutant killed · exit 1 · `cmd/mrw/main.go` · the flag is misspelt in write --help: a PATH caller cannot learn --strict-balance from the help, and TestWriteHelpNamesAdvisoriesAndStrictBalance must go red · acceptance-sha256:fe3c24d0059722491ecc54d4fc45ca3c0c16c78abe2777c44f661e3410f26743

## Invariants

- AckRule sentence in README untouched.
- Shared() / 4096 untouched.
- The pre-registration is written before any campaign runs (BACKLOG rule).

## Risks

- README AckRule test goes red if the Safety edit touches the MCP ack bullet. Do not.

## Stop Condition

If teaching requires raising 4096, stop.

## Out of Scope

- Engine (T1–T3)
- The campaign itself and any default

## Verification Log
(empty until execute)
- 2026-09-13 · a0ec9cc* · exit 1 · `set -o pipefail …` · acceptance-sha256:fe3c24d0059722491ecc54d4fc45ca3c0c16c78abe2777c44f661e3410f26743 · ms:381 · test-lock-sha256:b9c058f16788b52cf32330ccdf4df60fa74dae17bed15e57ea4e4ff4c07a842d · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvd3JpdGVoZWxwX3Rlc3QuZ28JVGVzdFdyaXRlSGVscE5hbWVzQWR2aXNvcmllc0FuZFN0cmljdEJhbGFuY2UJZjJjY2QzZTA5YjMxNGI2MjA1MGI0MGNlMGM3NzdiMTQ2MWFjYzQ3MmZjOTA1ZjFlZDJkOWVmMjExYzhiNmI0Mgpib2R5CWNtZC9tcncvd3JpdGVoZWxwX3Rlc3QuZ28JVGVzdFdyaXRlSGVscE5hbWVzQXBwbHlQYXRjaEZvcm1hdAljMzg2NTk3NjUyNjYxNWFjNDliYjk3MzkyODZjY2YyN2JhNzEzNmUyM2RiMjc1MGMxOTZmNDkzYzYzYWJjNzVlCmJvZHkJY21kL21ydy93cml0ZWhlbHBfdGVzdC5nbwlUZXN0V3JpdGVIZWxwTmFtZXNFY2hvUGFkCWI0OGMyMDkyMmNiMjJkMmUwYjY5NDg2OGNhNzM0YjIyODdhNzVhZDk5ZWRkMTVkOTZjMzIzNDBmYjRlYWYxODAKYm9keQljbWQvbXJ3L3dyaXRlaGVscF90ZXN0LmdvCVRlc3RXcml0ZUhlbHBOYW1lc0hvd1RvUXVvdGVBSGVhZGVyT3B0aW9uCTU3Njk4NjUwZDQ5NGJmNjgzNGIzMzgwYzQxYzg4MjliZTI3MDM4NWZjYTdlMzAzZjA4ZDcwNmMyZTJmOTZmMmUKYm9keQljbWQvbXJ3L3dyaXRlaGVscF90ZXN0LmdvCVRlc3RXcml0ZUhlbHBOYW1lc05vQ2hlY2sJODA5MzA0NzU1NGI3YTJkY2JjNjY4YTljOWQ2OWU3YTI0NDY0MzExYjA5MWQ5N2U3ZmZmN2Q1YTZjOGVhMWM5Ng
  ```
  --- last 10 line(s) of stdout (of 262 after folding 262 raw)
          
          A non-prose hunk whose {} () [] nets differ between the replaced lines and
          the body prints a balance row under ok. It does not fail the hunk and it is
          not a checker: braces in strings miscount, and a balanced insert landing
          inside a function is invisible to it — the check catches that, the row does
          not.
  --- FAIL: TestWriteHelpNamesAdvisoriesAndStrictBalance (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.166s
  FAIL
  ```
- 2026-09-13 · a0ec9cc* · exit 0 · `set -o pipefail …` · acceptance-sha256:fe3c24d0059722491ecc54d4fc45ca3c0c16c78abe2777c44f661e3410f26743 · ms:2220
