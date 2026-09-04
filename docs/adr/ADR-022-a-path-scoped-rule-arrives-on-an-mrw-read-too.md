# ADR-022: A path-scoped rule arrives on an mrw read too

**Status:** Accepted
**Accepted:** 2026-09-04 by M — the hook was one of three items flagged as M's call (issue #86, option 3, "project config"); M answered *"fix of course"*. The record exists because the hook keeps persistent state outside the tree and runs on every matching tool call for every clone, which `CONTRIBUTING.md` says needs one.
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Served-path change:** none to `mrw`. A Claude Code session in THIS repository receives a path-scoped `.claude/rules/*.md` file when a Bash, Write, `mrw_read` or `mrw_write` call touches a file its globs match — where before only the harness's Read tool delivered it.
**Cross-references:** ADR-004 (owns "nothing is left in the working tree": the hook's only state lives outside it), ADR-001 (owns the plan grammar the hook must read the way `mrw` does), issue #86 (the measurement), PR #84 (the rules this delivers), PR #87 (the sentence this supersedes as the fallback)
**Governs:** `.claude/hooks/**`, `.claude/settings.json`
**Enforced-by:** `cmd/mrw/ruleshook_test.go::TestThePathScopedRulesHookDeliversOnAnMrwRead`
**Invalidates:** none. #87's "Read one such file first" becomes the fallback when hooks are off, not the rule.

## Context

**Claude Code injects a path-scoped rule only when its own Read tool reads a matching file.** Measured
2026-09-04 in another repository with controls, and reproduced here: the same file read via
`mrw read` (Bash) and via `mrw_read` (MCP) delivered no rule; a Read-tool read delivered exactly the
two rules whose globs matched, and not the third. `cat`, Write and Edit deliver nothing either.

**This repository sends sessions through mrw** — `AGENTS.md` at three edits or two files, and M's rule
of the same day, *"we must use mrw as bash tool when we can"* — and added three path-scoped rules in
#84. A session obeying the guide received none of the rules written for the files it was in, and an
absent rule is silent.

**Two reviews of the first cut (PR #88) found what a hook that runs on every call has to get right.**
The harness's `cwd` follows a `cd`, so "the project" cannot be read off it; a plan header is a grammar
with quoted paths, a BOM and counted bodies, not a regex; two hooks fire concurrently for parallel tool
calls, so read-then-append dedup double-delivers; a glob with many `**` backtracks; and state in a
world-readable temp directory follows a pre-positioned symlink. Each is a decision, and they are made
below rather than left as a hook's private opinion.

**A third review, of the reworked hook, found what a mirror and a matcher have to get right.** A
served path with a space was cut at the space; the header tokeniser was `shlex`, which strips the
single quotes mrw keeps; the `**` memo rescanned the path on every globstar, so a long path made a
matcher described as linear take seconds; a relative cache base landed state in the checkout; a Bash
operand that named nothing from `cwd` was retried against the root and found a file the call never
read; the walk-up crossed a nested repository's `.git`; and "exactly once" was written where "once
while a claim can be filed" was true. Each is decided below, and the measurements are in §55.

**A fourth review found the mirror's own failure mode.** The third round had mirrored `Parse`'s and
`parseHeader`'s refusals so that a plan mrw refuses delivers nothing — and every place the mirror was
STRICTER than mrw (a pattern it would not compile, an integer Go rejects and Python accepts) was a
successful write that delivered nothing, the exact silence this record exists to remove. It also found
a served path with two consecutive spaces cut at the first gap, a per-segment regex of `[^/]*` runs
that backtracked past 2 s on sixteen stars, a leading `cd` applied to a command's operands but not to
the headers mrw printed for it, and a claim filed before the envelope was written, so a closed stdout
kept the claim and silenced the next read for a week. The acceptance mirror is gone (Decision 4); the
rest are decided below.

## Existing Primitives Audit

- **`$CLAUDE_PROJECT_DIR`:** Claude Code sets it for hooks and it is what `.claude/settings.json`
  already expands to locate the script. **The project root is read from it**, with a walk up from
  `cwd` to the nearest `.claude/rules` as the fallback; `cwd` itself is what relative paths in a
  Bash command resolve against.
- **`internal/plan`'s header grammar (ADR-001):** `splitHeader` ported line for line, because the
  tokeniser decides WHICH string is the path. **Mirrored, not imported** — the hook is Python so it can
  run without a Go build — and only the tokeniser is mirrored: whether mrw ACCEPTS the plan is not,
  for the reason Decision 4 gives. §55 asserts the tokeniser differentially: a single-quoted path, an
  unterminated quote, a pattern address with spaces.
- **`os.open(O_CREAT|O_EXCL)`:** the atomic claim. **Taken over read-then-append** because two hooks
  can run at once and one must lose.
- **`mrw read`'s `==> path` header:** printed once per served file on the CLI and in the MCP text
  block alike. **It is where a grep's files are**, since a grep names none in its input.
- **The contract's `python3`:** already a dependency of `scripts/contract.sh`. No new one.

## Decision

**1. The hook delivers on Bash, Write, `mrw_read` and `mrw_write`; Read and Edit are the harness's.**
Read is the native trigger; an Edit is refused unless the file was already Read, so it carries its
rules.

**2. Paths come from the CALL and from the RESULT.** Named paths are taken from the tool input (Bash
tokens, uncapped; a Write's path; `mrw_read` specs with their range stripped; plan headers); served
paths from every `==> path  NL  NB  sha …` header in the tool result, the path read back from that
suffix so any run of spaces inside it survives. A grep, a working-set read, a no-argument read —
anything whose input names no file — is still delivered for the files it served. **Every named path is
a guess that a file was read**, and a wrong guess — `echo docs/adr/x.md` names the record without
reading it — delivers a rule one call early. That is the side the hook errs on throughout: an early
delivery puts the rule in context; a path the hook fails to see loses it.

**3. The project root is `$CLAUDE_PROJECT_DIR`, else the nearest `.claude/rules` above `cwd`, and
the walk stops at the first `.git` it meets** — a nested repository does not inherit an enclosing
one's rules. A Bash command's operands AND the `==>` headers mrw printed for it resolve from where the
command ran — `cwd`, moved by a leading `cd DIR &&` (mrw's own `--root` defaults to `.`); a Write's
path, an MCP spec and an MCP result resolve from the root. One base per call, never retried against
the other: a session that has `cd`-ed into `internal/` still gets `../scripts/contract.sh`'s rule, a
`cd docs && mrw read --grep` delivers for the `adr/…` headers it printed, and `docs/adr/x.md` typed
from `cmd/mrw`, which read nothing, delivers nothing.

**4. Plan headers are tokenised as `internal/plan` tokenises them, and whether mrw accepts the plan
is not mirrored.** `splitHeader` is ported line for line — double quotes only, a backslash escaping a
quote or a backslash, a `/pattern/` address one token with its spaces — and a BOM is stripped once per
line as mrw strips it. Every header-shaped line's first field is then a candidate, counted and raw
bodies included: a body line that looks like a header delivers early for a file the plan did not touch,
and a plan mrw refuses — an unknown op, a guard given twice, `raw=true` without `body=` — delivers
early for the files it names. The third round mirrored `Parse`, `parseHeader` and their refusals
instead, so that a refused plan delivered nothing; the fourth removed it, because a mirror can only add
silence: everywhere it was stricter than mrw — a pattern it would not compile, an integer Go rejects
and Python accepts, `validate`'s rules it never reached — a successful write delivered nothing, and
everywhere it was looser the delivery was merely early. Decision 2 already prefers early.

**5. Globs match by segment, in an enumerated grammar.** A `**` segment stands for zero or more
directories — Git's boundary rule, and the only thing borrowed from Git; `*` and `?` stay inside
one segment; a flat `{a,b}` is expanded before the pattern is split, so an alternative may hold a
slash or a glob; a slash-less pattern is root-only. The edges do what this sentence says: a pattern
ending in `/` names a directory and so no file (write `dir/**`), a pattern with nested braces is taken
literally, and there is no negation. The matcher fills one row per pattern segment over the path
positions and matches each segment by the two-pointer walk rather than a regex, so its cost is bounded
by the product of the segment counts times the product of the segment lengths, and nothing backtracks —
measured through the hook at 40 ms for 300 globstars against 400 directories, where the memoised
recursion it replaced took 2.3 s, and inside a 1 s alarm for 24 stars in one segment against a
200-character name, where a regex of `[^/]*` runs ran past 2 s at sixteen. The native matcher's
behaviour on shapes outside this grammar is unmeasured; the hook claims only what it matches.

**6. Dedup is an atomic claim per rule per session per agent per project**: a file created with
`O_CREAT|O_EXCL|O_NOFOLLOW` under `claude-rules-on-read` in `$XDG_CACHE_HOME`, else `~/.cache`,
made `0700` (a chmod that fails on a directory that already existed is ignored; the claim files are
`0600` regardless), swept after seven days. The base must be absolute and outside the project — a
relative one would land under whatever `cwd` the hook was given, one inside the tree would break
ADR-004 — and a state directory that cannot be used, for those reasons or any other, delivers on
every call rather than on none. A claim is filed before the envelope is written, so when the write
fails — a closed stdout — the claims this call filed are withdrawn; otherwise the next real read would
be silent for a week. So: exactly once while a claim can be filed and the envelope reaches the
harness; two hooks that race for one rule, one delivers; and the failure mode is a repeat, never a
silence.

**7. Exit 0 is unconditional**, including a closed stdout, and stdin is read whole — no size cap, so a
large tool result cannot become a silent no-rules. A hook that breaks must not take the turn.

**8. Every delivery in the contract row is paired with a non-delivery**, the row decodes the hook's
JSON rather than grepping it, and both the row and the Enforced-by take the hook from
`.claude/settings.json` — exactly the four tools in the matcher, the command resolved through
`CLAUDE_PROJECT_DIR` to a file that exists — and drive THAT file, so an entry pointing at nothing
fails here and not in a session.

**Go/no-go, checked during execution:**

- **No engine change**: every `internal/*` directory and `cmd/mrw/main.go` stay byte-identical against
  a merge-base diff; the only Go added is the Enforced-by test.
- **No new dependency**; `go.mod` still declares exactly one requirement. The hook is stdlib Python.
- **`gofmt -l .` empty and `go vet ./...` clean**, in the fence.

## Alternatives Considered

- **Make the three rules unconditional.** No silent failure mode at all — the reviewer's point, and it
  is a real one. Rejected because it costs ~100 lines of context in every session, which is what path
  scoping exists to avoid, and the session that needs `adr.md` is not the one fixing a typo.
- **Read only** — #87's sentence: Read one matching file first. Kept as the fallback with hooks off.
  Rejected as the rule because it depends on the session remembering to do it, which is the failure
  mode being fixed.
- **Import `internal/plan` by building a Go hook.** Exact grammar for free. Rejected: a hook that needs
  `go build` before it runs is a hook that silently is not there on a fresh clone.
- **`git ls-files --cached --others --exclude-standard` as the existence test** (the peer's approach).
  Avoids matching an ignored artefact; omits an ignored file a session deliberately read. Existence
  is taken, and a rule scoped to a build directory is the author's choice.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `.claude/hooks` | Which reads deliver which rules, and the dedup state | Yes |
| `.claude/settings.json` | Wiring only | Wiring only |
| `internal/*`, `cmd/mrw/main.go` | Untouched | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| a PostToolUse hook in project settings, matching `Bash|Write|mcp__mrw__mrw_read|mcp__mrw__mrw_write` | New, applies to every clone that trusts the project | every Claude Code session here |
| dedup state under the user's cache directory, `0700`, swept after 7 days | New persistent state, outside the tree | the hook only |
| the matcher assumes the MCP server is registered as `mrw` | Documented assumption | hosts that register another name get the Bash and Write half |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the hook, its wiring, and the Enforced-by test | T1 | — | No |

## Implementation

One task. The reworked hook, the Go test that drives it as the harness does, the strengthened §55,
and the wording.

## Consequences

- **Positive:** the rules written for `docs/adr/`, tests and the contract reach the sessions working
  there, whichever reader they use.
- **Positive:** a grep — the Desktop-style read — is covered, because the served files are read off
  the result.
- **Negative:** one Python process per matching tool call, bounded by the 10 s hook timeout and in
  practice milliseconds. A machine without `python3` sees a hook error and no rules, which is the
  state before this record.
- **Negative:** state under the user's cache. Named, bounded, swept.
- **Neutral:** `mrw` itself is unchanged.

## Out of Scope

- Tools other than the four named (permanent: boundary: Decision 1 — the measured set; Grep and Glob were not measured and are not claimed)
- A host that registers the MCP server under another name (permanent: fact: the matcher names `mcp__mrw__*`; citation: file `.claude/settings.json:5`)
- Making the rules unconditional instead (permanent: boundary: Alternatives)
- Windows behaviour of the hook (deferred: docs/adr/BACKLOG.md — the rules-hook-on-Windows entry)
- Matching the shapes outside the enumerated grammar — nested braces, a trailing `dir/`, negation — the way the native matcher would (permanent: boundary: Decision 5 — here they are literal, a directory, and absent; what the native matcher does with them is unmeasured)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The hook is wired to a path that does not exist in a clone | Low | High | §55 asserts the settings entry names an existing file; the Enforced-by runs the same file |
| A plan or command large enough to make the hook slow | Low | Med | a table matcher over segments and a two-pointer walk inside each (§55: 300 globstars × 400 directories, and 24 stars in one segment, each inside a 1 s alarm), no regex over globs, stdin read whole, the 10 s timeout |
| Two parallel hooks both deliver | Med without the claim | Low | atomic claim files; §55 races two hooks and requires one delivery |
| The hook masks a Claude Code fix that makes it redundant | Low | Low | it delivers once per rule per session, as the harness does; a double delivery would show as a repeat |
| The tokeniser drifts from `internal/plan`'s `splitHeader` | Med, over time | Low | §55's differential rows; a drift is a defect here, not there — ADR-001 owns the grammar. Acceptance is not mirrored, so a drift can only mis-pick a path, never suppress a plan |
| Dedup state cannot be written | Low | Low | the rule is delivered on every call instead — a repeat, never a silence; §55 drives a file where the directory should be |

## Rollback

Delete `.claude/settings.json`'s hook entry. The script and state are inert without it; #87's sentence
is the behaviour that remains.

## Follow-ups

- [ ] If Claude Code starts delivering path-scoped rules on MCP and Bash reads natively, retire this
      record and the hook with it.
