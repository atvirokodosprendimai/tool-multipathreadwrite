# ADR-005: A write stays inside what the caller scoped, and inside what they saw

**Status:** Accepted
**Date:** 2026-09-01
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (the addresses this constrains), ADR-002 (this narrows its ledger from "mrw hashed it" to "the caller was shown it"), ADR-004 (same class of bug: a tool touching what nobody asked it to touch)
**Governs:** `internal/apply/**`, `internal/read/read.go`, `internal/seen/**`, `internal/plan/plan.go`
**Enforced-by:** `internal/adversarial/filesystem_test.go::TestAPlanCannotWriteOutsideTheRoot`, `internal/adversarial/filesystem_test.go::TestEditingThroughASymlinkKeepsTheSymlink`, `internal/adversarial/filesystem_test.go::TestACRLFFileKeepsItsLineEndings`, `internal/adversarial/ledger_test.go::TestARangedReadLicensesOnlyTheLinesItServed`, `internal/adversarial/planformat_test.go::TestAnOvercountedBodyIsRejected`
**Invalidates:** none — ADR-002 stands and is narrowed, not replaced; the refusal it defines now fires in strictly more cases and never in fewer
**Served-path change:** `mrw write` now refuses a hunk whose path leaves `--root`; follows a symlink instead of replacing it; preserves CRLF and lone-CR line endings; refuses an edit to lines the caller has not been shown (`--stat` and partial reads no longer license one); and refuses a `body=N` whose count would swallow a valid following header.

## Context

Six behaviours were pinned as `TestKnownGap_` by the adversarial suite, each
one found by trying to break the tool rather than by reading it. None was a
decision anybody had made; each was what the code happened to do.

They are one decision, not six, because they share a shape: **mrw did something
outside what the caller had described.** Outside the directory they pointed it
at. Outside the file they addressed, when a symlink was in the way. Outside the
bytes they had, when line endings were rewritten. Outside what they had read,
when a hash stood in for having looked. Outside the hunks they wrote, when an
overcounted `body=` ate the next one.

The tool's whole argument is that a batch is safe because every part of it is
checked and every part of it is reported. A batch that quietly reaches past its
description is the same failure the project exists to refuse, one level up from
the hunk.

## Decision

**1. `--root` is a boundary, not a starting point.** A hunk whose path resolves
outside the root is refused, per hunk, with both paths named. Symlinks are
resolved before the check, so following one out of the tree is refused too. A
path that climbs and returns (`sub/../sub/a.go`) is inside and applies.

**2. A write follows a symlink rather than replacing it.** The temp-file-plus-
rename that makes a crash mid-write safe now creates its temp file beside the
RESOLVED target. Previously the rename replaced the link with a regular file:
the edit landed somewhere new, the file the caller meant stayed untouched, and
nothing in the receipt said the tree's shape had changed.

**3. Line endings are the file's, not mrw's.** A file whose every newline is
`\r\n` is CRLF and comes back CRLF; a file with no `\n` but containing `\r` is
old-Mac and its interior is addressable rather than one unsplittable line;
everything else, including a file that MIXES conventions, is LF, where a stray
`\r` stays part of its line's content. That last case is what keeps untouched
lines byte-identical — which is not cosmetic: `internal/seen` hashes the raw
file, so a normalising editor would make every such file read as "changed
behind mrw's back".

**4. The ledger records what was SHOWN, not what was hashed.** ADR-002's reason
was always that "a range address like 42-58 only means something in the version
of the file those numbers were counted in" — and the caller is who counts them.
So an observation carries the spans a read actually printed: `--stat` observes
nothing, a ranged read observes its ranges, a withheld span is not observed, a
whole-file read observes everything, and a write observes everything, because
mrw produced every line. `--force` still bypasses it.

**5. An overcounted `body=N` is refused when it would swallow a header.** A
counted body line is rejected only if it parses as a COMPLETE, VALID header, so
prose about the plan format still passes through — including the README's own
examples, whose trailing text is not `key=value`.

## Alternatives Considered

- **Leave the root open and document it.** Rejected: the receipt prints the
  relative path, so an escape is invisible in the output as well as in the
  plan. A boundary nobody can see is not a documented behaviour, it is a trap.
- **Normalise all line endings to LF.** Rejected: it makes mrw a formatter, and
  it breaks the ledger's identity with the raw bytes.
- **Keep the whole-file ledger and document that `--stat` licenses edits.**
  Rejected: it makes the guard's name false. The point of ADR-002 is the
  caller's picture, and a stat leaves them without one.
- **Refuse every body line that begins with `@@ `.** Rejected: it breaks the
  escape hatch `body=` exists for, on exactly the documents that explain it.
- **Require reading whole files.** Rejected: ranged reads are the tool's
  economics. Span tracking keeps both.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/apply` | Where a write may land (root boundary, symlink resolution) and what bytes it writes (line endings) | Yes — changes when write policy changes |
| `internal/seen` | What an observation IS: sha plus served spans, and how two observations of one file combine | Yes |
| `internal/read` | Reporting which spans it actually printed; it decides nothing about permission | Yes |
| `internal/plan` | Refusing a `body=` count that would swallow a header | Yes |

`internal/apply` gains an import of `internal/seen` for the observation type
only, aliased as `apply.Seen`. The engine still decides; the ledger only says
what was seen.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `seen.Observation` (`SHA`, `Spans`) replacing a bare sha string | breaking API change | `internal/seen` | `internal/apply`, `internal/read`, `cmd/mrw` |
| Ledger file format `<sha>  <spans>  <path>` | backward-compatible format change — a two-field line is read as a whole-file observation | `internal/seen` | any reader of the ledger |
| `read.Run` returning `map[string]seen.Observation` | breaking API change | `internal/read` | `cmd/mrw`, tests |
| `apply.Options.Seen` typed `map[string]apply.Seen` | breaking API change | `internal/apply` | `cmd/mrw`, tests |
| `mrw seen` output gains a served column | output contract | `cmd/mrw` | callers reading it |
| A hunk path outside `--root` → exit 1 | new refusal | `internal/apply` | callers, CI |
| `mrw read --stat` no longer licenses a write | narrowed permission | `internal/read` | callers, `scripts/contract.sh` |

## Consequences

- A plan that used to reach outside the root now fails loudly. This is a
  breaking change for anyone who relied on it; `-C` at the higher directory is
  the honest spelling of that intent.
- `mrw read --stat` is now purely informational. Workflows that used it to
  satisfy the ledger must do a real read — `scripts/contract.sh` needed exactly
  that change, which is the clearest evidence the old behaviour was load-bearing
  in a way nobody had noticed.
- The ledger's on-disk format gains a spans column. A pre-spans ledger is read
  as a whole-file observation, so nothing is lost by upgrading.
- The ledger accumulates spans across reads of an unchanged file, and resets
  when the sha changes: spans counted in one version say nothing about another.
