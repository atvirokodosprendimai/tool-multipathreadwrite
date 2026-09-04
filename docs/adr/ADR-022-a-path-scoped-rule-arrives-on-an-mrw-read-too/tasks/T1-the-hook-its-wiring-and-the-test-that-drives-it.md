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
call touches a file its globs match and the call's own shape is one this hook reads — from a
subdirectory, through a grep, for a quoted plan path — once per session while its claim can be filed
and on every call when it cannot, and the hook can never take the turn down. A shape outside that
reading, such as a subshell that changes directory, delivers nothing and is named in the record.

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
   `.claude/rules`, stopping at the first `.git`; resolve a Bash command's operands — each offered
   both as written and with an mrw range stripped — from EVERY directory the command may have run in
   (`cwd`, a leading `cd`, every `env --chdir`), and the headers mrw printed from those crossed with
   EVERY root any mrw in the command was given, mrw found by NAME anywhere in it and control
   operators split out of the tokens; every other tool's paths from the project root, and relativise
   to the root. [proof: mutation]
3. [S3] Take served paths from every `==> path  NL  NB  sha …` header in the tool result as well as
   the named ones, the path read back from that suffix, so a grep delivers and any spaces survive.
   [proof: mutation]
4. [S4] Tokenise plan headers as `mrw` does — `splitHeader` ported, a BOM stripped once per line —
   and take the first field of every header-shaped line the tokeniser can split as a candidate,
   bodies included; do not mirror
   whether mrw accepts the plan (a refused plan delivers early, never silently). [proof: mutation]
5. [S5] Match globs by segment in an enumerated grammar: `**` at a boundary crosses directories,
   `*`/`?` within one by a two-pointer walk, flat `{a,b}` expanded before the split, a pattern
   holding a group inside a group literal (decided for the whole pattern first), an inline list's
   comment stripped before its brackets, a trailing `/` names no file, slash-less is a file at the
   root except a bare `**`, which is the whole tree; a table whose cost is the product of the segment
   counts. [proof: mutation]
6. [S6] Dedup by an atomic claim — `O_CREAT|O_EXCL|O_NOFOLLOW` under a `0700` cache directory keyed
   by session, agent, root and rule — swept after seven days; the base absolute and outside the
   project, else no claim and a delivery; `FileExistsError` read as a claim ONLY from the create, never
   from the directory; a claim withdrawn when the envelope cannot be written; two racing hooks deliver
   once. [proof: mutation]
7. [S7] Exit 0 unconditionally, closed stdout included; stdin read whole. [proof: acceptance]
8. [S8] §55: decode the JSON envelope exactly; take the hook from the settings entry (exactly the
   four tools, the command resolved through `CLAUDE_PROJECT_DIR`) and drive that file; compare path
   selection against the BUILT BINARY for five header shapes; the subdirectory, quoted-path, race and
   closed-stdout cases (the descriptor closed by the shell that execs the hook, never before `env`);
   the early deliveries for plans mrw refuses; the spaced and double-spaced headers; the `cd` before
   a grep and the explicit `mrw --root`; the 601-operand command; the relative, in-tree and unusable
   state bases; the withdrawn claim; the nested `.git`; the 300-globstar × 400-directory and 24-star
   alarms; the trailing slash, the nested braces, the flat-before-nested group and the inline list
   with a comment. [proof: acceptance]

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
- 2026-09-04 · 8a8b137 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S3: read the served path forward to the first gap again. §55's consecutive-space row fails: `docs/adr/my  file.md` arrives as `docs/adr/my` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8a8b137 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: apply a leading `cd` to the command's operands only, not to the headers it printed. §55's `cd docs && mrw read --grep` row fails: `adr/x.md` resolves from the wrong base · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8a8b137 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S4: tokenise headers with `shlex` in place of the `splitHeader` port. §55's single-quoted-path row fails: the quotes are stripped and the file delivers · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8a8b137 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: match each segment with a regex of `[^/]*` runs in place of the two-pointer walk. §55's 24-star row fails on the alarm (exit 142) · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8a8b137 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: drop the trailing-slash check. §55's `README.md/` row fails: a pattern naming a directory matched a file · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8a8b137 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: expand a brace group that holds another. §55's nested-brace rows fail both ways: `src/a/b.tsx` matches and the literal path does not · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8a8b137 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S6: keep the claims when the envelope cannot be written. §55's closed-stdout-then-read row fails: the next read in the session is silent · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · d6e8c95 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: resolve a served header from the command's cwd even when mrw was given `--root`. §55's `mrw -C .. read` row fails: the header mrw printed is lost · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · d6e8c95 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: decide brace nesting per group instead of for the whole pattern. §55's flat-before-nested row fails: `src/{a,b}/{c,{d,e}}/*.tsx` half-expands and matches a path it should not · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · d6e8c95 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: strip an inline list's brackets before its comment. §55's commented-list row fails: the glob becomes `docs/adr/**]` and matches nothing · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · d6e8c95 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S6: keep the claims when the envelope cannot be written. §55's closed-stdout-then-read row fails — now on every host, because the row closes the descriptor in the shell that execs the hook rather than before `env` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · d6e8c95 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S4: tokenise headers with `shlex` in place of the `splitHeader` port. §55's CROSS-LANGUAGE row fails: for `'docs/adr/x.md' 1 replace` the built binary names `'docs/adr/x.md'` and the hook names `docs/adr/x.md` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8c1e33e · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: look for mrw at token zero only. §55's `env FOO=1 mrw -C ..` row fails: the served header is lost behind the wrapper · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8c1e33e · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: keep the FIRST root when the flag is given twice. §55's `-C /nowhere --root ..` row fails, where the CLI itself takes the last · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8c1e33e · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: read `-C` as a root for any program. §55's `git -C .. log` row fails: a base moves for a program whose `-C` means something else · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 8c1e33e · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S5: make every slash-less pattern root-only, the bare `**` included. §55's `paths: ["**"]` row fails on a file two directories deep · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0817274 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: look for mrw at token zero again instead of by name. §55's wrapper loop fails for `/usr/bin/env FOO=1`, `nice -n 5` and the rest · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0817274 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: ignore `env --chdir`. §55's `env -C docs cat adr/x.md` row fails: the operand resolves from the wrong base · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0817274 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S6: put the directory and the create back in ONE try block, so `FileExistsError` from `makedirs` reads as a claim. §55's file-at-the-claim-directory row fails: both calls deliver nothing · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0a46d2c · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: keep only the FIRST mrw root and resolve every served header against it. §55's two-roots row loses the second read's rule, and its false-mrw-token row loses the real read's · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · 0a46d2c · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: strip an mrw range from every Bash token instead of offering both forms. §55's `cat docs/adr/note:1` row fails: a filename with a colon is cut at it · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · fddfb55 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: stop splitting control operators out of the tokens. §55's `;mrw` row fails: the second call arrives as one token and its root is never seen · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · fddfb55 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: compose repeated `env --chdir` values only, instead of offering each. §55's `env -C /nowhere -C docs` row fails, where real env takes the last · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · fddfb55 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: drop the composite token from the candidates. §55's quoted `semi;colon.md` row fails: splitting the operator loses the filename · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · c4fd825 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: offer every `--chdir` against every directory so far, the exponential first cut. §55's forty-flag row fails on the alarm (exit 142) and the rule is lost · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370
- 2026-09-04 · c4fd825 · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · S2: take the FIRST `--chdir` of an invocation instead of the last, which is not what env does. §55's `-C /nowhere -C docs` and forty-flag rows both fail · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370

