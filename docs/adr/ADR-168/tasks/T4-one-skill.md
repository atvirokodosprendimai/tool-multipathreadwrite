# Task ADR-168-T4: the binary's skill is the one skill

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the generated `.claude/skills/mrw/SKILL.md`, release §7's new step, the refreshed memoryd `craft` skill `mrw`
**Consumes:** `guide.Skill(version)` (T1), `mrw skill --dir` (T2)
**Data dependency:** hermetic for the gate; S4 writes memoryd, outside the repository
**Proof map:** v1
**Rests-on:** `the equality test run`, `the equality test passed`, `the skill-names-instructions test passed`, `ADR-063's description clause`, `ADR-161's recipe literal`, `the in-repo file is generated`, `the release step installs with --dir`, `the hand mirror step is gone`, `measure.sh moved`, `the dev note moved`, `the engine packages it does not own`

## Goal

Decision 3: `.claude/skills/mrw/SKILL.md` is `mrw skill`'s output, held equal by test to `guide.Skill(v)` where `v` is the file's own `mrw-version`. The repository facts it carried that AGENTS.md lacks move to AGENTS.md. Release §7 refreshes memoryd's `craft` skill `mrw` from the tagged binary, and regenerates the in-repo file with `--dir`. The memoryd skill is refreshed now from the merged build. The three gates that read the file stay green.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/skill168_test.go` | edit | the equality test |
| `.claude/skills/mrw/SKILL.md` | replace | removed with `mrw write` (it has no `generated-by` marker, so `--dir` refuses it by design), then written by `mrw skill --dir .claude/skills` |
| `AGENTS.md` | edit | the facts the old file held and AGENTS.md lacks: `scripts/measure.sh`; a `bin/mrw` built from a working tree reports `dev`, newer than any tag |
| `.claude/skills/release/SKILL.md` | edit | §7: `mrw skill` from the tagged binary to memoryd `craft`, and `mrw skill --dir .claude/skills`, replace the hand mirror |
| `cmd/mrw/agentsdoc_test.go` | edit | the comment and message that say the centralised skill mirrors AGENTS.md (`:19-23`, `:53`) |

## Ordered Steps

