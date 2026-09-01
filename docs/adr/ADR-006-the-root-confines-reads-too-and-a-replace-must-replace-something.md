# ADR-006: The root confines reads too, and a replace must replace something

**Status:** Accepted
**Date:** 2026-09-01
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (defines `replace` and the plan grammar this narrows), ADR-005 (drew the root boundary, on the write path only — this finishes the job it started)
**Governs:** `internal/rooted/**`, `internal/read/read.go`, `internal/apply/apply.go`, `internal/plan/plan.go`
**Enforced-by:** `internal/adversarial/filesystem_test.go::TestAReadCannotLeaveTheRoot`, `internal/adversarial/filesystem_test.go::TestAReadCannotFollowASymlinkOutOfTheRoot`, `internal/adversarial/planformat_test.go::TestAReplaceWithNoBodyIsRejected`, `internal/rooted/rooted_test.go::TestASiblingWithASharedPrefixIsOutside`
**Invalidates:** none — ADR-005's refusal is unchanged and now fires on the read path as well; ADR-001's `replace` keeps its address grammar and loses only the empty body
**Served-path change:** `mrw read` now refuses a path resolving outside `--root`, including through a symlink, where it previously served the file at exit 0; and a `replace` hunk with no body is a parse error (exit 2) where it previously deleted the addressed lines and reported `ok`.

## Context

Both were reported by someone using the tool rather than reading it, which is
the only way either would have been found: each looks correct in the diff and
wrong only when you run it.

**A boundary that holds on one path is not a boundary.** ADR-005 refused a hunk
whose path leaves the root, and the refusal is precise and well tested. It was
implemented in `internal/apply`, so it protected the write path and nothing
else: `mrw --root . read ../outside.txt` served the file at exit 0, and a
symlink in the tree pointing at `/etc/hosts` was followed and printed. The
identical path in a write plan was refused by name, in the same session, by the
same binary. Which of the two you got depended on which function you happened to
call.

**A `replace` with no body deletes.** `@@ f.txt 2 replace` with nothing after it
applied cleanly: exit 0, `--json` reporting `"status": "ok"`, and line 2 gone.
So a plan whose body was lost in transit — a truncated emission, an editor
eating the last line, a pipe closed early — removes code while the receipt a
hook reads says it succeeded. That is the failure this format exists to refuse,
reached through the one op that did not check.

The tell that it was an oversight rather than a design: the parser already
polices the mirror image. `delete` WITH a body is a hard parse error. One
direction was checked and the other was not.

## Decision

**1. `--root` confines reads.** A spec whose path resolves outside the root is
refused, counted as a problem, and observed by nothing — no ledger entry, no
content printed. The message names both paths and says to point `--root` where
you mean.

**2. The boundary lives in one place.** `internal/rooted.Resolve` is the single
implementation, used by both `internal/apply` and `internal/read`. Duplicating
it would have re-created the divergence in a slower form.

**3. A `replace` needs a body.** An empty one is a parse error naming `delete`,
which is what the caller meant if the emptiness was deliberate. `body=0` on a
replace fails identically, so the check cannot be sidestepped by spelling.

## Alternatives Considered

- **Leave reads unconfined and document it.** Rejected. The asymmetry is the
  problem: two paths into the same tree disagreeing is a rule nobody can hold
  in their head, and the read path is the one that leaks content.
- **Confine reads but follow symlinks.** Rejected — a symlink was the reported
  escape, not a hypothetical one.
- **Make an empty `replace` a warning that still applies.** Rejected. A warning
  on a path that deletes code is the "printed it and did it anyway" shape the
  project refuses everywhere else.
- **Treat an empty `replace` as `delete` silently.** Rejected for the same
  reason, plus it makes a truncated plan indistinguishable from a deliberate
  one — exactly the ambiguity the error resolves.
- **Copy `resolve` into `internal/read`.** Rejected: two copies of a boundary
  drift, and this ADR exists because one copy already did.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/rooted` | What "inside the root" means, for every caller | Yes — changes when the boundary rule changes |
| `internal/read` | Refusing and reporting a spec outside the root; it decides nothing about the rule | Yes |
| `internal/apply` | Same, for hunks; keeps its own message about what a plan may change | Yes |
| `internal/plan` | Refusing a hunk whose op and body disagree | Yes |

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `rooted.Resolve`, `rooted.Contains` | new internal API | `internal/rooted` | `internal/apply`, `internal/read` |
| `mrw read <path outside root>` → `REFUSED` line, exit 1 | new refusal | `internal/read` | callers, CI |
| `mrw read <symlink out of tree>` → `REFUSED` line, exit 1 | new refusal | `internal/read` | callers, CI |
| `@@ p N replace` with no body → exit 2 | new parse error | `internal/plan` | plan authors, hooks |

## Consequences

- A workflow that read files outside the checkout with `-C` pointed inside it
  now fails. Pointing `--root` at the directory you actually mean is the honest
  spelling, and the refusal says so.
- A plan that used an empty `replace` to delete lines must say `delete`. No
  capability is lost; the two spellings collapse to the one that is legible.
- `mrw read` gains a third reason to exit 1 (refused, alongside unreadable and
  no-match). The output distinguishes them; the exit status does not, which is
  the open question ADR-003's exit table already carries.
- The boundary is now testable in isolation, which is how the shared-prefix trap
  (`/repo-backup` counting as inside `/repo`) got a test rather than a comment.
