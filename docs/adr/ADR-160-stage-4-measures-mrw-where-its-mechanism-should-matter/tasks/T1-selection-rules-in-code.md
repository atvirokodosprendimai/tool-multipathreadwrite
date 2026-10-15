# Task ADR-160-T1: the selection rules in code

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench mine`, `bench swebench-select --salt --min-hunks --exclude --dataset`, `bench analyze --selection`, the shapes `wide-rename` and `check-coupled`, Stage 4's descriptive rows
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the rename predicate`, `the coupled predicate`, `the salted hunk-floor selection`, `the category rows`, `the usage text names bench mine`

## Goal

Every Stage 4 selection rule registered in BACKLOG ("Pre-registered for ADR-160", committed with this record) is code a reader can rerun.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/mine.go` | add | `MineWideRenames`, `MineCheckCoupled`: Decision 1's predicates, standard library only |
| `internal/bench/mine_test.go` | add | the predicates against fixture trees and a fixture history |
| `internal/bench/swebench.go` | edit | `SelectSWEBench` takes a salt; a dataset-reading selection with a hunk floor, an exclusion list and the unnamed-files subset |
| `internal/bench/swebench_test.go` | edit | the salted, floored selection and its subset |
| `internal/bench/task.go` | edit | `ShapeWideRename`, `ShapeCheckCoupled` accepted by the loader |
| `internal/bench/analyze.go` | edit | one descriptive row per Stage 4 category; the slice's subset row from `--selection` |
| `internal/bench/analyze_test.go` | edit | the rows |
| `cmd/bench/main.go` | edit | `bench mine`, the new `swebench-select` and `analyze` flags, the usage text (what selects them) |
| `docs/bench/README.md` | edit | the shapes and the commands |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. [proof: acceptance]
2. [S2] `MineWideRenames`: parse every `.go` file of the module (`go/parser`) and count declarations per identifier; skip generated files; scan tokens (`go/scanner`) per file; a candidate is declared once, appears in no string literal and no generated file, has every whole-word occurrence outside comments as an identifier token, and meets the floor counting comments. Ordered by `sha256("adr-160:" + identifier)`. [proof: mutation]
3. [S3] `MineCheckCoupled`: walk `git log -p --no-merges <commit>`; keep a commit of at most 4 files where every touched file's change is one string literal's text replaced by another (the same pair everywhere), with a non-test `.go` file and a `_test.go` file of the same package among them. Ordered by `sha256("adr-160:" + sha)`. [proof: mutation]
4. [S4] `SelectSWEBench` gains a salt (default `adr-156:`, so Stage 3's selection is unchanged); a dataset-reading selection keeps instances with at least N `^@@ -` hunk headers in `patch`, minus an exclusion list, and writes the subset whose problem statement contains none of the patched paths and none of their basenames but `__init__.py`. [proof: mutation]
5. [S5] `bench analyze` prints one descriptive row per Stage 4 category, and with `--selection` the slice's subset row. [proof: mutation]
6. [S6] Wire `bench mine` and the flags in `cmd/bench/main.go` and its usage text; `docs/bench/README.md` names them. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestAWideRenameCandidateIsDeclaredOnceAndNeverInAString|TestACheckCoupledCandidateIsALiteralSubstitutionWithItsTest|TestTheStage4SWESliceIsTheLowestSaltedHashesWithFiveHunks|TestAnalyzePrintsAStage4CategoryRow|TestTheSWEBenchSliceIsTheLowestTwentyHashes)$' 2>&1) \
  && for t in TestAWideRenameCandidateIsDeclaredOnceAndNeverInAString TestACheckCoupledCandidateIsALiteralSubstitutionWithItsTest TestTheStage4SWESliceIsTheLowestSaltedHashesWithFiveHunks TestAnalyzePrintsAStage4CategoryRow TestTheSWEBenchSliceIsTheLowestTwentyHashes; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'bench mine'
```

