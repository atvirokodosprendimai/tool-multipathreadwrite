# ADR-008: A delete says what it removed, and may say what it expected

**Status:** Proposed
**Date:** 2026-09-01
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (defines `delete` and the guards this extends), ADR-006 (the mirror image: a `replace` with an empty body must say `delete`), ADR-002 (the ledger, which already answers the half of this problem that can be automatic)
**Governs:** `internal/plan/plan.go`, `internal/apply/apply.go`, `README.md`, `scripts/contract.sh`
**Enforced-by:** None — nothing enforces this yet, because nothing is built. The tests that will are named in T1 and T2 and this header is updated when they exist, which is the Follow-up below.
**Invalidates:** none — checked. Grepped every accepted record for `delete`, `body=` and `anchor=`: ADR-001 defines the op and its guards and is extended rather than changed; ADR-006 rejects an empty-bodied `replace` and is the symmetric case, unaffected.
**Served-path change:** a `delete` hunk's receipt names the first and last line it removed, both trimmed, in the human output and in `--json`; and `delete` accepts a body, which is currently a hard parse error, meaning "these are the lines I expect to remove" — a mismatch refuses the whole plan.

## Context

Measured 2026-09-01, on this repository, while implementing ADR-007 T2. The
plan was to drop two unused lines:

    @@ internal/read/walk.go 201-204 delete

The range was one line too long and took the closing brace of the function
above. `mrw` did exactly what it was told and reported:

    ok   internal/read/walk.go 201-204 delete  -4 +0

Four lines removed, status `ok`. The next command — `go build` — failed with
`expected '}', found 'EOF'`, and the fix took another read and another hunk.

**The tool was not wrong, and that is the point.** Three separate mechanisms
were available and none of them applied:

- The **ledger** (ADR-002) catches a file that changed behind mrw's back. This
  file had not changed; the picture was current.
- **`anchor=` and `lines=`** catch an address that does not hold. They were
  available and I did not write them: `@@ … 201-204 delete lines=4
  anchor="var _ = fmt.Sprintf"` is refused, verified the same day.
- **`--check`** (ADR-003) catches a broken result. It was not chained on that
  write, and when it is, it reports after the tree has changed — which is
  correct and is exit 3's whole meaning, but it is not the same as not doing it.

So the gap is narrow and real: `delete` is the only op with **no body**. For
`replace` and `insert-after`, writing the replacement text is itself a check
that the caller knows what they are addressing — you cannot write the new lines
without looking at the old ones. For `delete`, the caller asserts nothing and
removes N lines sight unseen, and the receipt reports a count.

**What cannot be fixed by inference.** A guard mrw derives from the file it just
read is computed from the same bytes it would be checked against, so it always
passes: an auto-`anchor=` asserts nothing, and `lines=` is arithmetic on the
address. The information that would have caught this — what the caller believed
lines 201-204 contained — is not in the system. Only the caller has it.

## Existing Primitives Audit

- **`trim(s string)`** (`internal/apply/apply.go`): already trims a line to 60
  characters for anchor-failure messages. **Reused unchanged** — the receipt's
  bounds use the same function, so one convention covers both.
- **`check.TailLines` and `read`'s `!! N more line(s) withheld`:** the house
  idiom of *bounded output that says what it withheld*. **Reused as the
  principle**: the receipt names two lines whatever the size of the range.
- **`plan.Hunk.Body` and `validate()`:** the body already exists and is already
  rejected for `delete`. **Reshaped** — the rejection becomes a meaning.
- **`sha=`, `lines=`, `anchor=`:** the existing guards. **Unchanged**, and still
  the cheaper answer when a caller wants one; the body is for when the caller
  wants the whole removal pinned rather than its first line.
- **The ledger (ADR-002):** already automatic, already covers drift. **Untouched
  and deliberately not extended** — it answers "did the file change", never
  "did you mean this".

## Decision

**1. A delete receipt names its bounds.** `ok f.go 5-8 delete -4 +0` becomes
`ok f.go 5-8 delete -4 +0 from "}" to "var _ = 2"`, each end passed through the
existing `trim`. `--json` gains `removed_first` and `removed_last` on a delete
hunk. The cost is two short strings per delete hunk **regardless of how many
lines it removes**, which is the property that makes it affordable: mrw exists
to keep output proportional to what the caller needs, and a full echo of a
500-line delete would be the opposite.

A hash was considered and rejected in Alternatives: it is the same size as the
useful answer and carries none of it.

**2. `delete` may take a body, meaning "these are the lines I expect to
remove".** When present, the body must match the addressed lines exactly, or the
hunk fails and — as always — nothing in the plan is written. When absent,
`delete` behaves exactly as it does today.

This is opt-in on purpose. It costs the caller the tokens of the lines being
removed, which is worth it for a delete they want pinned and not worth it for
a two-line one, and that judgement belongs to whoever writes the plan.

