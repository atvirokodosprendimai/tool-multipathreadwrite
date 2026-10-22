# Task ADR-168-T5: what allowing mrw allows is said to the caller and the agent

**Depends-on:** T1, T4
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the skill's permissions sentence, README's and `docs/agents.md`'s "What allowing mrw allows"
**Consumes:** `guide.Skill(version)` (T1), the generated `.claude/skills/mrw/SKILL.md` (T4)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guide test run`, `the permissions test passed`, `the repository skill test run`, `the repository skill test passed`, `docs/agents.md has the section`, `README says it`, `docs/agents.md says it`, `the engine packages it does not own`

## Goal

Decision 6: the caller who configures a harness, and the agent that loads the skill, are told that allowing mrw allows writes anywhere the user can write and shell through `--then-sh`, `mrw check` and `--then`, and that a host's per-path Edit and Write rules do not reach it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | one sentence in `Skill`'s closing paragraph |
| `internal/guide/skill168_test.go` | edit | the permissions test |
| `.claude/skills/mrw/SKILL.md` | regenerate | `mrw skill --dir .claude/skills`, so T4's equality test holds |
| `README.md` | edit | a "Permissions" paragraph under "Tell your agent to use it" |
| `docs/agents.md` | edit | §7, "What allowing mrw allows" |

## Ordered Steps

1. [S1] Write `TestTheSkillSaysHostPathRulesDoNotReachMrw`: the closing paragraph of `Skill` says a host's per-path rules for its Edit and Write tools do not reach mrw, that their absence is not permission, and that mrw is not a way to reach a path the host refused. Confirm RED.
2. [S2] The sentence in `Skill`, then `mrw skill --dir .claude/skills`. Mutant: the sentence is dropped from `Skill`. [proof: mutation]
3. [S3] README and `docs/agents.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ -count=1 -timeout 300s -v -run 'TestTheSkillSaysHostPathRulesDoNotReachMrw$' 2>&1) \
  && grep -qE '^--- PASS: TestTheSkillSaysHostPathRulesDoNotReachMrw \(' <<<"$out" \
  && out=$(go test ./cmd/mrw/ -count=1 -timeout 300s -v -run 'TestTheRepositorySkillIsTheBinarysSkill$' 2>&1) \
  && grep -qE '^--- PASS: TestTheRepositorySkillIsTheBinarysSkill \(' <<<"$out" \
  && grep -qF '## 7. What allowing mrw allows' docs/agents.md \
  && grep -qF 'per-path rules for its Edit and Write tools do not reach' README.md \
  && grep -qF 'per-path rules for its Edit and Write tools do not reach' docs/agents.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSkillSaysHostPathRulesDoNotReachMrw` | `internal/guide/skill168_test.go` | the skill tells the agent that host path rules do not reach mrw and are not permission | none | S1, S2 |
| `TestTheRepositorySkillIsTheBinarysSkill` | `cmd/mrw/skill168_test.go` | the in-repo skill carries the sentence, regenerated | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the sentence; the two doc sections |
| 2 — something selects it | `mrw skill` prints it; a host loads the skill's body |
| 3 — the caller can discover it | README "Tell your agent to use it"; `docs/agents.md` §7 |
| 4 — it is used | none; no telemetry (ADR-009) |

## Invariants

- No behaviour changes: mrw's refusals and its reach are as they were; this task says what they are.
- `mrw instructions` and the MCP handshake are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if saying this needs a claim about a specific host's permission engine that was not measured: the text names what mrw does, not how each host evaluates a rule.

## Out of Scope

- How each host evaluates a rule for a shell command (permanent: boundary: the host owns its permission engine; the text says what mrw does when it runs)

## Mutation Log
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the permissions sentence is dropped from Skill · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade

## Verification Log
- 2026-10-22 · 3a2e963e* · exit 1 · `set -o pipefail …` · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade · ms:188 · test-lock-sha256:78abbfe2d0c221370e8f6ea4d2974aa4e463906021b1f834a9b49371c91247d4 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTQ2NjhlNjZhNjJjZGEzYzJmNzUzZjVhZDNmODcwNTI1YTExYWZkOTI4M2I1ZTg1Y2IxOTlhZTVjY2M1OGZmZQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bglhNjEyMDE5MGYyNWM1ZmEwZWU5ODQ0MWU4MzY0Y2FhNjkzMWRlOTU3ZjYzMWEwYzNiNDE2ZjBmNTE0OTk5MTkzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpclJlYWRzT25seVRoZVNoYXBlSXRQcmludHMJZmMyYmRkNmJhY2NmNjc5NGIxMDdiNzU4YjVjM2U5OTIyYjUzYzQ0NDkyNDYyZmM1NWY0NGJhYjY2MGY0OTE5Ngpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlUmVwb3NpdG9yeVNraWxsSXNUaGVCaW5hcnlzU2tpbGwJMDhhYjNmODdiMGYwNzZkNGM2ZGNiNGE0NjVhMTIxYTVjMThmNTU5ZTY1YWIwMTFhYTYwOGVlZDQxNDhlMGJjNQpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsQm9keUlzVGhlQ29yZUFuZFRoZVJlY2lwZXMJOTUzMjYwOWY4YTE5YzE5NDVhYTdjNTQyZDQxZDU2YmNiYTg2OGJlNGY3YzJiMTAyNTY1OWU5ZjUzMmY3YTI4Mgpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsSXNBVmFsaWRBZ2VudFNraWxsCWE3YTM2ZmM4MTY1Y2QxODMwODg1MTM5ZjA2YzExN2E1YjMzNTg2YmVjMjI4ODNlOWJlMWY3ZjJiOThiZjZiZmMKYm9keQlpbnRlcm5hbC9ndWlkZS9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVTa2lsbFNheXNIb3N0UGF0aFJ1bGVzRG9Ob3RSZWFjaE1ydwk3NThjMDQ0M2JjMjhlM2Q4ZDIzNTgyOWIxNjA2Zjg1YWNmNDgyNGNmYjRlNjBlOTAwNmRjNmFkYzlkMGNhY2Ux
  ```
  ```
- 2026-10-22 · 3a2e963e* · exit 1 · `set -o pipefail …` · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade · ms:2007
  ```
  ```
- 2026-10-22 · 3a2e963e* · exit 1 · `set -o pipefail …` · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade · ms:817
  ```
  ```
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade · ms:2009
- 2026-10-22 · f323c1c8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade · ms:0 · test-lock-sha256:546bfe2316968d2c64a4d68ce73a97599532c2d91996b55152c4d38d032834e2 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTllZTQ4Zjc3MjViYWNjYWEzMGM1Yjg3NTE3MzgwMGM1MGM3Nzg2ZWNmOTNlZWU3ZjMzNzg5MzNkZWY2NTZkYwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bglhNjEyMDE5MGYyNWM1ZmEwZWU5ODQ0MWU4MzY0Y2FhNjkzMWRlOTU3ZjYzMWEwYzNiNDE2ZjBmNTE0OTk5MTkzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpcktlZXBzV2hhdEFQZXJzb25DaGFuZ2VkCWFhMjA5MjljNmJkZjIzYTMwYjRhZGQ1MDBlMDRjZmU1OTE1ZGViNmU4MjJkOTNjMDY3YTY4YzQ0MzBkNjlmMDIKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlyUmVhZHNPbmx5VGhlU2hhcGVJdFByaW50cwlmYzJiZGQ2YmFjY2Y2Nzk0YjEwN2I3NThiNWMzZTk5MjJiNTNjNDQ0OTI0NjJmYzU1ZjQ0YmFiNjYwZjQ5MTk2CmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVSZXBvc2l0b3J5U2tpbGxJc1RoZUJpbmFyeXNTa2lsbAlhYzcyMTljOWYyNDYyZDBlNDVmZmU5ZTMyMWY1NDljZGUwYzIzYWMyY2YyYzFlNTZhNGMyNmRhYTkwODI4OWU3CmJvZHkJaW50ZXJuYWwvZ3VpZGUvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlU2tpbGxCb2R5SXNUaGVDb3JlQW5kVGhlUmVjaXBlcwk5NTMyNjA5ZjhhMTljMTk0NWFhN2M1NDJkNDFkNTZiY2JhODY4YmU0ZjdjMmIxMDI1NjU5ZTlmNTMyZjdhMjgyCmJvZHkJaW50ZXJuYWwvZ3VpZGUvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlU2tpbGxJc0FWYWxpZEFnZW50U2tpbGwJOTU5MGZiZjI5ZTcxMTJjNDk2ZDlhZTM5M2FlMTUwNGUxMDIxZjkyYzg3OWEyMzQ1MDY3ZTM5MDVmZWFjZjY2Mwpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsU2F5c0hvc3RQYXRoUnVsZXNEb05vdFJlYWNoTXJ3CTc1OGMwNDQzYmMyOGUzZDhkMjM1ODI5YjE2MDZmODVhY2Y0ODI0Y2ZiNGU2MGU5MDA2ZGM2YWRjOWQwY2FjZTE · test-lock-kind:replace
- 2026-10-22 · f323c1c8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade · ms:0 · test-lock-sha256:546bfe2316968d2c64a4d68ce73a97599532c2d91996b55152c4d38d032834e2 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTllZTQ4Zjc3MjViYWNjYWEzMGM1Yjg3NTE3MzgwMGM1MGM3Nzg2ZWNmOTNlZWU3ZjMzNzg5MzNkZWY2NTZkYwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bglhNjEyMDE5MGYyNWM1ZmEwZWU5ODQ0MWU4MzY0Y2FhNjkzMWRlOTU3ZjYzMWEwYzNiNDE2ZjBmNTE0OTk5MTkzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpcktlZXBzV2hhdEFQZXJzb25DaGFuZ2VkCWFhMjA5MjljNmJkZjIzYTMwYjRhZGQ1MDBlMDRjZmU1OTE1ZGViNmU4MjJkOTNjMDY3YTY4YzQ0MzBkNjlmMDIKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlyUmVhZHNPbmx5VGhlU2hhcGVJdFByaW50cwlmYzJiZGQ2YmFjY2Y2Nzk0YjEwN2I3NThiNWMzZTk5MjJiNTNjNDQ0OTI0NjJmYzU1ZjQ0YmFiNjYwZjQ5MTk2CmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVSZXBvc2l0b3J5U2tpbGxJc1RoZUJpbmFyeXNTa2lsbAlhYzcyMTljOWYyNDYyZDBlNDVmZmU5ZTMyMWY1NDljZGUwYzIzYWMyY2YyYzFlNTZhNGMyNmRhYTkwODI4OWU3CmJvZHkJaW50ZXJuYWwvZ3VpZGUvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlU2tpbGxCb2R5SXNUaGVDb3JlQW5kVGhlUmVjaXBlcwk5NTMyNjA5ZjhhMTljMTk0NWFhN2M1NDJkNDFkNTZiY2JhODY4YmU0ZjdjMmIxMDI1NjU5ZTlmNTMyZjdhMjgyCmJvZHkJaW50ZXJuYWwvZ3VpZGUvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlU2tpbGxJc0FWYWxpZEFnZW50U2tpbGwJOTU5MGZiZjI5ZTcxMTJjNDk2ZDlhZTM5M2FlMTUwNGUxMDIxZjkyYzg3OWEyMzQ1MDY3ZTM5MDVmZWFjZjY2Mwpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsU2F5c0hvc3RQYXRoUnVsZXNEb05vdFJlYWNoTXJ3CTc1OGMwNDQzYmMyOGUzZDhkMjM1ODI5YjE2MDZmODVhY2Y0ODI0Y2ZiNGU2MGU5MDA2ZGM2YWRjOWQwY2FjZTE · test-lock-kind:replace
- 2026-10-22 · f323c1c8* · exit 0 · `set -o pipefail …` · acceptance-sha256:65622eb9d930be4308ca3f00509235eeacd57ae2405ef1c2d37bb84c440d7ade · ms:506
