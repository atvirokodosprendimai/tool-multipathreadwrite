# Task ADR-160-T6: the pilot's fixes — B's interpreter rule, check-coupled graded by its tests, and the mine floor as a flag

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** B's hook judging an interpreter on its own command, `task.json` `test_package` and the oracle that runs that package's tests, `bench mine --min-occurrences --min-files`
**Consumes:** the shapes `wide-rename` and `check-coupled` (T1), the corpus (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an interpreter is judged on its own segment`, `a check-coupled attempt is graded by its package's tests`, `the floor is a flag`

## Goal

What T3's pilot found wrong in the bench is fixed before the pilot runs again: B's hook stops refusing an interpreter for text in another command's heredoc or for a comparison, `check-coupled` is graded by its package's tests (Zy, "Grade by the tests"), and the raised `wide-rename` floor (Decision 2) is drawn by `bench mine` itself.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/hook.go` | edit | `classify` judges an interpreter on its own segment, and on the heredoc bodies only when that segment reads one; the redirect alternative of `writeCall` needs a quote before `>` |
| `internal/bench/hook_test.go` | edit | the pilot's and Stage 3's false refusals pass; an interpreter that writes, inline or through its heredoc, is still refused |
| `internal/bench/task.go` | edit | `Task.TestPackage` (`test_package`): only on `check-coupled`, a relative directory, never with `exact` |
| `internal/bench/oracle.go` | edit | with `TestPackage`, a change outside that directory is outside, the regions are not consulted, and `go test` of the package must pass, bounded and interruptible |
| `internal/bench/oracle_test.go` | edit | the grading by tests; the corpus test's stray edit goes outside the package for such a task |
| `internal/bench/helpers_test.go` | edit | a `test_package` task is materialized whole |
| `cmd/bench/main.go` | edit | `bench mine --min-occurrences N --min-files N` (defaults 15 and 8), and the usage text |
| `cmd/bench/main_test.go` | add | the floor flags |
| `bench/tasks/am-coupled-*/task.json` | edit | the twelve take `test_package`, drop `exact`, and keep `require` and `forbid` on the source file only |
| `docs/bench/README.md` | edit | `test_package`, the flags, and the false refusals found beside Stage 3 |

## Ordered Steps

1. [S1] Write the three failing tests and see them red. [proof: acceptance]
2. [S2] The hook: an interpreter (`python`, `node`, `perl`, `ruby`) is refused when its own segment, or a heredoc that segment reads, matches `writeCall`; a heredoc another command reads, such as an `mrw write` plan, is not its text. The redirect alternative becomes `['"]>>?\s*[A-Za-z]`, a mode string such as perl's `">out"`, so a comparison (`a > b`) and an annotation (`-> str`) do not match. An interpreter's write is still refused wherever it goes, since the hook cannot resolve the target, as it refuses `> $TMPDIR/…`; a shell redirect run through `os.system` is no longer seen here, and the ledger's slipped-write count is the backstop for it. [proof: mutation]
3. [S3] `test_package`: the loader accepts it only on a `check-coupled` task, as a clean relative directory, and refuses it beside `exact`. With it, `Oracle` judges a change by its directory instead of the regions, skips `exact`, keeps `require`, `forbid` and the structural checks, and when nothing else failed runs `go test -count=1 ./<dir>` in the tree under a 10-minute bound; an interrupt stops it and grading reports `errGradingInterrupted`, as the SWE-bench grader does; a failure is a `missing` entry naming the package. The corpus test materializes such a task whole and puts its stray edit outside the package. [proof: mutation]
4. [S4] `bench mine` takes `--min-occurrences` and `--min-files` for `wide-rename`, defaults 15 and 8, so the T2 draw is unchanged and the raised floor (25 and 12) is drawn by the same code; the usage text names them. [proof: mutation]
5. [S5] Rebuild the twelve `check-coupled` tasks' `task.json`: `test_package`, no `exact`, `require` and `forbid` on the source file; every oracle subtest passes. Write beside Stage 3 in `docs/bench/README.md` the false refusals the hook made there: six B attempts, eight refusals, every one solved. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ ./cmd/bench/ -count=1 -v -run '^(TestAnInterpreterIsJudgedOnItsOwnSegmentAndTheHeredocItReads|TestACheckCoupledTaskIsGradedByItsPackageTests|TestTheMineFloorIsAFlag)$' 2>&1) \
  && for n in TestAnInterpreterIsJudgedOnItsOwnSegmentAndTheHeredocItReads TestACheckCoupledTaskIsGradedByItsPackageTests TestTheMineFloorIsAFlag; do grep -qE "^--- PASS: $n " <<<"$out" || exit 1; done \
  && test "$(grep -l '"test_package"' bench/tasks/am-coupled-*/task.json | wc -l | tr -d ' ')" = 12 \
  && ! grep -l '"exact": true' bench/tasks/am-coupled-*/task.json \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q -- '--min-occurrences'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnInterpreterIsJudgedOnItsOwnSegmentAndTheHeredocItReads` | `internal/bench/hook_test.go` | an `mrw write` plan whose body holds `-> str` beside a `python -c` check passes; a heredoc fed to python that compares `a > b` passes; `python3 - <<'EOF'` with `open('f','w')` and `python -c` writing a file are refused | none | S2 |
| `TestACheckCoupledTaskIsGradedByItsPackageTests` | `internal/bench/oracle_test.go` | a fixture module: the source and its test changed is solved; the source alone fails its test and is unsolved; the code made to accept both values with the tests untouched is solved; a change outside the package is outside; `test_package` beside `exact`, or on another shape, is refused by the loader | none | S3 |
| `TestTheMineFloorIsAFlag` | `cmd/bench/main_test.go` | a fixture repository whose identifier has 3 occurrences in 2 files is listed with `--min-occurrences 3 --min-files 2` and not with the defaults | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the hook rule, `Task.TestPackage`, the oracle branch, the flags |
| 2 — something selects it | the runner calls `Oracle` and B's hook on every attempt; `test_package` in the twelve tasks; `bench mine` parses the flags |
| 3 — the caller can discover it | `docs/bench/README.md`, the usage text |
| 4 — it is used | T3's calibration-2 and pilot rerun, then T4 |

## Mutation Log
(empty until execute)
- 2026-10-15 · 051219b1* · mutant killed · exit 1 · `internal/bench/hook.go` · S2: another command's heredoc body is judged as the interpreter's text again · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · covers:an interpreter is judged on its own segment
- 2026-10-15 · 051219b1* · mutant killed · exit 1 · `internal/bench/hook.go` · S2: a comparison or annotation reads as a redirect again · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204
- 2026-10-15 · 051219b1* · mutant killed · exit 1 · `internal/bench/oracle.go` · S3: the package tests never run, so a source-only change that breaks them is solved · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · covers:a check-coupled attempt is graded by its package's tests
- 2026-10-15 · 051219b1* · mutant killed · exit 1 · `cmd/bench/main.go` · S4: the flags parse and the floor stays 15 and 8 · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · covers:the floor is a flag

## Invariants

- The T2 draw is reproduced byte for byte by `bench mine` with no flags.
- No receipt already published is regraded; the six Stage 3 attempts are described beside the stage, not changed.

## Risks

- `go test` in the oracle makes grading slower and depends on the module cache; the bound keeps a hung test from holding the stage.

## Stop Condition

A check-coupled reference fails its own package's tests under the new oracle; or a fixed hook refuses a command the pilot showed to be a read.

## Out of Scope

- Resolving where an interpreter writes (permanent: boundary: the hook reads text, and the slipped-write count catches a write it missed)
- Regrading Stage 3 (permanent: boundary: its six affected attempts were all solved; the note says so beside the stage)

## Verification Log
(empty until execute)
- 2026-10-15 · 051219b1* · exit 1 · `set -o pipefail …` · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · ms:421 · test-lock-sha256:84c4d6abe74a58253d231e53cc9dd156af5bfc7addddb3d53997cac5276bfcd5 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL2JlbmNoL21haW5fdGVzdC5nbwlUZXN0VGhlTWluZUZsb29ySXNBRmxhZwk3OTNmNmZmNDgzMjkwZGQyNGI5ZjQyOTJhODdhZmFhN2I5ZTY0YTRkODU4M2IzNjQ0MzM3NGMyMTEzMTE1YTRhCmJvZHkJaW50ZXJuYWwvYmVuY2gvaG9va190ZXN0LmdvCVRlc3RBU2xpcHBlZFdyaXRlSXNGb3VuZEZyb21UaGVMZWRnZXIJYjY2ODY1MWEwYWE2Nzc1MDM3MDA5NTVhNDc1ZmQ4NmEzOGE4ZTNlMzY5MDQzYWM3NWQ0ZDQ0NjMyMDYzNjMwZQpib2R5CWludGVybmFsL2JlbmNoL2hvb2tfdGVzdC5nbwlUZXN0QVdyaXRlT3V0c2lkZVRoZVRhc2tUcmVlUGFzc2VzVGhlSG9vawljNDk4ZTY2NzAyY2Q5MjA4YWQ4MWZlMjg1MjUzNjU2ZjFhOTk0NTE4ZmFlMWNiY2JhOWMzNzg0MGY3MDkxYTA0CmJvZHkJaW50ZXJuYWwvYmVuY2gvaG9va190ZXN0LmdvCVRlc3RBbkludGVycHJldGVySXNKdWRnZWRPbkl0c093blNlZ21lbnRBbmRUaGVIZXJlZG9jSXRSZWFkcwk4YjdjMjNiYzJmYjFmY2QzMzkzY2RlZWUwMGUyZDZhMWFjMzM5NTQ4M2FiZmRjY2ZhZjU4MWYwNjlmYmM2ZWViCmJvZHkJaW50ZXJuYWwvYmVuY2gvaG9va190ZXN0LmdvCVRlc3RUaGVBcm1CSG9va1JlZnVzZXNBV3JpdGVCeUFueU1lYW5zQnV0TXJ3QW5kQ291bnRzUmVhZHMJNDA0MGQ2ZDFhMmI0MWQyMmVkZGI3YzQzNWQ3YTEwNzBiOWVjMTA1MzVhNWUzYzJjNDZhNGUwMGE1OTkxMGQyZQpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RBQ2hlY2tDb3VwbGVkVGFza0lzR3JhZGVkQnlJdHNQYWNrYWdlVGVzdHMJMjkwYTQ1MjI3ZjY4MzZlMjVhNDA4NGVmZTE2YjE2NWIzNmRkMDc0YWEwYzE2OTRjNGFjOWYwMGVkZjRlMDRiYgpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RBRGVjbGFyZWRTZWN0aW9uSXNEcm9wcGVkQW5kQU1pc3NpbmdPbmVSZWZ1c2VzCWVmZmI5MGYwMmVmYzBkOTMwYzI5MGY4NzcxNjNjNjk5YzFiZDAyYzExZmQyZGUyZWNlNjE1YjhjY2EzZTZhYWYKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0QWdlbnRzbWVtb3J5c1RyZWVMb3Nlc0JvdGhEZWNsYXJlZFNlY3Rpb25zCTA1ZjMwNDRiYzIwNjNkNmYxYjI2NTVlZDQ4OTBiOGMxZTI3YjI3M2E1NDI0OWQ2NDUyMDUyNTQwMmE3YTgwZGIKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0RXZlcnlPcmFjbGVGYWlsc09uQmFzZVBhc3Nlc09uRml4QW5kUmVmdXNlc0FTdHJheUVkaXQJMDkwNDgyYzQwMzcwN2NiMmVjNDIxY2I5MGZmNDkwNGVmMWM0M2I5OTU3ZWJlZjkyN2RmNGY3YTc4NzFiMDNjNApib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RFdmVyeVZhcmllZE11bHRpU2l0ZVRhc2tIYXNBUmVwZWF0ZWRBbmNob3JBY3Jvc3NUd29GaWxlcwlkMjM2NWJiMDc0YTJjYzZkMmY2MzIzMTU5Zjc1YWY2MmRhZjFmN2RiOWE1Nzc0MTQ1ZjBlMmY1NWRmNzZhMzQxCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZUNvcnB1c0hhc1RoZVJlZ2lzdGVyZWRTaGFwZUNvdW50cwk1NTU0YWY0NDAxNzI5MmY1YWZhODIwZWNhYjY0MzVkYTY4MGY5NGI5ZGRiYTQ5YTE2NjU3MTkwMWVlNTI0YjMwCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZU9yYWNsZVJlZnVzZXNXaGF0VGhlUmV2aWV3Rm91bmQJNDk5ZDFkYjZmNDc5ZDE2NjM5NThlY2FmNzEyMWMyMDA5NzZlOThlNmU2MWI0OTBhMmY2YmY4ZjUwZDM2MGNkYQpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RUaGVQb3N0U2VuZFJlbmFtZUxlYXZlc1RoZVRlc3RIZWxwZXIJMjFlZmQ0NGJjODIxNWQ0ZTRkY2ZiMTUxMGM1N2U0YjFmZjI4NDE4OTBiZDExMjdmYTRkMmJkYzBmOWFhNWMwOQpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RUaGVQcmVwYXJlZFRyZWVSZWRhY3RzSW5zdHJ1Y3Rpb25zQW5kU3RyaXBzQ29uZmlndXJhdGlvbgliMDYwZDE2ZWJkYzU1ZmZlZWE5MGYxZjA0YzQyOWY2NGJkNTgxZmM1NjEyNmZjNDZmZWQ0ZmUxYzNhMmRmZGZmCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZVZhcmllZE9yYWNsZXNHcmFkZVdoYXRUaGVSZXZpZXdPZjQxN0ZvdW5kCWQxMmU4YTJjNGVjNjMyM2E1NTc0NzQ0MWVmMzhhZmQ1MGI0N2ZhNjVmNWFlOThjY2QzYmE3NjNkYzExMDIzNTE
  ```
  ```
- 2026-10-15 · 051219b1* · exit 0 · `set -o pipefail …` · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · ms:2865
- 2026-10-15 · 051219b1* · exit 0 · `set -o pipefail …` · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · ms:2766
- 2026-10-15 · 051219b1* · exit 0 · `set -o pipefail …` · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · ms:2834
- 2026-10-15 · 051219b1* · exit 0 · `set -o pipefail …` · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · ms:2826
- 2026-10-15 · 051219b1* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b10fa7ba9dd7530262061b258321a9b500f899b7007bac71b49db895abeb6204 · ms:0 · test-lock-sha256:a7c81406f6d345199bb8509d2eed8f3086211dc87ad1be9fd7f6268a0345cc61 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL2JlbmNoL21haW5fdGVzdC5nbwlUZXN0VGhlTWluZUZsb29ySXNBRmxhZwk3OTNmNmZmNDgzMjkwZGQyNGI5ZjQyOTJhODdhZmFhN2I5ZTY0YTRkODU4M2IzNjQ0MzM3NGMyMTEzMTE1YTRhCmJvZHkJaW50ZXJuYWwvYmVuY2gvaG9va190ZXN0LmdvCVRlc3RBU2xpcHBlZFdyaXRlSXNGb3VuZEZyb21UaGVMZWRnZXIJYjY2ODY1MWEwYWE2Nzc1MDM3MDA5NTVhNDc1ZmQ4NmEzOGE4ZTNlMzY5MDQzYWM3NWQ0ZDQ0NjMyMDYzNjMwZQpib2R5CWludGVybmFsL2JlbmNoL2hvb2tfdGVzdC5nbwlUZXN0QVdyaXRlT3V0c2lkZVRoZVRhc2tUcmVlUGFzc2VzVGhlSG9vawljNDk4ZTY2NzAyY2Q5MjA4YWQ4MWZlMjg1MjUzNjU2ZjFhOTk0NTE4ZmFlMWNiY2JhOWMzNzg0MGY3MDkxYTA0CmJvZHkJaW50ZXJuYWwvYmVuY2gvaG9va190ZXN0LmdvCVRlc3RBbkludGVycHJldGVySXNKdWRnZWRPbkl0c093blNlZ21lbnRBbmRUaGVIZXJlZG9jSXRSZWFkcwkwN2MzNGJjM2NlNTgyNmEzNDM0ZTQzNDYzYTQ1M2NmYjVkOWFkMTBmNDQxZGVkYjE3NGNlYWIzMWU1ZGU4Zjc4CmJvZHkJaW50ZXJuYWwvYmVuY2gvaG9va190ZXN0LmdvCVRlc3RUaGVBcm1CSG9va1JlZnVzZXNBV3JpdGVCeUFueU1lYW5zQnV0TXJ3QW5kQ291bnRzUmVhZHMJNDA0MGQ2ZDFhMmI0MWQyMmVkZGI3YzQzNWQ3YTEwNzBiOWVjMTA1MzVhNWUzYzJjNDZhNGUwMGE1OTkxMGQyZQpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RBQ2hlY2tDb3VwbGVkVGFza0lzR3JhZGVkQnlJdHNQYWNrYWdlVGVzdHMJMjkwYTQ1MjI3ZjY4MzZlMjVhNDA4NGVmZTE2YjE2NWIzNmRkMDc0YWEwYzE2OTRjNGFjOWYwMGVkZjRlMDRiYgpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RBRGVjbGFyZWRTZWN0aW9uSXNEcm9wcGVkQW5kQU1pc3NpbmdPbmVSZWZ1c2VzCWVmZmI5MGYwMmVmYzBkOTMwYzI5MGY4NzcxNjNjNjk5YzFiZDAyYzExZmQyZGUyZWNlNjE1YjhjY2EzZTZhYWYKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0QWdlbnRzbWVtb3J5c1RyZWVMb3Nlc0JvdGhEZWNsYXJlZFNlY3Rpb25zCTA1ZjMwNDRiYzIwNjNkNmYxYjI2NTVlZDQ4OTBiOGMxZTI3YjI3M2E1NDI0OWQ2NDUyMDUyNTQwMmE3YTgwZGIKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0RXZlcnlPcmFjbGVGYWlsc09uQmFzZVBhc3Nlc09uRml4QW5kUmVmdXNlc0FTdHJheUVkaXQJNjAwNDViNjBjMjRlZmI1OWMzNjUyZDI3N2JmODVkMzVjOTBlODg2NDA2MWRhZGY1Nzg3NGQ5ZTRjYTdhNjAzOQpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RFdmVyeVZhcmllZE11bHRpU2l0ZVRhc2tIYXNBUmVwZWF0ZWRBbmNob3JBY3Jvc3NUd29GaWxlcwlkMjM2NWJiMDc0YTJjYzZkMmY2MzIzMTU5Zjc1YWY2MmRhZjFmN2RiOWE1Nzc0MTQ1ZjBlMmY1NWRmNzZhMzQxCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZUNvcnB1c0hhc1RoZVJlZ2lzdGVyZWRTaGFwZUNvdW50cwk1NTU0YWY0NDAxNzI5MmY1YWZhODIwZWNhYjY0MzVkYTY4MGY5NGI5ZGRiYTQ5YTE2NjU3MTkwMWVlNTI0YjMwCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZU9yYWNsZVJlZnVzZXNXaGF0VGhlUmV2aWV3Rm91bmQJNDk5ZDFkYjZmNDc5ZDE2NjM5NThlY2FmNzEyMWMyMDA5NzZlOThlNmU2MWI0OTBhMmY2YmY4ZjUwZDM2MGNkYQpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RUaGVQb3N0U2VuZFJlbmFtZUxlYXZlc1RoZVRlc3RIZWxwZXIJMjFlZmQ0NGJjODIxNWQ0ZTRkY2ZiMTUxMGM1N2U0YjFmZjI4NDE4OTBiZDExMjdmYTRkMmJkYzBmOWFhNWMwOQpib2R5CWludGVybmFsL2JlbmNoL29yYWNsZV90ZXN0LmdvCVRlc3RUaGVQcmVwYXJlZFRyZWVSZWRhY3RzSW5zdHJ1Y3Rpb25zQW5kU3RyaXBzQ29uZmlndXJhdGlvbgliMDYwZDE2ZWJkYzU1ZmZlZWE5MGYxZjA0YzQyOWY2NGJkNTgxZmM1NjEyNmZjNDZmZWQ0ZmUxYzNhMmRmZGZmCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZVZhcmllZE9yYWNsZXNHcmFkZVdoYXRUaGVSZXZpZXdPZjQxN0ZvdW5kCWQxMmU4YTJjNGVjNjMyM2E1NTc0NzQ0MWVmMzhhZmQ1MGI0N2ZhNjVmNWFlOThjY2QzYmE3NjNkYzExMDIzNTE · test-lock-kind:replace
