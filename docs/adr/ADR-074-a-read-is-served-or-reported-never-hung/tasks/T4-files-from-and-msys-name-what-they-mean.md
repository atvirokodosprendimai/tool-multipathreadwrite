# Task ADR-074-T4: `--files-from` names a long line; the MSYS hint names both variables

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the line number in `specList`'s error; the MSYS hint and prose
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the long line is named by number`, `the hint names both variables`, `the docs say when the rewrite fires`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A spec line over 8 MiB in a `--files-from` list failed with "token too long" and no line number. The
MSYS hint named one of the two variables that stop the rewrite, and the docs showed an example the
round measured the rewrite does not touch.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `specList` counts lines and names the long one |
| `cmd/mrw/filesfrom_long_test.go` | new | line 3 of a list is named |
| `internal/read/read.go` | edit | `msysHint` names `MSYS_NO_PATHCONV=1` |
| `internal/read/msys_hint_test.go` | new | both variables named |
| `AGENTS.md`, `README.md`, `internal/guide/guide.go` | edit | when the rewrite fires; both variables |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN; the guide and instructions tests stay green. [proof: mutation]
   Mutants: the line counter counts specs, not lines; the hint drops `MSYS_NO_PATHCONV`.
3. [S3] The prose in AGENTS.md, README and the guide. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ ./internal/read/ ./internal/guide/ -count=1 -timeout 180s -run 'TestAFilesFromLineOver8MiBNamesTheLine|TestTheMSYSHintNamesBothVariables|TestAMangledMSYSSpecSaysWhatHappened' -v 2>&1 | tee /tmp/adr074-T4.out \
  && missing=$(for t in TestAFilesFromLineOver8MiBNamesTheLine TestTheMSYSHintNamesBothVariables TestAMangledMSYSSpecSaysWhatHappened; do grep -qE "^--- PASS: $t \(" /tmp/adr074-T4.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q 'MSYS_NO_PATHCONV=1' AGENTS.md \
  && grep -q 'MSYS_NO_PATHCONV=1' README.md \
  && grep -q 'MSYS_NO_PATHCONV=1' internal/guide/guide.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/read internal/check internal/subproc ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' ':(exclude)internal/read/astgrep.go' ':(exclude)internal/read/fifo_unix_test.go' ':(exclude)internal/read/msys_hint_test.go' ':(exclude)internal/check/check.go' ':(exclude)internal/subproc/subproc.go' ':(exclude)internal/subproc/interrupt_unix_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/read internal/check internal/subproc ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' ':(exclude)internal/read/astgrep.go' ':(exclude)internal/read/fifo_unix_test.go' ':(exclude)internal/read/msys_hint_test.go' ':(exclude)internal/check/check.go' ':(exclude)internal/subproc/subproc.go' ':(exclude)internal/subproc/interrupt_unix_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFilesFromLineOver8MiBNamesTheLine` | `cmd/mrw/filesfrom_long_test.go` | "line 3" and "8 MiB" named | — | S1, S2 |
| `TestTheMSYSHintNamesBothVariables` | `internal/read/msys_hint_test.go` | both variables in the hint | — | S1, S2 |
| `TestAMangledMSYSSpecSaysWhatHappened` | `internal/read/read_test.go` | issue #45's hint, unchanged | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the counted line; the second variable |
| 2 — something selects it | every `--files-from`; every mangled spec |
| 3 — the caller can discover it | the error names the line; the hint names the fix |
| 4 — it is used | the round met both |

