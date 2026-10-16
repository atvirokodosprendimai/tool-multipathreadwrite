# Task ADR-148-T1: the null-device read names what is still licensed, and the docs stop recommending an unread file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `stillLicensed` in `cmd/mrw/nullread148.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a licensed file is named`, `an unlicensed or changed file is not named`, `the call site prints it`, `the docs stop recommending an unread file`, `spans reaching the last line license a whole-file read`, `another spelling of the file is found`, `the recount is bounded`

## Goal

Decisions 1–3 of the record, with a test that fails before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/nullread148.go` | add | `stillLicensed` and `licensedBy` |
| `cmd/mrw/main.go` | edit | the read action prints the line after ADR-133's |
| `cmd/mrw/nullread148_test.go` | add | the test |
| `scripts/contract.sh` | edit | §251 drives the built binary |
| `AGENTS.md`, `README.md` | edit | the read-before-write rule |

## Ordered Steps

1. [S1] Write the failing test `TestANullDeviceReadNamesWhatIsStillLicensed` and see it red: a file read to a file earlier, then read with another never read, both to the null device — stderr names the first and not the second, with the count; a write to the first applies and to the second is refused; after the first file's content changes, a null-device read names nothing; a file read on line 1 only, then read on 1-2 to the null device, is not named.
2. [S2] `stillLicensed` and its call after ADR-133's line. [proof: mutation]
3. [S3] AGENTS.md and README. [proof: mutation]
4. [S4] Contract §251: the named file's write applies, the unnamed file's is refused, through the built binary. The fence greps only its heading, and adr-verify refuses a heading mutant as comment-only. [proof: human: the executor runs ./scripts/contract.sh once and reads the three §251 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestANullDeviceReadNamesWhatIsStillLicensed|TestAReadSentToTheNullDeviceLicensesNothing|TestTheNullDeviceLineFollowsOnlyServedLines' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestANullDeviceReadNamesWhatIsStillLicensed \(' "$out" \
  && grep -qE '^--- PASS: TestAReadSentToTheNullDeviceLicensesNothing \(' "$out" \
  && grep -qE '^--- PASS: TestTheNullDeviceLineFollowsOnlyServedLines \(' "$out" \
  && grep -q '^# 251\. ' scripts/contract.sh \
  && ! grep -q 'Send it to a file, or read it, if you mean to edit' AGENTS.md \
  && grep -q 'a file you never open licenses all the same' AGENTS.md \
  && grep -q 'still licensed by an earlier read or write' README.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANullDeviceReadNamesWhatIsStillLicensed` | `cmd/mrw/nullread148_test.go` | a licensed file is named and its write applies; an unlicensed, a changed and a partly covered file are not named and their writes are refused; a whole-file read covered by earlier spans, and a file read through an in-root link, are named and their writes apply (the Codex review of #397); a file over the recount cap is not read whole to be named (its second review) | none | S1, S2 |
| `TestAReadSentToTheNullDeviceLicensesNothing` | `cmd/mrw/nullread133_test.go` | ADR-133 holds: nothing recorded, the line said | none | — |
| `TestTheNullDeviceLineFollowsOnlyServedLines` | `cmd/mrw/stat133_test.go` | ADR-133's line follows only a read that served lines | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `stillLicensed` |
| 2 — something selects it | the read action's null-device branch; contract §251 drives the built binary |
| 3 — the caller can discover it | the stderr line itself; AGENTS.md and README |
| 4 — it is used | nothing measures this; built because a peer session reported the confusion on 2026-10-13 |

## Mutation Log
(empty until execute)
- 2026-10-13 · afe2196* · mutant killed · exit 1 · `cmd/mrw/nullread148.go` · a file changed since its licence is named · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:an unlicensed or changed file is not named
- 2026-10-13 · afe2196* · mutant killed · exit 1 · `cmd/mrw/nullread148.go` · a licensed file is never named · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:a licensed file is named
- 2026-10-13 · afe2196* · mutant killed · exit 1 · `cmd/mrw/main.go` · the read action stops printing the line · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:the call site prints it
- 2026-10-13 · afe2196* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md recommends an unread file again · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:the docs stop recommending an unread file
- 2026-10-13 · ba6604f* · mutant inconclusive · exit 1 · `cmd/mrw/nullread148.go` · a whole-file read covered by earlier spans is never named · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:spans reaching the last line license a whole-file read
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-13 · ba6604f* · mutant killed · exit 1 · `cmd/mrw/nullread148.go` · a file licensed under another spelling is not found · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:another spelling of the file is found
- 2026-10-13 · ba6604f* · mutant killed · exit 1 · `cmd/mrw/nullread148.go` · a whole-file read covered by earlier spans is never named · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:spans reaching the last line license a whole-file read
- 2026-10-13 · 8ab5aee* · mutant killed · exit 1 · `cmd/mrw/nullread148.go` · the recount reads a file whole whatever its size · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · covers:the recount is bounded

## Verification Log
(empty until execute)
- 2026-10-13 · afe2196* · exit 1 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1676 · test-lock-sha256:fbe30282d28fce6eab09b4d03a05a1d092063f243b53109e9a1ad1bd279905ac · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9udWxscmVhZDEzM190ZXN0LmdvCVRlc3RBUmVhZFNlbnRUb1RoZU51bGxEZXZpY2VMaWNlbnNlc05vdGhpbmcJMDBlNmY1NzM3NTE1YzE3ODFhNmU2N2Q2ZTFhZmRhYzU0ZDE2YzY2NGYwZTMwOWQ3NDBkMWIzNDg1MmEwZDczMwpib2R5CWNtZC9tcncvbnVsbHJlYWQxNDhfdGVzdC5nbwlUZXN0QU51bGxEZXZpY2VSZWFkTmFtZXNXaGF0SXNTdGlsbExpY2Vuc2VkCTQxYmZiMjIxN2YxNTc1MTU0NDRlYzVkYWVlYTc5YWRiYjA4ZDhiOTNjNGZiNjExMzY5MGUxMDFjMWZhZWE3YmUKYm9keQljbWQvbXJ3L3N0YXQxMzNfdGVzdC5nbwlUZXN0VGhlTnVsbERldmljZUxpbmVGb2xsb3dzT25seVNlcnZlZExpbmVzCWYxZTJmZmU3OWIyOTNmYTRlNmJhN2JiMjY4MTZmYjllNDVmMDBkZmIxYTZlODMyNmEyNWM4NzhmOGFiZDA5NzA
  ```
  --- last 10 line(s) of stdout
  === RUN   TestAReadSentToTheNullDeviceLicensesNothing
  --- PASS: TestAReadSentToTheNullDeviceLicensesNothing (0.03s)
  === RUN   TestANullDeviceReadNamesWhatIsStillLicensed
      nullread148_test.go:90: the file an earlier read still licenses is not named with the count: "mrw: the answer went to the null device, so nothing was recorded: a write to these lines needs a read whose answer you see; on unix a stdout closed when mrw starts is the null device too (ADR-133)\n"
  --- FAIL: TestANullDeviceReadNamesWhatIsStillLicensed (0.03s)
  === RUN   TestTheNullDeviceLineFollowsOnlyServedLines
  --- PASS: TestTheNullDeviceLineFollowsOnlyServedLines (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.273s
  FAIL
  ```
- 2026-10-13 · afe2196* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1421
- 2026-10-13 · afe2196* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1217
- 2026-10-13 · afe2196* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1205
- 2026-10-13 · afe2196* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1225
- 2026-10-13 · human-observed · observed by the executor 2026-10-13: S4 ./scripts/contract.sh exit 0 ('contract holds'), §251 rows 'naming the file an earlier read still licenses, and only it', 'the named file's write applies', 'the pair: the unnamed file's write is refused' all PASS
- 2026-10-13 · human-observed · observed and approved by the executor 2026-10-13: S4 ./scripts/contract.sh exit 0 ('contract holds'); all three §251 rows PASS — the named file is named alone with its count, its write lands, and the pair row for the unnamed file passes
- 2026-10-13 · ba6604f* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1298
- 2026-10-13 · ba6604f* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1222
- 2026-10-13 · ba6604f* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1405
- 2026-10-13 · ba6604f* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:0 · test-lock-sha256:15f1f361b52bd8e934be7bcce3e13e77b96e0813a4dbc82656e640b3a1c4e512 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9udWxscmVhZDEzM190ZXN0LmdvCVRlc3RBUmVhZFNlbnRUb1RoZU51bGxEZXZpY2VMaWNlbnNlc05vdGhpbmcJMDBlNmY1NzM3NTE1YzE3ODFhNmU2N2Q2ZTFhZmRhYzU0ZDE2YzY2NGYwZTMwOWQ3NDBkMWIzNDg1MmEwZDczMwpib2R5CWNtZC9tcncvbnVsbHJlYWQxNDhfdGVzdC5nbwlUZXN0QU51bGxEZXZpY2VSZWFkTmFtZXNXaGF0SXNTdGlsbExpY2Vuc2VkCWJkYWUzNzVmOWVhM2QxOGExNDQxNWFlYTBjODI2M2Q5YWNlMjhiMjU0ODZlNjk1NmFkZGI5ZTQ4NDkxMDZlYzkKYm9keQljbWQvbXJ3L3N0YXQxMzNfdGVzdC5nbwlUZXN0VGhlTnVsbERldmljZUxpbmVGb2xsb3dzT25seVNlcnZlZExpbmVzCWYxZTJmZmU3OWIyOTNmYTRlNmJhN2JiMjY4MTZmYjllNDVmMDBkZmIxYTZlODMyNmEyNWM4NzhmOGFiZDA5NzA · test-lock-kind:replace
- 2026-10-13 · 8ab5aee* · exit 0 · `set -o pipefail …` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:1450
- 2026-10-13 · 8ab5aee* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:0 · test-lock-sha256:e1ac03aee03c5e48e86312c7c266ca3a586fdeb62c21d809bac9f30d0f2aecac · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9udWxscmVhZDEzM190ZXN0LmdvCVRlc3RBUmVhZFNlbnRUb1RoZU51bGxEZXZpY2VMaWNlbnNlc05vdGhpbmcJMDBlNmY1NzM3NTE1YzE3ODFhNmU2N2Q2ZTFhZmRhYzU0ZDE2YzY2NGYwZTMwOWQ3NDBkMWIzNDg1MmEwZDczMwpib2R5CWNtZC9tcncvbnVsbHJlYWQxNDhfdGVzdC5nbwlUZXN0QU51bGxEZXZpY2VSZWFkTmFtZXNXaGF0SXNTdGlsbExpY2Vuc2VkCWRkY2IwMjk0Zjg2YTVkMDBhYTIzM2RjMGM4YzI2YzQyYzZjNDllZTA5YzNlZmVkYjIyNTEzZmZkM2Y4OTI1NjgKYm9keQljbWQvbXJ3L3N0YXQxMzNfdGVzdC5nbwlUZXN0VGhlTnVsbERldmljZUxpbmVGb2xsb3dzT25seVNlcnZlZExpbmVzCWYxZTJmZmU3OWIyOTNmYTRlNmJhN2JiMjY4MTZmYjllNDVmMDBkZmIxYTZlODMyNmEyNWM4NzhmOGFiZDA5NzA · test-lock-kind:replace
- 2026-10-16 · c8106825* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:63fc3a019827d66ac99097fefdeb0134c0690f1a3721ffc83be495c9f9040c56 · ms:0 · test-lock-sha256:2f8d54ce2e6816620a319afc080da64ca49f05ed86a3af6a89fd5bbef35ed7e5 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9udWxscmVhZDEzM190ZXN0LmdvCVRlc3RBUmVhZFNlbnRUb1RoZU51bGxEZXZpY2VMaWNlbnNlc05vdGhpbmcJMDBlNmY1NzM3NTE1YzE3ODFhNmU2N2Q2ZTFhZmRhYzU0ZDE2YzY2NGYwZTMwOWQ3NDBkMWIzNDg1MmEwZDczMwpib2R5CWNtZC9tcncvbnVsbHJlYWQxNDhfdGVzdC5nbwlUZXN0QU51bGxEZXZpY2VSZWFkTmFtZXNXaGF0SXNTdGlsbExpY2Vuc2VkCTBlMDU4YjAxMjhkY2YxOGJlNTQ0M2I3NmIwYzE4NzJlYTVhN2VhMmNmMjY5OGM3YmZhMWRlMDg3ZDZiNWJjZmIKYm9keQljbWQvbXJ3L3N0YXQxMzNfdGVzdC5nbwlUZXN0VGhlTnVsbERldmljZUxpbmVGb2xsb3dzT25seVNlcnZlZExpbmVzCWYxZTJmZmU3OWIyOTNmYTRlNmJhN2JiMjY4MTZmYjllNDVmMDBkZmIxYTZlODMyNmEyNWM4NzhmOGFiZDA5NzA · test-lock-kind:replace

## Invariants

- ADR-133 holds: a null-device read records nothing, keeps its line and its exit code.
- Nothing under `internal/` changes.

## Risks

- None beyond the record's.

## Out of Scope

- The MCP surface (permanent: boundary: it licenses through acks, ADR-031)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
