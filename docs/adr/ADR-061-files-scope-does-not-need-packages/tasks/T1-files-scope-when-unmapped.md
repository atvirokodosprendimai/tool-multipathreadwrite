# Task ADR-061-T1: `{files}`-only scoped_check runs when packages() is empty; contract §113

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `{files}`-only scopes when unmapped; §113
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `files-only template scopes a .rs path`, `packages-only still falls back`, `mixed still falls back`, `empty path list still falls back`, `Go {files} unchanged`

## Goal

`command()` returns the `scoped_check` substitution when `packages()` is empty if that template contains `{files}` and does not contain `{packages}`. A `{packages}`-only or mixed template still falls back to `check`. Drive the built binary through §113.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `command()`: `{files}`-only arm when the map is empty. |
| `internal/check/check_test.go` | edit | Red tests for the four template shapes on a `.rs` path, plus empty paths. |
| `scripts/contract.sh` | edit | **§113** — next free after §112. Pair: `{files}`-only scopes / `{packages}`-only falls back. |
| `docs/adr/BACKLOG.md` | edit | Receipt: the 2026-09-15 Zeus leftover is this record. |
| `AGENTS.md` | edit | One sentence: `{files}`-only `scoped_check` runs on `.rs`. |

## Ordered Steps

1. [S1] Write `TestFilesPlaceholderDoesNotNeedPackages`, `TestPackagesOnlyNonGoStillFallsBack`, `TestMixedPlaceholdersFallBackWhenUnmapped`, and `TestFilesOnlyWithNoPathsStillFallsBack` — the first is RED; the others pin the fallbacks that must stay. [proof: mutation]
2. [S2] Implement the `{files}`-only arm in `command()`. S1 GREEN. Deleting the arm must fail S1. [proof: mutation]
3. [S3] Write §113 against a binary that still falls back on `{files}`-only and confirm it is RED, then rebuild and confirm GREEN. The `{packages}`-only half must fail if the binary scopes an empty map. [proof: mutation]
4. [S4] BACKLOG receipt and the AGENTS.md sentence. Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 113\. ' scripts/contract.sh \
  && go test ./internal/check/ -count=1 -v \
    -run 'TestFilesPlaceholderDoesNotNeedPackages|TestPackagesOnlyNonGoStillFallsBack|TestMixedPlaceholdersFallBackWhenUnmapped|TestFilesOnlyWithNoPathsStillFallsBack|TestFilesPlaceholder$' 2>&1 | tee /tmp/adr061-t1.out \
  && grep -q '^--- PASS: TestFilesPlaceholderDoesNotNeedPackages' /tmp/adr061-t1.out \
  && grep -q '^--- PASS: TestPackagesOnlyNonGoStillFallsBack' /tmp/adr061-t1.out \
  && grep -q '^--- PASS: TestMixedPlaceholdersFallBackWhenUnmapped' /tmp/adr061-t1.out \
  && grep -q '^--- PASS: TestFilesOnlyWithNoPathsStillFallsBack' /tmp/adr061-t1.out \
  && grep -qE '^--- PASS: TestFilesPlaceholder\b' /tmp/adr061-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr061-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/check/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestFilesPlaceholderDoesNotNeedPackages` | `internal/check/check_test.go` | `.rs` + `lint {files}` + `check FULL` → `lint a.rs`, scoped | — | S1, S2 |
| `TestPackagesOnlyNonGoStillFallsBack` | `internal/check/check_test.go` | `.rs` + `go test {packages}` → `FULL`, unscoped | — | S1, S2 |
| `TestMixedPlaceholdersFallBackWhenUnmapped` | `internal/check/check_test.go` | `.rs` + `{packages}` and `{files}` → `FULL`, unscoped | — | S1, S2 |
| `TestFilesOnlyWithNoPathsStillFallsBack` | `internal/check/check_test.go` | nil paths + `lint {files}` → `FULL`, unscoped (`--full`) | — | S1, S2 |
| `TestFilesPlaceholder` | `internal/check/check_test.go` | `.go` `{files}` still expands (regression) | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests and §113 |
| 2 — something selects it | `Run` calls `command()`; deleting the new arm fails S1 and §113 |
| 3 — the caller can discover it | `check --help` already names `{files}`; AGENTS.md names the `.rs` case |
| 4 — it is used | Zeus already declared `scoped_check`; ADR-009 refuses telemetry |

