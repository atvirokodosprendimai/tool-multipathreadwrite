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
  && go test ./internal/mcp/ -count=1 -v -run 'TestEverySurfaceCarriesTheOneRule|TestTheInstructionsTellAHostHowToAuthorAPlan' 2>&1 | tee /tmp/adr031-rule.out \
  && grep -q '^--- PASS: TestEverySurfaceCarriesTheOneRule' /tmp/adr031-rule.out \
  && grep -q '^--- PASS: TestTheInstructionsTellAHostHowToAuthorAPlan' /tmp/adr031-rule.out \
  && ! grep -qE 'no tests to run|^FAIL|^--- FAIL' /tmp/adr031-rule.out \
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

⚠ **THE RULE IS ONE CONSTANT NOW, AND THAT IS THE STRUCTURAL ANSWER.** Three paraphrases drifted
from the mechanism and two mutants survived by leaving a token, and then a heading, in place. The
third review of PR #132 then demonstrated the remaining hole directly: negating the operative clause
while keeping `ack`, `LICENSES NOTHING`, `BOTH its markers` and `numbered NNN| lines` still passed.
Listing tokens cannot express "says this". So `mcp.AckRule` is the sentence. The MCP instructions, the page footer, the refusal remedy and BOTH tool schemas are built from the constant; `README.md` and `AGENTS.md` carry the same text as static duplicates, which a test pins rather than derives — changing the constant does not change those two files, it makes their check FAIL until somebody updates them, and the fourth review of PR #132 was right that the first draft claimed otherwise.
`TestEverySurfaceCarriesTheOneRule` requires it BYTE FOR BYTE in the instructions, `README.md`,
`AGENTS.md` and both schema descriptions; `TestAPagedFooterCarriesTheOneRule` inspects a REAL paged
answer; and the contract row reads the constant out of the source and requires it on the WIRE.

⚠ **An `instructions.go` mutant SURVIVED TWICE, and the fence was the reason both times.**
The second run gutted the OPERATIVE CLAUSE — "only if you hold BOTH its markers AND counted the N
numbered NNN| lines between them" became "only if you received it" — while the fence checked that
the heading "A PAGE LICENSES NOTHING" was present, which it still was. The first survivor taught
that checking a WORD is not checking the rule; the second taught that checking the rule's HEADING is
not checking the rule either. The fence and the instruction-words contract row now match the clause
that carries the requirement.

⚠ **The first `instructions.go` mutant SURVIVED too, and the fence was the reason then as well.** It
gutted the rule — "A PAGE LICENSES NOTHING" became "A page licenses everything" — while the fence
only grepped for the token `ack`, which was still there. A gate that checks a word is present is not
a gate that checks the rule is stated. The fence now matches the rule itself, and the contract row
that already pins required instruction words carries `ack` and `LICENSES NOTHING` too, so the claim
is asserted against the built server rather than against a grep of the source.