**What would make this the wrong decision:** if the bounds in the receipt turn
out to be noise rather than signal — if, after this ships, a reader still cannot
tell a correct delete from a wrong one at a glance. That is checkable: the
contract row for it prints a receipt whose bounds are the brace from the case
above, and if the row has to explain itself in prose to be legible, the format
is wrong rather than the idea.

## Alternatives Considered

- **Echo every deleted line in the receipt:** rejected. Output would grow with
  the size of the range, which fights the reason this tool exists — a 500-line
  delete would print 500 lines back at a caller who is paying for them.
- **A hash of the removed content (`removed_sha`):** rejected, and this is the
  one worth writing down. A hash is informative only against an EXPECTATION, and
  the caller who miscounted a range has none — `removed_sha: a1b2c3d4` tells
  them exactly as much as `removed: 4` did. Where an expectation does exist, it
  is already covered: `sha=` pins the whole file and the ledger pins drift.
- **Have mrw derive `anchor=` from the file it just read:** rejected as
  circular, and the reason generalises: a guard the tool computes from the bytes
  it will check against always passes.
- **Refuse every unguarded `delete` (require `lines=` or `anchor=`):** the
  runner-up, and genuinely close. Rejected as the primary answer because it
  taxes every correct two-line delete to catch the rare wrong one, and because
  the body form is strictly more informative when a caller does want to pin.
  If the bounds in the receipt prove insufficient, this is what to do next.
- **A Go parser or static check inside mrw:** rejected. mrw edits `.go`, `.md`,
  `.sh`, `.json`, plan files and `.gitignore` — six kinds in one session — so a
  Go parser covers one of six while reading as a complete guarantee. It would
  also refuse every edit to an already-broken file, which is when a batch editor
  is most useful. "Does the result compile" is ADR-003's question and `--check`
  already answers it in whatever language the project declares.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/plan` | What a hunk may say, including that a `delete` body is now legal | Yes |
| `internal/apply` | Checking the expected body against the addressed lines, and reporting the bounds | Yes |
| `cmd/mrw` | Unchanged — it prints the receipt `apply` produces | Yes |

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `delete` receipt line gains `from "…" to "…"` | output contract | `internal/apply` | callers, hooks reading the human output |
| `HunkResult.RemovedFirst`, `HunkResult.RemovedLast` (`removed_first`, `removed_last`) | new JSON fields on a delete hunk | `internal/apply` | `--json` consumers, gates |
| `@@ path N-M delete` followed by a body | legal where it is currently a parse error | `internal/plan` | plan authors |
| A body that does not match the addressed lines | new refusal | `internal/apply` | plan authors |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `Hunk.Body` legal for `OpDelete` (T1) | T1 | T2 | No — T2 reads a field T1 stops rejecting |

## Implementation

See `ADR-008-a-delete-says-what-it-removed/tasks/README.md`. Two tasks: the
receipt bounds, then the expected body.

## Consequences

- **Positive:** the incident that produced this record becomes visible at write
  time rather than at build time, for every language and every file type.
- **Positive:** a caller who wants a destructive hunk pinned has a way to say
  so that reads as what it means, rather than an `anchor=` on the first line.
- **Negative:** two more strings in every delete receipt, including the ones
  nobody needed to look at. This is the cost being accepted deliberately; the
  measurement above is why it is bounded rather than full.
- **Negative:** a plan that today relies on `delete` rejecting a body — as a
  deliberate error, which nothing in this repository does — would change
  meaning. Grepped: no such plan exists here.
- **Neutral:** the guards, the ledger and `--check` are all unchanged. This adds
  a fourth mechanism rather than replacing any of the three.

## Out of Scope

- Requiring a guard on every `delete` (deferred: docs/adr/BACKLOG.md)
- Any language-aware validation inside mrw (permanent: boundary: mrw edits six file types in a single session and a validator covering one of them reads as a complete guarantee; "does the result compile" is ADR-003's question, answered by the project's own declared check)
- Echoing removed content in full, or on ops other than `delete` (permanent: boundary: `replace` and the insertions already carry a body the caller wrote, which is the check this adds to the one op that has none)
- A `removed_sha` in the receipt (permanent: boundary: a hash is informative only against an expectation the miscounting caller does not have, and where an expectation exists `sha=` and the ledger already carry it)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The bounds are noise a reader learns to skip | Med | Med | The falsifiable check in Decision: a contract row prints the real receipt from the incident, and if it needs prose to be legible the format is wrong |
| An expected body makes plans long enough that callers stop using `delete` | Low | Low | Opt-in; the unguarded form is unchanged and `anchor=` remains the cheap pin |
| Trimming hides the difference between two similar lines | Low | Low | `trim` is the existing convention at 60 characters, and the address is in the same line of output |

## Rollback

Revert the two commits. The receipt is additive text and two additive JSON
fields; the body is a rejection becoming a permission, so every plan legal
before stays legal. No persistent state and no ledger format change.

## Follow-ups

- [ ] Replace `Enforced-by` with the tests T1 and T2 introduce, once they exist