## Mutation Log
(empty until execute)
- 2026-09-16 · aabdbf8* · mutant killed · exit 1 · `internal/check/check.go` · the {files}-only arm is gone: TestFilesPlaceholderDoesNotNeedPackages must go red · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · covers:files-only template scopes a .rs path
- 2026-09-16 · aabdbf8* · mutant killed · exit 1 · `internal/check/check.go` · mixed templates would scope with an empty packages map: TestMixedPlaceholdersFallBackWhenUnmapped must go red · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · covers:mixed still falls back
- 2026-09-16 · aabdbf8* · mutant killed · exit 1 · `internal/check/check.go` · an empty path list would scope {files} with nothing named: TestFilesOnlyWithNoPathsStillFallsBack must go red · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · covers:empty path list still falls back
- 2026-09-16 · aabdbf8* · mutant killed · exit 1 · `internal/check/check.go` · any non-empty path list would take the files arm, so a {packages}-only template would not fall back: TestPackagesOnlyNonGoStillFallsBack must go red · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · covers:packages-only still falls back
- 2026-09-16 · aabdbf8* · mutant killed · exit 1 · `internal/check/check.go` · Go {files} would stop expanding when packages() maps: TestFilesPlaceholder must go red · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · covers:Go {files} unchanged

## Invariants

- `packages()` stays Go-only and is not edited.
- A `{packages}`-only template still falls back when the map is empty.
- A mixed template still falls back when the map is empty.
- An empty path list still falls back.
- `TestFilesPlaceholder` stays green.
- No Rust mapper. No `covers` glob.

## Risks

- A `{packages}`-shaped Zeus `scoped_check` still falls back. Named in the parent Consequences.

## Stop Condition

If the only way to go green is to scope a `{packages}`-only template with an empty map, stop — that is a silent PASS.
If the only way to go green is a language table in `packages()`, stop — ADR-054 rejected it.

## Out of Scope

- Teaching `check --help` beyond the existing `{files}` mention
- Relocking ADR-054 T1–T4 (parent Follow-ups)

