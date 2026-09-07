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

## Invariants

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
