# Task ADR-035-T2: The contract drives the refusal, and the docs teach it

**Depends-on:** T1
**Covers:** none — no spec
**Owner:** Zy
**Produces:** contract §73, and the plan grammar as documented in `README.md`, `AGENTS.md` and the MCP `plan` description
**Consumes:** the refusal in `apply.Apply`, and its message naming `anchor=`
**Proof map:** v1

**Rests-on:** `the built binary refusing an anchorless multi-line replace`, `the built binary still applying one that carries an anchor`, `every documented multi-line replace being a plan mrw would accept`, `the documentation check classifying a header exactly as the parser does`

## Goal

Prove the refusal reaches the shipped binary, and stop every document and the MCP
surface from teaching a grammar mrw no longer accepts.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | §73: the anchorless multi-line replace refused, paired with the anchored one applying. A unit test proves the function; it cannot prove the function is called. |
| `README.md` | edit | The plan grammar and its worked examples. |
| `AGENTS.md` | edit | Same, in the agent-facing copy. |
| `internal/mcp/mcp.go` | edit | The `plan` tool description — ADR-012: the MCP surface teaches the format it demands. |
| `scripts/break-campaign.sh` | edit | `@@ f.txt 1-3 replace` at line 67 measures overlap detection and would now fail for a different reason, which would silently stop measuring what it is there to measure. |
| `internal/adversarial/documented_plans_test.go` | add | The documentation check. It calls `plan.Parse` rather than approximating `splitHeader` with a regex — four shell cuts were each defeated by a header the parser accepts (a quoted path read as the op, a greedy match backtracking into one, tab separators, a QUOTED OP and a BOM). A contract row can drive the built binary but cannot tokenise a plan header, so this claim does not belong in one. |

## Ordered Steps

1. [S1] Write §73 driving `$MRW` and confirm it is RED against the binary built from T1's merge-base — an anchorless `3-6` replace must exit non-zero and leave the file byte-identical, and the same plan with `anchor=` must apply. ⚠ Assert the MESSAGE names `anchor=`, not merely that an error came back: `mrw write` exits non-zero for a dozen reasons, and asserting the exit alone scores the binary's error handling rather than this guard. Take the section number with `grep -oE '^# [0-9]+\. ' scripts/contract.sh | sort -k2,2n | tail -1` — keyed on the second field and with the trailing space, per `.claude/rules/lifecycle.md` — after confirming `# 73. ` returns zero hits. [proof: mutation]
2. [S2] Rebuild `bin/mrw` from the branch and confirm §73 goes GREEN. A contract row is evidence only once it has been seen to fail; a row written against an already-fixed binary has never been red. [proof: acceptance]
3. [S3] Sweep every documented multi-line `replace` and give each one an anchor, or say in the prose why the example is refused. ⚠ `@@ f.go 5-9999 replace` in both `README.md` and `AGENTS.md` illustrates the out-of-range refusal and now has TWO reasons to fail; the sentence must not imply the range is the only one. [proof: acceptance]
4. [S4] Update `scripts/break-campaign.sh:67`. [proof: acceptance]
5. [S5] State the requirement in the plan grammar itself in all three places, next to `sha=`, `lines=` and `anchor=` where they are introduced — a guard a caller learns about only from a refusal is one they meet at the worst moment. [proof: acceptance]
6. [S6] Run every gate, unpiped, including a full `./scripts/contract.sh`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 73\. ' scripts/contract.sh \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr035-t2c.out \
  && grep -q '^contract holds$' /tmp/adr035-t2c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§73` | `scripts/contract.sh` | The BUILT binary refuses an anchorless multi-line replace with a message naming `anchor=` and leaves the file byte-identical, and applies the same plan when it carries one | — | S1, S2 |