## Verification Log
(empty until execute)
- 2026-09-16 · aabdbf8* · exit 1 · `set -o pipefail …` · acceptance-sha256:190f518cba18617e9595f1b7edfb569ba447eb4ed0a810756b6aa63757f6b4ed · ms:840 · test-lock-sha256:c4085b3ca8d88ada61cb33f9a3e81d7fe81be27afc55e981b4ed788d6b64fde1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFDaGVja1RoYXRDYW5ub3RTdGFydERpZE5vdFJ1bglkMTNhNzBiYTcyNzYxNTI0M2Y4ZjcwOTRjMTAyYjA5YTNhMTY0ZDIzYTQzMGMzMzNlNTFlOGMyNmZiZmNmNjIyCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QURpcmVjdG9yeVRoYXRDYW5ub3RCZVJlYWRJc1JlZnVzZWROb3RUcmVhdGVkQXNFbXB0eQk2NzIzN2E5M2E1MTI2Yjc4YmNlYWIwMjJhZmQzOTE0MDM0MDlhZDE4ZjY1YjRkNjVlZTIyODY4ZmQxZjZhMjAyCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QUZhaWxpbmdDaGVja0tlZXBzSXRzTG9nCTlmMDQ2YmJkNzMwNGE1MjhjNDJhNGFkNTY2YTRlZjIxOGRhNzI4NDkxZjE5NTQyMWQ1ODEyMzdmODliMjA2MTEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFkZGVkQ2hlY2tJc1N0aWxsVGhlRGVjbGFyZWRDaGVjawk5OWUwMjYyNjkyZjMxMWVkYTVjNGNmZjVhMzBkMWMwOGYyODFjOWI5NTUzMjgxYTc2ZmExZGZiYWE3MWJjODE1CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVBhZGRlZFNjb3BlZENoZWNrSXNTdGlsbERlY2xhcmVkCTZjNmVlODBkODNiMmQ2YmU2NzZjM2Y5YzAyN2FmM2JhMGI5ZDA2MjI4MDkyZWE1YmJjMGEzYzI1ZDVlOWM4MDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFzc2luZ0NoZWNrTGVhdmVzTm9Mb2dCZWhpbmQJNTY2MWE2MDNlOGMwZGM5MzE5OTZkOGYzYTRhZDU1M2U5ZDExZDI4MDk0MWFmNWMyYzMyMTZkNTNiODEwMzdiNQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFQcmVzZW50VW5wbGFjZWFibGVJblJvb3RTY29wZVN0aWxsRmFsbHNCYWNrCTcxNWU4ODc0ZjRkZjkyZmVmODRlMTgwYjE2YTg5NDVhOTlmMTU3NWJkNDM3Y2I5YzVhMTJlNjM1OWFhMWQzZTQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUmVhZGFibGVEaXJlY3RvcnlTdGlsbFNjb3Blcwk0ODYxOTI3MDlhNTAyYmY5MTAxOTRkZGQ2MDM3OTgxNmE5ZjJiZjU1ZThjNGYzOGQ3NzA3NDZlYmEzYTA0ZGFmCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVJlZnVzZWRTY29wZURvZXNOb3RJbmhlcml0VGhlUm9vdHNWZXJkaWN0CTk5MjY2NmU2M2Y4ZTU1ZGJhYTI0ODQ5NGRlZTVlZTRjMDk3YWRhNGIyYTQyZTA1M2RhYTIzYjlmNWY1MTUxZDgKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBU2NvcGVPdXRzaWRlVGhlUm9vdElzUmVmdXNlZE5vdEZhbGxlbkJhY2tUbwlhYjc3ZjEwODdmMDg1YWU0YjdmNzIyY2Y3Y2Q2NjMwMmY2ZGU3NzJlNzZhNzgyNWVjZmY1NzJlMzY3ZDZjM2M1CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVNoZWxsSW5qZWN0ZWRTY29wZVN0aWxsRmFpbHMJZmUyOGFiYjc0ZWY1NmVmZmNiNmFiYmNjNGJmNDgzZWUzNWQzMjQ3YzA0N2FmYWU3MzM4ODdhYzYzNmU1ZDBhYwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFXaGl0ZXNwYWNlT25seUNoZWNrSXNOb3REZWNsYXJlZAlhOTA3ZWU0MzlmMTRkYzJlNDhkMmU0MTM4MDEwODc5MTUzM2E0YTRmMDhlMjRlOGQwZWYyZDRiMzU3OTFlYTQ0CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVdoaXRlc3BhY2VPbmx5U2NvcGVkQ2hlY2tJc05vdERlY2xhcmVkCWY2YmYwYzU0OWQ2NDZhNzUwNzUwZmZhNTEwYzk4ZjY5MzUxNThmODViMzcyMjYwZTg4NjI1MzViOWI0YWE1MWIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBbkluUm9vdE1pc3NJc1JlZnVzZWROb3RBU2lsZW50UGFzcwkwMzk3ZWIyOTc2ZTIzMWUwZmRlZTI4ZWY3NTBmYTQwMGY1ZGY3MjJiMWYyNmE4MDc4MDUxYjg0MmM2ZmRhNDIxCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QW5PdmVybGFyZ2VUaW1lb3V0SXNDbGFtcGVkTm90T3ZlcmZsb3dlZAlkMGIxOGU0NTcwMzRjZjJjMTI5MTZhNmZmNjA2MTRjNmRlMTUwNWRiNmFmZGVlZmJlZjJmNDAyN2QyNDVhMjliCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNPbmx5V2l0aE5vUGF0aHNTdGlsbEZhbGxzQmFjawkwMThiMTY4NTRlMTU3ZmQzN2E4ZTgwY2E4YmIzZmU1OWQ2NTM2OWQ1ZGNhOTdhMjJhZGQ0NGQzNjU5NmUyNGViCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNQbGFjZWhvbGRlcglmYTRiYTBhNmQ1NDM3MzYwMGYwMDI0NzliMDk4YTc1MTEyMTlhMjU0MDhjNmE2OTg0MTU0ZGZlMmUwNTkzN2MxCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNQbGFjZWhvbGRlckRvZXNOb3ROZWVkUGFja2FnZXMJODFjZmQ0MzU3NTM0ODBlOTZkZGUxNTIxN2JlZTJhYTg1OTA2MDllZDYzZTBlZjQ3ODA4ZjQ1NTkwNGY1NTY3ZApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRJbmZlcnNHb0NoZWNrQnV0U2F5c1NvCTA5OWU2YmZmM2ZlNzdhMzM5MDc1M2M1ZWZiYzdlMjE1NWQzYjY0MDY5YTk5YjJlMmRlMWVlZmRkMzllYjMzMWEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RMb2FkT25BQmFyZURpcmVjdG9yeUhhc05vQ2hlY2sJZjU1ZTJiM2ZlZWVmMzc4ZmJjOTI1OTRlZmM0ZmI1YmZkNjMzNGIzNzM1OTg1M2IzNTNjM2Y4Mjg1Y2MxNGU2Mwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRQcmVmZXJzVGhlRGVjbGFyZWRDaGVjawkyNWQxZjdjZTU5ZGU1M2Y4ZGJjYzg2Zjg4ZWJiNTVlNWNlMzRlNGU2ZWUyOTcxZmJhZDgzZTBmZTZjZTI5MDllCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0TWl4ZWRQbGFjZWhvbGRlcnNGYWxsQmFja1doZW5Vbm1hcHBlZAkxYWM3NzM4Yzk3OWM3MDgxYjcxYTIyODkwNjI2MzA0Njk4NmVjZjUyZDNiOWM2MTU3MmVkZDhjOTdhZDNkMzM3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0Tm9DaGVja0lzTm90QVBhc3MJYTNiMzNkYWU0MjdlYTZjNWE2YmIwODk5NDA4MjJkZTVhYzhiZTk0MmFlYjAyMTkwYmE1YTg0NjAzYjA1OGFiNgpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFBhY2thZ2VzT25seU5vbkdvU3RpbGxGYWxsc0JhY2sJMWJhMGIyNTMwYmZjOTNmNDg2NzEwZTkwMmU2Mjk2NGEzNzhkZTYxZTAzYzdkZGQ2YjRlZjhmN2E5ZjQ1ODM5ZQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFJ1blBhc3Nlcwk4YjIyZWMzNjdmNjBmOGFmYTg3NzEwZjFhYTgxMGI2OGZjOWQ5MGMxZWM4NjYyMTU5NTUwZDY4NjJhNDAyZmY1CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0UnVuUmVwb3J0c1RoZVJlYWxFeGl0Q29kZQlmYmNiNzgyN2Y1MDhiMDY3MDA1ZGYxZTEzZDBkYmVlMDc2Zjg0ZjZlZDU1YTU2NjlmNmZiYWUzYTlmMWQ1ODAwCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0U2NvcGVEZXJpdmF0aW9uCTY0ZThhYmY3OGM1ZTM4MzU2NmFmZDFlNTI5YjNlMGFjYzVhZWI2ZmVlYzhmMWMyOGU0MDdhN2EzNTkyNDNlMzQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RTdWJzdGl0dXRlZFBhdGhzQXJlT25lU2hlbGxBcmd1bWVudEVhY2gJZWMzZWRlZWU5NTViZGJjMDdjYThkZGJlYzI1NGEyNTgzMDMxYzMyZDRiMDIwZjY2NjM3ODUyMWQ0OWNkNjE1Ngpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFRhaWxBbm5vdW5jZXNXaGF0SXRMZWZ0T3V0CTVlYjAxMmUwYzRlOWFlODI2N2FjMDczNDQ3NTQyODJlMGE3MzU3ZWE0YzNjMGI5ZGVjMDBjZjIwM2FhYTRhNWYKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RUaW1lb3V0SXNSZXBvcnRlZEFzQUZhaWx1cmVOb3RBUGFzcwkyMTFmMWQ1YWQ5NDBhM2IxMjhhNjRkOTgzMDE2NmY0OGFjZDA3YWJiZDU5N2ZjYzVjOGQwZTA2M2FmM2IyMTAz
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  --- FAIL: TestFilesPlaceholderDoesNotNeedPackages (0.00s)
  === RUN   TestPackagesOnlyNonGoStillFallsBack
  --- PASS: TestPackagesOnlyNonGoStillFallsBack (0.00s)
  === RUN   TestMixedPlaceholdersFallBackWhenUnmapped
  --- PASS: TestMixedPlaceholdersFallBackWhenUnmapped (0.00s)
  === RUN   TestFilesOnlyWithNoPathsStillFallsBack
  --- PASS: TestFilesOnlyWithNoPathsStillFallsBack (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	0.198s
  FAIL
  ```
- 2026-09-16 · aabdbf8* · exit 1 · `set -o pipefail …` · acceptance-sha256:190f518cba18617e9595f1b7edfb569ba447eb4ed0a810756b6aa63757f6b4ed · ms:697
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
  === RUN   TestFilesPlaceholderDoesNotNeedPackages
  --- PASS: TestFilesPlaceholderDoesNotNeedPackages (0.00s)
  === RUN   TestPackagesOnlyNonGoStillFallsBack
  --- PASS: TestPackagesOnlyNonGoStillFallsBack (0.00s)
  === RUN   TestMixedPlaceholdersFallBackWhenUnmapped
  --- PASS: TestMixedPlaceholdersFallBackWhenUnmapped (0.00s)
  === RUN   TestFilesOnlyWithNoPathsStillFallsBack
  --- PASS: TestFilesOnlyWithNoPathsStillFallsBack (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	0.194s
  ```
- 2026-09-16 · aabdbf8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:865
- 2026-09-16 · aabdbf8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:738
- 2026-09-16 · aabdbf8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:1070
- 2026-09-16 · aabdbf8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:481
- 2026-09-16 · aabdbf8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:482
- 2026-09-16 · aabdbf8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:1110
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:0 · test-lock-sha256:e9a7f303b06a35e2bc930cfd4f7a568977b7c33b6ca84abd9766a74b4a753c44 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFDaGVja1RoYXRDYW5ub3RTdGFydERpZE5vdFJ1bglkMTNhNzBiYTcyNzYxNTI0M2Y4ZjcwOTRjMTAyYjA5YTNhMTY0ZDIzYTQzMGMzMzNlNTFlOGMyNmZiZmNmNjIyCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QURpcmVjdG9yeVRoYXRDYW5ub3RCZVJlYWRJc1JlZnVzZWROb3RUcmVhdGVkQXNFbXB0eQk2NzIzN2E5M2E1MTI2Yjc4YmNlYWIwMjJhZmQzOTE0MDM0MDlhZDE4ZjY1YjRkNjVlZTIyODY4ZmQxZjZhMjAyCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QUZhaWxpbmdDaGVja0tlZXBzSXRzTG9nCTk5MWY0MjFmMGU3N2FiODQwZjMyOTQyZTI2YmI2NGFhYmM3MDllNGM0Y2UwYzA5YmZiOWM4ZGZiNWYxYzgwZWUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFkZGVkQ2hlY2tJc1N0aWxsVGhlRGVjbGFyZWRDaGVjawk5OWUwMjYyNjkyZjMxMWVkYTVjNGNmZjVhMzBkMWMwOGYyODFjOWI5NTUzMjgxYTc2ZmExZGZiYWE3MWJjODE1CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVBhZGRlZFNjb3BlZENoZWNrSXNTdGlsbERlY2xhcmVkCTZjNmVlODBkODNiMmQ2YmU2NzZjM2Y5YzAyN2FmM2JhMGI5ZDA2MjI4MDkyZWE1YmJjMGEzYzI1ZDVlOWM4MDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFzc2luZ0NoZWNrTGVhdmVzTm9Mb2dCZWhpbmQJNzU5ZTdjM2Q0Y2M0MTExYzExYTQzYzM4YmMwNTFlYTIzNzM1MjY1ODRmMzdiNTM2OTAxOTk4ZTY0ZmFhYjA3MApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFQcmVzZW50VW5wbGFjZWFibGVJblJvb3RTY29wZVN0aWxsRmFsbHNCYWNrCTU2YTAyZGY4NjEwMmY3ZTliY2JlZTM4OGYwNDJiNTY3Y2IwODg3MzExNWI0OTI2MmFhN2Y1YWUxZGU3ZjhkYjkKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUmVhZGFibGVEaXJlY3RvcnlTdGlsbFNjb3BlcwljNmVhY2E2N2JkMTI2Yjg0NzQ5YzlkNmEzZWVmMDJlMzI2NDI0MzgzYzVlYWEyOGZmOTA5NGY5MGViMDc2Y2Q3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVJlZnVzZWRTY29wZURvZXNOb3RJbmhlcml0VGhlUm9vdHNWZXJkaWN0CTk5MjY2NmU2M2Y4ZTU1ZGJhYTI0ODQ5NGRlZTVlZTRjMDk3YWRhNGIyYTQyZTA1M2RhYTIzYjlmNWY1MTUxZDgKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBU2NvcGVPdXRzaWRlVGhlUm9vdElzUmVmdXNlZE5vdEZhbGxlbkJhY2tUbwlhYjc3ZjEwODdmMDg1YWU0YjdmNzIyY2Y3Y2Q2NjMwMmY2ZGU3NzJlNzZhNzgyNWVjZmY1NzJlMzY3ZDZjM2M1CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVNoZWxsSW5qZWN0ZWRTY29wZVN0aWxsRmFpbHMJMWFlYTYwNzMxOGYzNGExYjA4YmY1NDI1MzRiY2E2NTU2ZjFjN2YwN2ZhZmVjZWY0ZjFmNjQ3YTkxYjU4ZWMwZApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFXaGl0ZXNwYWNlT25seUNoZWNrSXNOb3REZWNsYXJlZAlhOTA3ZWU0MzlmMTRkYzJlNDhkMmU0MTM4MDEwODc5MTUzM2E0YTRmMDhlMjRlOGQwZWYyZDRiMzU3OTFlYTQ0CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVdoaXRlc3BhY2VPbmx5U2NvcGVkQ2hlY2tJc05vdERlY2xhcmVkCWY2YmYwYzU0OWQ2NDZhNzUwNzUwZmZhNTEwYzk4ZjY5MzUxNThmODViMzcyMjYwZTg4NjI1MzViOWI0YWE1MWIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBbkluUm9vdE1pc3NJc1JlZnVzZWROb3RBU2lsZW50UGFzcwkwMTc0YTk2NjE1OGYyYThkNmE5OTIxMWQ3MGVlMWIxMGUwYzBmODA4NTc4YzExZmE4NzQ2MzA3MDA1YjQ5Y2QxCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QW5PdmVybGFyZ2VUaW1lb3V0SXNDbGFtcGVkTm90T3ZlcmZsb3dlZAk2MjI5MzU0ZDk1MzJmZDBhYTU2NWIzMGI0NTRkZDc3ZWRiZmQ1OWQ1YmNkNjZiNzRjY2QxOTlkZDIyMzg1YmI0CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNPbmx5V2l0aE5vUGF0aHNTdGlsbEZhbGxzQmFjawkwMThiMTY4NTRlMTU3ZmQzN2E4ZTgwY2E4YmIzZmU1OWQ2NTM2OWQ1ZGNhOTdhMjJhZGQ0NGQzNjU5NmUyNGViCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNQbGFjZWhvbGRlcglmYTRiYTBhNmQ1NDM3MzYwMGYwMDI0NzliMDk4YTc1MTEyMTlhMjU0MDhjNmE2OTg0MTU0ZGZlMmUwNTkzN2MxCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNQbGFjZWhvbGRlckRvZXNOb3ROZWVkUGFja2FnZXMJODFjZmQ0MzU3NTM0ODBlOTZkZGUxNTIxN2JlZTJhYTg1OTA2MDllZDYzZTBlZjQ3ODA4ZjQ1NTkwNGY1NTY3ZApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRJbmZlcnNHb0NoZWNrQnV0U2F5c1NvCTA5OWU2YmZmM2ZlNzdhMzM5MDc1M2M1ZWZiYzdlMjE1NWQzYjY0MDY5YTk5YjJlMmRlMWVlZmRkMzllYjMzMWEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RMb2FkT25BQmFyZURpcmVjdG9yeUhhc05vQ2hlY2sJZjU1ZTJiM2ZlZWVmMzc4ZmJjOTI1OTRlZmM0ZmI1YmZkNjMzNGIzNzM1OTg1M2IzNTNjM2Y4Mjg1Y2MxNGU2Mwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRQcmVmZXJzVGhlRGVjbGFyZWRDaGVjawkyNWQxZjdjZTU5ZGU1M2Y4ZGJjYzg2Zjg4ZWJiNTVlNWNlMzRlNGU2ZWUyOTcxZmJhZDgzZTBmZTZjZTI5MDllCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0TWl4ZWRQbGFjZWhvbGRlcnNGYWxsQmFja1doZW5Vbm1hcHBlZAkxYWM3NzM4Yzk3OWM3MDgxYjcxYTIyODkwNjI2MzA0Njk4NmVjZjUyZDNiOWM2MTU3MmVkZDhjOTdhZDNkMzM3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0Tm9DaGVja0lzTm90QVBhc3MJYTNiMzNkYWU0MjdlYTZjNWE2YmIwODk5NDA4MjJkZTVhYzhiZTk0MmFlYjAyMTkwYmE1YTg0NjAzYjA1OGFiNgpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFBhY2thZ2VzT25seU5vbkdvU3RpbGxGYWxsc0JhY2sJMWJhMGIyNTMwYmZjOTNmNDg2NzEwZTkwMmU2Mjk2NGEzNzhkZTYxZTAzYzdkZGQ2YjRlZjhmN2E5ZjQ1ODM5ZQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFJ1blBhc3Nlcwk4ZWI3MGZmOGY4OTFjOTAzY2JlNmQ4YjkyMDI1Y2M5OTc0NzU1NmE5MmJkY2EzMGM4YzRhNjEwOGY1ZTU4ZmE3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0UnVuUmVwb3J0c1RoZVJlYWxFeGl0Q29kZQlhMGE0YzBmNjEwZDk1Y2Y0NDFjMDBkODViZDgyNGE4NWQ1Mzk2NjBhYjI3ZDYzYjkwZDI0Njg0MWMxNTdhZGQ3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0U2NvcGVEZXJpdmF0aW9uCTY0ZThhYmY3OGM1ZTM4MzU2NmFmZDFlNTI5YjNlMGFjYzVhZWI2ZmVlYzhmMWMyOGU0MDdhN2EzNTkyNDNlMzQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RTdWJzdGl0dXRlZFBhdGhzQXJlT25lU2hlbGxBcmd1bWVudEVhY2gJZWMzZWRlZWU5NTViZGJjMDdjYThkZGJlYzI1NGEyNTgzMDMxYzMyZDRiMDIwZjY2NjM3ODUyMWQ0OWNkNjE1Ngpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFRhaWxBbm5vdW5jZXNXaGF0SXRMZWZ0T3V0CTQ3NDk0M2RjNjE1MzlhMmViNDVhOGM3OTMzOGI5ZTk2OGY4YmNkMmViMDkwZTE3ZTcwYjg3ZGM4Y2JjODZkODUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RUaW1lb3V0SXNSZXBvcnRlZEFzQUZhaWx1cmVOb3RBUGFzcwlmZjdkMDYwYzg3NWVkN2U1ZDlmNmNiYjEzMjIyY2FiY2U0MjI5ZGY3MDZmMmMwY2MyZDBhNTk3ZTk0NGFlMmFl · test-lock-kind:replace
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:a29a06959ebbbccce6faec0f38bb005b7ee42d005e01fd0f37aca6a3255424b7 · ms:521
