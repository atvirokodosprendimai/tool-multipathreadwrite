# Task ADR-167-T1: the plan reads `from=` as a source

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `plan.Hunk.BodyFrom`, `plan.Hunk.BodyFromAddr`, set only by `from=` and never with `CountedBody`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the accepted forms`, `the refused shapes`, `no CountedBody for a from= hunk`, `body=@ unchanged`, `the engine packages it does not own`

## Goal

Decisions 1 and 2, the parser's half: `from=PATH:ADDR` parses into a source path and a numeric address; every shape the record refuses is refused at parse, naming the fix; a `from=` hunk never sets `CountedBody`; `body=@` parses exactly as before, a colon in its path included.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `parseHeader`: the `from` key — `rooted.IsRooted` first, a value holding `:/` refused as a pattern, the last-colon split, `ParseAddr` held to `N`/`N-M`/`A,+N`/`$`/`N-$`; refused beside `body=`/`body=@`/a body line, on `delete`/`unlink`/`rename`, and as a working-set pointer; `BodyFrom`/`BodyFromAddr` on `Hunk`; `CountedBody` stays false; the empty-body deferral (`:816`) waits on `BodyFrom`; `LoadBodyFiles` skips a `BodyFrom` hunk; the unknown-option message names `from` |
| `internal/plan/fromcopy167_test.go` | add | the tests |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestAFromCopyParsesIntoASourceAndRefusesWhatItCannotCopy (table, one row each: `from=a.go:3-5`, `:7`, `:2,+3`, `:$`, `:4-$` on replace, insert-after, insert-before and create parse to `BodyFrom` "a.go", the address, `CountedBody` false and no Body; `from=a:b.txt:3` gives "a:b.txt"; refused, each naming the value: `:0`, `:-`, `:4-`, `:$-2`, `:$,+2`, `:1-2,5-6`, `:/x/`, `:/a/,/b/`, `:/a:b/`, `:$.k`, `a.go`, `C:\x`, `/x:3`, `C:\x:3`, `@1:3-5`; refused: `from=` with `body=2`, with `body=@b.txt`, with a body line under the header, with `raw=true`; on `delete`, `unlink`, `rename`; `LoadBodyFiles` leaves a `from=` hunk's Body empty and opens no file; a `create` with `from=` and validate's empty-body check: not refused at parse). TestABodyFileWhoseNameHoldsAColonStillLoads (`body=@a:3` loads the file named `a:3` through `LoadBodyFiles`). [proof: acceptance]
2. [S2] The parse branch, the fields, the deferral, the loader skip. Mutants: accept a pattern source; set `CountedBody` for a `from=` hunk; split at the first colon. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/plan/ -count=1 -timeout 300s -v -run 'TestAFromCopyParsesIntoASourceAndRefusesWhatItCannotCopy$|TestABodyFileWhoseNameHoldsAColonStillLoads$|TestBodyAtPathLoadsTheFile$' 2>&1) \
  && for n in TestAFromCopyParsesIntoASourceAndRefusesWhatItCannotCopy TestABodyFileWhoseNameHoldsAColonStillLoads TestBodyAtPathLoadsTheFile; do grep -qE "^--- PASS: $n \(" <<<"$out" || exit 1; done \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFromCopyParsesIntoASourceAndRefusesWhatItCannotCopy` | `internal/plan/fromcopy167_test.go` | each accepted form and op; each refused shape; no `CountedBody`; the loader skip | none | S1, S2 |
| `TestABodyFileWhoseNameHoldsAColonStillLoads` | `internal/plan/fromcopy167_test.go` | `body=@a:3` is still the file `a:3` | none | S1 |
| `TestBodyAtPathLoadsTheFile` | `internal/plan/bodyfile_test.go` | ADR-060's whole-file form, unchanged | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the parse branch and the two fields |
| 2 — something selects it | every plan header passes `parseHeader`; T2 consumes the fields |
| 3 — the caller can discover it | the refusals name the fix; T3 teaches the form |
| 4 — it is used | nothing yet: M's request of 2026-10-22 is the evidence for building it |

## Invariants

- `body=@` parses exactly as before.
- No hunk carries a body and `BodyFrom` together; no `BodyFrom` hunk has `CountedBody`.

## Risks

- `ParseAddr` accepts more than the subset (patterns, `-`, JSON paths): the subset is enforced after it, and the table has a row for each excluded form.

## Out of Scope

- Resolving or licensing the source (permanent: boundary: T2 does it in apply, where the ledger is)

## Stop Condition

Stop if `ParseAddr` cannot tell `$` as a start from `$` as an end without a change to `internal/addr`: name that package in the record first.

