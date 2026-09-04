# ADR-018: A root nobody named is not a root

**Status:** Accepted
**Accepted:** 2026-09-04 by M — asked how #81 should be dealt with, shown that the issue's own framing ("refuse `/`") targets the symptom rather than the defect, and given the choice of three rules. M chose *"refuse pathological fallbacks"*: guard the FALLBACK, honour anything explicit.
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Served-path change:** a server that nobody told which tree to serve, and that landed somewhere which cannot be a project, refuses to start and says how to name one — instead of serving the whole filesystem correctly.
**Cross-references:** ADR-011 (owns `ResolveRoot` and the `Source` this guard keys on), ADR-006 (owns the confinement that keeps working and is exactly why the defect is silent), ADR-005 (owns the scope a write stays inside), ADR-017 (capability half of "loosen the MCP"; this is not its reach half)
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/root_test.go::TestAFallbackRootThatIsNotAProjectIsRefused`
**Invalidates:** none — no earlier record decided what a fallback root may be. README's description of the fallback becomes incomplete and moves in the same commit.

## Context

**Issue #81, split out of #75.** Reproduced 2026-09-04 at `2e6b7e7`:

```sh
$ cd / && env -u CLAUDE_PROJECT_DIR mrw mcp
mrw mcp: serving / (from the working directory)
```

`ResolveRoot` takes `--root`, else `CLAUDE_PROJECT_DIR`, else the working directory. Claude Code sets
the variable; a host that does not — Claude Desktop is the case reported — gets the third.

**Everything downstream is then correct and useless.** Confinement still works, every ADR-006 refusal
is still right, the ledger still records honestly — about the entire filesystem, on a surface that
also WRITES. That is the same class as binding to the wrong repository, which ADR-011-T1 already
fixed for Claude Code: correct-looking answers about a tree nobody asked about. Here the wrong tree
is everything.

**The issue's own framing is the symptom, not the defect.** #75 suggested refusing `/`, "or at least
warning loudly on stderr". But `/` is not what went wrong: what went wrong is that NOBODY NAMED A
TREE. A fallback onto `/Applications` is equally unintended, and a rule that enumerates unpleasant
paths is a rule this repository would then have to defend by taste. The axis that matters is already
in the code — `ResolveRoot` returns a `Source` — and it is **explicit versus accidental**.

**And "warn loudly" is the option that does not work**, for the reason this reached a user at all: a
line on stderr is what host logs swallow. The reporter found this by reading the code and
constructing the condition, not by noticing a warning.

## Existing Primitives Audit

- **`mcp.ResolveRoot` and `mcp.Source` (ADR-011):** already distinguish `SourceFlag`,
  `SourceProjectDir` and `SourceWorkingDir`. **Reused unchanged, and this is what makes the record
  small**: the guard keys on the value that already exists rather than inferring intent from a path.
- **`rooted.Resolve` (ADR-006):** the containment primitive. **Untouched** — this decides which root
  is acceptable, not what a path may reach once a root is chosen. Confining to `/` is not a
  confinement bug; it is a correct answer to a question nobody asked.
- **`filepath.Dir(p) == p`:** the portable test for a filesystem root. **Taken instead of comparing
  to `"/"`**, because `filepath.IsAbs` is already recorded in this codebase as returning FALSE for
  `/etc/hosts` on Windows, and a guard that names `/` literally would silently pass a Windows volume
  root.

## Decision

**1. A root reached by FALLBACK is refused when it cannot be a project.** Only two cases qualify: a
filesystem root, and the home directory itself. Both are refused before the server starts, at exit
`2` — the usage status, because the fix is an argument.

**2. Anything EXPLICIT is honoured, whatever it names.** `--root /` starts and serves. So does
`--root "$HOME"`, and so does a host-set `CLAUDE_PROJECT_DIR` pointing at either. **Explicitness is
the licence**, and this clause is the one that keeps the record from becoming a list of paths
somebody found distasteful. It also protects the population ADR-017 was written for: a Desktop
analyst's documents really do live under the home directory, and naming it is allowed.

**3. The refusal names the fix.** It says which directory it refused, why that directory cannot be a
project, and the exact flag to set. A refusal without a way forward is the dead end this corpus keeps
refusing.

**4. An ordinary directory reached by fallback still serves.** `cd myrepo && mrw mcp` works today and
keeps working. The guard is narrow on purpose: it fires only where the answer is certainly wrong,
so it can be shipped without a migration.

**5. This is NOT the reach half.** Multi-root, repeatable `--root`, and the per-hunk ledger question
belong to the record that owns the root model, which is still unwritten. ⚠ Earlier notes in this
corpus and in ADR-017's own text say "ADR-018 owns reach". That is now wrong: this record took 018
because it is small and shippable, and blocking a safety guard behind a larger design is backwards.
Reach is ADR-019.

**Go/no-go, checked during execution:**

- **No engine change.** `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`,
  `internal/check` and `internal/state` stay byte-identical against a merge-base diff.
- **No new dependency**; `go.mod` still declares exactly one requirement.
- **The CLI is untouched.** `mrw read`, `mrw write` and every other subcommand take `--root` and are
  not guarded: a shell user who types a root has stated intent by definition, and there is no
  fallback-from-a-host problem to solve there.
- **`gofmt -l .` empty and `go vet ./...` clean**, in the fence.

## Alternatives Considered

- **Refuse `/` and nothing else.** The issue's literal suggestion. Rejected: it fixes the reported
  symptom and leaves the defect, which is an unnamed root of any shape.
- **Refuse EVERY fallback — require `--root` or `CLAUDE_PROJECT_DIR`.** The simplest rule, and the
  one with no taste in it at all. Put to M and not chosen: it breaks `cd myrepo && mrw mcp`, which
  works today and is a reasonable thing for a shell user to do. Worth revisiting if the narrow guard
  proves insufficient — the evidence would be a second report of an unintended root that is neither
  a filesystem root nor a home directory.
- **Warn loudly on stderr, refuse nothing.** Zero breakage and no contract change. Rejected because
  host logs swallow stderr, which is precisely how the reported case reached a user.
- **Refuse any directory with no project marker** (no `.git`, no `go.mod`, …). Rejected: it is the
  taste-based rule in disguise, it would refuse a folder of CSVs — the Desktop case this project is
  trying to serve — and the marker set would need defending forever.
- **Decide this inside the reach record.** Tempting, since both are about roots. Rejected because it
  makes a small safety fix wait on a larger design, and because the two are separable: this record
  says nothing about how MANY roots there may be.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | Which root the server will accept, and now which it will refuse | Yes |
| `internal/rooted` | Unchanged — what a path may reach, given a root | Untouched |
| `cmd/mrw` | One call site: the guard runs before `Serve` | Wiring only |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| `mrw mcp` exits 2 when a fallback root is a filesystem root or the home directory | Public contract, BREAKING for that case | Hosts that set neither `--root` nor `CLAUDE_PROJECT_DIR` and launch from such a directory |
| `mcp.CheckRoot(root, src) error` | New exported function | `cmd/mrw` |
| An explicit `--root` is honoured whatever it names | Unchanged, now asserted | — |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the fallback-root guard and its refusal text | T1 | — | Yes, deliberately — see Consequences |

## Implementation

One task. The guard, its wiring, the contract row, and the documentation that currently describes an
unconditional fallback.

## Consequences

- **Positive:** the worst configuration this server can be in — writable, bound to everything, by
  accident — stops being reachable silently.
- **Positive:** the refusal is actionable, so the fix is one flag rather than an investigation.
- **Negative, and it is a real break:** a host that launched `mrw mcp` from `/` or from the home
  directory and relied on the fallback now fails to start. That is the point, and it is why the
  message names the flag. Anyone who MEANT it can say so with `--root`.
- **Negative:** two more paths could deserve refusing (a system directory, a mount point) and this
  record refuses neither. Narrow beats speculative; the Alternatives name the evidence that would
  widen it.
- **Neutral:** the CLI is unchanged.

## Out of Scope

- Multi-root, repeatable `--root`, and the per-hunk ledger question (deferred: docs/adr/BACKLOG.md — the reach entry, which is ADR-019 and not this record)
- Guarding the CLI's `--root` (permanent: boundary: a typed root is a stated intent, and there is no host fallback to protect against)
- Refusing on project markers such as `.git` (permanent: boundary: named and rejected in Alternatives — it would refuse the document folders this project is trying to serve)
- Changing `ResolveRoot`'s precedence (permanent: boundary: ADR-011 owns it and this record only reads its `Source`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The guard fires on a legitimate setup and a working host stops | Med | Med | Only two directories qualify and only by fallback; the message names `--root`, which restores any of them |
| It is written but never wired, so the binary keeps serving `/` | Low | **High** | §53 drives the BUILT BINARY from `/` and asserts exit 2 — a unit test on `CheckRoot` alone would pass with nothing calling it |
| "Explicit is honoured" quietly stops being true and the guard becomes a path blacklist | Low | Med | Asserted in both the unit test and §53: an explicit `--root /` must exit 0 |
| The home directory comparison misses because of symlinks | Med | Low | Both sides are resolved with `EvalSymlinks` before comparison — `/tmp` vs `/private/tmp` on macOS is the case that motivated it |

## Rollback

Revert the commit. `CheckRoot` is one call in `cmd/mrw`; removing the call restores the previous
behaviour exactly, and nothing else depends on it.

## Follow-ups

- [ ] If a second unintended root is reported that is neither a filesystem root nor a home directory,
      revisit "refuse every fallback" — the alternative M did not take, with the evidence it needs.
