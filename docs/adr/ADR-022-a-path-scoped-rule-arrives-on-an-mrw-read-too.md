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

## Existing Primitives Audit

- **`$CLAUDE_PROJECT_DIR`:** Claude Code sets it for hooks and it is what `.claude/settings.json`
  already expands to locate the script. **The project root is read from it**, with a walk up from
  `cwd` to the nearest `.claude/rules` as the fallback; `cwd` itself is what relative paths in a
  Bash command resolve against.
- **`internal/plan`'s header grammar (ADR-001):** first field quoted or bare, a BOM stripped, `body=N`
  always honoured, `raw=true` needing `body=`. **Mirrored, not imported** — the hook is Python so it
  can run without a Go build — and the contract row asserts the mirror against the cases the
  reviewer found.
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
tokens, a Write's path, `mrw_read` specs with their range stripped, plan headers); served paths are
taken from every `==> path` header in the tool result. A grep, a working-set read, a no-argument
read — anything whose input names no file — is still delivered for the files it served.

**3. The project root is `$CLAUDE_PROJECT_DIR`, else the nearest `.claude/rules` above `cwd`; relative
paths resolve from `cwd`, then relativise to the root.** A session that has `cd`-ed into `internal/`
still gets `../scripts/contract.sh`'s rule.

**4. Plan headers are read the way `mrw` reads them.** A leading BOM is stripped; the first field may be
quoted; every `body=N` skips N lines whether or not `raw=true`; `raw=true` without `body=` is a plan
mrw refuses and delivers nothing.

**5. Globs match by segment, not by regex.** `**` crosses directories only at a segment boundary
(Git's rule), `*` and `?` stay within one, `{a,b}` alternates, a slash-less pattern is root-only. A
segment matcher is linear in the path and cannot backtrack.

**6. Dedup is an atomic claim per rule per session per agent per project**, a file created with
`O_CREAT|O_EXCL|O_NOFOLLOW` under a `0700` directory in the user's cache, swept after seven days. Two
hooks that race for one rule: exactly one delivers.

**7. Exit 0 is unconditional**, including a closed stdout. A hook that breaks must not take the turn.

**8. Every delivery in the contract row is paired with a non-delivery**, and the row decodes the
hook's JSON rather than grepping it, and asserts `.claude/settings.json` names a hook file that exists.

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

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The hook is wired to a path that does not exist in a clone | Low | High | §55 asserts the settings entry names an existing file; the Enforced-by runs the same file |
| A plan or command large enough to make the hook slow | Low | Med | linear matchers, no regex over globs, 10 s timeout |
| Two parallel hooks both deliver | Med without the claim | Low | atomic claim files; §55 races two hooks and requires one delivery |
| The hook masks a Claude Code fix that makes it redundant | Low | Low | it delivers once per rule per session, as the harness does; a double delivery would show as a repeat |

## Rollback

Delete `.claude/settings.json`'s hook entry. The script and state are inert without it; #87's sentence
is the behaviour that remains.

## Follow-ups

- [ ] If Claude Code starts delivering path-scoped rules on MCP and Bash reads natively, retire this
      record and the hook with it.