`TestTheSWEBenchSliceIsTheLowestTwentyHashes` exists and passes today: it is the guard that Stage 3's selection does not move, not a clause this task turns green.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWideRenameCandidateIsDeclaredOnceAndNeverInAString` | `internal/bench/mine_test.go` | a fixture module: an identifier declared twice, one in a string, one in a generated file, one used as a selector on a standard-library value, one under the floor are refused; one mentioned in comments that qualifies is listed with its comments counted; salted-hash order | none | S2 |
| `TestACheckCoupledCandidateIsALiteralSubstitutionWithItsTest` | `internal/bench/mine_test.go` | a fixture history: a commit replacing one literal in a source and its test is listed; one that also changes code, one with no test file, one whose test is in another package, one of 5 files, a merge are not | none | S3 |
| `TestTheStage4SWESliceIsTheLowestSaltedHashesWithFiveHunks` | `internal/bench/swebench_test.go` | the `adr-160:` salt, the 5-hunk floor, the exclusion list and the unnamed-files subset | none | S4 |
| `TestTheSWEBenchSliceIsTheLowestTwentyHashes` | `internal/bench/swebench_test.go` | Stage 3's selection is unchanged by the salt's default | none | S4 |
| `TestAnalyzePrintsAStage4CategoryRow` | `internal/bench/analyze_test.go` | one descriptive row per category and the subset row, never in the verdict's place | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `cmd/bench/main.go`'s dispatch; the fence runs `go run ./cmd/bench` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T2 draws the corpus with it |

## Mutation Log
(empty until execute)
- 2026-10-15 · 7b6e351a* · mutant killed · exit 1 · `internal/bench/mine.go` · an identifier declared twice is listed · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · covers:the rename predicate
- 2026-10-15 · 7b6e351a* · mutant killed · exit 1 · `internal/bench/mine.go` · a literal change with no test in its package is listed · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · covers:the coupled predicate
- 2026-10-15 · 7b6e351a* · mutant killed · exit 1 · `internal/bench/swebench.go` · an instance under the hunk floor is selected · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · covers:the salted hunk-floor selection
- 2026-10-15 · 7b6e351a* · mutant killed · exit 1 · `internal/bench/analyze.go` · an earlier stage gets category rows · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · covers:the category rows
- 2026-10-15 · 7b6e351a* · mutant killed · exit 1 · `cmd/bench/main.go` · the usage text does not name bench mine · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · covers:the usage text names bench mine

## Invariants

- `bench swebench-select --n 20` with no new flag prints exactly what it printed for Stage 3.
- `go.mod` keeps exactly one requirement.

## Risks

- `go/scanner` sees no types: a module name equal to a standard-library field (`Header`) passes the token rule. The fixture carries the case and documents it, and T2's build gate is what filters it.

## Stop Condition

The predicates need a dependency outside the standard library.

## Out of Scope

- Building the tasks (T2).

