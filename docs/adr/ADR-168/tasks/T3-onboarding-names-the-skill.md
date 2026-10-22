# Task ADR-168-T3: onboarding names the skill

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the first help line, the last sentence of `mrw instructions`, the docs
**Consumes:** `mrw skill` (T1), `mrw skill --dir` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the help test run`, `the help test passed`, `the no-undeclared-flag test passed`, `README names --skill --dir`, `docs/agents.md names --skill --dir`, `AGENTS.md names --skill --dir`, `the engine packages it does not own`

## Goal

Decisions 4 and 5: `mrw --help` and a bare `mrw` open with `AGENTS: READ THE SKILL FIRST: mrw --skill`; `mrw instructions` ends naming `mrw skill`; README, AGENTS.md and `docs/agents.md` say how to install it, and that it goes beside the standing trigger.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `CustomRootCommandHelpTemplate`: the agents line, a blank line, then `cli.RootCommandHelpTemplate`; the root description's install line |
| `internal/guide/guide.go` | edit | one sentence at the end of `CLI()`, after the `--core` one (ADR-141's place for additions), naming `mrw skill` and not `--skill` |
| `cmd/mrw/skill168_test.go` | edit | the help test |
| `README.md` | edit | "Tell your agent to use it": `mrw --skill --dir`, beside the trigger, the hosts measured 2026-10-22 |
| `docs/agents.md` | edit | §1: a skill row and a paragraph — what loads when, keep the trigger |
| `AGENTS.md` | edit | §5's `mrw skill` bullet gains `--dir`; the "none of these commands writes to your checkout" paragraph names `mrw skill --dir` |

## Ordered Steps

1. [S1] Write `TestRootHelpPointsAgentsAtTheSkill`, and confirm RED. It checks:
   - Line 1 of `mrw --help` output, and of a bare `mrw`, is `AGENTS: READ THE SKILL FIRST: mrw --skill`.
   - The help names `mrw --skill --dir` and `mrw instructions --core`, and is at most 29 lines (25 measured on 2026-10-22).
   - `guide.CLI()` ends with a sentence naming `mrw skill` and contains no `--skill`.
   - `guide.Shared()` (the MCP handshake) contains no `skill`.
2. [S2] The template, the description line and the sentence. Mutants: the template prefix is deleted; the agents line is put after the help body instead of first; the MCP handshake (`Shared()`) names the skill. A mutant that moves the sentence inside the raw string is refused by `adr-verify` as comment-only, so the handshake mutant is written as code. [proof: mutation]
3. [S3] README, `docs/agents.md`, AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -timeout 300s -v -run 'TestRootHelpPointsAgentsAtTheSkill$|TestTheInstructionsNameNoFlagTheCLILacks$' 2>&1) \
  && grep -qE '^--- PASS: TestRootHelpPointsAgentsAtTheSkill \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheInstructionsNameNoFlagTheCLILacks \(' <<<"$out" \
  && grep -qF 'mrw --skill --dir' README.md \
  && grep -qF 'mrw --skill --dir' docs/agents.md \
  && grep -qF 'mrw --skill --dir' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestRootHelpPointsAgentsAtTheSkill` | `cmd/mrw/skill168_test.go` | the first line of help and of a bare `mrw`; the help's length; the full contract names the skill; the handshake does not | none | S1, S2 |
| `TestTheInstructionsNameNoFlagTheCLILacks` | `cmd/mrw/teach094_test.go` | the new sentence names no undeclared flag | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the template, the description line, the sentence |
| 2 — something selects it | `--help` and a bare `mrw` render the root template |
| 3 — the caller can discover it | it is the discovery |
| 4 — it is used | none; no telemetry (ADR-009) |

## Invariants

- `mrw instructions --core` and `guide.Shared()` are unchanged.
- Subcommand help (`mrw read --help`) is unchanged: the template is the root's only.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the root help would exceed 29 lines.

## Out of Scope

- Rewriting the trigger text in `docs/agents.md` (permanent: boundary: Decision 5 keeps it)

## Mutation Log
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the template prefix is deleted · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the agents line is not the first line (put after the help body) · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the MCP handshake (Shared) names the skill · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2

