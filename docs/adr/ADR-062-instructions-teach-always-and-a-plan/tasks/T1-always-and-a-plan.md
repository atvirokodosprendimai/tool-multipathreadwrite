# Task ADR-062-T1: Shared() always + plan; CLI() cookbook; contract §114

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** always + plan on Shared(); cookbook on CLI() only; §114
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `Shared first sentence is always and a plan`, `CLI cookbook names @@ path 0 create`, `Shared does not carry the cookbook`, `3 or more edits is gone from CLI`

## Goal

`guide.Shared()`'s first sentence is `Use mrw always: plan the activity as one read of every site, then one plan, then one write.` `guide.CLI()` teaches `@@ path 0 create` after the operator traps. The MCP handshake contains Shared() and does not contain that cookbook needle. Contract §114 drives the shipped binary.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | Shared() first sentence; CLI() cookbook after the traps. |
| `internal/guide/guide_test.go` | edit | Shared needles; `TestCLITeachesAlwaysAndAPlan`. |
| `internal/mcp/instructions.go` | edit | `triggerRule`; handshake must not say "below that, don't". |
| `internal/mcp/mcp.go` | edit | descriptions pitch always + plan; drop "Below that a single read is cheaper". |
| `internal/mcp/mcp_test.go` | edit | descriptions still carry `triggerRule`; drop the "when NOT to reach for mrw" assertion. |
| `scripts/contract.sh` | edit | §43 / §75 / §85 Shared tuples; **§114** — next free after §113. |
| `AGENTS.md` | edit | "Using mrw" opening matches Shared()'s first sentence. |
| `BESTPRACTICES.md` | edit | When section is always + plan. |
| `docs/comparison.md` | edit | one-file one-edit names the byte cost; does not teach "stay out". |
| `.claude/skills/mrw/SKILL.md` | edit | description no longer says 3+ / stay on Edit. |
| `docs/adr/BACKLOG.md` | edit | Receipt: centralised skill update is deferred. |

## Ordered Steps

1. [S1] Write `TestCLITeachesAlwaysAndAPlan` and update `TestEverySurfaceContainsTheSharedSentences` so both are RED on the current pamphlet. [proof: mutation]
2. [S2] Change Shared(), CLI(), `triggerRule`, MCP descriptions / handshake wording, and the AGENTS.md / BESTPRACTICES / comparison / skill mirrors. S1 GREEN. Restoring the 3+ first sentence must fail S1. Deleting `@@ path 0 create` from CLI() must fail S1. [proof: mutation]
3. [S3] Rewrite §43 / §75 / §85 Shared tuples. Write §114 against a binary that still prints 3+ and confirm it is RED, then rebuild and confirm GREEN. The handshake half must fail if `initialize` carries `@@ path 0 create`. [proof: mutation]
4. [S4] BACKLOG receipt. Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 114\. ' scripts/contract.sh \
  && go test ./internal/guide/ -count=1 -v \
    -run 'TestEverySurfaceContainsTheSharedSentences|TestCLIContainsSharedAndTheOperatorTraps|TestCLITeachesAlwaysAndAPlan' 2>&1 | tee /tmp/adr062-t1.out \
  && grep -q '^--- PASS: TestEverySurfaceContainsTheSharedSentences' /tmp/adr062-t1.out \
  && grep -q '^--- PASS: TestCLIContainsSharedAndTheOperatorTraps' /tmp/adr062-t1.out \
  && grep -q '^--- PASS: TestCLITeachesAlwaysAndAPlan' /tmp/adr062-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr062-t1.out \
  && go test ./internal/mcp/ -count=1 -run 'TestTheDescriptionsSayWhenToReachForTheTool' \
  && [ -z "$(gofmt -l internal/guide internal/mcp)" ] \
  && go vet ./internal/guide/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEverySurfaceContainsTheSharedSentences` | `internal/guide/guide_test.go` | Shared() first sentence is always + plan | — | S1, S2 |
| `TestCLIContainsSharedAndTheOperatorTraps` | `internal/guide/guide_test.go` | CLI() still contains Shared() and the traps | — | S1, S2 |
| `TestCLITeachesAlwaysAndAPlan` | `internal/guide/guide_test.go` | CLI() has always + plan and `@@ path 0 create`; not 3+; Shared has no cookbook | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §114 |
| 2 — something selects it | `mrw instructions` prints `guide.CLI()`; handshake prepends `guide.Shared()`; restoring 3+ fails S1 and §114 |
| 3 — the caller can discover it | `mrw instructions --help` already names the command; AGENTS.md "Using mrw" matches Shared() |
| 4 — it is used | ADR-009 refuses telemetry; PATH agents are why this record exists |