## Verification Log
(empty until execute)
- 2026-10-15 · 7b6e351a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:2011
- 2026-10-15 · 7b6e351a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:1836
- 2026-10-15 · 7b6e351a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:2048
- 2026-10-15 · 7b6e351a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:1992
- 2026-10-15 · 7b6e351a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:1824
- 2026-10-15 · 7b6e351a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:1207
- 2026-10-15 · 7b6e351a* · exit 1 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:232 · test-lock-sha256:dcc2ef2764168fba8363a0c9480624c1af5c8c46475fd73caeb57444b9df96a0 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUGlsb3RJc05ldmVyQ291bnRlZAkxNzExNDE3YjczMzQ5NzMxMzgwMzNjODE4ZWM1NGFjM2Y2M2Q2NWUyYWUyYjFhNzhkOThiYTU1NjFlYjIwYTVhCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUm93V2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlSXNEZXNjcmlwdGl2ZQkwYTc3OGYzZDBjZjE3NWRiMTkyNzEwOWI4YWMwNDRjM2JjOGJhMGJkNDM0OTE4YWY0YjA1ZWUyMTU1MjhlNzY0CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBbmFseXplUHJpbnRzQVN0YWdlNENhdGVnb3J5Um93CWU0MzJiODc4Njg0YjcyZjA3YjVjMzBkOGUzZDI0MWFmM2JhMjZhY2IwNjliZmE3MmJmY2FjZDBkZTEyMjY4ZjIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdE9ubHlUaGVSZWdpc3RlcmVkRXhjbHVzaW9uc0FyZUFkbWl0dGVkCWRiODQ3NDI0Yzc2NzRlNzBhZTU0Nzc3ZTk5MzQ0NjFhZThkNTkwMmJmZDZlMmRjOTBiNGI0MTZiYWM4YmJmNWYKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFN0YWdlT25lc1B1Ymxpc2hlZFZlcmRpY3RJc1VuY2hhbmdlZAk5NjA5YmJlZTM2OTBiYTFkMWIxY2UyMjVjYmZmOTQzNTFmNmYwY2Q0NzNkOTljOTdhNTViNjFkMmU2NTJjYTQ1CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RUaGVBbmFseXNpc1BhaXJzQnlUYXNrQW5kS2VlcHNFdmVyeUF0dGVtcHQJYjA5MDNmZGRmZjVmMzJmOWRjOGE0YWU1NjI5M2ZlMWUwNDI0MDIwZjUxNTU4ZWY5MDI2ZmM0NzcxMGMyNzAxYgpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlUHVibGlzaGVkU3RhZ2VzV2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlCTgwYWE5ZTk2ZjIyMGNiMDJjMzUwMTFiOGUzZGMyYTlmM2EyMDNkYWE3Yjk5OTc2ZmZkNGZiNmY1N2U4MDk2ZmIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFRoZVZhcmllZE11bHRpU2l0ZVJvd0lzRGVzY3JpcHRpdmUJNmVlZjQzMWFhNmU3YzliZGJmYjk4ZmJlNzhjN2EzMTZiN2JjMGU0ZmNjYWFiMWNlNjA1ZWExZTZiN2UyOTU0MQpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlVmVyZGljdEZvbGxvd3NUaGVSZWdpc3RlcmVkQmFyCTUxNjUxNjM0MmUwNmE3YjYzOTVkZjA4NDJhNzQ5N2ZjMTdiNGQ1MDcwZDAxZDIzM2Y1NWZkMzIwYjg1MzM5ZTMKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFR3b1N0YWdlc0FyZU5ldmVyUG9vbGVkCTU4MTBmOTZmYTYyNmNmMzhiODg2NTlmMjljZDVjNTA5ZDg5NmQ2MWNkNmRmODZlYmUwYzhmZmQzZWZiZjFmMDYKYm9keQlpbnRlcm5hbC9iZW5jaC9taW5lX3Rlc3QuZ28JVGVzdEFDaGVja0NvdXBsZWRDYW5kaWRhdGVJc0FMaXRlcmFsU3Vic3RpdHV0aW9uV2l0aEl0c1Rlc3QJODg3NTNlNTIxNDVlZjZlZjQ1Zjk0MzUyZjc3NWI3MTA0YzkzODA2NDg3MzY5YWMwMjQ2ZjliYmI1MjEzODRiMwpib2R5CWludGVybmFsL2JlbmNoL21pbmVfdGVzdC5nbwlUZXN0QVdpZGVSZW5hbWVDYW5kaWRhdGVJc0RlY2xhcmVkT25jZUFuZE5ldmVySW5BU3RyaW5nCWExMzYwNjY5ZWU5MDE2MzZmMjhkODEwMGQxZWRmZDllZmI1NzRjZjZlMGFiOWFkMzZiNTFhMzFmZTc4NWNhODUKYm9keQlpbnRlcm5hbC9iZW5jaC9zd2ViZW5jaF90ZXN0LmdvCVRlc3RBU1dFQmVuY2hTdGFnZUlzR3JhZGVkQnlUaGVIYXJuZXNzT25lSW1hZ2VBdEFUaW1lCTQzYTQ1ZjI2YjY4NmQxZWY4MTIxZDU3MmU2ZTI4YzJhMjFkNDk2MWM4MmM1M2FjNjkxNTQ1MmFiNjRkMTA4M2MKYm9keQlpbnRlcm5hbC9iZW5jaC9zd2ViZW5jaF90ZXN0LmdvCVRlc3RBVGltZWRPdXRIYXJuZXNzUmVtb3Zlc0l0c0NvbnRhaW5lcgk4OTFiMTBkZjY0YmMxMTQ3MzlmMGViOTg1MTRjNjQ5MWFlNWU5NGI2OTlkNzgwMDIxMDljMTQ1MzE1MTJlYzQ1CmJvZHkJaW50ZXJuYWwvYmVuY2gvc3dlYmVuY2hfdGVzdC5nbwlUZXN0QW5FbXB0eVNXRUJlbmNoRGlmZklzVW5zb2x2ZWRXaXRob3V0VGhlSGFybmVzcwk1ZTc1NWI2MjU5NWQyNmYwMGMzMmVkYjk5ZDA2ZjUxNGFmOWQzMGE3YmVkZTBjZTA3NDQ2MzE1Mzg2Yzk3NzJhCmJvZHkJaW50ZXJuYWwvYmVuY2gvc3dlYmVuY2hfdGVzdC5nbwlUZXN0QW5JbnRlcnJ1cHRXaGlsZUdyYWRpbmdTdG9wc1RoZUhhcm5lc3NBbmRJdHNDb250YWluZXIJODA3YzJmOWY4MWU2ODExYzFiMmMxY2QwZTRkZDdhZGQyOWY2NTVkOTg4NTBkMmI2NGU2YTc0OTVhYWY3YTFmNApib2R5CWludGVybmFsL2JlbmNoL3N3ZWJlbmNoX3Rlc3QuZ28JVGVzdEFuSW50ZXJydXB0ZWRTV0VCZW5jaEF0dGVtcHRJc05vdEdyYWRlZAliNTViNDgzYzhjMDI0MjM5YWMxM2RkNDk5NTQ5ODExMTllZDBkODkxYTYyOTIwMGJkNzMyNzE5ZGM1NjYzNGNiCmJvZHkJaW50ZXJuYWwvYmVuY2gvc3dlYmVuY2hfdGVzdC5nbwlUZXN0VGhlU1dFQmVuY2hTbGljZUlzVGhlTG93ZXN0VHdlbnR5SGFzaGVzCTY1YTlmMjJlNzIyN2YzZDczMmE5Mjg5MGQ3OWFjODA4ZDBhMTdjYTcyNzBhZmNlYTg2NjNjNDI1YzBlMjIwMDgKYm9keQlpbnRlcm5hbC9iZW5jaC9zd2ViZW5jaF90ZXN0LmdvCVRlc3RUaGVTV0VCZW5jaFRhc2tJc1RoZVByb2JsZW1TdGF0ZW1lbnRBbG9uZQlhOTI1ODJhYWVjYTc4OTY2NzgwMGI3ZGIyYjVjOTA2MTJhOWZkMzE2ODU3NTYyZmNhMDM0NTFmODE3NGViMzliCmJvZHkJaW50ZXJuYWwvYmVuY2gvc3dlYmVuY2hfdGVzdC5nbwlUZXN0VGhlU3RhZ2U0U1dFU2xpY2VJc1RoZUxvd2VzdFNhbHRlZEhhc2hlc1dpdGhGaXZlSHVua3MJMzcxNmUzN2E2ZjgzNzg3ODk2NzA5ZGEzNDM3YWRjMDBlMmU4ZDBlMzk1MmM0YjkyM2VhNWRjYjM3YjNmYTJmNQ
  ```
  ```
- 2026-10-15 · 7b6e351a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ed1ea022329e0bc0788a7d1cd80d6c816db237cd02a5afe7a1914554dac7fe39 · ms:1381