## Verification Log
- 2026-10-22 · 3a2e963e* · exit 1 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:385 · test-lock-sha256:5d2d4297f5cead622d601237bebee316eeb878fa00f2eafe0997dcba837969cc · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTQ2NjhlNjZhNjJjZGEzYzJmNzUzZjVhZDNmODcwNTI1YTExYWZkOTI4M2I1ZTg1Y2IxOTlhZTVjY2M1OGZmZQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bgk1Y2M3YmMwNzA1NmQzZGQxOWY4ZGU0ZTdmYjUyMjM3ZjIyYmQyMGE0NDAxMjE2M2Y1MDY5ZTJmYWZjMWU3Mzc0CmJvZHkJY21kL21ydy90ZWFjaDA5NF90ZXN0LmdvCVRlc3RFdmVyeVN1cmZhY2VUZWFjaGVzV2hhdEFTdGVwQ2hlY2tlZAk3NjBhNDg5NzM2OTMxMTA0N2EzYWU3YWNlMTE1Y2MzZjA0MmNkN2RiNzZmZGZiOThhMTNkZGViZWRiOWZiZTUzCmJvZHkJY21kL21ydy90ZWFjaDA5NF90ZXN0LmdvCVRlc3RUaGVJbnN0cnVjdGlvbnNOYW1lTm9GbGFnVGhlQ0xJTGFja3MJNzQ0ZjdmMjZmOTdlNzc0YzBmMWUyYmQ4ZTU1ODA0MzhmYWUzMWZiMmU4YzFjODZjOWRjYzFkOTE4MTM4NzQ4Nw
  ```
  ```
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:1181
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:1208
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:1568
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:341
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:407
- 2026-10-22 · 3a2e963e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:0 · test-lock-sha256:900101e725726b5c3b4bec249fbdeb9fa43a7ee502e4907e127f1ee8bb55c1ec · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTQ2NjhlNjZhNjJjZGEzYzJmNzUzZjVhZDNmODcwNTI1YTExYWZkOTI4M2I1ZTg1Y2IxOTlhZTVjY2M1OGZmZQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bglhNjEyMDE5MGYyNWM1ZmEwZWU5ODQ0MWU4MzY0Y2FhNjkzMWRlOTU3ZjYzMWEwYzNiNDE2ZjBmNTE0OTk5MTkzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpclJlYWRzT25seVRoZVNoYXBlSXRQcmludHMJZmMyYmRkNmJhY2NmNjc5NGIxMDdiNzU4YjVjM2U5OTIyYjUzYzQ0NDkyNDYyZmM1NWY0NGJhYjY2MGY0OTE5Ngpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlUmVwb3NpdG9yeVNraWxsSXNUaGVCaW5hcnlzU2tpbGwJMDhhYjNmODdiMGYwNzZkNGM2ZGNiNGE0NjVhMTIxYTVjMThmNTU5ZTY1YWIwMTFhYTYwOGVlZDQxNDhlMGJjNQpib2R5CWNtZC9tcncvdGVhY2gwOTRfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlVGVhY2hlc1doYXRBU3RlcENoZWNrZWQJNzYwYTQ4OTczNjkzMTEwNDdhM2FlN2FjZTExNWNjM2YwNDJjZDdkYjc2ZmRmYjk4YTEzZGRlYmVkYjlmYmU1Mwpib2R5CWNtZC9tcncvdGVhY2gwOTRfdGVzdC5nbwlUZXN0VGhlSW5zdHJ1Y3Rpb25zTmFtZU5vRmxhZ1RoZUNMSUxhY2tzCTc0NGY3ZjI2Zjk3ZTc3NGMwZjFlMmJkOGU1NTgwNDM4ZmFlMzFmYjJlOGMxYzg2YzlkY2MxZDkxODEzODc0ODc · test-lock-kind:replace
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:439
- 2026-10-22 · f323c1c8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:0 · test-lock-sha256:d4998d06a2410155964ee9d64b25dedb0fbf283481093e6b22af6c715e0289d8 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTllZTQ4Zjc3MjViYWNjYWEzMGM1Yjg3NTE3MzgwMGM1MGM3Nzg2ZWNmOTNlZWU3ZjMzNzg5MzNkZWY2NTZkYwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bglhNjEyMDE5MGYyNWM1ZmEwZWU5ODQ0MWU4MzY0Y2FhNjkzMWRlOTU3ZjYzMWEwYzNiNDE2ZjBmNTE0OTk5MTkzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpcktlZXBzV2hhdEFQZXJzb25DaGFuZ2VkCWFhMjA5MjljNmJkZjIzYTMwYjRhZGQ1MDBlMDRjZmU1OTE1ZGViNmU4MjJkOTNjMDY3YTY4YzQ0MzBkNjlmMDIKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlyUmVhZHNPbmx5VGhlU2hhcGVJdFByaW50cwlmYzJiZGQ2YmFjY2Y2Nzk0YjEwN2I3NThiNWMzZTk5MjJiNTNjNDQ0OTI0NjJmYzU1ZjQ0YmFiNjYwZjQ5MTk2CmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVSZXBvc2l0b3J5U2tpbGxJc1RoZUJpbmFyeXNTa2lsbAlhYzcyMTljOWYyNDYyZDBlNDVmZmU5ZTMyMWY1NDljZGUwYzIzYWMyY2YyYzFlNTZhNGMyNmRhYTkwODI4OWU3CmJvZHkJY21kL21ydy90ZWFjaDA5NF90ZXN0LmdvCVRlc3RFdmVyeVN1cmZhY2VUZWFjaGVzV2hhdEFTdGVwQ2hlY2tlZAk3NjBhNDg5NzM2OTMxMTA0N2EzYWU3YWNlMTE1Y2MzZjA0MmNkN2RiNzZmZGZiOThhMTNkZGViZWRiOWZiZTUzCmJvZHkJY21kL21ydy90ZWFjaDA5NF90ZXN0LmdvCVRlc3RUaGVJbnN0cnVjdGlvbnNOYW1lTm9GbGFnVGhlQ0xJTGFja3MJNzQ0ZjdmMjZmOTdlNzc0YzBmMWUyYmQ4ZTU1ODA0MzhmYWUzMWZiMmU4YzFjODZjOWRjYzFkOTE4MTM4NzQ4Nw · test-lock-kind:replace
- 2026-10-22 · f323c1c8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:0 · test-lock-sha256:d4998d06a2410155964ee9d64b25dedb0fbf283481093e6b22af6c715e0289d8 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTllZTQ4Zjc3MjViYWNjYWEzMGM1Yjg3NTE3MzgwMGM1MGM3Nzg2ZWNmOTNlZWU3ZjMzNzg5MzNkZWY2NTZkYwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bglhNjEyMDE5MGYyNWM1ZmEwZWU5ODQ0MWU4MzY0Y2FhNjkzMWRlOTU3ZjYzMWEwYzNiNDE2ZjBmNTE0OTk5MTkzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpcktlZXBzV2hhdEFQZXJzb25DaGFuZ2VkCWFhMjA5MjljNmJkZjIzYTMwYjRhZGQ1MDBlMDRjZmU1OTE1ZGViNmU4MjJkOTNjMDY3YTY4YzQ0MzBkNjlmMDIKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlyUmVhZHNPbmx5VGhlU2hhcGVJdFByaW50cwlmYzJiZGQ2YmFjY2Y2Nzk0YjEwN2I3NThiNWMzZTk5MjJiNTNjNDQ0OTI0NjJmYzU1ZjQ0YmFiNjYwZjQ5MTk2CmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVSZXBvc2l0b3J5U2tpbGxJc1RoZUJpbmFyeXNTa2lsbAlhYzcyMTljOWYyNDYyZDBlNDVmZmU5ZTMyMWY1NDljZGUwYzIzYWMyY2YyYzFlNTZhNGMyNmRhYTkwODI4OWU3CmJvZHkJY21kL21ydy90ZWFjaDA5NF90ZXN0LmdvCVRlc3RFdmVyeVN1cmZhY2VUZWFjaGVzV2hhdEFTdGVwQ2hlY2tlZAk3NjBhNDg5NzM2OTMxMTA0N2EzYWU3YWNlMTE1Y2MzZjA0MmNkN2RiNzZmZGZiOThhMTNkZGViZWRiOWZiZTUzCmJvZHkJY21kL21ydy90ZWFjaDA5NF90ZXN0LmdvCVRlc3RUaGVJbnN0cnVjdGlvbnNOYW1lTm9GbGFnVGhlQ0xJTGFja3MJNzQ0ZjdmMjZmOTdlNzc0YzBmMWUyYmQ4ZTU1ODA0MzhmYWUzMWZiMmU4YzFjODZjOWRjYzFkOTE4MTM4NzQ4Nw · test-lock-kind:replace
- 2026-10-22 · f323c1c8* · exit 0 · `set -o pipefail …` · acceptance-sha256:20684f0ca85b86ae35b134e09aa31cd560cdd2d9ef51dda9b34d75321f2e1aa2 · ms:360
