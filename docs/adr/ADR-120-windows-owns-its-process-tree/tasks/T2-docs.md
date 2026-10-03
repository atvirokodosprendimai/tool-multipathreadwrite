# Task ADR-120-T2: the docs say a Windows child stops with mrw

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the Windows clause of AGENTS.md's steps paragraph
**Consumes:** `subproc.runInJob` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the docs say a Windows child stops with mrw`

## Goal

AGENTS.md, README and the record trail say that on Windows a step's background children are killed with it and none can leave, where they said none are.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `AGENTS.md`, `README.md` | edit | the Windows clause |
| `docs/adr/ADR-080-*.md`, `docs/adr/ADR-095-*.md` | edit | a dated note that ADR-120 supersedes the Windows boundary |
| `docs/adr/BACKLOG.md` | edit | the job-object deferral closed |

## Ordered Steps

1. [S1] Confirm the fence is RED: AGENTS.md still says "on Windows none are". [proof: acceptance]
2. [S2] The edits. [proof: mutation]

## Acceptance

```bash
set -o pipefail
! grep -q 'on Windows none are' AGENTS.md \
  && grep -q 'on Windows they are killed with it too (a job object, ADR-120)' AGENTS.md \
  && grep -q 'ADR-120' README.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | prose, checked by the fence | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the sentence |
| 2 — something selects it | AGENTS.md is the tool guide every agent reads |
| 3 — the caller can discover it | the centralised `mrw` skill mirrors it |
| 4 — it is used | the Windows peers read it |

## Mutation Log
- 2026-10-02 · e6be523* · mutant killed · exit 1 · `AGENTS.md` · S2: the Windows clause says the job kills them · acceptance-sha256:22151b30de5fdaadb6bc42333893211d6f9aca38748fdb71fc7245a1eccafbf1

## Invariants

- The ast-grep sentence "on Windows it is killed at 2 s" stays true and unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a doc must promise more than the windows CI shards showed.

## Out of Scope

- The skill resync (permanent: boundary: the release drill resyncs the skill, step 7 of .claude/skills/release/SKILL.md, not a task)

## Verification Log
- 2026-10-02 · e6be523* · exit 0 · `set -o pipefail …` · acceptance-sha256:22151b30de5fdaadb6bc42333893211d6f9aca38748fdb71fc7245a1eccafbf1 · ms:93
- 2026-10-02 · fd17179* · exit 0 · `set -o pipefail …` · acceptance-sha256:22151b30de5fdaadb6bc42333893211d6f9aca38748fdb71fc7245a1eccafbf1 · ms:34
- 2026-10-02 · fd17179* · exit 1 · `set -o pipefail …` · acceptance-sha256:22151b30de5fdaadb6bc42333893211d6f9aca38748fdb71fc7245a1eccafbf1 · ms:31 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-02 · fd17179* · exit 0 · `set -o pipefail …` · acceptance-sha256:22151b30de5fdaadb6bc42333893211d6f9aca38748fdb71fc7245a1eccafbf1 · ms:39
- 2026-10-02 · 68615a6* · exit 0 · `set -o pipefail …` · acceptance-sha256:22151b30de5fdaadb6bc42333893211d6f9aca38748fdb71fc7245a1eccafbf1 · ms:70