- 2026-09-07 · d95d79e* · mutant killed · exit 1 · `internal/mcp/ack.go` · the interleave stops emitting checkpoints mid-page, so a whole page carries one marker at the end — the single-token design ADR-031 rejects, and the one that survives a middle cut · acceptance-sha256:feeb54c07d5d3a5bd8cac1ce2bc0923b87a2e0bb17f1242b543f44abfc86e34f · covers:the built server licensing only acked segments
- 2026-09-07 · d95d79e* · mutant survived · exit 0 · `internal/mcp/instructions.go` · the instructions stop teaching the rule while the server still enforces it, so a caller meets the refusal with no idea what ack is — the ADR-015 failure this task exists to prevent · acceptance-sha256:feeb54c07d5d3a5bd8cac1ce2bc0923b87a2e0bb17f1242b543f44abfc86e34f · covers:the instructions naming ack within the byte bound
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-07 · d95d79e* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the instructions stop teaching the rule while the server still enforces it. It SURVIVED on its first run because the fence only grepped for the token ack; the fence now matches the rule, and the instruction-words contract row carries it too · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · covers:the instructions naming ack within the byte bound
- 2026-09-07 · ba6aecd* · mutant survived · exit 0 · `internal/mcp/instructions.go` · the instructions stop requiring both markers and the count, so a caller that received half a span acknowledges it — the mechanism brackets and the GUIDANCE defeats it, which is the P0 the second review of PR #132 found: an implementation is worth what its instructions say · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · covers:the instructions naming ack within the byte bound
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-07 · ba6aecd* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the instructions stop requiring both markers and the count. It SURVIVED twice before: once when the fence checked only the token ack, and again when it checked only the heading A PAGE LICENSES NOTHING. Checking a word is not checking the rule; checking the rule HEADING is not checking the rule either · acceptance-sha256:15bb7ed8ce28c05a2a8c3715a4d97e617c7ffbfe8bd56fae9890ff4b25cf3156 · covers:the instructions naming ack within the byte bound
- 2026-09-07 · f0865c8* · mutant killed · exit 1 · `internal/mcp/ack.go` · the canonical rule is weakened at its single source, which now changes every surface at once — instructions, footer, README and AGENTS. Three earlier paraphrases drifted and two mutants survived against token and heading checks; this is the shape those gates could not see · acceptance-sha256:cdb4e95e7f640dff5456dd1b3028eba624c69c724d0e68ef2d46bc0e6c06f3bb · covers:the instructions naming ack within the byte bound

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
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:feeb54c07d5d3a5bd8cac1ce2bc0923b87a2e0bb17f1242b543f44abfc86e34f · ms:30983
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:feeb54c07d5d3a5bd8cac1ce2bc0923b87a2e0bb17f1242b543f44abfc86e34f · ms:29834
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:feeb54c07d5d3a5bd8cac1ce2bc0923b87a2e0bb17f1242b543f44abfc86e34f · ms:29480
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · ms:32291
- 2026-09-07 · d95d79e* · exit 0 · `set -o pipefail …` · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · ms:29731
- 2026-09-07 · dbe88d0* · exit 0 · `set -o pipefail …` · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · ms:30208
- 2026-09-07 · ba6aecd* · exit 0 · `set -o pipefail …` · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · ms:31749
- 2026-09-07 · ba6aecd* · exit 0 · `set -o pipefail …` · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · ms:35585
- 2026-09-07 · ba6aecd* · exit 0 · `set -o pipefail …` · acceptance-sha256:37066136ae42398cf0f39614a5e406b4c3ee603f14675da395711b9f98d180da · ms:31928
- 2026-09-07 · ba6aecd* · exit 0 · `set -o pipefail …` · acceptance-sha256:15bb7ed8ce28c05a2a8c3715a4d97e617c7ffbfe8bd56fae9890ff4b25cf3156 · ms:30121
- 2026-09-07 · ba6aecd* · exit 0 · `set -o pipefail …` · acceptance-sha256:15bb7ed8ce28c05a2a8c3715a4d97e617c7ffbfe8bd56fae9890ff4b25cf3156 · ms:30024
- 2026-09-07 · f0865c8* · exit 1 · `set -o pipefail …` · acceptance-sha256:c7c5e0b3c7e1f1518efb2173c3d163a4cbaaa13075391d67c199a08a8d591411 · ms:331
  ```
  --- last 1 line(s) of stdout
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.178s
  ```
- 2026-09-07 · f0865c8* · exit 0 · `set -o pipefail …` · acceptance-sha256:214ecdbe31cd12cef63b3c559c4b1aebed27c127ddd4fbf74766a7b37e7b886d · ms:30868
- 2026-09-07 · f0865c8* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdb4e95e7f640dff5456dd1b3028eba624c69c724d0e68ef2d46bc0e6c06f3bb · ms:30876
- 2026-09-07 · f0865c8* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdb4e95e7f640dff5456dd1b3028eba624c69c724d0e68ef2d46bc0e6c06f3bb · ms:30079
- 2026-09-07 · 7c7b0b7* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdb4e95e7f640dff5456dd1b3028eba624c69c724d0e68ef2d46bc0e6c06f3bb · ms:32483
- 2026-09-07 · c617934* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdb4e95e7f640dff5456dd1b3028eba624c69c724d0e68ef2d46bc0e6c06f3bb · ms:30160
- 2026-09-07 · fc95241* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdb4e95e7f640dff5456dd1b3028eba624c69c724d0e68ef2d46bc0e6c06f3bb · ms:32184
- 2026-09-07 · fc95241* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdb4e95e7f640dff5456dd1b3028eba624c69c724d0e68ef2d46bc0e6c06f3bb · ms:32108
