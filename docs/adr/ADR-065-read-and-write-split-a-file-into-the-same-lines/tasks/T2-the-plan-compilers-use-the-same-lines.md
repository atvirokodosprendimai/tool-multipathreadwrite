# Task ADR-065-T2: apply_patch and search_replace compile against the same lines; contract §121

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the plan compilers read targets through `lines.Split`; §121
**Consumes:** `lines.Split` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `apply_patch edits a CRLF file`, `search_replace edits a CR-only file`, `the binary applies both`

## Goal

`fileLines` (`internal/ingest/applypatch.go:269-286`) and `searchFileLines` (`internal/ingest/searchreplace.go:126-143`) take the target's lines from `lines.Split`. A patch whose context is an LF-normalised document (ADR-051 F-9) then finds its lines in a CRLF or CR-only target and compiles to the addresses the write engine uses. Today the same edit is refused with "matched no lines", exit 2 (probed 2026-09-24 on v1.22.3).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/applypatch.go` | edit | `fileLines` uses `lines.Split` |
| `internal/ingest/searchreplace.go` | edit | `searchFileLines` uses `lines.Split` |
| `internal/ingest/applypatch_test.go` | edit | `TestCompileApplyPatchRules`'s F-10 half now asserts an LF patch compiles against a CR LF file (ADR-051 F-10 superseded) |
| `docs/adr/ADR-051-foreign-plan-grammars-compile-to-plan-hunks/apply-patch-ingest.md` | edit | F-10 and its risk row marked superseded by ADR-065 |
| `internal/ingest/lines_agree_test.go` | new | two tests below |
| `scripts/contract.sh` | edit | §121 |

## Ordered Steps

1. [S1] Write `TestApplyPatchEditsACRLFFile` and `TestSearchReplaceEditsACROnlyFile`, and confirm both RED on T1's tree: each compile returns "matched no lines". [proof: mutation]
2. [S2] Rewire the two helpers; confirm GREEN, and that the existing `internal/ingest` tests stay green. [proof: mutation]
   Mutants, one per helper: revert it to breaking at `"\n"`, which kills its test.
3. [S3] §121 (next after §120). [proof: mutation]
   - Good: `read crlf.txt`, then an `--format=apply_patch` edit of `two` exits 0, and the file is `one\r\nTWO\r\nthree\r\n`. `read cr.txt`, then a `--format=search_replace` edit of `two` exits 0, and the file is `one\rTWO\rthree\r`.
   - Must-fail: the apply_patch document whose old side names a line the file lacks exits 2, with the file unchanged.
   - Confirm RED against v1.22.3 (both good cases exit 2 there), then GREEN.
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 121\. ' scripts/contract.sh \
  && go test ./internal/ingest/ -count=1 -v 2>&1 | tee /tmp/adr065-t2.out \
  && grep -q '^--- PASS: TestApplyPatchEditsACRLFFile ' /tmp/adr065-t2.out \
  && grep -q '^--- PASS: TestSearchReplaceEditsACROnlyFile ' /tmp/adr065-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr065-t2.out \
  && ./scripts/contract.sh > /tmp/adr065-t2-contract.out 2>&1 \
  && grep -q '^  PASS  apply_patch edits a CRLF file' /tmp/adr065-t2-contract.out \
  && grep -q '^  PASS  search_replace edits a CR-only file' /tmp/adr065-t2-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/seen internal/check internal/state internal/guide \
  && [ -z "$(gofmt -l internal/ingest)" ] \
  && go vet ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestApplyPatchEditsACRLFFile` | `internal/ingest/lines_agree_test.go` | an LF patch compiles against a CRLF target | — | S1, S2 |
| `TestSearchReplaceEditsACROnlyFile` | `internal/ingest/lines_agree_test.go` | a SEARCH block compiles against a CR-only target | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §121 |
| 2 — something selects it | `mrw write --format=apply_patch` and `--format=search_replace` call the two helpers; reverting either fails its test |
| 3 — the caller can discover it | the compiled plan applies instead of "matched no lines" |
| 4 — it is used | found while enumerating ADR-065's class; ADR-009 refuses telemetry |