## Mutation Log
(empty until execute)
- 2026-10-22 · f862d8a3* · mutant survived · exit 0 · `internal/plan/plan.go` · a pattern source is accepted · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · covers:the refused shapes
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/plan/plan.go` · a from= hunk is counted · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · covers:no CountedBody for a from= hunk
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/plan/plan.go` · the split is at the first colon · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · covers:the accepted forms
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/plan/plan.go` · a pattern source is accepted · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · covers:the refused shapes
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/plan/plan.go` · body=@ splits at a colon · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · covers:body=@ unchanged
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/read/read.go` · internal/read changes · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · covers:the engine packages it does not own

## Verification Log
(empty until execute)
- 2026-10-22 · f862d8a3* · exit 1 · `set -o pipefail …` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:437 · test-lock-sha256:53dcb8c6610f3e4256075e3d441116c41ab7b9431ff0cfe0bdfba0763ec1c96c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5ZmlsZV90ZXN0LmdvCVRlc3RCb2R5QXRQYXRoTG9hZHNUaGVGaWxlCWEyM2FiOWViOGIzYmZhMzg0ZWJhM2UyZWE2MTI3YzEwNjE1MjRmYzc2YzQ4NTRhZmUyNDdhNTRmMGUxODAzMTkKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3Rlc3QuZ28JbG9hZHMJNzBjOTZlYjM0MWNjYzQ0MjYyM2Y5NzkwOWM3NzM4YjY0MzlkM2IyMTNjMTBjNmM4YmQwZjkyNmYwOGE2NGIyMgpib2R5CWludGVybmFsL3BsYW4vYm9keWZpbGVfdGVzdC5nbwlvdXRzaWRlCTAyZTZhYmFmY2I1NTk4NjgxMTlmYTdmM2U5N2EwOWMzYThiODcwYzg1NzBmODE2N2VkZDI3NmQwMGFhMTkzYzcKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3Rlc3QuZ28Jcm9vdGVkCTFkM2RjMWExNmMzOThmY2NhZTliMjE2OTQzNzExNTI5MmU1MGMyYmVmODkyOWUzZjc1ZmRkOTZiYTBlZDQwODMKYm9keQlpbnRlcm5hbC9wbGFuL2Zyb21jb3B5MTY3X3Rlc3QuZ28JVGVzdEFCb2R5RmlsZVdob3NlTmFtZUhvbGRzQUNvbG9uU3RpbGxMb2Fkcwk4ZTk1YTI4MzE3MTA5MGMxZTgzZWJhNTllY2EyYmYzN2M3NDNjNTJhNDllMDEwODBjZjQ5OWM3MTU3NmYyOGQzCmJvZHkJaW50ZXJuYWwvcGxhbi9mcm9tY29weTE2N190ZXN0LmdvCVRlc3RBRnJvbUNvcHlQYXJzZXNJbnRvQVNvdXJjZUFuZFJlZnVzZXNXaGF0SXRDYW5ub3RDb3B5CTFhOWI2ZmJlY2Q3ZGY5N2U0Zjk1MDA0ZmNlYjU2MGU3NDdhOWFjY2NkNmU4OTYwMzljYjEwNDBjY2QzMTllOTAKYm9keQlpbnRlcm5hbC9wbGFuL2Zyb21jb3B5MTY3X3Rlc3QuZ28JdGhlIGxvYWRlciBvcGVucyBub3RoaW5nIGZvciBhIGZyb209IGh1bmsJYjZhYzY2MDZhMmNiZDI5ZDZkNWNmNTJiZWZlZjljMjA3ZDE3ZDJiYTNlNGZmODUxMzZiZGU5Y2ZmMjU4OGY2Ng
  ```
  ```
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:1350
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:1113
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:964
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:958
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:1849
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:948
- 2026-10-22 · f862d8a3* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:0 · test-lock-sha256:192a40d175d6106089870f7f2f517148a5619da33305e28579a69f922d8ffa6f · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5ZmlsZV90ZXN0LmdvCVRlc3RCb2R5QXRQYXRoTG9hZHNUaGVGaWxlCWEyM2FiOWViOGIzYmZhMzg0ZWJhM2UyZWE2MTI3YzEwNjE1MjRmYzc2YzQ4NTRhZmUyNDdhNTRmMGUxODAzMTkKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3Rlc3QuZ28JbG9hZHMJNzBjOTZlYjM0MWNjYzQ0MjYyM2Y5NzkwOWM3NzM4YjY0MzlkM2IyMTNjMTBjNmM4YmQwZjkyNmYwOGE2NGIyMgpib2R5CWludGVybmFsL3BsYW4vYm9keWZpbGVfdGVzdC5nbwlvdXRzaWRlCTAyZTZhYmFmY2I1NTk4NjgxMTlmYTdmM2U5N2EwOWMzYThiODcwYzg1NzBmODE2N2VkZDI3NmQwMGFhMTkzYzcKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3Rlc3QuZ28Jcm9vdGVkCTFkM2RjMWExNmMzOThmY2NhZTliMjE2OTQzNzExNTI5MmU1MGMyYmVmODkyOWUzZjc1ZmRkOTZiYTBlZDQwODMKYm9keQlpbnRlcm5hbC9wbGFuL2Zyb21jb3B5MTY3X3Rlc3QuZ28JVGVzdEFCb2R5RmlsZVdob3NlTmFtZUhvbGRzQUNvbG9uU3RpbGxMb2Fkcwk4ZTk1YTI4MzE3MTA5MGMxZTgzZWJhNTllY2EyYmYzN2M3NDNjNTJhNDllMDEwODBjZjQ5OWM3MTU3NmYyOGQzCmJvZHkJaW50ZXJuYWwvcGxhbi9mcm9tY29weTE2N190ZXN0LmdvCVRlc3RBRnJvbUNvcHlQYXJzZXNJbnRvQVNvdXJjZUFuZFJlZnVzZXNXaGF0SXRDYW5ub3RDb3B5CTBjOWFhMDMzYzY4YTc2Y2M3MWJmNzE0YmU4ZDY4YjEwODczNWZlMWEwZTc2NTBhZDU1OGZjYTdiMmVjN2Y2ZTkKYm9keQlpbnRlcm5hbC9wbGFuL2Zyb21jb3B5MTY3X3Rlc3QuZ28JdGhlIGxvYWRlciBvcGVucyBub3RoaW5nIGZvciBhIGZyb209IGh1bmsJYjZhYzY2MDZhMmNiZDI5ZDZkNWNmNTJiZWZlZjljMjA3ZDE3ZDJiYTNlNGZmODUxMzZiZGU5Y2ZmMjU4OGY2Ng · test-lock-kind:replace
- 2026-10-22 · 874387ba* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:a6e92fa2644a0525eca9e0232fd1315d23346eb41ba7dbcd54ab4d385613f805 · ms:0 · test-lock-sha256:c6980bfd323f62221c2481b0f4fa0c9e8207f43f356f2b897fef1105ebfb5704 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5ZmlsZV90ZXN0LmdvCVRlc3RCb2R5QXRQYXRoTG9hZHNUaGVGaWxlCWEyM2FiOWViOGIzYmZhMzg0ZWJhM2UyZWE2MTI3YzEwNjE1MjRmYzc2YzQ4NTRhZmUyNDdhNTRmMGUxODAzMTkKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3Rlc3QuZ28JbG9hZHMJNzBjOTZlYjM0MWNjYzQ0MjYyM2Y5NzkwOWM3NzM4YjY0MzlkM2IyMTNjMTBjNmM4YmQwZjkyNmYwOGE2NGIyMgpib2R5CWludGVybmFsL3BsYW4vYm9keWZpbGVfdGVzdC5nbwlvdXRzaWRlCTAyZTZhYmFmY2I1NTk4NjgxMTlmYTdmM2U5N2EwOWMzYThiODcwYzg1NzBmODE2N2VkZDI3NmQwMGFhMTkzYzcKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3Rlc3QuZ28Jcm9vdGVkCTFkM2RjMWExNmMzOThmY2NhZTliMjE2OTQzNzExNTI5MmU1MGMyYmVmODkyOWUzZjc1ZmRkOTZiYTBlZDQwODMKYm9keQlpbnRlcm5hbC9wbGFuL2Zyb21jb3B5MTY3X3Rlc3QuZ28JVGVzdEFCb2R5RmlsZVdob3NlTmFtZUhvbGRzQUNvbG9uU3RpbGxMb2Fkcwk4ZTk1YTI4MzE3MTA5MGMxZTgzZWJhNTllY2EyYmYzN2M3NDNjNTJhNDllMDEwODBjZjQ5OWM3MTU3NmYyOGQzCmJvZHkJaW50ZXJuYWwvcGxhbi9mcm9tY29weTE2N190ZXN0LmdvCVRlc3RBRnJvbUNvcHlQYXJzZXNJbnRvQVNvdXJjZUFuZFJlZnVzZXNXaGF0SXRDYW5ub3RDb3B5CTdiNzg1NmM0M2ZkNTRkMGY4ZGY4NzkyNzJlZjZmZjViOTJlOTI1YjgyNmQ5YTcwNTNmMzVmMTBiMDQzNGRkMjQKYm9keQlpbnRlcm5hbC9wbGFuL2Zyb21jb3B5MTY3X3Rlc3QuZ28JYSBvbmUtbGV0dGVyIHNvdXJjZSBpcyBhIGZpbGUsIG5vdCBhIGRyaXZlCTIxNzdkMzEzNGUxM2I0ZDhiZGIzNjUxMTBhZjY0NmJkMGI4Yzc3OGRmZGU5YzljZGEwOWRjYzcwZWQxZjg4OTgKYm9keQlpbnRlcm5hbC9wbGFuL2Zyb21jb3B5MTY3X3Rlc3QuZ28JdGhlIGxvYWRlciBvcGVucyBub3RoaW5nIGZvciBhIGZyb209IGh1bmsJYjZhYzY2MDZhMmNiZDI5ZDZkNWNmNTJiZWZlZjljMjA3ZDE3ZDJiYTNlNGZmODUxMzZiZGU5Y2ZmMjU4OGY2Ng · test-lock-kind:replace
