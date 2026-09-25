# Task ADR-071-T5: The prose says what the code does

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the junction paragraph; the write exception to the per-line rule; BACKLOG dispositions
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a file mrw wrote is wholly known`, `the prose names junctions`, `the engine packages are unchanged`

## Goal

The round read "after any write the ledger licenses the whole file" as a broken promise. It is
ADR-002's decision and ADR-005 §4's ("a write observes everything, because mrw produced every
line"); M chose on 2026-09-25 to keep it and correct the prose, which says "per line" without the
exception in four places. Say where junctions stand, too, and close the round's BACKLOG items
this record owns.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `AGENTS.md` | edit | the rule list and "The rules that will bite you": the write exception; Portability: junctions |
| `README.md` | edit | "Per-line licence": the write exception; confinement: junctions |
| `internal/guide/guide.go` | edit | `mrw instructions`: the write exception |
| `internal/guide/guide_test.go` | edit | pins the sentence |
| `docs/adr/BACKLOG.md` | edit | dispositions for the round's items 1, 4, 9, 10 and the Windows suite |

## Ordered Steps

1. [S1] Add `TestInstructionsSayAWrittenFileIsWhollyKnown`; RED. [proof: mutation]
2. [S2] Edit the four sites and the BACKLOG; GREEN. [proof: mutation]
   Mutant: the guide sentence removed.

## Acceptance

```bash
set -o pipefail
go test ./internal/guide/ -count=1 -timeout 120s -run 'TestInstructionsSayAWrittenFileIsWhollyKnown' -v 2>&1 | tee /tmp/adr071-T5.out \
  && missing=$(for t in TestInstructionsSayAWrittenFileIsWhollyKnown; do grep -qE "^--- PASS: $t \(" /tmp/adr071-T5.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q 'junction' AGENTS.md \
  && grep -q 'junction' README.md \
&& git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInstructionsSayAWrittenFileIsWhollyKnown` | `internal/guide/guide_test.go` | `mrw instructions` states the write exception | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the sentences |
| 2 — something selects it | `mrw instructions`, the handshake, AGENTS.md |
| 3 — the caller can discover it | it is in the served text |
| 4 — it is used | two sessions of the round read the old wording as a defect |

## Verification Log
(empty until execute)
- 2026-09-25 · 77a408a* · exit 1 · `set -o pipefail …` · acceptance-sha256:bd2651fa22bc6e1f08efb1162c17ed07393c877ef51bd08defced5473447685e · ms:394 · test-lock-sha256:87d6d5cc7a8b595eff085a96e2154321de583fcc53830824d936ca452fcb77d7 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSUNvbnRhaW5zU2hhcmVkQW5kVGhlT3BlcmF0b3JUcmFwcwk2NTZiMWY2MzU0ZDRjYzY0NmEzNjAxZDBkYWUzY2Y4MmNkNzlhMGNhMGY2MTM0NzQ0ZTJkODU1ZjQwY2FlNzU1CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJVGVhY2hlc0Fsd2F5c0FuZEFQbGFuCTg3NmQ4ZDVlZWU2MGIwODk5ZjI0ODhmYzdhOGU3YzM4YzQ5Mzk1MWY0ZTRmODcxNDcxZGIzMGM1ZjBmN2U3MTEKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElUZWFjaGVzVGhlUmVhZFNpZGUJMDFmYzAzYTE1MGMyMTdjMTExZDNlZjE2MDAyMDliMjNlYmY2YTBlZTY1NzA4ZjRiMGNiMmIxNzAzN2QxZjlkMwpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdEV2ZXJ5U3VyZmFjZUNvbnRhaW5zVGhlU2hhcmVkU2VudGVuY2VzCWM0ZTMxNjkzMWUzNWYxNWJlODljZmU0ZWEzYTRmNWQ1ODZiNzI3NDE0OTE5ODYwYjNkYTMxMzY3MTUzY2Y3NzAKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RJbnN0cnVjdGlvbnNTYXlBV3JpdHRlbkZpbGVJc1dob2xseUtub3duCTJhOGU4NmQyODRjYTljYjNlZmY0NjEyNzYxODc5YjA4NDEwNjMxYmQ1Y2NmYjMyNjY5NjRmMzA0MTNjOTBiYmUKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RJbnN0cnVjdGlvbnNUZWFjaFRoZVBhZGRlZFBhdGhSdWxlCTg2NTgyMDMwNjU2ZTg2ZTdkN2U3MGZiY2Y3NGU1ZDNhMTRiNDQxN2I3NGE2YzY1NjA3NTVjY2I4ZTc5NTI4YTcKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3ROb1NlcnZlZFRleHRQcm9taXNlc05vdGhpbmdXcml0dGVuT25BbnlGYWlsdXJlCTBjMzUzN2I1OTc0ODZmZmY2OTM5NTEwZDI2NjgzOTc1NDIzYmI2MzgwZjEzOTJjNGVmZjRjNGFhYTY3MGI5ZGQKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RTaGFyZWRTYXlzQUZhaWxlZENvbW1pdElzUmVwb3J0ZWRQYXJ0aWFsCWRmOTFhNzIyZTU1ODExMDcyNzFhMDkxZDFhZDEyMzYwMWJlZDlhMjBhNzA5NWJmM2ZmODc1M2Q1OGJiNDE4ZGM
  ```
  --- last 6 line(s) of stdout
  === RUN   TestInstructionsSayAWrittenFileIsWhollyKnown
      guide_test.go:161: mrw instructions do not state the write exception to the per-line rule
  --- FAIL: TestInstructionsSayAWrittenFileIsWhollyKnown (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide	0.161s
  FAIL
  ```
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:bd2651fa22bc6e1f08efb1162c17ed07393c877ef51bd08defced5473447685e · ms:665
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:bd2651fa22bc6e1f08efb1162c17ed07393c877ef51bd08defced5473447685e · ms:219
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:bd2651fa22bc6e1f08efb1162c17ed07393c877ef51bd08defced5473447685e · ms:719
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:f8b0cfae886d5e975d754cfed2a2d937dce1594d0ede879452b65ed7559136a8 · ms:377
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:f8b0cfae886d5e975d754cfed2a2d937dce1594d0ede879452b65ed7559136a8 · ms:240

## Mutation Log
(empty until execute)
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/guide/guide.go` · mrw instructions no longer state the write exception to the per-line rule · acceptance-sha256:bd2651fa22bc6e1f08efb1162c17ed07393c877ef51bd08defced5473447685e · covers:a file mrw wrote is wholly known
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/guide/guide.go` · mrw instructions no longer state the write exception to the per-line rule · acceptance-sha256:f8b0cfae886d5e975d754cfed2a2d937dce1594d0ede879452b65ed7559136a8 · covers:a file mrw wrote is wholly known

## Invariants

- ADR-002 and ADR-005 are unchanged; only the prose moves.

## Risks

- None.

## Out of Scope

- Narrowing the write observation to served spans (permanent: boundary: M, 2026-09-25, kept ADR-002)

## Stop Condition

Stop if the prose would change what the code does.