## Mutation Log
(empty until execute)
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · apply_patch reads the target at \n alone again: TestApplyPatchEditsACRLFFile and §121 must go red · acceptance-sha256:15dd3d3d58db94e35ca1904afa26be4b859734f4671e6344c60c6d2c94df3ba1 · covers:apply_patch edits a CRLF file
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/ingest/searchreplace.go` · search_replace reads the target at \n alone again: TestSearchReplaceEditsACROnlyFile and §121 must go red · acceptance-sha256:15dd3d3d58db94e35ca1904afa26be4b859734f4671e6344c60c6d2c94df3ba1 · covers:search_replace edits a CR-only file

## Invariants

- The document is still normalised to LF before parsing (ADR-051 F-9).
- An LF target compiles exactly as before.
- No `§NN` row in the Tests table.

## Risks

- A mixed-ending target keeps its stray `\r` in a line, so a patch line without it does not match; that is unchanged, and the refusal still names it.

## Stop Condition

- If going green needs a change to the document parsers (`applypatch.go:64`, `searchreplace.go:24`), stop.

## Out of Scope

- Normalising a mixed-ending target (permanent: boundary: ADR-005 §3 keeps a stray `\r` as content so untouched lines stay byte-identical)

## Verification Log
(empty until execute)
- 2026-09-24 · cd91659* · exit 1 · `set -o pipefail …` · acceptance-sha256:15dd3d3d58db94e35ca1904afa26be4b859734f4671e6344c60c6d2c94df3ba1 · ms:293 · test-lock-sha256:3c57a93389ba7cac4b891f8e93eefb0aa97cf90b2640546f893eee2b8c7402af · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2luZ2VzdC9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RBcHBseVBhdGNoRWRpdHNBQ1JMRkZpbGUJMDQzNDE0NjZjYTE4ODU5YmVmNjQwMTc2MjgwZWE2NDc2ZmMyZTliZTdkYjliMDZmOTc0MjExMzMwMjY2Yjc3YQpib2R5CWludGVybmFsL2luZ2VzdC9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RTZWFyY2hSZXBsYWNlRWRpdHNBQ1JPbmx5RmlsZQkwM2I2MTI3MjhiMTVkODJiNWYyMzdlMjVjN2JjMDBlOTk4ZTlkZWJjMTJiZTNhNDNkNjE4NDc1ZmM5M2IyOWUx
  ```
  --- last 10 line(s) of stdout (of 61 after folding 61 raw)
  --- PASS: TestANearMissSearchIsACompileRefusal (0.00s)
  === RUN   TestAnAmbiguousSearchIsACompileRefusal
  --- PASS: TestAnAmbiguousSearchIsACompileRefusal (0.00s)
  === RUN   TestEmptySearchOnExistingFileIsACompileRefusal
  --- PASS: TestEmptySearchOnExistingFileIsACompileRefusal (0.00s)
  === RUN   TestEmptySearchOnMissingFileIsCreate
  --- PASS: TestEmptySearchOnMissingFileIsCreate (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest	0.013s
  FAIL
  ```
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:15dd3d3d58db94e35ca1904afa26be4b859734f4671e6344c60c6d2c94df3ba1 · ms:29726
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:15dd3d3d58db94e35ca1904afa26be4b859734f4671e6344c60c6d2c94df3ba1 · ms:36739
- 2026-09-24 · cd91659* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:15dd3d3d58db94e35ca1904afa26be4b859734f4671e6344c60c6d2c94df3ba1 · ms:0 · test-lock-sha256:171b614f8ec47cd0c4b1942a34f3f74071a5c0d89e1b02554d1da906086a4ab8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2luZ2VzdC9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RBcHBseVBhdGNoRWRpdHNBQ1JMRkZpbGUJNDkzMTFjNDRhMjY0ZTk1MzcwNDg5MTFlN2JkNGI3MTdiZjYzNjk5Y2EzZDg1ODg5ZDdiYjFlZjAzODJiNWZjNwpib2R5CWludGVybmFsL2luZ2VzdC9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RTZWFyY2hSZXBsYWNlRWRpdHNBQ1JPbmx5RmlsZQkwM2I2MTI3MjhiMTVkODJiNWYyMzdlMjVjN2JjMDBlOTk4ZTlkZWJjMTJiZTNhNDNkNjE4NDc1ZmM5M2IyOWUx · test-lock-kind:replace
