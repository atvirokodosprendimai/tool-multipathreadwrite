# Task ADR-193-T5: on SHIP, mrw instructions --host claude prints the measured paragraph

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `mrw instructions --host claude`, `guide.ClaudeEdits`, contract section 333
**Consumes:** the frozen paragraph (T3), the verdict line (T4)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the verdict is SHIP`, `the host form is the text S ran`, `an unknown host is refused`, `the core takes no host`, `without --host nothing changes`, `the contract row exists`, `the docs name it`

## Goal

`mrw instructions --host claude` prints `CLI()`, a newline and the paragraph S ran; any other host, and `--core` with
`--host`, is a usage error, exit 2; `docs/agents.md` names the flag as an option for Claude hosts (ADR-193 Decision 3).

This task runs only when T4's `verdict.txt` ends `verdict: SHIP` (the fence's first clause). On NO SHIP it is marked
`blocked` with that line cited, and nothing in the guide changes. It does not edit the standing trigger in
`docs/agents.md` §1, the skill, or the owner's global instructions (ADR-193 Decision 3).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `ClaudeEdits()`, the paragraph; its flags held to the binary as `Core()`'s are |
| `cmd/mrw/main.go` | edit | `instructionsCmd`: `--host`, taking `claude` only and refused beside `--core`; usage text |
| `cmd/mrw/instructions_test.go` | edit | the three tests |
| `internal/guide/core141_test.go` | edit | the paragraph's flags are flags of `write` or `read` |
| `scripts/contract.sh` | edit | section 333 (the next free on 2026-11-03; renumber in this fence before its first run if taken): the binary's `--host claude` output ends in the paragraph, and `--host codex` and `--core --host claude` exit 2 |
| `docs/agents.md` | edit | one line outside §1's trigger: Claude hosts may run `mrw instructions --host claude` for the full text with the measured paragraph |
| `AGENTS.md` | edit | §5's `mrw instructions` bullet names `--host claude` |
| `README.md` | edit | one line beside `mrw instructions` |

## Ordered Steps

1. [S1] Write the three failing tests and the contract row and see them red. [proof: acceptance]
2. [S2] `ClaudeEdits()` equal to `bench.EditParagraph` (the test compares them), and `--host` in `instructionsCmd`. [proof: mutation]
3. [S3] An unknown host, and `--core` with `--host`, are usage errors, exit 2, naming `claude` and that the core has no host form. [proof: mutation]
4. [S4] Section 333 drives `$MRW`: the good case and both refusals. The fence checks the section exists; the row itself runs in `./scripts/contract.sh`, which `lifecycle.md` §6 requires before the commit. [proof: mutation]
5. [S5] The docs: one line in `docs/agents.md`, AGENTS.md §5, README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
V=docs/bench/stage-8/verdict.txt C=scripts/contract.sh G=docs/agents.md
grep -qx 'verdict: SHIP' "$V" \
  && out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestInstructionsForClaudeIsTheTextConfigurationSMeasured|TestInstructionsRefusesAHostItDoesNotKnow|TestInstructionsRefusesCoreWithAHost)$' 2>&1) \
  && for t in TestInstructionsForClaudeIsTheTextConfigurationSMeasured TestInstructionsRefusesAHostItDoesNotKnow TestInstructionsRefusesCoreWithAHost; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go test ./internal/guide/ ./cmd/mrw/ ./internal/bench/ -count=1 \
  && grep -q '^# 333\. ' "$C" \
  && grep -q 'instructions --host claude' "$G"
```

The file operands are variables on purpose: `scripts/fence-prose.py` runs every task's plain grep over a tracked file,
pending or done, so a literal operand would be red until T5 lands — and the verdict clause red for good on NO SHIP.
The cost: once T5 is done, fence-prose no longer guards these two prose greps.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | T5 runs only on SHIP (Decision 5), so its tests do not exist until it starts and are not named in this table before then (`internal/adversarial/record_test.go` holds every named test to existing). The list below is what S1 writes; each enters this table, in this row's place, in the commit that starts T5 | none | — |

The tests S1 writes, in `cmd/mrw/instructions_test.go`:
- TestInstructionsForClaudeIsTheTextConfigurationSMeasured: `--host claude` prints `guide.CLI()`, a newline and `bench.EditParagraph`; without `--host`, `instructions` and `--core` print what the same build's `CLI()` and `Core()` are (S2).
- TestInstructionsRefusesAHostItDoesNotKnow: `--host codex` and `--host ''` exit 2 naming `claude`, printing nothing on standard output (S3).
- TestInstructionsRefusesCoreWithAHost: `--core --host claude` exits 2 saying the core has no host form, printing nothing on standard output (S3).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests |
| 2 — something selects it | the `--host` flag in `instructionsCmd`; section 333 drives the binary |
| 3 — the caller can discover it | `mrw instructions --help`, the line in `docs/agents.md` |
| 4 — it is used | Claude Code hosts that choose it; the evidence for building it is T4's verdict, not adoption (ADR-009) |

## Mutation Log

## Invariants

- `mrw instructions` and `mrw instructions --core` without `--host` print what the same build prints without the flag: `CLI()` and `Core()`, unchanged by this task.
- `Core()` stays within ADR-141's 300 words, counted by `strings.Fields`.

## Risks

- A Claude host that keeps running `--core` sees no change; wiring the paragraph into the triggers is Zy's decision after SHIP (BACKLOG).
- S's prompt ends without a final newline (`EditParagraph` has none, T1); `--host claude` prints the same bytes, so a test that expects a trailing newline measures other text than S ran.

## Stop Condition

`ClaudeEdits()` cannot equal the frozen paragraph without changing it: stop to Zy, since the measured text is the one that ships.

## Out of Scope

- `--core --host claude`, the triggers and the skill (ADR-193 Out of Scope).

## Verification Log
