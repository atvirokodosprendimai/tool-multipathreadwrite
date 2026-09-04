# Task ADR-022-T1: The hook, its wiring, and the test that drives it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (one script, one settings entry, one Go test, one contract section)
**Owner:** unassigned
**Produces:** the hook, its wiring, and the Enforced-by test
**Consumes:** `$CLAUDE_PROJECT_DIR` and the PostToolUse contract (Claude Code), `mrw read`'s `==> path` header (unchanged), the plan header grammar (ADR-001, mirrored)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a rule arrives on a read the harness does not follow`, `the project root is not cwd`, `a header is read as mrw reads it`, `one delivery per rule per session even under a race`, `the hook never fails the turn`

## Goal

A session in this repository receives a path-scoped rule when a Bash, Write, `mrw_read` or `mrw_write`
call touches a file its globs match — from a subdirectory, through a grep, for a quoted plan path —
exactly once, and the hook can never take the turn down.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `.claude/hooks/rules-on-read.py` | edit | the eight decisions of the parent record |
| `.claude/settings.json` | unchanged | the wiring from #88's first cut stands |
| `cmd/mrw/ruleshook_test.go` | create | **the ADR's Enforced-by** — drives the hook as the harness does, from a subdirectory, with a second call in the same session |
| `scripts/contract.sh` | edit | §55 strengthened: decoded JSON, the settings entry, the root/grammar/race/closed-stdout cases |
| `AGENTS.md`, `CLAUDE.md` | edit | name the record |

## Ordered Steps

1. [S1] Write the failing test first (TDD red): the hook run with `cwd` = a subdirectory and
   `CLAUDE_PROJECT_DIR` = the project delivers the rule for `../docs/adr/x.md`; the current hook
   looks for rules under `cwd` and delivers nothing. [proof: acceptance]
2. [S2] Read the project root from `$CLAUDE_PROJECT_DIR`, else walk up from `cwd` to the nearest
   `.claude/rules`; resolve relative paths from `cwd` and relativise to the root. [proof: mutation]
3. [S3] Take served paths from every `==> path` header in the tool result as well as the named ones,
   so a grep delivers. [proof: mutation]
4. [S4] Read plan headers as `mrw` does: strip a BOM, quoted first field, every `body=N` honoured,
   `raw=true` without `body=` delivers nothing. [proof: mutation]
5. [S5] Match globs by segment: `**` at a boundary crosses directories, `*`/`?` within one, `{a,b}`
   alternates, slash-less is root-only; linear in the path. [proof: mutation]
6. [S6] Dedup by an atomic claim — `O_CREAT|O_EXCL|O_NOFOLLOW` under a `0700` cache directory keyed
   by session, agent, root and rule — swept after seven days; two racing hooks deliver once.
   [proof: mutation]
7. [S7] Exit 0 unconditionally, closed stdout included. [proof: acceptance]
8. [S8] §55: decode the JSON envelope exactly; assert the settings entry names an existing file; the
   subdirectory, quoted-path, counted-body, race and closed-stdout cases; state under the fixture.
   [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./cmd/mrw/ -run 'TestThePathScopedRulesHookDeliversOnAnMrwRead' -v 2>&1 | tee /tmp/adr022-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr022-t1.out \
  && [ "$(grep -cE '^--- (PASS|SKIP): (TestThePathScopedRulesHookDeliversOnAnMrwRead)\b' /tmp/adr022-t1.out)" = "1" ] \
  && grep -q '^# 55\.' scripts/contract.sh \
  && python3 -c 'import json; h=json.load(open(".claude/settings.json"))["hooks"]["PostToolUse"][0]; assert "mcp__mrw__mrw_read" in h["matcher"]; assert "rules-on-read.py" in h["hooks"][0]["command"]' \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal cmd/mrw/main.go)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal cmd/mrw/main.go \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name and the Go
identifier `ruleshook`. **§55 is the row** — it already existed on #88's first cut and is strengthened
here rather than renumbered. The test may SKIP where `python3` is absent (the Windows runner), which is
why the fence accepts `SKIP` for it; the contract row, on Linux, never skips.

**`internal` and `cmd/mrw/main.go` stay byte-identical**: this record adds a test file under `cmd/mrw`
and touches no engine code.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestThePathScopedRulesHookDeliversOnAnMrwRead` | `cmd/mrw/ruleshook_test.go` | **the ADR's Enforced-by** — from a subdirectory, a Bash mrw read of a globbed file delivers its rule once; a second call in the same session delivers nothing | — | S1, S2, S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `.claude/settings.json` wires it for every matching tool call; §55 asserts the entry names the file |
| 3 — the caller can discover it | the delivered context names the rule and the hook that sent it; `AGENTS.md` and `CLAUDE.md` say it exists |
| 4 — it is used | every session in this repository; the measurement that motivated it is issue #86 |

## Mutation Log
<!-- filled during execution -->
- 2026-09-04 · 48b2e9f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: take cwd for the project root. TestThePathScopedRulesHookDeliversOnAnMrwRead fails: from a subdirectory no rule is found — the first cut's defect · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 48b2e9f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S3: ignore the `==>` headers in the result. §55's grep row fails: a grep names no file in its input · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 48b2e9f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S4: stop stripping the BOM. §55's quoted-path-behind-a-BOM row fails · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 48b2e9f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: make `**` no longer cross directories. §55's `**/*_test.go` and `docs/adr/**` rows fail · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 48b2e9f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S6: let a losing claim deliver anyway. The Enforced-by's second call and §55's dedup and race rows fail · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370

## Invariants

- A rule is delivered once per session per agent per project, whichever of the four tools read the file.
- The project root is never assumed to be `cwd`.
- A plan header is read as `mrw` reads it.
- The hook exits 0 whatever happens.
- No engine change; `go.mod` declares exactly one requirement.

## Risks

- `python3` absent on a contributor's machine: the hook errors, nothing is delivered, which is the state before this record; `AGENTS.md` says so.
- The mirrored header grammar drifts from `internal/plan`. Mitigated by §55's cases and by ADR-001 owning the grammar; a drift is a defect here, not there.

## Stop Condition

Stop if delivering rules correctly needs the hook to read the plan through `internal/plan` — i.e. a Go
build. That is a different tool, and the record's Alternatives say why it was not chosen.

## Out of Scope

- Tools other than the four (permanent: boundary: parent ADR, Decision 1)
- Windows behaviour of the hook (deferred: docs/adr/BACKLOG.md — the rules-hook-on-Windows entry)
- Making the rules unconditional (permanent: boundary: parent ADR, Alternatives)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 48b2e9f · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:46599
