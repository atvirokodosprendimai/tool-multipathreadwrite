# ADR-015: A refusal names the fix, for the two mistakes the syntax invites

**Status:** Accepted
**Accepted:** 2026-09-04 by M — *"good, solve the remaining things"*, given after the six defects were enumerated and D1–D3 were recorded. These are D4 and D5, the two that were classed cheap.
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (owns the plan grammar and the `body=`/`raw=` escape this points at), ADR-007 (owns the read whose path error this improves), ADR-013 (also governs `internal/plan`, for addressing rather than diagnostics)
**Governs:** `internal/plan/**`, `internal/read/**`
**Enforced-by:** `internal/plan/plan_test.go::TestABodyLineThatLooksLikeAHeaderSaysSo`
**Invalidates:** none — checked. Neither the grammar nor the read syntax changes; only what mrw says when a caller gets them wrong. ADR-013 governs `internal/plan` for addressing and is untouched: no address form gains or loses meaning here.
**Served-path change:** the two mistakes mrw's own syntax invites now produce a refusal that names the escape, instead of a cascade of errors about something else.

## Context

**This project's stated ethos is that a refusal is the tool working** — *"it names the file, the plan
line and the reason; read it rather than reaching for a bigger hammer"* (AGENTS.md). Two refusals do
not meet that bar, and both were hit repeatedly by this session's own author while writing this
repository's documentation.

**D5 — a body line beginning with `@@` is read as a header, and the error is about something else.**
Measured 2026-09-04 at `14d28b3`:

    $ printf '@@ f.md 1 replace\n@@ this line starts with at-at\nsecond body line\n' | mrw write -
    mrw: <stdin>: plan has 3 error(s):
      line 2: unknown op "starts" (want replace, insert-after, insert-before, delete or create)
      line 3: text before the first @@ header: "second body line"
      line 1: replace with an empty body would delete 1 — …

Three errors, none of which mentions `body=<n> raw=true` — the escape that exists precisely for this
and is documented in AGENTS.md, README and the MCP `instructions`. A caller who has not read those
is told their op is `starts`. This is not hypothetical: it happened twice today while editing
AGENTS.md, because documenting the plan format means writing plan syntax into a body.

