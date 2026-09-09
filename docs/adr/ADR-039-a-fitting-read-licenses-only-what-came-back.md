# ADR-039: A fitting read licenses only what came back

**Status:** Accepted
**Date:** 2026-09-09
**Owner:** M
**Accepted:** 2026-09-09 by M — *"accpted"*
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-024-a-page-is-known-by-its-served-text.md`, `docs/adr/ADR-028-a-guard-does-not-read-back-what-was-not-served.md`, `docs/adr/ADR-031-a-page-licenses-only-what-came-back.md`, `docs/adr/ADR-032-the-ceiling-is-the-callers.md`, `docs/adr/BACKLOG.md`
**Governs:** `internal/mcp/tools.go`, `internal/mcp/ack.go`, `internal/mcp/mcp.go`, `internal/mcp/instructions.go`
**Enforced-by:** `internal/mcp/ack_test.go::TestAFittingReadLicensesOnlyWhatCameBack`
**Invalidates:** ADR-031 — the Served-path clause that a read that fits is unchanged and carries no checkpoints, and its Out of Scope "Applying checkpoints to small reads that fit whole"
**Served-path change:** an MCP `mrw_read` that FIT in one answer now carries the same `-- ck` brackets a page does, records nothing until the caller echoes them, and a following write without `ack` is refused exactly as a paged one is.

## Context

**The class this record governs.** Every MCP `mrw_read` that serves numbered file content and
currently calls `seen.Record` before returning, without paging. Enumerated 2026-09-09 with

```
rg -n 'seen\.Record' --glob '*.go'
```

Five live call sites, one function (`seen.Record`; `save` is private). Enumerated 2026-09-09.
This record changes **one**: `internal/mcp/tools.go` in `readTool`, the fitting-serve return that
still records on serve. Members left out, and why:

- `cmd/mrw/main.go` CLI read — no host sits between stdout and the caller. ADR-031 made that a
  permanent boundary for acknowledgement; this record keeps it.
- `cmd/mrw/main.go` CLI write — records what was *written*, not what was shown.
- `internal/mcp/tools.go` MCP write — same: records what was written.
- `internal/mcp/ack.go` `promote` — already the pending→ledger path. Reused, not replaced.
- A grep INDEX, a served-nothing error (ADR-025), and a page (`firstPage` → `hold`) — none of
  those call `seen.Record` on the fitting return.

M said *"ok, start"* after the recommendation that a fitting read has no ack/checkpoint path, so
“I saw this” is uneven vs paged reads. The deferral lives in `docs/adr/BACKLOG.md` under
**From ADR-032**, titled "A read that FITS could record on serve rather than on acknowledgement",
and was first filed as the remaining half of ADR-024 Decision 4 / ADR-031 Out of Scope.

**Why pages have checkpoints, and why fitting reads do not — today.** ADR-031 exists because a host
can cut a result after mrw returned and before the model reads it. Measured 2026-09-05 on Claude
Code 2.1.261 (`docs/curve/reading-18-result.md`): mrw served lines 1-2727, the host kept 1-90 and
2644-2727, the ledger recorded 1-2727, and `@@ f.txt 1500 replace` applied at exit 0. mrw cannot
observe that from inside the server (ADR-024): a cut result and a delivered one are identical to
it. Checkpoints do not detect truncation; they make an honest caller *able to tell*, by bracketing
each run so a middle cut leaves neither end intact. Fitting reads skipped that path because
nothing was *paged* — and "nothing was cut" is the same server-side belief ADR-031 rejected for
pages. The BACKLOG said so, and called it a consistency argument rather than a measured defect,
because no host has yet been measured truncating a result *under* the advertised ceiling.

**What licenses a write today for an unpaged MCP read.** `readTool` composes the result, confirms
`encodedSize` is under the ceiling, then `seen.Record(root, observed)` and returns. No `ack`, no
pending entry, no markers. `TestAReadUnderTheLimitIsUnchanged` asserts that ledger entry. A CLI
`mrw read` of the same file licenses the same way at `cmd/mrw/main.go`, and that is not this hole:
there is no envelope to cut.

**The cost argument in the BACKLOG is wrong in the 2-call recipe.** It feared a second round trip
on every read. `ack` already rides on `mrw_write` (and on the next `mrw_read`): promote runs at the
start of both tools. Today's MCP recipe is read then write. The new recipe is read then
write-with-ack. Still two calls. The extra call appears only if a caller acknowledges on a
separate read they did not otherwise need.

**`hold` is not ready as-is.** `internal/mcp/ack.go` `hold` takes one path out of the observation
map — *"One observation per path here — a paged read serves one file."* A fitting serve can be
many files, and `interleave` groups by consecutive `NNN|` numbers without knowing which file they
belong to, so line numbers restarting at 1 on file B would merge into file A's span. Closing the
class without fixing that licenses the wrong file.

## Existing Primitives Audit

- **`-- ck` brackets, `interleave`, `promote`, `AckRule`, the pending store, `ack` on both tools
  (`internal/mcp/ack.go`, ADR-031):** **reused as-is** for format, randomness, consumption-on-promote,
  and the honest-caller rule. Not a second acknowledgement design.
- **`hold`:** **reshaped.** Same pending records (`path`, `sha`, span, seq). It must key each
  checkpoint to the file whose lines it bracketed, not to the first map entry.
- **`readTool`'s fitting return and `encodedSize` check:** **the selection site.** Markers and any
  footer are composed *before* `hold`, and an encode that now exceeds the ceiling takes the existing
  overflow path (page / index / refusal) and holds nothing — the same order `firstPage` already
  uses. Inventing a second cap would be the ADR-031 three-attempt failure again.
- **`anchor=` as the echo-back:** audited and **not** taken. ADR-031 rejected it: per-write, not
  per-read, and ADR-028 made it strictly weaker than the ledger.
- **Optional checkpoints while still recording on serve:** audited and **not** taken. Ceremony. The
  hole is the Record-on-serve, not the missing markers.

## Decision

**An MCP read that served numbered file content, whether it paged or fit, interleaves the same
unguessable checkpoints ADR-031 already defined, holds them pending per file, and records only the
spans whose checkpoints the caller echoes back.**

1. After a fitting encode that will actually be sent, split the served report on the existing
   `==> path` headers (`internal/read/read.go`), `interleave` each file's slice, and `hold` each
   file's spans against that file's path and sha. Reassemble the marked report. Do not call
   `seen.Record` on this path.
2. Measure `encodedSize` of the composed result *after* markers (and a one-line "this serve
   licenses nothing until ack" footer built from `AckRule`, the page's existing sentence retargeted).
   If it no longer fits, do not hold; take the existing overflow path. A read that was under the
   ceiling by a few marker lines becomes a page or a refusal, never a truncated fitting serve.
3. `ack` does not change shape. Promote already consumes a pending id. A fitting serve with no
   numbered lines (empty observed, index, refusal) still carries none and still records none.
4. Teaching retargets: schema copy, handshake, and the page footer that currently say "paged read"
   mean "read that served lines". `AckRule` itself does not change. `maxInstructionsChars` stays
   4096; if the retarget will not fit, shorten MCP-only prose, never raise the bound.

**On a following `mrw_write` without `ack`, a fitting MCP read licenses NOTHING.** That is a
breaking change for every existing MCP caller that writes after a small read and never sends `ack`.
It fails safe, and it is the same break ADR-031 already shipped for pages.

**What this does not claim.** It does not claim a host has been measured truncating an under-ceiling
result. It claims the licensing premise is the same belief ADR-031 already refused to encode. It
does not detect truncation (ADR-024). It does not make `anchor=` a substitute (ADR-028, ADR-031).
It does not lock the target file (ADR-002). It does not generate `AGENTS.md` from `guide.Shared()`.

**What Accept settles as written, unless M names a fork:** MCP-only (CLI unchanged); required
ack, not optional checkpoints-while-still-Recording; `hold` keyed per file on a multi-file
fitting serve. Naming a fork is what changes the record; *"ok, start"* authorised drafting, not
those three.

**Go/no-go, checked during execution. If any fails, the task is withdrawn rather than shipped:**

- **`internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`,
  `internal/state` stay byte-identical** against the merge-base. This record owns `internal/mcp`.
- **`maxInstructionsChars` is still the literal 4096.**
- **`go.mod` still declares exactly one requirement.**
- **`AckRule` is still the ADR-031 sentence.** Checkpoints stay random 16-hex, bracketed, consumed
  on promote. No second marker design.
- **A CLI `mrw read` of a fitting file still licenses a following write without `ack`.**

**What would falsify this:** a host that truncates a fitting result and then reconstructs both
markers and the stated count of numbered lines between them. Same falsifier as ADR-031. Or: a
fitting two-file serve whose checkpoints all promote onto one of the files — that is `hold` as it
stands today, and T1's two-file fixture is the data that can produce the failure.

## Alternatives Considered

- **Leave fitting reads recording on serve.** Today's state, and the BACKLOG's "until a host is
  measured truncating under the ceiling". Rejected by M's *"ok, start"* on the uneven-proof
  recommendation. The measurement may never come: a result under the ceiling is the case a host is
  *least* likely to cut, and waiting for it leaves ADR-002's promise host-shaped.
- **Emit checkpoints but still `seen.Record` on serve (optional ack).** Rejected: the hole is the
  Record, not the missing tokens. A caller that ignores markers writes exactly as today.
- **Checkpoints only when the result is "close to the ceiling".** Rejected: invents a threshold mrw
  cannot justify, and "cannot observe truncation" (ADR-024) is why a size heuristic is not a
  delivery proof.
- **Apply the same ack requirement to the CLI.** Rejected: ADR-031's Out of Scope, restated by
  ADR-032. There is no host between `mrw read` and the caller. Reopen only if a CLI consumer is
  measured dropping stdout the way Claude Code dropped a page.
- **`anchor=` as the fitting-read echo-back.** Rejected by ADR-031: it licenses a hunk, not a read.
- **A second pending format for small reads (one token at the end).** Rejected by the reading-18
  cut, which kept both ends. Small does not change the shape of a middle cut.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | The MCP read surface, pending store, and ack teaching | Yes — this is the fitting-serve hole |
| `internal/seen` | Unchanged. `promote` still calls `Record` | No — byte-identical go/no-go |
| `cmd/mrw` | Unchanged CLI licensing | No — Out of Scope |
| `internal/apply` / `internal/plan` | Unchanged | No — do not lock the target file; do not touch the apply/plan engine |

No new module. No architecture-doc delta (this repository has no `docs/architecture.md`).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| MCP `mrw_read` fitting return | markers + pending; no `seen.Record` | T1 | callers, T2, T3 |
| `hold` | one pending entry per (checkpoint, path) | T1 | `promote` (already) |
| `ack` schema copy | "paged read" → "read that served lines" | T3 | schema-driven hosts |
| handshake `instructions` | same retarget; bound stays 4096 | T3 | MCP hosts |
| `scripts/contract.sh` | new `# 77.` | T3 | CI |
| `TestAReadUnderTheLimitIsUnchanged` | asserts markers and an empty ledger, not a Record | T1 | the old contract's own test |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| fitting serves held pending per file | T1 | T2, T3 | ⚠ Yes for MCP callers: a fitting read licenses nothing until acked |
| CLI fitting reads still license without ack | T2 | T3 | No — T3's non-MCP rows must stay green |