## Invariants

- A rule is delivered once per session per agent per project while its claim can be filed and its envelope reaches the harness, and on every matching call when a claim cannot be filed — never on none — whichever of the four tools read the file.
- A shape the reading does not cover delivers nothing rather than something wrong, and the record names it: the served header makes up for an unparsed command only when its path resolves against a directory the reading recognised.
- The candidate directory list is linear in the number of `--chdir` flags, never exponential.
- The project root is never assumed to be `cwd`; a Bash operand resolves from where the command ran, a header mrw printed from any root an mrw in that command was given, every other path from the project root.
- A plan header is tokenised as `mrw` tokenises it, pinned against the built binary; whether mrw accepts the plan is not mirrored, so a refused plan delivers early and never silently.
- The hook exits 0 whatever happens.
- No engine change; `go.mod` declares exactly one requirement.

## Risks

- `python3` absent on a contributor's machine: the hook errors, nothing is delivered, which is the state before this record; `AGENTS.md` says so.
- The mirrored tokeniser drifts from `internal/plan`. Mitigated by §55's cross-language row, which drives the built binary, and by ADR-001 owning the grammar; a drift is a defect here, not there.

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
- 2026-09-04 · 8a8b137* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:35773
- 2026-09-04 · d6e8c95* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:49058
- 2026-09-04 · 8c1e33e* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:33581
- 2026-09-04 · 0817274* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:33053
- 2026-09-04 · 0817274* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:31943
- 2026-09-04 · 0817274* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:32504
- 2026-09-04 · 0a46d2c* · exit 2 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:26715
  ```
  --- last 10 line(s) of stdout (of 434 after folding 434 raw)
    PASS  and behind 'time -p', where a positional reading of the command loses it
    PASS  and behind 'exec -a custom', where a positional reading of the command loses it
    PASS  and behind 'nohup', where a positional reading of the command loses it
    PASS  and behind 'sudo -u nobody', where a positional reading of the command loses it
    PASS  and behind 'timeout 30', where a positional reading of the command loses it
    PASS  env --chdir moves the base the operands resolve against, as a leading cd does
    PASS  a data token that spells mrw does not take the root away from the mrw that ran
    PASS  two mrw calls with two roots both deliver: the headers of the second are not resolved against the first
    PASS  a filename containing a colon is read as itself, not cut at the colon
    PASS  and a root given twice delivers, because both are tried rather than one being chosen
  --- last 2 line(s) of stderr
  ./scripts/contract.sh: line 2695: syntax error near unexpected token `||'
  ./scripts/contract.sh: line 2695: `  || bad "a repeated root did not take the last value: $ctx"'
  ```
- 2026-09-04 · 0a46d2c* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:32990
- 2026-09-04 · fddfb55* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:32437
- 2026-09-04 · c4fd825* · exit 0 · `set -o pipefail …` · acceptance-sha256:8996f83fd6e2fa313b22b02ec081c26c34c9c04542feff85dd0572b923ece370 · ms:83902