1. [S1] Write `TestTheRepositorySkillIsTheBinarysSkill`: read `mrw-version` from `.claude/skills/mrw/SKILL.md` and compare the file to `guide.Skill` of that value. Confirm RED against today's file.
2. [S2] List each fact in the old file and where AGENTS.md or the new skill carries it. Move to AGENTS.md those neither carries. Remove the old file with `mrw write` (`@@ .claude/skills/mrw/SKILL.md - unlink`), then run `mrw skill --dir .claude/skills` from the built binary. Mutant: one line of the in-repo body edited by hand. [proof: mutation]
3. [S3] Release §7 and the `agentsdoc_test.go` wording. [proof: acceptance]
4. [S4] Refresh memoryd's `craft` skill `mrw` from `mrw skill` of the merged build (`update_skill`). Record the new skill version in the sign-off. [proof: human: memoryd is outside the repository; the sign-off names the craft skill version and the mrw version it was printed by]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -timeout 300s -v -run 'TestTheRepositorySkillIsTheBinarysSkill$|TestReadmeAndSkillNameInstructions$' 2>&1) \
  && grep -qE '^--- PASS: TestTheRepositorySkillIsTheBinarysSkill \(' <<<"$out" \
  && grep -qE '^--- PASS: TestReadmeAndSkillNameInstructions \(' <<<"$out" \
  && python3 -c "import sys,re; t=open('.claude/skills/mrw/SKILL.md').read(); d=re.search(r'^description: >-\n((?:  .*\n)+)', t, re.M).group(1); d=' '.join(l.strip() for l in d.splitlines()); sys.exit(0 if '--grep' in d and len(d.encode()) < 1024 else 1)" \
  && grep -qF "mrw write --grep '\bOld\b' --sub 'New' ." .claude/skills/mrw/SKILL.md \
  && grep -qF 'generated-by: "mrw skill"' .claude/skills/mrw/SKILL.md \
  && grep -qF 'mrw skill --dir' .claude/skills/release/SKILL.md \
  && ! grep -qF 'then mirror it' .claude/skills/release/SKILL.md \
  && grep -qF 'scripts/measure.sh' AGENTS.md \
  && grep -qF 'newer than any tag' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRepositorySkillIsTheBinarysSkill` | `cmd/mrw/skill168_test.go` | the in-repo skill is the binary's, for its own stamp | none | S1, S2 |
| `TestReadmeAndSkillNameInstructions` | `cmd/mrw/agentsdoc_test.go` | the generated file still names `mrw instructions` | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the generated file; the refreshed memoryd skill |
| 2 — something selects it | Claude Code loads `.claude/skills/mrw/SKILL.md` in this checkout; memoryd `load_skill("mrw")` elsewhere |
| 3 — the caller can discover it | the skill list a session is shown |
| 4 — it is used | none; no telemetry (ADR-009) |

## Invariants

- No fact the old file carried is lost: each is in AGENTS.md or the new skill (S2's list, in the PR body).
- ADR-063 T1's and ADR-161 T4's done fences stay green.

## Risks

- The memoryd refresh is a release step after this one: a release that skips it leaves `craft` one behind, and the stamp now says so.

## Stop Condition

Stop and ask if a fact in the old file belongs neither in AGENTS.md nor in the skill.

## Out of Scope

- A gate on the memoryd copy (permanent: boundary: memoryd is outside the repository; release §7 and the stamp carry it)

## Mutation Log
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `.claude/skills/mrw/SKILL.md` · S2: one line of the in-repo body edited by hand · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1

## Verification Log
- 2026-10-22 · 3a2e963e* · exit 1 · `set -o pipefail …` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:413 · test-lock-sha256:dd8af014a5250acce427aaebe4ee08fdd5c16ea85a43a7725ba16b2a050ce7a8 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJYzM4MzI4OTM0MDM3ZGYxNTlhYWFkMjlmN2Q5NjIyNWYyNjc0Yjc3MzkzNDNiNTE0YjZmODJlNDYwMDRkOWU2Ywpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0Um9vdEhlbHBQb2ludHNBZ2VudHNBdFRoZVNraWxsCTNjY2ZkMDlmNWJlMjMwZDdmMTE1N2M1NGFmMDczNTMxMjhhODg1OTk4ZWNkYmMzMmVjYzEzNDVjY2JiOGI4ZTEKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsQ29tbWFuZFByaW50c1RoZVNraWxsCTk0NjY4ZTY2YTYyY2RhM2MyZjc1M2Y1YWQzZjg3MDUyNWExMWFmZDkyODNiNWU4NWNiMTk5YWU1Y2NjNThmZmUKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlySW5zdGFsbHNBbmRSZXBsYWNlc09ubHlJdHNPd24JNWNjN2JjMDcwNTZkM2RkMTlmOGRlNGU3ZmI1MjIzN2YyMmJkMjBhNDQwMTIxNjNmNTA2OWUyZmFmYzFlNzM3NApib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlUmVwb3NpdG9yeVNraWxsSXNUaGVCaW5hcnlzU2tpbGwJMDhhYjNmODdiMGYwNzZkNGM2ZGNiNGE0NjVhMTIxYTVjMThmNTU5ZTY1YWIwMTFhYTYwOGVlZDQxNDhlMGJjNQ
  ```
  ```
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:1149
- 2026-10-22 · 3a2e963e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:0 · test-lock-sha256:a7d9a2868ab39ec64b56a813d218290c7a01e65a72635b07c4b8ac77d7bd958d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJNmZjNzQ4MmZjMDE4ZTRkMjRiMWI0ZDFkNWRhNDhlOTZhYTJhNzZiOTI2ZDM2ODEzNmQ1NjA5MWFkYWQ5OTRkOQpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0Um9vdEhlbHBQb2ludHNBZ2VudHNBdFRoZVNraWxsCTNjY2ZkMDlmNWJlMjMwZDdmMTE1N2M1NGFmMDczNTMxMjhhODg1OTk4ZWNkYmMzMmVjYzEzNDVjY2JiOGI4ZTEKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsQ29tbWFuZFByaW50c1RoZVNraWxsCTk0NjY4ZTY2YTYyY2RhM2MyZjc1M2Y1YWQzZjg3MDUyNWExMWFmZDkyODNiNWU4NWNiMTk5YWU1Y2NjNThmZmUKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlySW5zdGFsbHNBbmRSZXBsYWNlc09ubHlJdHNPd24JNWNjN2JjMDcwNTZkM2RkMTlmOGRlNGU3ZmI1MjIzN2YyMmJkMjBhNDQwMTIxNjNmNTA2OWUyZmFmYzFlNzM3NApib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlUmVwb3NpdG9yeVNraWxsSXNUaGVCaW5hcnlzU2tpbGwJMDhhYjNmODdiMGYwNzZkNGM2ZGNiNGE0NjVhMTIxYTVjMThmNTU5ZTY1YWIwMTFhYTYwOGVlZDQxNDhlMGJjNQ · test-lock-kind:replace
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:375
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:379
- 2026-10-22 · 3a2e963e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:0 · test-lock-sha256:100afdf2a3a8cb4056666595c05e4a941c532cf59e44cd4277c2fb6d0ecd67c1 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJNmZjNzQ4MmZjMDE4ZTRkMjRiMWI0ZDFkNWRhNDhlOTZhYTJhNzZiOTI2ZDM2ODEzNmQ1NjA5MWFkYWQ5OTRkOQpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0Um9vdEhlbHBQb2ludHNBZ2VudHNBdFRoZVNraWxsCTNjY2ZkMDlmNWJlMjMwZDdmMTE1N2M1NGFmMDczNTMxMjhhODg1OTk4ZWNkYmMzMmVjYzEzNDVjY2JiOGI4ZTEKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsQ29tbWFuZFByaW50c1RoZVNraWxsCTk0NjY4ZTY2YTYyY2RhM2MyZjc1M2Y1YWQzZjg3MDUyNWExMWFmZDkyODNiNWU4NWNiMTk5YWU1Y2NjNThmZmUKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlySW5zdGFsbHNBbmRSZXBsYWNlc09ubHlJdHNPd24JYTYxMjAxOTBmMjVjNWZhMGVlOTg0NDFlODM2NGNhYTY5MzFkZTk1N2Y2MzFhMGMzYjQxNmYwZjUxNDk5OTE5Mwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJSZWFkc09ubHlUaGVTaGFwZUl0UHJpbnRzCWZjMmJkZDZiYWNjZjY3OTRiMTA3Yjc1OGI1YzNlOTkyMmI1M2M0NDQ5MjQ2MmZjNTVmNDRiYWI2NjBmNDkxOTYKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVJlcG9zaXRvcnlTa2lsbElzVGhlQmluYXJ5c1NraWxsCTA4YWIzZjg3YjBmMDc2ZDRjNmRjYjRhNDY1YTEyMWE1YzE4ZjU1OWU2NWFiMDExYWE2MDhlZWQ0MTQ4ZTBiYzU · test-lock-kind:replace
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:708
- 2026-10-22 · f323c1c8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:0 · test-lock-sha256:0edf77c5ed21f813a606be884c1ca606bad40fe2354c453c313d90009c2c2eb5 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJNmZjNzQ4MmZjMDE4ZTRkMjRiMWI0ZDFkNWRhNDhlOTZhYTJhNzZiOTI2ZDM2ODEzNmQ1NjA5MWFkYWQ5OTRkOQpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0Um9vdEhlbHBQb2ludHNBZ2VudHNBdFRoZVNraWxsCTNjY2ZkMDlmNWJlMjMwZDdmMTE1N2M1NGFmMDczNTMxMjhhODg1OTk4ZWNkYmMzMmVjYzEzNDVjY2JiOGI4ZTEKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsQ29tbWFuZFByaW50c1RoZVNraWxsCTk5ZWU0OGY3NzI1YmFjY2FhMzBjNWI4NzUxNzM4MDBjNTBjNzc4NmVjZjkzZWVlN2YzMzc4OTMzZGVmNjU2ZGMKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlySW5zdGFsbHNBbmRSZXBsYWNlc09ubHlJdHNPd24JYTYxMjAxOTBmMjVjNWZhMGVlOTg0NDFlODM2NGNhYTY5MzFkZTk1N2Y2MzFhMGMzYjQxNmYwZjUxNDk5OTE5Mwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJLZWVwc1doYXRBUGVyc29uQ2hhbmdlZAlhYTIwOTI5YzZiZGYyM2EzMGI0YWRkNTAwZTA0Y2ZlNTkxNWRlYjZlODIyZDkzYzA2N2E2OGM0NDMwZDY5ZjAyCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpclJlYWRzT25seVRoZVNoYXBlSXRQcmludHMJZmMyYmRkNmJhY2NmNjc5NGIxMDdiNzU4YjVjM2U5OTIyYjUzYzQ0NDkyNDYyZmM1NWY0NGJhYjY2MGY0OTE5Ngpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlUmVwb3NpdG9yeVNraWxsSXNUaGVCaW5hcnlzU2tpbGwJYWM3MjE5YzlmMjQ2MmQwZTQ1ZmZlOWUzMjFmNTQ5Y2RlMGMyM2FjMmNmMmMxZTU2YTRjMjZkYWE5MDgyODllNw · test-lock-kind:replace
- 2026-10-22 · f323c1c8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:0 · test-lock-sha256:0edf77c5ed21f813a606be884c1ca606bad40fe2354c453c313d90009c2c2eb5 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJNmZjNzQ4MmZjMDE4ZTRkMjRiMWI0ZDFkNWRhNDhlOTZhYTJhNzZiOTI2ZDM2ODEzNmQ1NjA5MWFkYWQ5OTRkOQpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0Um9vdEhlbHBQb2ludHNBZ2VudHNBdFRoZVNraWxsCTNjY2ZkMDlmNWJlMjMwZDdmMTE1N2M1NGFmMDczNTMxMjhhODg1OTk4ZWNkYmMzMmVjYzEzNDVjY2JiOGI4ZTEKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsQ29tbWFuZFByaW50c1RoZVNraWxsCTk5ZWU0OGY3NzI1YmFjY2FhMzBjNWI4NzUxNzM4MDBjNTBjNzc4NmVjZjkzZWVlN2YzMzc4OTMzZGVmNjU2ZGMKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlySW5zdGFsbHNBbmRSZXBsYWNlc09ubHlJdHNPd24JYTYxMjAxOTBmMjVjNWZhMGVlOTg0NDFlODM2NGNhYTY5MzFkZTk1N2Y2MzFhMGMzYjQxNmYwZjUxNDk5OTE5Mwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJLZWVwc1doYXRBUGVyc29uQ2hhbmdlZAlhYTIwOTI5YzZiZGYyM2EzMGI0YWRkNTAwZTA0Y2ZlNTkxNWRlYjZlODIyZDkzYzA2N2E2OGM0NDMwZDY5ZjAyCmJvZHkJY21kL21ydy9za2lsbDE2OF90ZXN0LmdvCVRlc3RTa2lsbERpclJlYWRzT25seVRoZVNoYXBlSXRQcmludHMJZmMyYmRkNmJhY2NmNjc5NGIxMDdiNzU4YjVjM2U5OTIyYjUzYzQ0NDkyNDYyZmM1NWY0NGJhYjY2MGY0OTE5Ngpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlUmVwb3NpdG9yeVNraWxsSXNUaGVCaW5hcnlzU2tpbGwJYWM3MjE5YzlmMjQ2MmQwZTQ1ZmZlOWUzMjFmNTQ5Y2RlMGMyM2FjMmNmMmMxZTU2YTRjMjZkYWE5MDgyODllNw · test-lock-kind:replace
- 2026-10-22 · f323c1c8* · exit 0 · `set -o pipefail …` · acceptance-sha256:87cdfc8106ee1f0cc8b914757d1ace20829c6819a5723ab25e51bdeb7aa13ae1 · ms:366