## Mutation Log
(empty until execute)
- 2026-09-16 · 9e50697* · mutant killed · exit 1 · `internal/guide/guide.go` · Shared first sentence restored to the 3+ threshold: TestEverySurfaceContainsTheSharedSentences and TestCLITeachesAlwaysAndAPlan must go red · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · covers:Shared first sentence is always and a plan
- 2026-09-16 · 9e50697* · mutant killed · exit 1 · `internal/guide/guide.go` · CLI cookbook no longer names @@ path 0 create: TestCLITeachesAlwaysAndAPlan must go red · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · covers:CLI cookbook names @@ path 0 create
- 2026-09-16 · 9e50697* · mutant killed · exit 1 · `internal/guide/guide.go` · Shared absorbed the CLI cookbook: TestCLITeachesAlwaysAndAPlan must go red · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · covers:Shared does not carry the cookbook
- 2026-09-16 · 9e50697* · mutant killed · exit 1 · `internal/guide/guide.go` · CLI again teaches the 3+ threshold: TestCLITeachesAlwaysAndAPlan must go red · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · covers:3 or more edits is gone from CLI

## Invariants

- Shared() stays five sentences. The why is not a sixth.
- `maxInstructionsChars` stays 4096.
- Extra args to `mrw instructions` stay exit 2.
- MCP handshake does not contain `@@ path 0 create`.
- Engine packages are not edited.
- No §NN in this Tests table.

## Risks

- Handshake size: Shared grew one short sentence; cookbook is CLI-only.

## Stop Condition

If the only way to go green is to raise 4096, stop.
If the only way to go green is to put the cookbook on `initialize`, stop.
If the only way to go green is to keep a 3+ / "below that, don't" sentence on a live teaching surface, stop.

## Out of Scope

- Centralised skill POST (parent Follow-ups; BACKLOG receipt)
- Relocking ADR-037 T1–T3

## Verification Log
(empty until execute)
- 2026-09-16 · 9e50697* · exit 0 · `set -o pipefail …` · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · ms:2022
- 2026-09-16 · 9e50697* · exit 0 · `set -o pipefail …` · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · ms:1076
- 2026-09-16 · 9e50697* · exit 0 · `set -o pipefail …` · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · ms:1104
- 2026-09-16 · 9e50697* · exit 0 · `set -o pipefail …` · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · ms:1385
- 2026-09-16 · 9e50697* · exit 0 · `set -o pipefail …` · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · ms:1077
- 2026-09-16 · 9e50697* · exit 1 · `set -o pipefail …` · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · ms:765 · test-lock-sha256:e559394c1a5945585d40cd26f8269f82fbb186ac5f917222f904df21309305c4 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSUNvbnRhaW5zU2hhcmVkQW5kVGhlT3BlcmF0b3JUcmFwcwk2NTZiMWY2MzU0ZDRjYzY0NmEzNjAxZDBkYWUzY2Y4MmNkNzlhMGNhMGY2MTM0NzQ0ZTJkODU1ZjQwY2FlNzU1CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJVGVhY2hlc0Fsd2F5c0FuZEFQbGFuCTg3NmQ4ZDVlZWU2MGIwODk5ZjI0ODhmYzdhOGU3YzM4YzQ5Mzk1MWY0ZTRmODcxNDcxZGIzMGM1ZjBmN2U3MTEKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RFdmVyeVN1cmZhY2VDb250YWluc1RoZVNoYXJlZFNlbnRlbmNlcwljNGUzMTY5MzFlMzVmMTViZTg5Y2ZlNGVhM2E0ZjVkNTg2YjcyNzQxNDkxOTg2MGIzZGEzMTM2NzE1M2NmNzcw
  ```
  --- last 10 line(s) of stdout (of 58 after folding 58 raw)
          lines= is a guard on how many lines the ADDRESS covers, and is not body=.
          The checkout is named by global -C DIR or --root DIR before the subcommand (mrw -C repo write plan). After read, -C is context lines, not a checkout.
          Ops: replace, insert-after, insert-before, delete, create, unlink, rename.
          @@ path 0 create makes a new file; empty is body=0. A new file is not a reason to skip mrw.
          A multi-line replace requires anchor= taken from the served first line.
      guide_test.go:67: CLI still teaches the threshold that trains agents never to use mrw
  --- FAIL: TestCLITeachesAlwaysAndAPlan (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide	0.240s
  FAIL
  ```
- 2026-09-16 · 9e50697* · exit 0 · `set -o pipefail …` · acceptance-sha256:ff0203b62456a35d953a3ca2ca125df05626d69773b654ed655077c6a40892f9 · ms:2283
