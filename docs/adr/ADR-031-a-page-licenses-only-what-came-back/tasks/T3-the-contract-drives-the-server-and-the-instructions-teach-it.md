# Task ADR-031-T3: The contract drives the server, and the instructions teach it

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S (a contract section and the instruction text)
**Owner:** Zy
**Produces:** contract §68 and the documented `ack` rule
**Consumes:** the checkpoint format and the pending store (T1), `ack` on both tools, and promotion (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built server licensing only acked segments`, `the instructions naming ack within the byte bound`

## Goal

Prove it in the BUILT server, over the wire, and tell a caller the rule where it will read it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 68.` section driving `mrw mcp` over stdio: serve a paged read, ack the ends, write the middle and the end, and assert the two verdicts differ. A unit test proves `promote`; it cannot prove the SERVER calls it. |
| `internal/mcp/instructions.go` | edit | The rule and the shape. ⚠ The instructions are under a hard 4,096-byte bound with its own test; the text is paid for by compressing prose, never by raising the bound — the bound's own test says shorten. |
| `README.md`, `AGENTS.md` | edit | The MCP sections gain the rule, since a caller reading either must not meet the refusal without the remedy. |

## Ordered Steps

1. [S1] Confirm §68 does not exist and that the row is RED against a server built before T2 — where the middle write APPLIES. [proof: acceptance]
2. [S2] Write §68 driving the built server over stdio: a paged read, the checkpoints parsed out of the served text, an ack of the first and last, then two writes — one to the middle and one to an acked span — asserting refusal and success respectively. [proof: acceptance]
3. [S3] Add the rule to the instructions and confirm `TestTheInstructionsTellAHostHowToAuthorAPlan`, which asserts the 4,096-byte bound, still passes, having compressed elsewhere. [proof: acceptance]
4. [S4] Document in `README.md` and `AGENTS.md`. [proof: human: read both passages against §68's fixture and confirm each names `ack`, says an unacked page licenses nothing, and does not claim the CLI takes it]
5. [S5] Run every gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 68\. ADR-031: a page licenses only what came back\.' scripts/contract.sh \
  && grep -q 'ack' internal/mcp/instructions.go \
  && grep -q 'ack' README.md \
  && grep -q 'ack' AGENTS.md \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr031-t3.out \
  && grep -q '^contract holds$' /tmp/adr031-t3.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr031-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§68` | `scripts/contract.sh` | The built server records only acked segments: a write to an unacked span is refused naming `ack`, and a write to an acked span applies | — | S1, S2 |
| `TestTheInstructionsTellAHostHowToAuthorAPlan` | `internal/mcp/mcp_test.go` | Unchanged: it asserts the instructions stay under `maxInstructionsChars` (4,096), which now has to hold with the new rule in them | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §68 in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section against the binary it builds |
| 3 — the caller can discover it | The instructions, `README.md` and `AGENTS.md`, checked in S4 |
| 4 — it is used | The contract runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

## Invariants

- The row drives `$MRW`, never a Go test, and it drives the SERVER rather than the library.
- Both halves in one section: the refused middle and the applied end. A row asserting only the refusal passes against a server that licenses nothing at all.
- The instruction byte bound is not raised. Compressing prose is the only permitted way to pay for the new text.

## Risks

- A section asserting only the refusal would pass against a server that had stopped licensing anything. S2 pairs it with the acked write.
- The instructions could describe the rule without naming what to send, which is the failure ADR-015 exists to prevent. S4's human proof reads both passages against the fixture.

## Stop Condition

Stop and ask if §68 cannot be made red against the pre-ADR-031 server: a row green before the feature
exists asserts nothing, and finding that out here is the point of S1.

## Out of Scope

- The mechanism — T1 and T2
- Any change to `MaxResultChars` (deferred: `docs/adr/BACKLOG.md` — its own record)

## Verification Log