| `TestEveryDocumentedReplaceCarriesItsAnchor` | `internal/adversarial/documented_plans_test.go` | Every `@@` line in `README.md` and `AGENTS.md` is classified by the real parser, so a documented replace addressing more than one line without `anchor=` fails the build | — | S3, S5 |
| `TestTheDocumentedPlanCheckRejectsWhatItMustReject` | `internal/adversarial/documented_plans_test.go` | The gate on the gate: ten headers that must be flagged — including the quoted op, the BOM and the tabbed range that defeated the shell versions — and ten that must pass | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §73 |
| 2 — something selects it | `./scripts/contract.sh` runs every section; §73 fails if the built binary accepts the anchorless plan |
| 3 — the caller can discover it | the plan grammar in `README.md`, `AGENTS.md`, and the MCP `plan` description |
| 4 — it is used | the row drives `$MRW`, which is what ships |

## Mutation Log

- 2026-09-08 · 3245e1a* · mutant killed · exit 1 · `internal/apply/apply.go` · the guard stops firing in the SHIPPED binary, so §73 sees an anchorless multi-line replace apply — the wiring a unit test cannot check, because it proves the function and not that the function is reached · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · covers:the built binary refusing an anchorless multi-line replace
- 2026-09-08 · 3245e1a* · mutant killed · exit 1 · `internal/apply/apply.go` · the guard refuses EVERY multi-line replace, anchored or not, so the requirement becomes a ban — which the refusal half of §73 cannot see on its own, and only the paired applying case catches · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · covers:the built binary still applying one that carries an anchor
- 2026-09-08 · 3245e1a* · mutant killed · exit 1 · `AGENTS.md` · a documented example goes back to the old grammar, so AGENTS.md teaches a plan mrw now refuses — the drift nothing else here would catch, since no gate reads README.md or AGENTS.md · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · covers:every documented multi-line replace being a plan mrw would accept
- 2026-09-08 · 2254c4a* · mutant inconclusive · exit 1 · `internal/adversarial/documented_plans_test.go` · the documentation check classifies every replace as single-line, so it returns "" for everything and passes on documentation that teaches a refused shape — the vacuous-gate shape that three shell cuts of this check each had in a different disguise · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · covers:the documentation check classifying a header exactly as the parser does
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-08 · 2254c4a* · mutant inconclusive · exit 1 · `internal/adversarial/documented_plans_test.go` · the documentation check classifies every replace as single-line, so it returns "" for everything and passes on documentation that teaches a refused shape — the vacuous-gate shape that four shell cuts of this check each had in a different disguise · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · covers:the documentation check classifying a header exactly as the parser does
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-08 · 2254c4a* · mutant killed · exit 1 · `internal/adversarial/documented_plans_test.go` · the documentation check returns "" for every header, so it passes on documentation that teaches a refused shape — the vacuous-gate shape that four shell cuts of this check each had in a different disguise, and the reason it carries a gate on the gate · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · covers:the documentation check classifying a header exactly as the parser does

## Invariants

- §73 drives `$MRW`, never a Go test helper.
- Every multi-line `replace` shown in `README.md`, `AGENTS.md` and the MCP `plan` description is a plan mrw would accept, or the prose says why it would not.
- The MCP handshake `instructions` stay within `maxInstructionsChars`; if the clause does not fit, it goes in the `plan` tool description, which is not bounded by that budget.

## Risks

- The row could pass because the anchorless plan failed for an unrelated reason — an unread line is the likely one, since §73 must read the fixture first. S2 asserts the message names `anchor=`.
- A documentation example could be corrected in one file and not the other. `README.md` and `AGENTS.md` carry the same grammar by design; S3 sweeps both.

## Stop Condition

Stop and ask if the MCP handshake cannot carry the requirement within
`maxInstructionsChars` without weakening a claim a test already guards — the last
attempt at that traded a guarded claim for a new sentence, and the answer was to
defer with a receipt, not to shorten what is guarded.

## Out of Scope

- The guard itself — T1
- Any requirement on `delete` (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:38449
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:33361
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:33542
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:33524
- 2026-09-08 · e9165d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:34286
- 2026-09-08 · 0681115* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:36219
- 2026-09-08 · 195f72f* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:39249
- 2026-09-08 · 2254c4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:33811
- 2026-09-08 · 2254c4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:33329
- 2026-09-08 · 2254c4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:33325
- 2026-09-08 · 2254c4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0dd175e616442c631057034b9b434721c41060feb945a3ce1742eae2d3678402 · ms:35246
