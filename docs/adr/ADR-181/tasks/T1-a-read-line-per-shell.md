# Task ADR-181-T1: the Windows run receipt prints a labelled read line per shell

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `runReadLines`; `TestTheWindowsRunReceiptLabelsAReadLineForEachShell`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Decisions 1-2: on Windows the receipt prints `read (PowerShell):`, `read (cmd):` and `read (Git Bash):` lines, the first byte-identical to ADR-177's; unix prints its one `read:` line unchanged; no Windows line when a path holds `"` or `%`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/run177_test.go` | edit | the red test beside `:137-167`'s string tests; the existing Windows-string assertion (`:150-151`) keeps the PowerShell form; the empty-log assertion (`:212`) rejects every read-line label, not only `read:` |
| `cmd/mrw/run177.go` | edit | `runReadLine` (`:232`) returns labelled lines for Windows; `printRunReceipt` (`:215`) prints each |
| `README.md` | edit | `:683-688` names the three Windows lines |
| `AGENTS.md` | edit | the `mrw run` entry names them |
| `internal/guide/guide.go` | edit | recipe 12 (`:66`): "a four-line receipt" says the Windows receipt names a read line per shell |
| `.claude/skills/mrw/SKILL.md` | edit | regenerated from `mrw skill` after the guide changes (`:54`) |
| `docs/agents.md` | edit | `:151`, the `mrw run` row: the same |

## Ordered Steps

1. [S1] Write `TestTheWindowsRunReceiptLabelsAReadLineForEachShell`: for `C:\Program Files\mrw\mrw.exe` and `C:\u's\$x\b`c.log`, 100 lines, the Windows lines are exactly the three of Decision 1 (`-:61-$`), in order, labelled; for linux the one `read:` line is unchanged; a `"` or `%` in either path prints none; a `!` prints the PowerShell and Git Bash lines and no cmd line. See it RED.
2. [S2] The lines. Mutant: the cmd line keeps PowerShell's backtick escapes — the test must go red. [proof: mutation]
3. [S3] Mutant: the Git Bash line double-quotes the paths — the `$x` path must make it go red. [proof: mutation]
4. [S4] The empty-log assertion rejects every label; README, AGENTS.md, the guide's recipe 12, `docs/agents.md`, and `.claude/skills/mrw/SKILL.md` regenerated. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestTheWindowsRunReceiptLabelsAReadLineForEachShell)$' 2>&1) \
  && grep -qE '^--- PASS: TestTheWindowsRunReceiptLabelsAReadLineForEachShell \(' <<<"$out" \
  && grep -qF 'read (Git Bash):' README.md \
  && grep -qF 'read (Git Bash)' AGENTS.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWindowsRunReceiptLabelsAReadLineForEachShell` | `cmd/mrw/run177_test.go` | the three labelled Windows lines, the unchanged unix line, none for `"`/`%` | none | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every `mrw run` on Windows with a non-empty log |
| 3 — the caller can discover it | the receipt itself; README, AGENTS.md |
| 4 — it is used | the v1.77.1 Windows validation pasted the line into Git Bash and cmd, where it failed (memoryd 73843a76) |

## Mutation Log
- 2026-10-29 · 9611d43b* · mutant killed · exit 1 · `cmd/mrw/run177.go` · S3: the Git Bash line double-quotes the paths, so $x expands · acceptance-sha256:e8a920860b373d4b5be607c800891759697e5dc7e2807caf32cfed587208c3a9
- 2026-10-29 · 9611d43b* · mutant killed · exit 1 · `cmd/mrw/run177.go` · S2: the cmd line keeps the PowerShell backtick escape on $ · acceptance-sha256:e8a920860b373d4b5be607c800891759697e5dc7e2807caf32cfed587208c3a9

## Invariants

- The PowerShell line is ADR-177's byte for byte; the unix line is unchanged.
- No line for a path holding `"` or `%`.
- `--json` is unchanged.

## Risks

- None beyond ADR-181's.

## Stop Condition

The fence exits 0. A contract row is not possible here: `scripts/contract.sh` is POSIX shell on Linux, where the unix line is unchanged; T2 is this task's binary-through-the-shell proof.

## Out of Scope

- Running the lines in their shells (deferred: docs/adr/ADR-181/tasks/T2-each-line-runs-in-its-shell.md)

## Verification Log
- 2026-10-29 · 9611d43b* · exit 0 · `set -o pipefail …` · acceptance-sha256:e8a920860b373d4b5be607c800891759697e5dc7e2807caf32cfed587208c3a9 · ms:3267
- 2026-10-29 · 9611d43b* · exit 0 · `set -o pipefail …` · acceptance-sha256:e8a920860b373d4b5be607c800891759697e5dc7e2807caf32cfed587208c3a9 · ms:3635
- 2026-10-29 · 9611d43b* · exit 0 · `set -o pipefail …` · acceptance-sha256:e8a920860b373d4b5be607c800891759697e5dc7e2807caf32cfed587208c3a9 · ms:3032
- 2026-10-29 · 9611d43b* · exit 1 · `set -o pipefail …` · acceptance-sha256:e8a920860b373d4b5be607c800891759697e5dc7e2807caf32cfed587208c3a9 · ms:400 · test-lock-sha256:71ecd533c6396d31a395ed6d43b5985258e5fd9aed9af9de5153e6911676eedc · test-lock-file:docs/adr/locks/71ecd533c6396d31a395ed6d43b5985258e5fd9aed9af9de5153e6911676eedc.lock
  ```
  ```
- 2026-10-29 · 9611d43b* · exit 0 · `set -o pipefail …` · acceptance-sha256:e8a920860b373d4b5be607c800891759697e5dc7e2807caf32cfed587208c3a9 · ms:1546
