# Task ADR-033-T2: The contract drives both spellings

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (a contract section and two documents)
**Owner:** Zy
**Produces:** contract §69 and the documented meaning of zero
**Consumes:** `MaxLines *int` and the zero-is-a-cap rule (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built binary serving nothing for a cap of zero`, `the built binary serving the whole file with no cap`

## Goal

Prove both spellings in the BUILT binary, and say what zero means where a caller reads it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 69.` section pairing `--max-lines 0` with an absent flag. A row asserting only the zero case would pass against a binary that had stopped serving anything. |
| `cmd/mrw/main.go` | edit | The flag's usage string says zero means zero. |
| `README.md` | edit | The flag table says the same, since a caller who meets an empty read otherwise reads it as a bug. |

## Ordered Steps

1. [S1] Confirm §69 does not exist and that the row is RED against a binary built before T1 — where `--max-lines 0` serves the file whole. [proof: acceptance]
2. [S2] Write §69: `--max-lines 0` serves no content line, reports the withheld count, and exits non-zero; the same read with no flag serves every line and exits 0. [proof: acceptance]
3. [S3] Document in the usage string and the README. [proof: human: read both against §69's fixture and confirm each says zero is a cap of zero and that omitting the flag is how to ask for no cap]
4. [S4] Run every gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 69\. ADR-033: a cap of zero is a cap\.' scripts/contract.sh \
  && grep -q 'zero means zero' cmd/mrw/main.go \
  && grep -q 'zero means zero' README.md \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr033-t2.out \
  && grep -q '^contract holds$' /tmp/adr033-t2.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr033-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§69` | `scripts/contract.sh` | The built binary serves nothing and reports the withholding for `--max-lines 0`, and serves the whole file, line for line and in order, when the flag is absent | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §69 in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section against the binary it builds |
| 3 — the caller can discover it | The usage string and the README table, checked in S3 |
| 4 — it is used | The contract runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · 73649a9* · mutant killed · exit 1 · `internal/read/read.go` · the BUILT binary stops honouring a cap of zero, so §69 sees it serve the file and report nothing withheld — the row drives $MRW, which a unit test cannot · acceptance-sha256:b3d83a81ad6b0687f4877ebc85e4464a3f6525ba15bc3883789b9cd6cfc97175 · covers:the built binary serving nothing for a cap of zero
- 2026-09-07 · 2fda99d* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI stops distinguishing an ABSENT flag from an explicit zero, which is the whole wiring — the Go test drives read.Run directly and cannot see it, so §69 is the only gate that can. Its absence as a receipt was the review of PR #133 finding a [proof: mutation] step with no mutant against cmd/mrw · acceptance-sha256:b3d83a81ad6b0687f4877ebc85e4464a3f6525ba15bc3883789b9cd6cfc97175 · covers:the built binary serving nothing for a cap of zero

## Invariants

- Both spellings in one section: a row asserting only the zero case passes against a binary that serves nothing at all.
- The row drives `$MRW`, never a Go test.

## Risks

- A row asserting only the refusal would pass against a tool that had stopped serving content. Mitigated by S2 requiring the no-flag case in the same section.
- The documentation could state the change without naming how to ask for no cap, which is ADR-015's failure. Mitigated by S3's human proof.

## Stop Condition

Stop and ask if §69 cannot be made red against the pre-ADR-033 tree: a row green before the change
exists asserts nothing.

## Out of Scope

- The semantics — T1
- `MaxResultChars` (deferred: `docs/adr/BACKLOG.md` — its own record)

## Verification Log
- 2026-09-07 · 73649a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:b3d83a81ad6b0687f4877ebc85e4464a3f6525ba15bc3883789b9cd6cfc97175 · ms:29836
- 2026-09-07 · 73649a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:b3d83a81ad6b0687f4877ebc85e4464a3f6525ba15bc3883789b9cd6cfc97175 · ms:32507
- 2026-09-07 · 2fda99d* · exit 0 · `set -o pipefail …` · acceptance-sha256:b3d83a81ad6b0687f4877ebc85e4464a3f6525ba15bc3883789b9cd6cfc97175 · ms:33041
- 2026-09-07 · 2fda99d* · exit 0 · `set -o pipefail …` · acceptance-sha256:b3d83a81ad6b0687f4877ebc85e4464a3f6525ba15bc3883789b9cd6cfc97175 · ms:31547
- 2026-09-07 · fd4ce17* · exit 0 · `set -o pipefail …` · acceptance-sha256:b3d83a81ad6b0687f4877ebc85e4464a3f6525ba15bc3883789b9cd6cfc97175 · ms:34524