## Implementation

See `docs/adr/ADR-039-a-fitting-read-licenses-only-what-came-back/tasks/README.md`.

## Consequences

- **Positive:** MCP licensing is one rule. "I saw this" is the same proof for a 3-line file and a
  2,727-line page.
- **Positive:** the 2-call recipe does not grow a round trip. `ack` already rides on `mrw_write`.
- **Negative, and the reason this needs M's word:** every MCP caller that today writes after a
  fitting read without `ack` starts getting ledger refusals. Fail-safe, breaking, same class as
  ADR-031.
- **Negative:** marker lines (~1% at `ckEvery=200`, more on a tiny file because a 3-line serve still
  pays an open, a close, and a footer) can push a just-under-ceiling result onto the overflow path.
- **Neutral:** CLI `mrw read` / `scripts/contract.sh` non-MCP rows behave identically.
- **Neutral:** a grep index still carries no checkpoints and still licenses nothing.

## Out of Scope

- The CLI path (permanent: boundary: no host sits between `mrw read` and the caller; ADR-031 drew this line for acknowledgement and this record does not reopen it)
- Detecting truncation from inside the server (permanent: fact: a cut result and a delivered one are identical to the server; citation: file `docs/adr/ADR-024-a-page-is-known-by-its-served-text.md:28`)
- A grep INDEX carrying checkpoints (permanent: boundary: an index serves no numbered lines, so bracketing has nothing to cover, and it licenses nothing today)
- Requiring `ack` on a CLI fitting read (permanent: boundary: same host-absence as the CLI path above)
- Locking the target file a plan writes (permanent: boundary: ADR-002 parked that; this record must not touch `internal/apply`)
- Raising `maxInstructionsChars` / 4096 (permanent: boundary: M's standing instruction and ADR-037's go/no-go; shorten MCP-only prose)
- Generating `AGENTS.md` from `guide.Shared()` (permanent: boundary: ADR-037 deferred that; T3 edits the static ack paragraph, it does not generate the file)
- ADR-019 Desktop reach / multi-root (permanent: boundary: that number is reserved and this is 039)
- Making `anchor=` the fitting-read echo-back (permanent: boundary: ADR-031 rejected it; ADR-028 made the guard weaker than the ledger)
- A host-truncation measurement of an under-ceiling result (deferred: `docs/adr/BACKLOG.md`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| `hold` still takes one path, so a two-file fitting serve licenses the wrong file | **High** | High | T1's fixture is two files; ack only A's checkpoints; write to B must be refused. A single-file test is green against today's `hold` |
| `TestAReadUnderTheLimitIsUnchanged` stays green because it still wants a ledger entry | **High** | High — the old test protects the hole | T1 rewrites that test in the same change that removes `seen.Record`; a leftover Record fails the new assertion |
| Marker bytes push a near-ceiling fitting read over the cap | Med | Med | Measure `encodedSize` after markers, before `hold`; overflow uses the existing page/index/refusal path |
| Handshake overflows 4096 | Med | High | Go/no-go: retarget "paged" to "served", do not add a paragraph; Stop Condition if the bound is the proposed fix |
| Callers echo every checkpoint without counting | Med | High | Unpreventable server-side, stated in ADR-031; `AckRule` is unchanged; T3 must not weaken it |
| A fixture that only cuts the tail proves a single end-marker | Low | High | Fitting reads still use ADR-031 brackets. T1 does not invent a one-token small-read design |

## Rollback

Revert the commit. Fitting MCP reads `seen.Record` on serve again. Pending entries already held
are orphaned in the state directory and harmless (same as ADR-031 rollback). CLI unaffected. No
ledger format change.

## Follow-ups

- [ ] If a host is measured truncating a result *under* the advertised ceiling, file the measurement
      beside reading-18 rather than treating this record as that evidence — this is a consistency
      close, not that measurement.
