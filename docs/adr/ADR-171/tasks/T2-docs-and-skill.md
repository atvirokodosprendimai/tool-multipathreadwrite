# Task ADR-171-T2: the docs and the skill say what the binary says, and README shows a non-Go check

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the new trigger sentences in `docs/agents.md`; README's Ruby `scoped_check` example; AGENTS.md's recipe 5; the regenerated `.claude/skills/mrw/SKILL.md`; the BACKLOG entry
**Consumes:** the new core rule 4, core header and recipe 5 text (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the binary builds`, `the checked-in skill is the binary's`, `docs/agents.md says when to re-read`, `docs/agents.md allows a plan file through the host's file tool`, `README shows a rubocop scoped_check`, `AGENTS.md's recipe 5 sends a plan file`, `the BACKLOG receipt exists`

## Goal

ADR-171 Decisions 2-4 where adopters read them: the trigger and the subagent brief in `docs/agents.md`, README's check section, AGENTS.md's recipe block, the checked-in skill; and the receipt for the record's BACKLOG deferral.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/agents.md` | edit | the trigger (`:51-52`), the paragraph after it (`:59-62`), and one sentence in §1 and §2 that a plan file is the caller's input and may be written with the host's file tool |
| `README.md` | edit | the check section: a tree that declares no check gets none; the `rubocop` `scoped_check` example |
| `AGENTS.md` | edit | recipe 5 in the recipe block (`:152`) |
| `.claude/skills/mrw/SKILL.md` | regenerate | `mrw skill --dir .claude/skills` from the T1 binary |
| `docs/adr/BACKLOG.md` | edit | "A one-line answer to a repeated `mrw instructions`" under a new "From ADR-171" heading |

## Ordered Steps

1. [S1] Run the fence and see it RED against today's docs. [proof: acceptance]
2. [S2] Edit the documents, add the BACKLOG entry, regenerate the skill. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go build -o bin/mrw ./cmd/mrw \
  && out=$(go test ./cmd/mrw/ -count=1 -v -run '^TestTheRepositorySkillIsTheBinarysSkill$' 2>&1) && grep -q '^--- PASS: TestTheRepositorySkillIsTheBinarysSkill ' <<<"$out" \
  && grep -qF 'After a compaction, re-read the lines you will edit' docs/agents.md \
  && grep -qF 'host'"'"'s file tool' docs/agents.md \
  && grep -qF 'bundle exec rubocop --force-exclusion {files}' README.md \
  && grep -qF 'mrw write "$TMPDIR/plan.txt"' AGENTS.md \
  && grep -qF 'From ADR-171' docs/adr/BACKLOG.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRepositorySkillIsTheBinarysSkill` | `cmd/mrw/skill168_test.go` | the checked-in skill is what the binary prints | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the sentences in the files |
| 2 — something selects it | adopters copy the `docs/agents.md` trigger |
| 3 — the caller can discover it | README and `docs/agents.md` |
| 4 — it is used | not measured here: the external team's re-run is the measure |

## Mutation Log
(empty until execute)
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `AGENTS.md` · S2 AGENTS.md recipe 5 back to stdin · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7

## Invariants

- No other recipe changes.

## Risks

- None beyond ADR-171's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The owner's host triggers outside this repository (permanent: boundary: the owner changes them)

## Verification Log
(empty until execute)
- 2026-10-24 · 8278fcda* · exit 1 · `set -o pipefail …` · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7 · ms:539 · test-lock-sha256:4e14e8b9c49e36a21a834de72b7d56225fb1f7c897241505f2ccee75115958d8 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3REaXJSZWFzb25LZWVwc1RoZUVuZ2luZVJlYXNvbldob2xlCWE0ZjUxZTk1NjJhY2I3NjY5MDdmYTE4NGQ5ZGU2OWRiYmMzZDQ1NzY5NTk5MDcyNTdkMzI3OTM2YjFiOGZmYzYKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdEtlcHREaWZmZXJlbmNlTmFtZXNPbmx5V2hhdElzS2VwdAllZWFiMjg3N2JjZWZjZjE0MGNmNTlmNTk2OGY3NDY5Zjg2NmVhMTJjN2ZhODg1ZTcxMTRkYWM1NmZjMjQyOWIzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RSb290SGVscFBvaW50c0FnZW50c0F0VGhlU2tpbGwJM2NjZmQwOWY1YmUyMzBkN2YxMTU3YzU0YWYwNzM1MzEyOGE4ODU5OThlY2RiYzMyZWNjMTM0NWNjYmI4YjhlMQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTllZTQ4Zjc3MjViYWNjYWEzMGM1Yjg3NTE3MzgwMGM1MGM3Nzg2ZWNmOTNlZWU3ZjMzNzg5MzNkZWY2NTZkYwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJJbnN0YWxsc0FuZFJlcGxhY2VzT25seUl0c093bglhNjEyMDE5MGYyNWM1ZmEwZWU5ODQ0MWU4MzY0Y2FhNjkzMWRlOTU3ZjYzMWEwYzNiNDE2ZjBmNTE0OTk5MTkzCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpcktlZXBzVGhlVHJpZ2dlckFuZFRoZUJPTQkzMGI2N2Q2NzQ0YmI0MjhkMDU0NThlYTZlYzk5OTlhOGNhMzhkOWU2MDc4Nzk4Zjc3ZDRlM2Y4ZmMxZjdjMzc4CmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpcktlZXBzV2hhdEFQZXJzb25DaGFuZ2VkCTRkNmNjYjU1YTFkYzVjNmMyODkxNDYyNTQ0MjNkMzkxM2NiNzIwOTA3MjZlM2NjZmQyZGU5YWI2MWI5M2M3ZjkKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlyUmVhZHNPbmx5VGhlU2hhcGVJdFByaW50cwlmYzJiZGQ2YmFjY2Y2Nzk0YjEwN2I3NThiNWMzZTk5MjJiNTNjNDQ0OTI0NjJmYzU1ZjQ0YmFiNjYwZjQ5MTk2CmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVSZXBvc2l0b3J5U2tpbGxJc1RoZUJpbmFyeXNTa2lsbAlhYzcyMTljOWYyNDYyZDBlNDVmZmU5ZTMyMWY1NDljZGUwYzIzYWMyY2YyYzFlNTZhNGMyNmRhYTkwODI4OWU3CmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RVbmtlcHRDb21wb25lbnRGaW5kc0FUcmFpbGluZ0RvdE9yU3BhY2UJNjUwNDEyYjRlYWRiY2YyZWMyOTZlOTY1MDAzODg0MGU4ZTRhMzAxZTQyNmQ4NzliZjE2NTkwZDk1YmIzMDVlOA
  ```
  ```
- 2026-10-24 · 8278fcda* · exit 1 · `set -o pipefail …` · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7 · ms:568
  ```
  ```
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7 · ms:483
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7 · ms:1787
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7 · ms:775
- 2026-10-24 · 954b7562* · exit 0 · `set -o pipefail …` · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7 · ms:823
- 2026-10-27 · e9186d50* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:a3fe73c466aeca3fc703be3c41f1dcdc9e1a775c36d1300ca3a2cc4d46cbe8f7 · ms:0 · test-lock-sha256:f4c85f11f7e5d6ba73786044d1aa35cd9f6b79054bda8547881274bf40254d22 · test-lock-file:docs/adr/locks/f4c85f11f7e5d6ba73786044d1aa35cd9f6b79054bda8547881274bf40254d22.lock · test-lock-kind:replace
