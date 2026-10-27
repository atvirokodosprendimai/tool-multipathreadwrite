# Task ADR-175-T2: `docs/agents.md` gives the compaction hook and the re-read rule

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the hook recipe and the re-read rule in `docs/agents.md`; the flag in README and AGENTS.md
**Consumes:** `mrw seen --forget` (T1)
**Data dependency:** a live Claude Code session for S2's observation; the fence is hermetic
**Proof map:** v1
**Rests-on:** `the binary builds`, `the help names the flag`, `the help is read`, `the shell is strict`, `the hook block is extracted from its <!-- ADR-175 hook --> marker`, `the docs are read`, `the hook block parses and names the event, matcher and command`, `the hook reads cwd`, `docs/agents.md excludes --if-changed`, `docs/agents.md excludes --quiet`, `README names the flag`, `AGENTS.md names the flag`

## Goal

Decisions 3 and 4: a reader of `docs/agents.md` can install the Claude Code hook that, on `SessionStart` with matcher `compact|clear`, reads the hook input's `cwd` and runs `mrw --root "$cwd" seen --forget`; and knows that after it a caller-authored edit needs a re-read that shows the lines (not `--if-changed`, not `--quiet`), while `--sub`/`--rewrite` plans need none.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/agents.md` | edit | a section after §7 with the hook JSON and the re-read rule |
| `README.md` | edit | `mrw seen`'s paragraph names `--forget` |
| `AGENTS.md` | edit | §5's `mrw seen` bullet names `--forget` |

## Ordered Steps

1. [S1] The words, red first: the fence below fails until they exist. [proof: acceptance]
2. [S2] Install the hook in a scratch project, let Claude Code compact (`/compact`), and confirm the next write to a line read before it is refused; repeat after entering a git worktree of that project, confirming the worktree's ledger is the one forgotten; record the Claude Code version in the sign-off. [proof: human: a host compaction cannot be driven from a hermetic test]

## Acceptance

```bash
set -o pipefail
go build -o bin/mrw ./cmd/mrw \
  && h=$(./bin/mrw seen --help 2>&1) && grep -qF -- '--forget' <<<"$h" \
  && hook=$(awk '/<!-- ADR-175 hook -->/{f=1;next} f&&/^```json/{g=1;next} g&&/^```/{exit} g' docs/agents.md) \
  && jq -e '.hooks.SessionStart[0].matcher == "compact|clear" and (.hooks.SessionStart[0].hooks[0].command | test("seen --forget"))' <<<"$hook" >/dev/null \
  && jq -e '.hooks.SessionStart[0].hooks[0].command | test("\\.cwd")' <<<"$hook" >/dev/null \
  && grep -qF 'not --if-changed' docs/agents.md \
  && grep -qF 'not --quiet' docs/agents.md \
  && grep -qF 'mrw seen --forget' README.md \
  && grep -qF 'mrw seen --forget' AGENTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | prose; the fence greps it, and S2 is observed by hand | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the section |
| 2 — something selects it | a reader setting up an agent host |
| 3 — the caller can discover it | `docs/agents.md` is where AGENTS.md sends agent setups |
| 4 — it is used | not yet |

## Mutation Log
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `docs/agents.md` · S1: the hook matches compact only · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `docs/agents.md` · S1: the hook reads no cwd · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `docs/agents.md` · S1: the re-read rule loses --quiet · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc
- 2026-10-24 · 8278fcda* · mutant killed · exit 1 · `AGENTS.md` · S1: AGENTS.md does not name the flag · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc

## Invariants

- mrw installs no hook.

## Risks

- The hook's event names are the host's and may change; S2 records the version.

## Stop Condition

The fence exits 0 and S2 is signed off with the Claude Code version.

## Out of Scope

- Other hosts' hooks (deferred: docs/adr/BACKLOG.md, "ADR-175: compaction hooks for Codex and Cursor")

## Verification Log
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:439
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:1726
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:1447
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:1532
- 2026-10-24 · 8278fcda* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:1609
- 2026-10-27 · human-observed · S2 observed by the coordinator, 2026-10-27: Claude Code 2.1.311, mrw built from adr-175 39d6ff31, a scratch git project whose .claude/settings.json is the docs/agents.md hook block, driven headless (claude -p --setting-sources project). After mrw read f.txt, a startup session left the ledger row (the matcher does not fire on startup). claude -p --resume <id> /compact exit 0; the ledger then held no f.txt row and @@ f.txt 2 replace was refused (f.txt has not been read), exit 1, NOTHING WRITTEN. In a git worktree of the project, f.txt read in both checkouts, compacting a session started in the worktree emptied the worktree's ledger (1 to 0 rows), left the main checkout's (1 to 1), and the same write in the worktree was refused. · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:693
- 2026-10-27 · human-observed · S2 signed off by Zy, 2026-10-27, after re-observation by the coordinator with the guarded hook command d=$(jq -r '.cwd // empty'); [ -z "$d" ] || mrw --root "$d" seen --forget (review finding P3-4): Claude Code 2.1.311, mrw built from the adr-175 working tree, headless claude -p --setting-sources project. f.txt licensed by mrw read --quiet in a scratch checkout and in its git worktree (1 row each). A startup session forgot nothing (1). /compact in the checkout: its ledger 1 to 0, the worktree's kept 1, the next write refused exit 1. /compact of a session started in the worktree: the worktree's 1 to 0, the checkout's kept 1, the write refused exit 1. Hook input with no cwd ({}): exit 0, nothing forgotten (1). · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc
- 2026-10-27 · 39d6ff31* · exit 4 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:365 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-27 · 39d6ff31* · exit 4 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:237
  ```
  ```
- 2026-10-27 · 39d6ff31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:720
- 2026-10-27 · 4973c064* · exit 0 · `set -o pipefail …` · acceptance-sha256:ccef9a443465d86f66b5f4cfd48c88443896596aac28f266cc4f1cab69036abc · ms:270
