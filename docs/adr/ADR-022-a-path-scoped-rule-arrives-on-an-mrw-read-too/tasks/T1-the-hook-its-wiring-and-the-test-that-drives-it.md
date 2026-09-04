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
once per session while its claim can be filed and on every call when it cannot, and the hook can
never take the turn down.

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
   `.claude/rules`, stopping at the first `.git`; resolve a Bash command's paths from `cwd` and every
   other tool's from the root, one base per call, and relativise to the root. [proof: mutation]
3. [S3] Take served paths from every `==> path` header in the tool result as well as the named ones,
   the path read up to the two spaces mrw prints after it, so a grep delivers and a space survives.
   [proof: mutation]
4. [S4] Read plan headers as `mrw` does: `splitHeader` ported, `Parse`'s and `parseHeader`'s refusals
   mirrored (a BOM stripped once per line, every `body=N` honoured, `raw=true` without `body=`, a
   duplicate or unknown guard, an unknown op and an unterminated quote all deliver nothing), a
   pattern checked for shape and not compiled. [proof: mutation]
5. [S5] Match globs by segment in an enumerated grammar: `**` at a boundary crosses directories,
   `*`/`?` within one, flat `{a,b}` expanded before the split, slash-less is root-only; a table
   whose cost is the product of the segment counts. [proof: mutation]
6. [S6] Dedup by an atomic claim — `O_CREAT|O_EXCL|O_NOFOLLOW` under a `0700` cache directory keyed
   by session, agent, root and rule — swept after seven days; the base absolute and outside the
   project, else no claim and a delivery; two racing hooks deliver once. [proof: mutation]
7. [S7] Exit 0 unconditionally, closed stdout included; stdin read whole. [proof: acceptance]
8. [S8] §55: decode the JSON envelope exactly; take the hook from the settings entry (exactly the
   four tools, the command resolved through `CLAUDE_PROJECT_DIR`) and drive that file; the
   subdirectory, quoted-path, counted-body, race and closed-stdout cases; the differential grammar
   rows; the spaced header; the relative, in-tree and unusable state bases; the nested `.git`; the
   300-globstar × 400-directory alarm. [proof: acceptance]

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
- 2026-09-04 · ee58dbb · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: make `**` no longer cross directories (it degrades to `*`). ⚠ THE FIRST ATTEMPT SURVIVED and was logged as killed before the log was corrected: every globbed fixture sat exactly one directory deep, where `*` and `**` agree. §55 gained a root-level `top_test.go` (zero directories) and `pkg/sub/deep_test.go` (two), and the mutant now fails both · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 48b2e9f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S6: let a losing claim deliver anyway. The Enforced-by's second call and §55's dedup and race rows fail · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0df776f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: drop the `.git` stop in the walk-up. §55's nested-repository row fails: the enclosing project's rule is delivered into `inner/` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0df776f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S3: cut the served path at the first whitespace (`\S+`) again. §55's spaced-header row fails: `docs/adr/my file.md` is lost · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0df776f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S4: tokenise headers with `shlex` in place of the `splitHeader` port. §55 fails twice: the single-quoted path is stripped and delivers, and the pattern address with spaces splits the header so the op is `(s` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0df776f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: make `**` no longer cross directories (`if False:` on the globstar branch). §55's zero-directory, two-directory and 400-directory rows fail · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0df776f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: put the memoised recursion back in place of the table. ⚠ IT SURVIVED TWICE FIRST: a 200-globstar fixture under a 5 s alarm let it through at 1.6 s, and a 500-globstar one killed it by Python's recursion limit rather than by time — the wrong reason, with the alarm row passing. Measured, then sized: 300 globstars × 400 directories under a 1 s alarm; the recursion takes 2.3 s through the hook and exit 142 fails the row, the table takes 40 ms · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0df776f · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S6: accept a relative cache base. §55's relative-base row fails: `rel55/` is created under cwd while the delivery still happens · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370

## Invariants

- A rule is delivered once per session per agent per project while its claim can be filed, and on every matching call when it cannot — never on none — whichever of the four tools read the file.
- The project root is never assumed to be `cwd`; a Bash path resolves from `cwd`, every other from the root, and no path from both.
- A plan header is tokenised as `mrw` tokenises it, and a plan mrw refuses delivers nothing, a bad pattern excepted.
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
- 2026-09-04 · ee58dbb* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:30941
- 2026-09-04 · 0df776f* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:25969
