# Task ADR-129-T1: a case-only rename applies; a hard link and a directory-only respelling stay refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `respelling` (the Decision 1 test) in `internal/apply/pathop.go`
**Consumes:** none
**Data dependency:** hermetic — the respelling tests need a filesystem that folds case and skip, saying so, where it does not; the Windows CI shard and a local macOS run are where they run
**Proof map:** v1
**Rests-on:** `a case-only rename applies; a hard link and a directory-only respelling stay refused`

## Goal

`@@ a.txt - rename` / `A.txt` applies on a folding filesystem and the listing shows `A.txt`; every other existing destination is refused as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/pathop.go` | edit | `respelling`; `planPathOp` allows it; `renameOne` commits it and checks the listing |
| `internal/apply/respell129_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §230 |
| `AGENTS.md` | edit | §2's rename sentence |

## Ordered Steps

1. [S1] Write `TestACaseOnlyRenameApplies`, `TestAHardLinkUnderAnotherNameIsStillRefused`, `TestARespellingOfOnlyTheDirectoryIsRefused`, `TestARespellingTheFilesystemKeptIsUndone`. Confirm RED on this Mac. [proof: mutation]
2. [S2] `respelling(full, destFull)` per Decision 1; `planPathOp` allows a respelling and names an `otherSpelling` source; `renameOne` re-asks Decision 1 at commit, and once the rename is recorded requires the source's spelling to have left the listing. Then the review's three: the same-file count, the source-listed test, and the mixed-content report. Mutants: the same-file count dropped (a hard link under another spelling renames); the source-listed test dropped (a source spelled otherwise is renamed); the post-commit check dropped (the kept spelling reports ok). [proof: mutation]
3. [S3] Contract §230 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestACaseOnlyRenameApplies|TestAHardLinkUnderAnotherNameIsStillRefused|TestARespellingOfOnlyTheDirectoryIsRefused|TestARespellingTheFilesystemKeptIsUndone|TestAHardLinkUnderAnotherSpellingIsRefused|TestASourceNamedInAnotherSpellingIsRefused|TestAKeptRespellingAfterAContentEditIsReportedPartial|TestARespellingThatAlsoRespellsTheDirectoryIsRefused|TestAnEditedFileWhoseRespellingWasKeptIsNotReportedApplied' -v 2>&1 | tee "$out" \
  && grep -qE '^--- (PASS|SKIP): TestACaseOnlyRenameApplies \(' "$out" \
  && grep -qE '^--- PASS: TestAHardLinkUnderAnotherNameIsStillRefused \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestARespellingOfOnlyTheDirectoryIsRefused \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestARespellingTheFilesystemKeptIsUndone \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestAHardLinkUnderAnotherSpellingIsRefused \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestASourceNamedInAnotherSpellingIsRefused \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestAKeptRespellingAfterAContentEditIsReportedPartial \(' "$out" \
  && go test ./internal/apply/ ./internal/adversarial/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 230\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted internal/mcp \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACaseOnlyRenameApplies` | `internal/apply/respell129_test.go` | on a folding filesystem `a.txt` → `A.txt` applies and the listing holds `A.txt` only | none | S1, S2 |
| `TestAHardLinkUnderAnotherNameIsStillRefused` | `internal/apply/respell129_test.go` | a rename onto a hard link of the source is refused "already exists", nothing renamed | none | S1, S2 |
| `TestARespellingOfOnlyTheDirectoryIsRefused` | `internal/apply/respell129_test.go` | `d/a.txt` → `D/a.txt` on a folding filesystem is refused as the source | none | S1, S2 |
| `TestARespellingTheFilesystemKeptIsUndone` | `internal/apply/respell129_test.go` | a commit rename that succeeds and keeps the old spelling fails the hunk and writes nothing | none | S1, S2 |
| `TestAHardLinkUnderAnotherSpellingIsRefused` | `internal/apply/respell129_test.go` | a hard link `B.txt` to the source, with the plan asking `b.txt`, is refused "already exists" (the Codex review of the record) | none | S2 |
| `TestASourceNamedInAnotherSpellingIsRefused` | `internal/apply/respell129_test.go` | a source the plan spells otherwise than its directory is refused, so no undo restores a name it never had (the Codex review of the record) | none | S2 |
| `TestAKeptRespellingAfterAContentEditIsReportedPartial` | `internal/apply/respell129_test.go` | a kept respelling after a content edit landed reports the edit written and undoes the rename (the Codex review of the record) | none | S2 |
| `TestARespellingThatAlsoRespellsTheDirectoryIsRefused` | `internal/apply/respell129_test.go` | `a/x.txt` → `A/X.txt` is refused, not renamed to `a/X.txt` under a receipt naming `A/X.txt` (the in-process review of #346) | none | S2 |
| `TestAnEditedFileWhoseRespellingWasKeptIsNotReportedApplied` | `internal/apply/respell129_test.go` | the edited-and-renamed return path runs the post-commit check too (the in-process review of #346) | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `respelling` and the commit check |
| 2 — something selects it | `planPathOp` for every rename hunk; `renameOne` for every rename |
| 3 — the caller can discover it | the receipt; AGENTS.md §2 |
| 4 — it is used | BACKLOG "From the Windows peers"; no telemetry (ADR-009) |

## Invariants

- Where two spellings name two files, every rename refuses and applies exactly as before.
- No exit code and no receipt key changes.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the Windows shard shows `os.Root.Rename` refusing a case-only rename: the intermediate-name fallback then needs its own undo step.

## Out of Scope

- A respelling of a directory component (permanent: boundary: the record's)

## Mutation Log
- 2026-10-06 · 135eb2e* · mutant killed · exit 1 · `internal/apply/pathop.go` · S2: the same-file count dropped — a hard link under another spelling renames · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666
- 2026-10-06 · 135eb2e* · mutant killed · exit 1 · `internal/apply/pathop.go` · S2: the source-listed test dropped — a source spelled otherwise is renamed · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666
- 2026-10-06 · 135eb2e* · mutant inconclusive · exit 1 · `internal/apply/pathop.go` · S2: the post-commit check dropped — a kept spelling reports ok · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-06 · 135eb2e* · mutant killed · exit 1 · `internal/apply/pathop.go` · S2: the post-commit check dropped — a kept spelling reports ok · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666
- 2026-10-06 · 6d5dbe8* · mutant killed · exit 1 · `internal/apply/pathop.go` · S2: the edited-and-renamed path skips confirm() — a kept respelling of an edited file reports applied (the reviews of #346) · acceptance-sha256:bfa46bd7bd99113ddcdf672a469244c84e307166ccd41ff4e24eb579865baaca

## Verification Log
- 2026-10-06 · 135eb2e* · exit 1 · `set -o pipefail …` · acceptance-sha256:e82cc1f4b57a9ce640fcfca4336fe63e1d44e470f29b653d057ae713f711a41f · ms:356 · test-lock-sha256:27d8364763b26617a46acf767087968027f0abe30a1e5831ec86d09dc1df18aa · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVzcGVsbDEyOV90ZXN0LmdvCVRlc3RBQ2FzZU9ubHlSZW5hbWVBcHBsaWVzCWMxNDcxYWI5NDIyMjI1MjVkMTJjNjhjZDYxMmFmNmQ3NTQ2NWRjZDg1Mzg5MDZiNDI5YTY0Nzk3Y2FkOWM1ZDAKYm9keQlpbnRlcm5hbC9hcHBseS9yZXNwZWxsMTI5X3Rlc3QuZ28JVGVzdEFIYXJkTGlua1VuZGVyQW5vdGhlck5hbWVJc1N0aWxsUmVmdXNlZAk3ZTAzZjM0ODA1MzM4MGM4M2Q0ZjdhOGMxMTE3ZGUzNDNmNmY4MzlmYjg3ZTY2ZGVjNmM4YTBkZWE4ZThiNjM1CmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVzcGVsbDEyOV90ZXN0LmdvCVRlc3RBUmVzcGVsbGluZ09mT25seVRoZURpcmVjdG9yeUlzUmVmdXNlZAliMzNjY2U1N2YzYjNjOWY5OWQ4MDYxODMxZDUzNTRmNmU0NTA5OTkyYjIyZDgxMTkzOWQ0NGZhMmU4M2NhODhhCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVzcGVsbDEyOV90ZXN0LmdvCVRlc3RBUmVzcGVsbGluZ1RoZUZpbGVzeXN0ZW1LZXB0SXNVbmRvbmUJMzMyOWVhYWMwNGI5ZjMyYzk0MDAzMGE2ZDE2YjFiZTM2ZmNjZDBhYzc2MDJkMzQwYzQ3NDQwMTJiODJlYzNlNA
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  --- PASS: TestAHardLinkUnderAnotherNameIsStillRefused (0.00s)
  === RUN   TestARespellingOfOnlyTheDirectoryIsRefused
      respell129_test.go:93: a directory-only respelling was not refused as the source: applied=false "rename dest D/a.txt already exists — unlink it in this plan, or pick another path"
  --- FAIL: TestARespellingOfOnlyTheDirectoryIsRefused (0.00s)
  === RUN   TestARespellingTheFilesystemKeptIsUndone
      respell129_test.go:114: a respelling the filesystem kept reported success: err=<nil> applied=false
  --- FAIL: TestARespellingTheFilesystemKeptIsUndone (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.079s
  FAIL
  ```
- 2026-10-06 · 135eb2e* · exit 0 · `set -o pipefail …` · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666 · ms:42200
- 2026-10-06 · 135eb2e* · exit 0 · `set -o pipefail …` · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666 · ms:38012
- 2026-10-06 · 135eb2e* · exit 0 · `set -o pipefail …` · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666 · ms:36292
- 2026-10-06 · 135eb2e* · exit 0 · `set -o pipefail …` · acceptance-sha256:c7269dd7c0dcee41bf3687b65083069f4191ed9c0fd91e44df053b7fd2fca666 · ms:37713
- 2026-10-06 · 6d5dbe8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:bfa46bd7bd99113ddcdf672a469244c84e307166ccd41ff4e24eb579865baaca · ms:0 · test-lock-sha256:35179a1d67e9b2b2f2ca62a6df494761a1d317e838063bdbc9bf641cd9060e32 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVzcGVsbDEyOV90ZXN0LmdvCVRlc3RBQ2FzZU9ubHlSZW5hbWVBcHBsaWVzCWMxNDcxYWI5NDIyMjI1MjVkMTJjNjhjZDYxMmFmNmQ3NTQ2NWRjZDg1Mzg5MDZiNDI5YTY0Nzk3Y2FkOWM1ZDAKYm9keQlpbnRlcm5hbC9hcHBseS9yZXNwZWxsMTI5X3Rlc3QuZ28JVGVzdEFIYXJkTGlua1VuZGVyQW5vdGhlck5hbWVJc1N0aWxsUmVmdXNlZAk3ZTAzZjM0ODA1MzM4MGM4M2Q0ZjdhOGMxMTE3ZGUzNDNmNmY4MzlmYjg3ZTY2ZGVjNmM4YTBkZWE4ZThiNjM1CmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVzcGVsbDEyOV90ZXN0LmdvCVRlc3RBSGFyZExpbmtVbmRlckFub3RoZXJTcGVsbGluZ0lzUmVmdXNlZAk4MDI4MmYxYWZkYzVmMjg1MGQ4MjEyNDc1MDY1NDZlZjlhNzY2ZjRiNTVlOWJlNTNmNTlhNTJjM2QzNGE4OTYzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVzcGVsbDEyOV90ZXN0LmdvCVRlc3RBS2VwdFJlc3BlbGxpbmdBZnRlckFDb250ZW50RWRpdElzUmVwb3J0ZWRQYXJ0aWFsCWQwNGZiYTRkMDlmMDFiYjY5YWM3ODY5MzJhOTBkOGFhYjNmOTA4NDA1MDk4ODEyYjM5N2RmZmY1NzQ0NDQ5NjkKYm9keQlpbnRlcm5hbC9hcHBseS9yZXNwZWxsMTI5X3Rlc3QuZ28JVGVzdEFSZXNwZWxsaW5nT2ZPbmx5VGhlRGlyZWN0b3J5SXNSZWZ1c2VkCWIzM2NjZTU3ZjNiM2M5Zjk5ZDgwNjE4MzFkNTM1NGY2ZTQ1MDk5OTJiMjJkODExOTM5ZDQ0ZmEyZTgzY2E4OGEKYm9keQlpbnRlcm5hbC9hcHBseS9yZXNwZWxsMTI5X3Rlc3QuZ28JVGVzdEFSZXNwZWxsaW5nVGhhdEFsc29SZXNwZWxsc1RoZURpcmVjdG9yeUlzUmVmdXNlZAliYzFhOGQwZThmZGMwYzA4MTNiYzdlNzVhYzQ5YjllZjcyMjFhMTIxZGE3OTI2MGM2NzE5Y2MyNGMxODVkZGU3CmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVzcGVsbDEyOV90ZXN0LmdvCVRlc3RBUmVzcGVsbGluZ1RoZUZpbGVzeXN0ZW1LZXB0SXNVbmRvbmUJMzMyOWVhYWMwNGI5ZjMyYzk0MDAzMGE2ZDE2YjFiZTM2ZmNjZDBhYzc2MDJkMzQwYzQ3NDQwMTJiODJlYzNlNApib2R5CWludGVybmFsL2FwcGx5L3Jlc3BlbGwxMjlfdGVzdC5nbwlUZXN0QVNvdXJjZU5hbWVkSW5Bbm90aGVyU3BlbGxpbmdJc1JlZnVzZWQJNWRjNDQyZjRiYjQ2MzM4NzUxNDZhYjIyZTBmYzkxMDEyMjQ0NmNmNzk3YjdlODMxYTI4MmNiNDU5Yjc1Mzg3MApib2R5CWludGVybmFsL2FwcGx5L3Jlc3BlbGwxMjlfdGVzdC5nbwlUZXN0QW5FZGl0ZWRGaWxlV2hvc2VSZXNwZWxsaW5nV2FzS2VwdElzTm90UmVwb3J0ZWRBcHBsaWVkCTg3ZmU1ZmI4MDVlMTQwZmNiZjZkYWVjMDZhNGFjYWMyNzk0M2JjZjIwOTBlMjNiMmUyNmZlYzIxOTgzNWRlM2Y · test-lock-kind:replace
- 2026-10-06 · 6d5dbe8* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfa46bd7bd99113ddcdf672a469244c84e307166ccd41ff4e24eb579865baaca · ms:36648
