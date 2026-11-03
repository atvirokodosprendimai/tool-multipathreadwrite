# Task ADR-193-T5: on SHIP, mrw instructions --host claude prints the measured paragraph

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `mrw instructions --host claude`, `guide.ClaudeEdits`, contract section 333
**Consumes:** the frozen paragraph (T3), the verdict line (T4)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the verdict is SHIP`, `the host form is the text S ran`, `an unknown host is refused`, `without --host nothing changes`, `the binary prints it`, `the contract row exists`, `the Claude Code trigger names it`

## Goal

`mrw instructions --host claude` prints `CLI()`, a newline and the paragraph S ran, `--core --host claude` prints
`Core()`, a newline and the same paragraph, any other host is refused, and `docs/agents.md` §1 tells Claude Code
hosts to run the host form (ADR-193 Decision 3).

This task runs only when T4's `verdict.txt` ends `verdict: SHIP` (the fence's first clause). On NO SHIP it is marked
`blocked` with that line cited, and nothing in the guide changes.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `ClaudeEdits()`, the paragraph; its flags held to the binary as `Core()`'s are |
| `cmd/mrw/main.go` | edit | `instructionsCmd`: `--host`, taking `claude` only; usage text |
| `cmd/mrw/instructions_test.go` | edit | the two tests |
| `internal/guide/core141_test.go` | edit | the paragraph's flags are flags of `write` or `read` |
| `scripts/contract.sh` | edit | section 333 (the next free on 2026-11-03; renumber in this fence before its first run if taken): the binary's `--host claude` output ends in the paragraph, and `--host codex` exits 2 |
| `docs/agents.md` | edit | §1's Claude Code trigger runs `mrw instructions --core --host claude` |
| `AGENTS.md` | edit | §5's `mrw instructions` bullet names `--host claude` |
| `README.md` | edit | one line beside `mrw instructions` |
| `docs/adr/BACKLOG.md` | edit | the memoryd `craft` skill mirror refreshed by whoever merges |

## Ordered Steps

1. [S1] Write the two failing tests and the contract row and see them red. [proof: acceptance]
2. [S2] `ClaudeEdits()` equal to `bench.EditParagraph` (the test compares them), and `--host` in `instructionsCmd`. [proof: mutation]
3. [S3] An unknown host is a usage error, exit 2, naming `claude`. [proof: mutation]
4. [S4] Section 333 drives `$MRW`: the good case and the refused one. [proof: mutation]
5. [S5] The docs: `docs/agents.md` §1, AGENTS.md §5, README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
V=docs/bench/stage-8/verdict.txt C=scripts/contract.sh G=docs/agents.md
grep -qx 'verdict: SHIP' "$V" \
  && out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestInstructionsForClaudeIsTheTextConfigurationSMeasured|TestInstructionsRefusesAHostItDoesNotKnow)$' 2>&1) \
  && for t in TestInstructionsForClaudeIsTheTextConfigurationSMeasured TestInstructionsRefusesAHostItDoesNotKnow; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go test ./internal/guide/ ./cmd/mrw/ ./internal/bench/ -count=1 \
  && grep -q '^# 333\. ' "$C" \
  && grep -q 'instructions --core --host claude' "$G"
```

The file operands are variables on purpose: `scripts/fence-prose.py` runs every task's plain grep over a tracked file,
pending or done, so a literal operand would be red until T5 lands — and the verdict clause red for good on NO SHIP.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInstructionsForClaudeIsTheTextConfigurationSMeasured` | `cmd/mrw/instructions_test.go` | `--host claude` prints `guide.CLI()`, a newline and `bench.EditParagraph`; with `--core`, `guide.Core()`, a newline and the same; without `--host`, `CLI()` and `Core()` unchanged | none | S2 |
| `TestInstructionsRefusesAHostItDoesNotKnow` | `cmd/mrw/instructions_test.go` | `--host codex` and `--host ''` exit 2 naming `claude`, printing nothing on standard output | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | the `--host` flag in `instructionsCmd`; section 333 drives the binary |
| 3 — the caller can discover it | `mrw instructions --help`, `docs/agents.md` §1 |
| 4 — it is used | Claude Code sessions whose trigger follows §1; the evidence for building it is T4's verdict, not adoption (ADR-009) |

## Mutation Log

## Invariants

- `mrw instructions` and `mrw instructions --core` without `--host` are byte-identical to v1.83.0's.
- `Core()` stays within ADR-141's 300 words.

## Risks

- The `--core --host claude` form was not measured (ADR-193 Decision 3); Zy may ship the full form only.

## Stop Condition

`ClaudeEdits()` cannot equal the frozen paragraph without changing it: stop to Zy, since the measured text is the one that ships.

## Out of Scope

- `mrw skill --host` (ADR-193 Out of Scope).

## Verification Log
