# Task ADR-108-T3: a body file is split like every other text

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** a body file is split like every other text
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a body file arrives as sent`

## Goal

`LoadBodyFiles` splits with `lines.Split`, so a body file's terminators are stripped exactly as a file's are.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `LoadBodyFiles` |
| `internal/plan/body108_test.go` | add | the test below |
| `scripts/contract.sh` | edit | §203 |

## Ordered Steps

1. [S1] Write the failing test(s) `TestABodyFileKeepsTheTargetsLineEndings`; confirm RED. [proof: mutation]
2. [S2] Split with `lines.Split`; contract §203 through `$MRW`. Mutant: the \\n-only split restored. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/plan/ -count=1 -timeout 300s -run 'TestABodyFileKeepsTheTargetsLineEndings' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestABodyFileKeepsTheTargetsLineEndings \(' "$out" \
  && grep -q '^# 203\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/iter internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestABodyFileKeepsTheTargetsLineEndings` | `internal/plan/body108_test.go` | a body file with LF, CRLF, CR-only, mixed, no final newline and empty content loads the same lines `lines.Split` gives, no line holding a `\\r` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write or acknowledgement that reaches it |
| 3 — the caller can discover it | the refusal or the receipt names it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review |

## Mutation Log
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/plan/plan.go` · the \n-only split restored: a CRLF body line keeps its \r · acceptance-sha256:257f75afd9099be383b5e1e4249345dcbdaf53012d4efff18fbaeb3b058dae3b

## Invariants

- Exit codes keep their meanings; inputs the defect did not touch behave as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 98feab5* · exit 1 · `set -o pipefail …` · acceptance-sha256:257f75afd9099be383b5e1e4249345dcbdaf53012d4efff18fbaeb3b058dae3b · ms:221 · test-lock-sha256:17600d6446d92441a546478dd10552efb3c1997fd9e6e11de5544dccf5853f9b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3BsYW4vYm9keTEwOF90ZXN0LmdvCVRlc3RBQm9keUZpbGVLZWVwc1RoZVRhcmdldHNMaW5lRW5kaW5ncwlmMGQ5NWU3ZjJiYTg0ZTY4NzgyNTMxZDNiOTk1Mzk1NWEzM2E4M2Q3M2Q3M2QzZDkxYjczMmU1NmQzZTU5MGM3
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan.test]
  internal/plan/bound108_test.go:14:9: undefined: maxBodyBytes
  internal/plan/bound108_test.go:15:21: undefined: maxBodyBytes
  internal/plan/bound108_test.go:16:2: undefined: maxBodyBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan [build failed]
  FAIL
  ```
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:257f75afd9099be383b5e1e4249345dcbdaf53012d4efff18fbaeb3b058dae3b · ms:324
- 2026-10-01 · 98feab5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:257f75afd9099be383b5e1e4249345dcbdaf53012d4efff18fbaeb3b058dae3b · ms:0 · test-lock-sha256:02e8d5d49e7bbad4cb9c4dc745a126848fb1b42da52ecdef6bfc990fed93750f · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3BsYW4vYm9keTEwOF90ZXN0LmdvCVRlc3RBQm9keUZpbGVLZWVwc1RoZVRhcmdldHNMaW5lRW5kaW5ncwk5OWRkZGUwMTRkOTQ1MzZmMTVlMTUwNjI2ODc3YTMyYmQ0Zjc5NzY4MDNmODg3ZTM1NTc5ZTA1YjA3ODExZWE2 · test-lock-kind:replace
- 2026-10-01 · human-observed · Zy's session observed the relock: after red the no-terminator loop exempts the mixed case, where lines.Split keeps the minority terminator by ADR-065; equality with lines.Split still holds for every case