## Verification Log
(empty until execute)
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:ba35add1f68cbf6ae4ac2b845cf51eedb7f2611388cfd1fb3ef1f500a8f2d0de · ms:463 · test-lock-sha256:7c1b90816e9df6c9040a48984ebb9f6dcd58e5496f161925bd438fd3a75ffef7 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZmlsZXNmcm9tX2xvbmdfdGVzdC5nbwlUZXN0QUZpbGVzRnJvbUxpbmVPdmVyOE1pQk5hbWVzVGhlTGluZQk4OWVkNzgzN2ZmM2ZmYTJmOWE3YTQyZmUxNTc2Y2ExMDE4MjA5NjlmOTJhYmY4NDMyY2EwNTMwYTY2ZWMwYWVkCmJvZHkJaW50ZXJuYWwvcmVhZC9tc3lzX2hpbnRfdGVzdC5nbwlUZXN0VGhlTVNZU0hpbnROYW1lc0JvdGhWYXJpYWJsZXMJYjNmZGE5MjM5MTMzZmU5YmZjMmIxNWQ0YWExYTY5NTVjOTFjMDU1ZDBiNjA3NTFkZDE2NGNiY2Y5NTNkNzc4Mwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBQ2FwT2ZaZXJvU2VydmVzTm90aGluZwk1MWYwYTg1NjA2NDE0ZTYwYTU4MzM3Njc1ZWU0NzBmNWI2Mjc1YTQyOWJiZWY0NmI3YTY3ZmQ4MDU2MGE1YjZlCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdEFHbG9iUGF0aEtlZXBzVGhlR2xvYkhpbnROb3RUaGVFbmdsaXNoV29yZEhpbnQJNmU5Y2U0MjgwMzg2ZWFjZDdmZTVjMWRmMGY3YTM2NDRiMDc5NzU1NDlhYTg2NWM0NmFlZTA4NGIwY2VlYjdmMwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBR2xvYlRoYXRUaGVTaGVsbERpZE5vdEV4cGFuZFNheXNTbwkzY2Y4YjM2ZjJjMGU0ZTY4MTFiNWVjMzBmMGE3NzVlNGQyZjQ4YWQyM2M2MmMwMzM4NWMxYTE2MWE5M2I4ZWIyCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdEFHbG9iVW5yZWFkYWJsZVBhdGhOYW1lc1RoZUdsb2JFc2NhcGUJMmNjM2ZkMjZiMDRmMzI3YjI0NTFmMTM3NTljNDkyN2Y5OTdkZWRiZGEzNGE1ZjA1NDViZGI1NjcxMTc2MDk0Ywpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBTWFsZm9ybWVkUGF0dGVybkFkZHJlc3NJc1JlZnVzZWQJMDg3NmFkZTAzYzg0ZGNkMzkxNjg1ZmQyNjA5ZDJmMzA2YzVlMjBkMDVlNjRhOWFhN2E2ZTRjYjgxOTFlYzg2Ngpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBTWFuZ2xlZE1TWVNTcGVjU2F5c1doYXRIYXBwZW5lZAliMmU0NTRjZDVhOTMxZGYzM2IyYzU4OTI3MGYyNTRkNWJkNmI5MWJkMTdmYjcxNjY3Mzk5MjY4Y2MzM2Q5M2JmCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdEFQYWlyZWRQYXR0ZXJuRW5kc0F0T3JBZnRlckl0c1N0YXJ0CTg5NTBlZTVmNmY4ZThhMTlkMmZmODUwN2JhYzg3Yjg0ZTFkNzI0NTk5YmU1MWE0OGIzOWQ4OWIzMzlkZmNjODUKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVBhaXJlZFBhdHRlcm5XaXRoTm9FbmRJc1JlZnVzZWRSYXRoZXJUaGFuRXh0ZW5kZWRUb0VPRglkMWY4ODUyZTI3YmZlN2I2ZTJmMGNlMzExZDRmZjZhMGFlNTVmOTMzNzAzY2VhMjBmNjM4MTM2MGVhN2EzYjQxCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdEFQYXR0ZXJuRW5kaW5nSW5BQmFja3NsYXNoSXNDbG9zZWQJM2JhOWRiNDk0NjliNzFlZTc5OTNhZjBiODEwN2FiYjY3NzRkYzc0Y2RiMjc0YmYxOWU5OGE0MjBmMDkxNjQ3Ngpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBUmFuZ2VBZ2FpbnN0QW5FbXB0eUZpbGVJc1JlcG9ydGVkCWQ4OWRhYTQyZjk5ODliZDY1YmQxMzVkZGQwMGViNzQxNDc0MjA4NWM1ODUxYmFhZjNmZDcxNjEwYjVlMjVkZjYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJhbmdlVGhhdE1hdGNoZXNOb3RoaW5nT2JzZXJ2ZXNOb3RoaW5nCWRmZTYyYTgzYzUxYjlkODEyMmI3ZjlmMzc3N2JlYmI5YzJlODc2N2UyYjJkOWJmYWQzNTRjNDI1YWZjNWE5OWIKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kQXRUaGVJbnRlZ2VyQm91bmRhcnlDbGFtcHNPbkFSZWFkCWRlMmYxZDIwM2JjMTE5ZGI3MDgzN2RiNDBiNTg5ZjdmZTRiOTFhODhhZWFjN2ExMzZhYjA3NTM3YjdkNWJhMDcKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kT2ZaZXJvSXNSZWZ1c2VkCTc1ZDg4NjhmYzhmNDZmMTAzMjFiOGY1NGFiOTQ3OGRjMTU3NWNlOGZjNDA0ZjgyNTk4ODkyNDlhOTk2MzBhNGIKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kUGFzdFRoZUxhc3RMaW5lQ2xhbXBzCWExNzQzZmFlMDNhY2FmYzBlYThkMmFhZTY0MjYwOWE3OWRhOGIxN2VlNDg2YTRlNTQ5ZjRjYTA0OTM4MzMyOGUKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kU2VydmVzVGhlTGluZXNBZnRlclRoZVN0YXJ0CWZlODU2NGEyYzMwNTE2ZDJiYzdkYzY1MGQyZDJjYjQ2NzMzNDRlNjllMDk0MmJhOGZkYjA0YjY0MWFjNjg1NDkKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kV2l0aE5vdGhpbmdCZWZvcmVJdElzUmVmdXNlZAllZjRiM2E5MGNkMDkyMzU1ZGMzMzgwZDc5ODQxYzU4ZGNlN2ZmZDQwNmJhNmE3YThjYTJiNzU2OGE5YTIzNjBkCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdEFSZXNvbHZlZFNwYW5Eb2VzTm90U3VwcHJlc3NBTGF0ZXJNaXNzaW5nRW5kCWU3NjBkN2NjMmFlMDE2MzI5MDVlOGI0MDZlODU3NGIwMmYyMWIyZDk5YzkxNTZmNDczMDc1ODMyYmU1YzVlMjYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QVJldmVyc2VkUmFuZ2VXcml0dGVuV2l0aERvbGxhcklzTm90U2VydmVkCTgzYjgwNDU5NzE1NjIzNzA4YWY1ZTZlZmE2YWMwNmM5ZTVmNWQ4MWQ4Mzg4N2FiZmUyN2I5MDZkY2QyYWNjMjMKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QW5BYnNvbHV0ZVBhdGhJc0FXaG9sZVBhdGhPblRoaXNQbGF0Zm9ybQllZTA1YTA1YmFlYjZmNDM2MzI4MzEyOTAxZDQxMGZkNjA3ZGViYzZhZmI3MWIxYmY4NzI1MzAxMWRiYTYxNmIwCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdEFuT3JkaW5hcnlCYWRTcGVjR2V0c05vTVNZU0hpbnQJYjNkNmFjODVmYWIyNDY5NmM2MDBlZmFkZDdiNDJiNWJiYzY5MGM2Mjg0MmVhNGNjYTI4NDBjNmM5NzA1NzVmMQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBbk9yZGluYXJ5TWlzc2luZ0ZpbGVHZXRzTm9FbmdsaXNoV29yZEhpbnQJNjUzYmZmM2E3ZDhkZTQ3MDhhMmYwNzhhM2E4YjY0MTNmMTUzNjMwNzQxYjg1ZDU3NGE1OWRmMjg4M2ZhMGQyNgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBbk9yZGluYXJ5TWlzc2luZ0ZpbGVHZXRzTm9HbG9iSGludAlmZmYwZjY1MjFkZjI0NzRmNzJiMDVkNGFiODk2OGZhMGEzN2RmZmM1NWM1NzMzOWYzOTU2Y2E0YWI0Yzc3M2UzCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdENvbnRleHRBcm91bmRTaW5nbGVQYXR0ZXJuCWEwNmFkODA3YjFhNTlhOGU4YTFmMDlmNjAzNTdkYmZlOTRhNDI1ZmNlZGE2NzMxYzhiOGNlMTFkMzkwM2NjMmYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0Q29udGV4dEF0VGhlSW50ZWdlckJvdW5kYXJ5Q2xhbXBzCWFiMWU5M2U1NjM0OGM2ZWVmMmE3MmRiOWMwNGMyYTA1ZjA3YzdjNGYzNWNjYzZjYzA3MTdkYTA3MzcxMjQ0OTEKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0RGVzY2VuZGluZ1Jhbmdlc0FyZVNvcnRlZEFuZE1lcmdlZAk3MDY0YjQxYTc4OTEzYjdmZDcwYTc5YjYwZWI1NWQwMTIyYjhlMzFjZGRhNDYzYTJhMDgxZTM2MGVmZGNlZjUwCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdERvbGxhckFzQW5FbmRTdGlsbFJ1bnNUb0VPRglmOTYwNjVmNGI5NmM3NTc3OTFlZWE3OTEzMGRjZWZiZjhiNmVhY2NkMzM3Y2ZmOGMzZDIzOTU5MDcyM2Q2NjQ1CmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdERvbGxhcklzVGhlTGFzdExpbmVOb3RUaGVXaG9sZUZpbGUJNGIzNGYzMTFhZmI1NjAyOWY3YzUxYjA5ZTA4NGJmODIzZGEyNDgwYTA4ZTMyZWI2YTljMDcxMDlhNjdmMmY3Zgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RNYW55RmlsZXNPbmVDYWxsCTYzNGQ3ZjczMmNhYmYxOTNhNTU2YWFmODg3YzZlMTQ2MWY3NzM2ZWNkNjU2ZjNjZTdiODNkYjdjYTVjMDY2OGEKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0TWFueVVuc29ydGVkUmFuZ2VzTWVyZ2VDb3JyZWN0bHkJZGQzMmQ1ZmRjMDQ2ODk3ZTJjMzVhN2NjM2M5OGE5NjMzYzc4ZWI1YTM0MTE3MzNkNWQ4NGI5MjcwMTc4NmYwYgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RNYXhMaW5lc1JlcG9ydHNXaGF0SXRXaXRoaGVsZAkwNDczYjE1ZmRhOTFhNDY2MjNjMDIzMmQ5NzU0ZTkzNTk5NDRjZWE2NzEyNjAwYWFiMzI2MmVlY2QxMjEzODNhCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdE51bWVyaWNSYW5nZXNBbmRIZWFkZXIJMWE4MDgxOWI3NDNhOTJjZTQ3Y2QwYTg5NmQxOWU4Y2ZjNzQ2YzYyYTY4MzAxNzM0MTI2Y2ZkODllN2Q1NmY3YQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RPdmVybGFwcGluZ1Jhbmdlc0FyZU1lcmdlZE5vdFJlcGVhdGVkCTdlOTNmMjQ4NDM5Y2Q1MDk4ZmVlY2NiMzZhOWZhYmViYTQ4OWNhYjc5MzQ4ODM2NmIzYjNhNDRkOGMwZTYwZTEKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0UGFyc2VTcGVjCWU3Y2Q1ZWZiMDJhMWM3ZmE5MTJkYmQ2YzMyMWRjMGNmY2VkZDExY2Y3NTMxZDFlY2IzYmNlODA3OGI2NWQzY2MKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0UmVhZEhpbnRzQU11bHRpTGluZVJlcGxhY2VOZWVkc1RoZU5leHRMaW5lCTgxMTljYWIyYzg5MDQ5OTA4NGU0ZjFjODgwMDEyYmIzMGJmMzU2MTNjNzE4OTcyOTNiNjc3MjliNWE3N2Q5NmYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0UmVnZXhwUmFuZ2UJMjMwMzQxYjNkOTljNzg3NzdhMzViOTI5ZmQ2NjA1NjExNjEzYjQ0ZDlmZGQ0MDQxYzA3MGExN2E2NGYyNzJmNQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RTZXZlcmFsRW5nbGlzaFdvcmRVbnJlYWRhYmxlUGF0aHNOYW1lUXVvdGluZwljMjZhNDA1MjA2YWFiMmVlMTZiZGM3ZWJjNGM5NGRiMmI2NTZiN2I3NzQwMGY2MzIzMmM5MDdlNmFjZDJiM2IzCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdFN0YXRBc2tzRm9yVGhlRmFjdE5vdFRoZUFydGlmYWN0CTQwZWJmNjVhYzBiYWFiNThkNTA1MjdmMTBkZGEzNzVmZGMwNDlhYTEyZDFlNzUyZDA4NzUzOTg3N2EyY2ZlZWIKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0VW5tYXRjaGVkUGF0dGVybklzUmVwb3J0ZWQJZThlYTM3MWNjZmI4YzczODBiODE5ZGY5ZThjODUwOWMxMjlmZWUxNDZlNDc5NzdjZGEwOGFiMGFkOTc5YzJjMQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RXaG9sZUFuZFBhcnRpYWxPYnNlcnZhdGlvbnNBcmVVbmNoYW5nZWQJNmI2ZTM0YzIwNGM2ZjQxZTAyYTA1YjJmYTFlZjQwYTliNTM2YWExM2QzMjNhOGIwNGQ3MGJkYjRjMGZkNzgzNgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCXJhbmdlZAljMTAyZjg4NGJkZTBhODg1ZmM3YWUxYzI5YzQ3NTRmMTA3MGNkNjk4MzQ0ZGM5MjU0NTZjMmM2NmRlMmVhYWMyCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28Jc2luZ2xlCWY4NDIzMWY3MzRhM2ZmMjg1YmI5YTdjZWFjNjgyNjFmMmVmZTI4OWY0MGFlYWY5OTQwY2M3YzY1MDZiMmFiZWYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwl3aG9sZQkzZDJlNTllOTNkMDUzNGUwMmIyNWIwODhiNWZkNGMyMGVkNDJiNGMzY2QzYjJlMmE5Y2NhNDM1MmI2OWU1Yjdl
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
          "cmd\\mrw\\main.go;C:\\Users\\me\\AppData\\Local\\Programs\\Git\\^func main\\": bad line number "\\Users\\me\\AppData\\Local\\Programs\\Git\\^func main\\" — this looks like MSYS2 argument conversion (Git Bash): your ':' became ';' and a /pattern/ was expanded against the Git install prefix. Quoting does not stop it. Set MSYS2_ARG_CONV_EXCL='*', or use PowerShell or WSL
  --- FAIL: TestTheMSYSHintNamesBothVariables (0.00s)
  === RUN   TestAMangledMSYSSpecSaysWhatHappened
  --- PASS: TestAMangledMSYSSpecSaysWhatHappened (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.066s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide	0.162s [no tests to run]
  FAIL
  ```
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba35add1f68cbf6ae4ac2b845cf51eedb7f2611388cfd1fb3ef1f500a8f2d0de · ms:401
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba35add1f68cbf6ae4ac2b845cf51eedb7f2611388cfd1fb3ef1f500a8f2d0de · ms:459
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:ba35add1f68cbf6ae4ac2b845cf51eedb7f2611388cfd1fb3ef1f500a8f2d0de · ms:334

## Mutation Log
(empty until execute)
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `cmd/mrw/main.go` · the over-long --files-from line is named as line 1 · acceptance-sha256:ba35add1f68cbf6ae4ac2b845cf51eedb7f2611388cfd1fb3ef1f500a8f2d0de · covers:the long line is named by number
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/read/read.go` · the MSYS hint names one of the two variables again · acceptance-sha256:ba35add1f68cbf6ae4ac2b845cf51eedb7f2611388cfd1fb3ef1f500a8f2d0de · covers:the hint names both variables

## Invariants

- Every read that served before this record serves the same bytes.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-074 Out of Scope)

## Stop Condition

Stop if the change needs a package the record does not govern.
