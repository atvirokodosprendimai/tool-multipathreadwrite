# Task ADR-015-T1: Say the escape, on both failure paths

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (two hints, two docs, one contract row)
**Owner:** unassigned
**Produces:** the two hints and their conditions
**Consumes:** —
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the body-line hint`, `the glob hint`, `both stay quiet when they should`

## Goal

The two mistakes mrw's own syntax invites produce a refusal naming the escape, and neither hint
fires on an ordinary failure.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | a `@@` line that fails to parse AFTER a hunk has been seen adds the `body=`/`raw=` hint |
| `internal/plan/plan_test.go` | edit | **the ADR's Enforced-by**, and the silence case |
| `internal/read/read.go` | edit | an unreadable path holding a glob metacharacter adds the glob hint |
| `internal/read/read_test.go` | edit | the glob hint, and its silence case |
| `AGENTS.md` | edit | the zsh trap, beside the MSYS trap that is already documented |
| `README.md` | edit | the same trap where README already documents the MSYS one, so the two shell traps are found together |
| `scripts/contract.sh` | edit | §49 — both hints, through the built binary |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a body line beginning with `@@` produces an error
   naming `body=` and `raw=`; an ordinary bad op on the FIRST line does not; an unreadable path
   containing `*` names the glob problem; an ordinary missing file does not. [proof: acceptance]
2. [S2] Condition the body-line hint on a previous hunk having been seen. A hint on every parse
   failure is noise, and noise is how a real hint stops being read. [proof: mutation]
3. [S3] Condition the glob hint on the path holding `*`, `?` or `[` AND having failed to open. It is
   advice on a path that is already broken, not a claim about what the caller meant. [proof: mutation]
4. [S4] Document the zsh trap in AGENTS.md next to the MSYS one, including the part that surprises:
   an address suffix stops the shell matching, so quoting does not help and `--grep` or
   `--files-from` is the answer. [proof: acceptance]
5. [S5] Add contract §49: both hints, driven through the built binary, plus both silence cases.
   [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/plan/ ./internal/read/ -v 2>&1 | tee /tmp/adr015-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr015-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestABodyLineThatLooksLikeAHeaderSaysSo|TestAGlobThatTheShellDidNotExpandSaysSo|TestTheHintsStayQuietOnOrdinaryFailures|TestAnOrdinaryMissingFileGetsNoGlobHint)\b' /tmp/adr015-t1.out)" = "4" ] \
  && grep -q '^# 49\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the three test names,
`# 49.`, and the Go identifier `hintUnexpandedGlob`. §49 was confirmed free — 44 is the highest on
`main`, ADR-013 holds 45/46 on #74 and ADR-014 holds 47/48 on #76, so this takes the next free one
rather than colliding on merge.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestABodyLineThatLooksLikeAHeaderSaysSo` | `internal/plan/plan_test.go` | **the ADR's Enforced-by** — the D5 error names the escape | — | S1, S2 |
| `TestAGlobThatTheShellDidNotExpandSaysSo` | `internal/read/read_test.go` | S3 — the D4 error names the real problem | — | S1, S3 |
| `TestTheHintsStayQuietOnOrdinaryFailures` | `internal/plan/plan_test.go` | S2 — the plan-side silence case: a hint that always fires is noise | — | S1, S2 |
| `TestAnOrdinaryMissingFileGetsNoGlobHint` | `internal/read/read_test.go` | S3 — the read-side silence case. It existed and passed but was not in this table, so `both stay quiet when they should` had only half a test bound to it | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests above |
| 2 — something selects it | both hints sit on paths every failing plan and every unreadable path take |
| 3 — the caller can discover it | the hint IS the discovery surface — it is read at the moment of the mistake |
| 4 — it is used | nothing counts whether a hint shortened a caller's recovery, and nothing will: ADR-009 refused telemetry, and ADR-012's Context records why the local tally cannot attribute anything |

## Mutation Log

- 2026-09-04 · a1059d2 · mutant killed · exit 1 · `internal/plan/plan.go` · S2: fire the body-line hint unconditionally — a hint on every parse failure is noise, and noise is how a real hint stops being read · acceptance-sha256:ded8db4c20c1c4361ed3a11675c59c0fc04af80b8305d4a1d986a0f36fa0cd45
- 2026-09-04 · a1059d2 · mutant killed · exit 1 · `internal/read/read.go` · S3: fire the glob hint for every unreadable path — an ordinary missing file would be told to check its shell, which is advice about a problem it does not have. Killed by TestAnOrdinaryMissingFileGetsNoGlobHint, which the fence did not count until review pointed it out · acceptance-sha256:ded8db4c20c1c4361ed3a11675c59c0fc04af80b8305d4a1d986a0f36fa0cd45

## Invariants

- Every plan that parsed before parses to the same hunks.
- The body-line hint appears only after a hunk header has been seen.
- The glob hint appears only for a path that failed to open and holds a metacharacter.
- No new dependency.

## Risks

- A hint becomes noise and stops being read. Mitigated by testing the silence cases, which is the
  half that is easy to skip.

## Stop Condition

Stop if either hint needs the grammar or the spec syntax to change. This record is diagnostics; a
grammar change is ADR-001's and needs its own record.

## Out of Scope

- A heredoc body terminator (deferred: docs/adr/BACKLOG.md)
- Expanding globs inside mrw (permanent: boundary: stated in the parent ADR)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 14d28b3* · exit 0 · `set -o pipefail …` · acceptance-sha256:176af12d72a278b779c4ef03dd9b03a3203b7c1b510f11895408699dfc01168c · ms:10835
- 2026-09-04 · a1059d2* · exit 0 · `set -o pipefail …` · acceptance-sha256:ded8db4c20c1c4361ed3a11675c59c0fc04af80b8305d4a1d986a0f36fa0cd45 · ms:13644
