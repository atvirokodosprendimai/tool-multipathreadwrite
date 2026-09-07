# ADR-033: A cap of zero is a cap

**Status:** Proposed
**Date:** 2026-09-07
**Owner:** M
**Accepted:** pending
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-006-the-root-confines-reads-too-and-a-replace-must-replace-something.md`, `docs/adr/ADR-025-a-read-that-served-nothing-is-an-error.md`, `docs/adr/ADR-027-an-empty-file-is-created-on-purpose-or-not-at-all.md`
**Governs:** `internal/read/read.go`, `cmd/mrw/main.go`
**Enforced-by:** `internal/read/read_test.go::TestACapOfZeroServesNothing`
**Invalidates:** none — checked
**Served-path change:** `mrw read --max-lines 0` now serves no lines and reports every line withheld, where it used to mean "no cap" and serve the file whole.

## Context

`--max-lines 0` means UNLIMITED today: `internal/read/read.go:442` and `:449` both guard on
`opt.MaxLines > 0`, so a cap of zero is indistinguishable from no cap at all. Two consequences, and
the second is the one that matters:

1. There is no way to say "serve me the header and nothing else" — a request with a cap of zero
   serves everything.
2. **Nothing is reported as withheld**, though `README.md` promises "whatever is withheld is always
   reported". A caller who asked for nothing and received a 3,000-line file was not told the cap was
   ignored. That is a promise the tool makes and does not keep.

**This repository has decided the same question the other way twice.** `body=0` in a plan means an
EMPTY body, not an unbounded one, and ADR-027 made a `create` with no body a refusal precisely
because an exhausted count read as "keep going" once before. `lines=0` is a real assertion about a
zero-length span, not a disabled guard. `--max-lines 0` was the last place where zero meant infinity.

**Against changing it**, and recorded because it is a genuine argument rather than a strawman:
`0 = unlimited` is a widespread CLI convention, and `--stat` already serves the "header only" need.
M decided on 2026-09-07 for consistency with `body=0` and `lines=0`, and because the unreported
withholding is a broken promise regardless of which meaning zero takes.

## Existing Primitives Audit

- **`read.Options.MaxLines int` (`internal/read/read.go:76`)** — the field whose zero value carries
  two meanings. **Changed to `*int`**: nil is "no cap", and a pointer to any value including zero is
  a cap. This is the smallest change that makes the two states distinguishable, and it makes the
  wrong thing impossible to write by omission — the three call sites that leave the field unset get
  nil, which is what they already meant.
- **The `WITHHELD` line (`:443`)** — already the exact reporting this record needs; a cap of zero
  now reaches it instead of being skipped. **Reused unchanged.**
- **`cmd.IsSet` from the CLI library** — how the command layer tells "flag absent" from "flag given
  as 0". **Reused**; no new flag, no new syntax.
- **ADR-025's rule that a read serving nothing is an error** — already decides what a zero cap
  RETURNS: the read reports every line withheld and exits non-zero, as an unsatisfiable read does.
  **Inherited, not re-decided.**

## Decision

**A cap of zero is a cap, and "no cap" is spelled by not passing one.**

`read.Options.MaxLines` becomes `*int`. Nil means unbounded. A non-nil pointer is a budget, and zero
is a budget of zero: every span is reported `WITHHELD`, nothing is served, and the read exits
non-zero under ADR-025 because it served nothing.

`--max-lines` is read through `cmd.IsSet`, so the flag's absence and `--max-lines 0` stop being the
same input. A negative value stays the usage error it already is.

**What would falsify this:** a caller found in the wild passing `--max-lines 0` to mean "no cap" —
scripted from the convention rather than from this tool's docs. Then the change costs them a silent
behaviour flip, and the answer is a refusal telling them to drop the flag, not a return to zero
meaning infinity. Nothing in this repository or its docs uses the form.

## Alternatives Considered

- **Keep `0 = unlimited`, and only fix the unreported withholding.** Rejected: there is nothing to
  report, because with a cap of zero nothing is withheld — the cap is not applied at all. The broken
  promise is a symptom of the overloaded zero, not a separate bug.
- **A sentinel `-1` for "no cap".** Rejected: the zero value of the struct would then mean "serve
  nothing", so the three call sites that omit the field would break by omission. A pointer makes the
  default correct.
- **A separate `Capped bool` beside the int.** Rejected: two fields that must agree, and nothing
  stops a caller setting one without the other. The pointer carries both facts in one place.
- **Add `--max-lines=none` as an explicit spelling.** Rejected: a new surface for something the
  flag's absence already says.

## Component / Boundary Impact

None structural. One field's type changes inside `internal/read`, and `cmd/mrw` learns to distinguish
an absent flag from a zero one. `internal/apply`, `internal/plan`, `internal/seen` and
`internal/state` are untouched; the MCP surface takes no `--max-lines` and is unaffected.

## Wiring & Contract Changes

- `read.Options.MaxLines` changes from `int` to `*int`. It is an internal package, so no published
  API moves.
- `README.md`'s flag table and the `--max-lines` usage string say what zero means.
- Contract §69 drives both spellings through the built binary.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `MaxLines *int` and the zero-is-a-cap rule | T1 | T2 | ⚠ Yes for a caller passing `--max-lines 0` to mean "no cap" |

## Implementation

See `docs/adr/ADR-033-a-cap-of-zero-is-a-cap/tasks/README.md`.

## Consequences

- **Positive:** "serve the header and nothing else" becomes expressible, and the withholding promise
  is kept for every cap including zero.
- **Positive:** zero means zero in all three places this format uses a count — `body=`, `lines=` and
  now `--max-lines`. A reader no longer has to remember which one is the exception.
- **Negative:** a caller who wrote `--max-lines 0` meaning "no cap" now gets nothing served and a
  non-zero exit. It fails loudly rather than silently, which is the direction this tool chooses, but
  it is a behaviour change and it is why this is a record rather than a commit.
- **Neutral:** the three call sites that never set the field are unaffected — nil is what they meant.

## Out of Scope

- A cross-file `--max-lines` budget rather than per-file (deferred: `docs/adr/BACKLOG.md`)
- `MaxResultChars`, the MCP-side ceiling (deferred: `docs/adr/BACKLOG.md` — M chose it on 2026-09-07 and it is its own record)
- Any change to `-C`/context, whose negative-value refusal is already settled (permanent: fact: a negative context is a usage error today and this record does not touch it; citation: file `cmd/mrw/main.go:423`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A call site omits the field and gets "serve nothing" | Low | High | The pointer makes nil the zero value, so omission means unbounded — the same as today. The alternative sentinel had this failure and is rejected above for it |
| The test passes because the read served nothing for an unrelated reason | Medium | High | The fixture asserts the WITHHELD line names the right count as well as the exit code, so an empty file or a missing path cannot satisfy it |
| A real caller relied on `0 = unlimited` | Low | Medium | Nothing in this repository or its docs uses the form; the README and the usage string now say what zero means |

## Rollback

Revert the commit; `MaxLines` returns to `int` and zero to "unlimited". No persistent state is
involved.

## Follow-ups

- [ ] If a caller reports `--max-lines 0` breaking a script, add a refusal naming "drop the flag" rather than restoring zero-means-infinity.