**D4 — a shell glob that never expanded is reported as a missing file.** Measured the same day:

    $ mrw read "internal/mcp/*.go:1-3"
    ==> internal/mcp/*.go  UNREADABLE  open …/internal/mcp/*.go: no such file or directory

The message is true and useless. And the unquoted form fails before mrw is started at all — zsh
refuses `internal/mcp/*.go:1-3` with *"no matches found"*, because the `:1-3` suffix means the
pattern matches no file. AGENTS.md warns about MSYS mangling on Windows and says nothing about this,
though zsh is macOS's default shell.

**What these have in common** is the reason they are one record: in both, mrw holds the information
the caller needs and declines to say it. Neither is a grammar defect. Both are diagnostics.

## Existing Primitives Audit

- **`body=` / `raw=` (ADR-001):** the escape already exists and works. **Reused unchanged** — this
  record points at it, and deliberately does not add a second way to delimit a body. A heredoc
  terminator was considered and rejected below.
- **`parseHeader`'s "unknown op" error (`internal/plan`):** already knows the line and the token it
  could not read. **Extended** with a hint, only when the line looks like a body line rather than a
  header.
- **`internal/read`'s UNREADABLE report (ADR-007):** already names the path and the OS error.
  **Extended** with a hint when the path holds a glob metacharacter, because "no such file" is
  technically right and answers the wrong question.
- **Detecting the shell before it runs:** not available and not attempted. mrw sees an argument, not
  the shell that produced it, so the hint is triggered by the ARGUMENT's shape.
- **A heredoc-style body terminator, `body=<<END`:** audited and **NOT taken.** See Alternatives.

## Decision

**1. A failed header that looks like a body line says so.** When a `@@` line fails to parse as a
header AND a previous hunk header has already been seen, the error adds: this looks like a body line
beginning with `@@`; declare `body=<n> raw=true` on the hunk it belongs to. The hint is added to the
existing error, not substituted for it — the parse failure is still reported exactly as before.

**2. An unreadable path holding a glob metacharacter says so.** When a path cannot be opened and
contains `*`, `?` or `[`, the report adds: this looks like a shell glob your shell did not expand;
quote it only if the file really has that name, and note that an address suffix like `:1-3` stops
most shells matching. `--grep` and `--files-from` are named as the tools for the job.

**3. Neither is a new mechanism, and neither changes what is accepted.** Every plan that parsed
before parses now; every read that succeeded before succeeds now. This record only changes the words
on the failure paths.

**Go/no-go, checked during execution:**

- **No accepted input changes.** A corpus of the plans in `contract.sh` parses to identical hunks.
- **The hints are conditional, and the condition is tested from both sides.** A hint that fires on
  every parse error would be noise; the tests assert it appears for the body-line case and does NOT
  appear for an ordinary bad op on a real header.
- **No new dependency**; `go.mod` still declares exactly one requirement.
- **`gofmt -l .` empty and `go vet ./...` clean**, in the fence.

## Alternatives Considered

- **Leave both and improve the docs.** Free, and it is what "documented escape" already means. The
  docs say it; the caller who needs it is by definition the one who has not read them, and today's
  message routes them to the wrong problem. Documentation is also part of this record — it is just
  not sufficient on its own.
- **Add a heredoc body terminator, `body=<<END … END`.** The ergonomic fix for D5: no counting. It
  is a genuine grammar addition, it gives the format a second way to say one thing, and it does not
  remove the need for `body=` because a terminator can itself appear in a body. Rejected as a
  bigger change than the defect: what hurt was not counting lines, it was not being told to. If the
  hint lands and counting is still the friction, that is a separate record with evidence behind it.
- **Make `@@` in a body legal by requiring headers to be otherwise well-formed.** Rejected: it makes
  the grammar ambiguous in exactly the case where a caller most needs it not to be, and a plan whose
  meaning depends on whether a body line happens to parse is worse than one that refuses.
- **Have mrw expand globs itself.** Rejected: it already has `--grep` and `--files-from` for finding
  files, and a tool that second-guesses the shell's expansion produces different results depending
  on which shell invoked it — the class of bug the MSYS note already documents.
- **Detect the shell and tailor the message.** Rejected: not knowable from an argument, and the hint
  is useful without it.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/plan` | The grammar and, now, a hint on one failure path | Yes — ADR-013 owns addressing here; this owns what a parse failure says |
| `internal/read` | Serving ranges and, now, a hint on the unreadable path | Yes |
| `AGENTS.md`, `README.md` | The zsh trap is documented where the MSYS trap already is | Yes |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| A parse error may carry a `body=`/`raw=` hint | Public contract, additive text | Anyone reading a parse error |
| An UNREADABLE line may carry a glob hint | Public contract, additive text | Anyone reading a read report |
| No input's meaning changes | — | — |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the two hints and their conditions | T1 | — | No — additive text |

## Implementation

One task. The two hints are independent code paths but one idea, one review, and one contract row;
splitting them would triple the ceremony for four lines of Go each.

## Consequences

- **Positive:** the two mistakes the syntax invites route the caller to the escape instead of to a
  wrong diagnosis.
- **Positive:** the D5 hint is what a caller writing documentation about mrw hits first, which is a
  population this project has exactly one member of and will have more.
- **Negative:** two more conditional branches on failure paths, each of which can be wrong. Mitigated
  by testing both the fires-when-it-should and the stays-quiet-when-it-should cases.
- **Neutral:** the underlying escapes and tools are unchanged, so nothing a caller already does stops
  working.

## Out of Scope

- A heredoc body terminator (deferred: docs/adr/BACKLOG.md — revisit only if the hint lands and counting is still the friction)
- Expanding globs inside mrw (permanent: boundary: `--grep` and `--files-from` exist, and second-guessing the shell is shell-dependent by construction)
- Any change to what a plan or a spec MEANS (permanent: boundary: ADR-001 and ADR-007 own those; this is diagnostics only)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The body-line hint fires on ordinary bad headers and becomes noise | Med | Low | Conditional on a previous hunk having been seen; a test asserts silence on a first-line bad op |
| The glob hint fires on a legitimate filename containing `*` | Low | Low | It is a hint on a path that ALREADY failed to open, and it says "quote it only if the file really has that name" |
| A hint is written and never read | Med | Low | §49 asserts both through the built binary, which is where a caller meets them |

## Rollback

Revert the commit. Both hints are additive text on paths that already failed; nothing depends on
their presence.

## Follow-ups

- [ ] If the body-line hint proves insufficient — if callers still lose time to `@@` in a body —
      revisit the heredoc terminator with that as the evidence
