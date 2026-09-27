# Task ADR-088-T3: the analysis runs after every commit

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `.claude/hooks/static-after-commit.py`; `.claude/rules/static-analysis.md`; the hook's registration
**Consumes:** `scripts/static.sh` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a commit that moves HEAD reports the analysis`, `an unmoved HEAD runs nothing`, `the hook is registered`

## Goal

A finding waited for a manual pass; the model that introduced it was not told.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `.claude/hooks/static-after-commit.py` | new | the hook |
| `.claude/settings.json` | edit | registers it on Bash |
| `.claude/rules/static-analysis.md` | new | what to do with a finding, with hooks and without |
| `cmd/mrw/statichook_test.go` | new | drives the hook against a scratch repository |

## Ordered Steps

1. [S1] Write `TestTheStaticHookReportsAfterACommit` and `TestTheStaticHookIsQuietWhenHEADDidNotMove`; both fail on `d352ba7`, where the hook does not exist. [proof: mutation]
2. [S2] The hook, its registration and the rule. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestTheStaticHook' -v 2>&1 | tee /tmp/adr088-T3.out \
  && missing=$(for t in TestTheStaticHookReportsAfterACommit TestTheStaticHookIsQuietWhenHEADDidNotMove; do grep -qE "^--- PASS: $t \(" /tmp/adr088-T3.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && python3 -c 'import json,sys; s=json.load(open(".claude/settings.json")); sys.exit(0 if any("static-after-commit.py" in h["command"] for e in s["hooks"]["PostToolUse"] for h in e["hooks"]) else 1)' \
  && grep -q 'scripts/static.sh' .claude/rules/static-analysis.md \
  && [ -z "$(gofmt -l .)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheStaticHookReportsAfterACommit` | `cmd/mrw/statichook_test.go` | a commit that moved HEAD returns the script's verdict and output as context | — | S1, S2 |
| `TestTheStaticHookIsQuietWhenHEADDidNotMove` | `cmd/mrw/statichook_test.go` | the first call and an unmoved or non-commit HEAD run nothing | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `.claude/hooks/static-after-commit.py` |
| 2 — something selects it | `.claude/settings.json` PostToolUse on Bash |
| 3 — the caller can discover it | `.claude/rules/static-analysis.md`, loaded every session |
| 4 — it is used | every commit in a Claude Code session in this checkout |

## Verification Log
(empty until execute)
- 2026-09-27 · d352ba7* · exit 1 · `set -o pipefail …` · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · ms:5347 · test-lock-sha256:4df96a869464dc74da249af698639b048bcc87cb4b0adb2b770ddc6430257b2d · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvc3RhdGljaG9va190ZXN0LmdvCVRlc3RUaGVTdGF0aWNIb29rSXNRdWlldFdoZW5IRUFERGlkTm90TW92ZQk2ZGFlYjBhMTEzYjRkMGRkMTk3YzM0ZjZhOTg3NjY5NjYxYjZiMmU1YTg4NDlkOWEzZTQ1MDZhNzA4NDBjOTFhCmJvZHkJY21kL21ydy9zdGF0aWNob29rX3Rlc3QuZ28JVGVzdFRoZVN0YXRpY0hvb2tSZXBvcnRzQWZ0ZXJBQ29tbWl0CTU3OGU2NDBlM2Y2MDQ2NGY1MWUzODllMzEzMzgyZWEwNWEwNjdhYTQxMjEzMGFiOTc0OGY2NTNlNmZjODhkN2Q
  ```
  --- last 9 line(s) of stdout
  === RUN   TestTheStaticHookReportsAfterACommit
      statichook_test.go:20: .claude/settings.json registers no static-after-commit.py hook on Bash
  --- FAIL: TestTheStaticHookReportsAfterACommit (0.00s)
  === RUN   TestTheStaticHookIsQuietWhenHEADDidNotMove
      statichook_test.go:39: .claude/settings.json registers no static-after-commit.py hook on Bash
  --- FAIL: TestTheStaticHookIsQuietWhenHEADDidNotMove (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.216s
  FAIL
  ```
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · ms:2670
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · ms:1890
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · ms:5531
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · ms:7857

## Mutation Log
(empty until execute)
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `.claude/hooks/static-after-commit.py` · the verdict is computed and never handed over · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · covers:a commit that moves HEAD reports the analysis
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `.claude/hooks/static-after-commit.py` · an unmoved HEAD is analysed again · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · covers:an unmoved HEAD runs nothing
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `.claude/settings.json` · the registration names another file · acceptance-sha256:c1c33bf3c78551ce62011e1072aeeba48f763643974477bc32d3e9e9ffb1dec6 · covers:the hook is registered

## Invariants

- The hook exits 0 whatever happens; it never takes the turn down.

## Risks

- None beyond the record's.

## Out of Scope

- The same trigger for Codex and Cursor sessions (permanent: boundary: the rule tells an agent without hooks to run the script after committing)

## Stop Condition

The fence exits 0.
