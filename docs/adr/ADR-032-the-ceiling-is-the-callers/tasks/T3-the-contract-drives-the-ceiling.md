# Task ADR-032-T3: The contract drives the ceiling

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S (a contract section and the documentation)
**Owner:** Zy
**Produces:** contract §70 and the documented knob
**Consumes:** the configured budget (T1), the bounded write receipt (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built server obeying a caller-set ceiling`, `the built server advertising what it enforces`

## Goal

Prove it in the BUILT server, at the default and at a caller-set budget, and document the knob.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 70.` section: at the default and at a small `--max-result-chars`, both tools' encoded results fit, and `_meta` names the value in force. |
| `README.md`, `AGENTS.md` | edit | The knob, its default, and that `0` means zero. |
| `internal/mcp/instructions.go` | edit | One line, within the 4,096-byte bound — paid for by compressing prose, never by raising it. |

## Ordered Steps

1. [S1] Confirm §70 does not exist and is RED against a server built before T1 — where `mrw_write` returns 453,632 characters against an advertised 200,000. [proof: acceptance]
2. [S2] Write §70 driving `mrw mcp` at the default and with `--max-result-chars` set small: measure the ENCODED result, and compare `_meta` to the configured value. [proof: acceptance]
3. [S3] Document in `README.md`, `AGENTS.md` and the instructions. [proof: human: read each passage against §70's fixture and confirm it names the flag, the env var, the default, and that `0` means zero]
4. [S4] Run every gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 70\. ADR-032: the ceiling is the caller' scripts/contract.sh \
  && grep -q 'max-result-chars' README.md \
  && grep -q 'max-result-chars' AGENTS.md \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr032-t3.out \
  && grep -q '^contract holds$' /tmp/adr032-t3.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr032-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§70` | `scripts/contract.sh` | The built server keeps both tools' encoded results within the advertised value, at the default and at a caller-set budget, and advertises what it enforces | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §70 in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section against the binary it builds |
| 3 — the caller can discover it | `README.md`, `AGENTS.md`, the instructions and `_meta`, checked in S3 |
| 4 — it is used | The contract runs in CI on every push |

## Mutation Log

- 2026-09-07 · b36aea1* · mutant killed · exit 1 · `cmd/mrw/main.go` · the built server ignores --max-result-chars and serves the default, so section 70 measures a ceiling nobody set and the advertisement no longer matches the value in force · acceptance-sha256:fab6668ddc2955165a55b18c16dbb97375c085e9f65e9d190aacf3e81fca4e76 · covers:the built server obeying a caller-set ceiling

## Invariants

- ⚠ A SURVIVOR LINE WAS WRITTEN HERE IN ERROR AND REMOVED, and the removal is recorded rather than
  hidden. The "mutant" appended a trailing COMMENT to a contract line, so it changed no mechanism and
  could not have failed; logged, it reads as a finding about `# 70.` when it is a finding about the
  person who typed it. The append-only Mutation Log is for evidence, and a line that is evidence of
  nothing costs the next reader a real investigation. The killed mutant above it is the one that
  binds this section: with `--max-result-chars` ignored, §70 measures a ceiling nobody set.
- ⚠ S1's RED was measured against a binary built from `main`, not by removing §70 from this tree: with
  the pre-ADR-032 server the same three calls returned 76,262 / **976,338** / 989 bytes against an
  advertised 200,000, and `mcp --max-result-chars 20000` was refused with "flag provided but not
  defined". So the Verification Log below carries no red entry by construction — the fence cannot be
  red against a tree that already has the fix, and the red is the measurement above.
- The row measures the ENCODED result, the same quantity T1 bounds.
- Both tools in one section: a row covering only reads passes against today's tree.
- The instruction byte bound is not raised.

## Risks

- A row that measures the report text would pass against the defect. S2 measures the encoded JSON.
- Documenting the knob without saying what `0` does repeats ADR-033's question. S3's human proof requires it.

## Stop Condition

Stop and ask if §70 cannot be made red against the pre-ADR-032 server.

## Out of Scope

- The mechanism — T1 and T2

## Verification Log
- 2026-09-07 · b36aea1* · exit 0 · `set -o pipefail …` · acceptance-sha256:fab6668ddc2955165a55b18c16dbb97375c085e9f65e9d190aacf3e81fca4e76 · ms:51224
- 2026-09-07 · human-observed · Zy, 2026-09-07: read README.md:472-496, AGENTS.md:104-109 and the instructions text against contract section 70's fixture. Each names --max-result-chars, MRW_MAX_RESULT_CHARS, the 200,000 default and that 0 means zero; the README and AGENTS passages also say the write receipt elides successes and never a failure. The instructions carry the ceiling in one sentence at 4,089 of 4,096 bytes, paid for by compressing five existing sentences and not by raising the bound.
- 2026-09-07 · b36aea1* · exit 0 · `set -o pipefail …` · acceptance-sha256:fab6668ddc2955165a55b18c16dbb97375c085e9f65e9d190aacf3e81fca4e76 · ms:46799
- 2026-09-07 · b36aea1* · exit 0 · `set -o pipefail …` · acceptance-sha256:fab6668ddc2955165a55b18c16dbb97375c085e9f65e9d190aacf3e81fca4e76 · ms:39052
