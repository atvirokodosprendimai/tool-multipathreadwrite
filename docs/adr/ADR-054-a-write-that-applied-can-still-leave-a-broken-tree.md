# ADR-054: A write that applied can still leave a broken tree

**Status:** Accepted
**Accepted:** 2026-09-13 by M — *"accepted"*
**Date:** 2026-09-13
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-009, ADR-044, ADR-048, ADR-052, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `internal/apply/apply.go`, `internal/authoring/authoring.go`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/writecheck_test.go::TestWriteRunsTheCheckByDefault`
**Invalidates:** ADR-009 — the clause of its Decision reading "the decision to leave `--check` opt-in"
**Served-path change:** `mrw write` runs the project's check after a successful apply when at least one written path is not prose, unless `--no-check`; a delimiter-balance delta prints on a non-prose hunk and does not fail it; `mrw stats` always prints `failed_check`, including zero, plus how many landed writes then failed the check.
**Notes:** Zeus field report 2026-09-13, filed to BACKLOG under From ADR-052. The author read `internal/apply/apply.go:960` before proposing and dropped two ideas after that read. M, 2026-09-13: *"okay, design these features, they seem important"*. Codex reviews of this Proposed record, 2026-09-13: Names() claim held; `packages()` is Go-only; balance on markdown would be loud; landed denominator includes `check_not_run`; arm 2 does not catch a balanced insert (one of three Zeus cases); data files (`.jsonl`) are not prose and still pay the suite. Folded below. Execute only on Accept. Branch from `origin/main` (ADR-053 is there; this working tree may not be). 4096 stays. 019 A stands. No parser. No MCP check. No default echo. No Rust `packages()`. No harness `covers` glob.

## Context

**The class this record governs.** Every CLI write that can land a plan, the receipt of an applied hunk, and the authoring tally of what became of those plans. Enumerated 2026-09-13 with

```
git ls-files cmd/mrw/main.go internal/apply/apply.go internal/authoring/authoring.go scripts/contract.sh
```

Four tracked files (1+1+1+1). Members left out: `internal/mcp` (ADR-044: MCP never runs `--check`; a third tool is cargo), Shared() (4096; teach on `write --help`), a target-syntax parser (ADR-048), the neighbour license (ADR-052 already refuses `end > start` without End+1; extending it to single-line addresses catches none of the Zeus cases).

**Why this is a record.** ADR-052 refuses the wrap-tail miss at `apply.go:960` (`end > start` and ledger does not cover End+1). `--check`, `--echo-pad`, `sha=` and the ledger already exist. The three Zeus breakages were **single-line addresses with multi-line bodies**, so `end > start` is false; End+1 had been served. No ledger rule catches what the body did to the structure. That checkout's tally: 259 plans, 251 applied (96.9%). Three of those 251 left the tree uncompilable and mrw reported success on every one. ADR-003 already named the caller who most needs the check as the one who will skip it. ADR-009 then left `--check` opt-in, so `failed_check` never incremented.

## Existing Primitives Audit

- **`mrw write --check` / `internal/check`.** Reused. ADR-003's exit table, no-revert, process-is-the-verdict, inferred `go test` when `go.mod` exists, and missing-check = exit 2 when a check was *demanded*. This record changes who has to ask, and when the default is allowed to ask.
- **`command()` / `packages()`.** Reused unchanged. `command()` at `internal/check/check.go:347` returns the scoped form only when `ScopedCheck` is non-empty and `packages()` maps every path. `packages()` is Go-specific: its comment says a `.go` FILE maps to its own directory (`check.go:385`). A markdown path returns nil and the caller falls back to the whole-project `Check`. Zeus is Rust; its `.quality-harness.json` declares only `check` (`cargo clippy --workspace --all-targets … && cargo nextest run --workspace …`). Measured 2026-09-13 in that checkout: 58–107 seconds for the nextest half alone. This record does not invent a Rust mapper and does not add a harness `covers` glob.
- **`authoring.FailedCheck` / `check_not_run`.** Already in the closed vocabulary (`internal/authoring/authoring.go`). Not a sixth counter. `Names()` iterates only keys present in the tally, so a counter that never incremented has no key and cannot print — Zeus read three names, no `failed_check`. `CheckNotRun` means written, but no check could run (exit 2).
- **`HunkResult.Echo` (ADR-052).** Precedent for visibility that does not fail the hunk. The balance delta is the same shape: a field on the hunk, omitted when there is nothing to say.
- **A parser / AST / brace-in-strings checker.** Audited and rejected (ADR-048). Naive rune counts will miscount braces inside string literals, which is why this reports rather than refuses.
- **MCP `mrw_write` running the check.** Audited and rejected (ADR-044). Default check is CLI `write` only.

## Decision

Three arms, one record. Neighbour-license-on-single-line is not an arm.

**Prose paths (one closed list, both arm 1 and arm 2).** A path is prose when `filepath.Ext` lowercased is one of `.md`, `.markdown`, `.txt`, `.rst`, `.adoc`. Growing this list is a new record, not a silent execute change.

**1. CLI `write` runs the project's check after a successful apply only when the plan touched a file the check could plausibly cover, unless `--no-check`.**

`--check` stays as an explicit demand. `--no-check` opts out. Both together is usage (exit 2).

The default runs only when **all** of these hold: the apply succeeded (`res.Applied && res.Failed == 0`); a check **exists** (declared in `.quality-harness.json`, or inferred from `go.mod` — `check.Load` today); `--no-check` was not set; and **at least one written path is not prose**. A markdown-only plan in a harnessed tree is Applied, exit 0, does not spawn the check, and records `applied` (the same Outcome as `--no-check` — not `check_not_run`, which is exit 2). A mixed plan (prose + code) runs the check.

This is option 2 of three, taken after the 2026-09-13 review. Option 1 — default only when `command()` returns scoped — would never fire in Zeus: `ScopedCheck` is empty and `packages()` does not map `.rs`. Option 3 — run the whole-project command on every write and name the cost in Consequences — was rejected because the motivating checkout's 259 plans were mostly markdown ADR and spec edits, and that command is a full workspace clippy+nextest (58–107s for nextest alone, measured 2026-09-13).

When the plan is eligible and a check exists, run it after apply. Failures stay exit 3; the tree is not reverted (ADR-003). `{packages}` / `{files}` stay as `command()` already substitutes them. When `packages()` cannot map (non-Go paths), the whole-project `Check` runs — a Zeus `.rs` write pays clippy+nextest; that is the remaining cost, named in Consequences, and `--no-check` is the escape. Do not invent a Rust `packages()`. Do not add a `covers` glob to the harness.

When **no check exists** (no harness, no `go.mod` — Desktop / CSV trees): the default does **not** invent exit 2. The write is Applied, exit 0. Explicit `--check` with no command still means "I demanded a check" and stays exit 2 (ADR-003 rule 2). That split is what keeps 019's population working.

Explicit `--check` on a prose-only plan still runs the check: the caller demanded it.

`--dry-run` implies no check (nothing was written). Explicit `--check --dry-run` stays usage, as today.

MCP `mrw_write` is unchanged: it still does not run a check. `internal/check.command` and `packages` stay as they are.

**2. A delimiter-balance delta on the receipt — visibility, not a refuse; omitted on prose.**

For an applied `replace`, `insert`, or `delete` whose path is **not** prose, count net `{`/`}` `(`/`)` `[`/`]` in the original addressed lines and in the body (insert: original net is 0; delete: body net is 0). If any family's net differs, the hunk carries a `balance` string naming the family and the two nets, e.g. `{ +1 → 0}`. The hunk stays `ok`. Quiet hides `ok` lines and their delta, the same as Echo. Naive counting; no string-literal lexer. YAML indent-reparent has no tokens and will not report — that miss stays BACKLOG.

A prose path omits `Balance` even when the nets differ. Markdown ADR bodies in Zeus contained `{packages}`, `{files}`, JSON blocks and shell fences; a delta there would have fired on a large share of 259 plans and meant nothing. Unrestricted, that trains callers to ignore the line (High).

Being code is not the same as producing a delta. Of the three Zeus breakages, two report: a stub opening line (addressed `{` net +1, body net 0) and a stale-address replace in `message.rs` (balanced line, body with an unmatched `}`). The third does not: an `insert-before` whose body was a run of complete `#[test] fn …() { … }` definitions — net 0 — landing inside an existing function. Original net for an insert is 0 by this Decision's own rule, so the delta is 0 and nothing prints. The file was left with one function unclosed and a stray `}` at the end; cargo said `unexpected closing delimiter`. Arm 1 catches it; arm 2 does not. That is the same test this record applied to the neighbour license, and it does not weaken the arm: a balanced insert in the wrong place is invisible to delimiter arithmetic.

**3. `mrw stats` always prints `failed_check`, including zero, and one derived line about landed writes.**

Do not add a sixth `Outcome`. The five names are already the exit-status projection. Print all five even at zero, so an all-green tally cannot hide the column. Derived line, with the denominator named:

`landed writes: N; failed_check F of those (x%).`

`N = applied + failed_check + check_not_run`. `check_not_run` is in N because its doc comment is "written, but no check could run" — the tree changed (exit 2). Landed is not "wrote and was checked". `--no-check` and the prose skip both record `applied`, so they sit in N as successes, not as `failed_check`. F/N is failed checks among landed writes, not among verified writes; a reader who assumes the latter misreads the percentage in the direction that flatters the tool. `--no-check` still records `applied`; that is the opt-out, and it is visible because `failed_check` is no longer omitted at zero. JSON: the five keys always present; add `landed` and `failed_check_of_landed` so a script does not re-derive.

## Alternatives Considered

- **Default only when `command()` returns scoped.** Rejected: Zeus never scopes (`ScopedCheck` empty, `packages()` is Go-only). The three `.rs` breakages would still report success.
- **Default on every write that has a check command, and name the whole-project cost in Consequences.** Rejected: the field report's own tree would pay clippy+nextest on markdown ADR/spec plans (259 of them). The Risk row that said "Keep scoped `{packages}`" does not exist there and cannot be written there.
- **Default check always, missing command = exit 2.** Rejected: breaks Desktop / non-Go trees that ADR-019 exists to serve. Explicit `--check` already means that.
- **Default echo instead of default check.** Rejected: ADR-052 already rejected default echo on cost; a cover-gated check runs the project's own command when the plan touched code.
- **Extend the neighbour license to single-line addresses.** Rejected as a *fix*: the three Zeus cases would not have fired (address was one line; End+1 was served). Remains an open question on BACKLOG, not armed.
- **Refuse on a balance delta.** Rejected: string literals and generated code will false-positive; ADR-048. Report only.
- **Balance on every file type, and tell the reader to expect prose noise.** Rejected: High risk of training callers to ignore the line. Restricted to non-prose instead.
- **A sixth Outcome `check_skipped` for `--no-check` or for prose.** Rejected: the five names match the five exits; a sixth is a second opinion. Opt-out and prose skip stay `applied`.
- **Add `.jsonl` / `.yml` / `.json` to the prose list.** Rejected: the list stays closed. `.toml` must trigger (`Cargo.toml`). Name the data-file cost in Consequences instead.
- **A harness `covers` glob / a Rust `packages()`.** Rejected: new config is cargo (ADR-044 adjacent); a language table is a parser's cousin. The closed prose list is the cover heuristic.
- **MCP `mrw_write` grows a `check` flag.** Rejected: ADR-044. Cargo.
- **Revert on failed check.** Rejected: ADR-003.

## Component / Boundary Impact

`cmd/mrw` `writeCmd` selects default check / `--no-check` / prose skip. `internal/check` is unchanged as a library (`command` / `packages` stay). `internal/apply` computes the delta onto `HunkResult` and omits it on prose. `internal/authoring` vocabulary is unchanged; `statsCmd` rendering changes. No architecture doc exists; no new bounded context.

C4: same write container. The check subprocess is the trust boundary ADR-003 already named.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` default check | run `check.Load`+`Run` when a command exists, the plan has a non-prose written path, and `--no-check` is off | `cmd/mrw/main.go` `writeCmd` | CLI; contract §89 |
| `mrw write --no-check` | opt out | T1 | CLI; contract §89 |
| prose skip | markdown-only apply does not spawn the check; records `applied` | T1 | CLI; contract §89 |
| `HunkResult.Balance` | omitempty string; omitted on prose; hunk stays `ok` | `internal/apply/apply.go` | `report`; JSON; MCP receipt; contract §90 |
| `mrw stats` | always five names; landed / failed_check line | `cmd/mrw/main.go` `statsCmd` | CLI; `--json`; contract §91 |
| contract §89–§91 | next free after §88 | `scripts/contract.sh` | `adr-verify`, CI |

Exit codes unchanged: 0 / 1 / 2 / 3 mean what ADR-003 already says. `--no-check` with a failing tree is 0 on purpose.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| default check / `--no-check` (T1) | T1 | T3, T4 | Yes — CLI default |
| `HunkResult.Balance` (T2) | T2 | T4 | No — additive |
| stats five names + landed line (T3) | T3 | T4 | No — rendering |

T2 does not depend on T1. T3 is useful before T1 (zeros print) and fills after T1. Execute T1 first: it is the arm that would have caught the three Zeus `.rs` breakages in the turn; markdown plans do not spawn the check.

## Implementation

See `docs/adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree/tasks/README.md`.

## Consequences

- **Positive:** a write that touches code in a harnessed (or inferred-Go) checkout cannot apply a plan that breaks its own check without exit 3, in the same turn. A markdown-only plan does not pay the project suite. A brace-net mismatch on a non-prose replace is visible on the receipt even when the check is skipped; a balanced insert in the wrong place is not. Zeus-shaped tallies cannot hide `failed_check`.
- **Negative:** a non-Go code write whose `packages()` cannot map pays the whole-project command (Zeus `.rs`: full workspace clippy plus 3096 tests; 58–107s for the nextest half alone, measured 2026-09-13). Data files are not in the prose list: a Zeus `docs/lineage.jsonl` append cannot break a Rust build and still pays that suite — `.yml` and `.json` the same. `.toml` must keep triggering (`Cargo.toml`, `rust-toolchain.toml`). The list is not changed; a reader who meets this will think the gate is broken. `--no-check` is a new flag callers must learn. Naive balance will still false-positive on braces in strings in code. F/N on the landed line mixes unverified `applied` writes with verified ones.
- **Neutral:** MCP applied plans still do not increment `failed_check`. Shared() unchanged. `internal/check.command` / `packages` unchanged.

## Out of Scope

- MCP `mrw_write` running a check (permanent: fact: ADR-044 stays two tools; citation: file `docs/adr/ADR-044-mcp-cargo-stays-two-tools.md:7`)
- Extending the neighbour license to single-line addresses (permanent: boundary: not a fix for the Zeus cases; open question remains on docs/adr/BACKLOG.md)
- Indent-reparent / YAML with no delimiter token (permanent: boundary: Decision 2 has nothing to count; deferred: docs/adr/BACKLOG.md)
- A parser or refuse-on-delta (permanent: boundary: ADR-048)
- Default echo (permanent: fact: ADR-052 opted in; citation: file `docs/adr/ADR-052-echo-pad-and-neighbour-license.md:51`)
- Revert on failed check (permanent: fact: ADR-003; citation: file `docs/adr/ADR-003-a-checks-verdict-comes-from-the-process-never-its-output.md:102`)
- A sixth authoring Outcome (permanent: boundary: five names are the five exits)
- Raising 4096 / rewriting Shared() (permanent: boundary: teach on `write --help`)
- Reopening ADR-019 B/C (permanent: fact: pick A stands; citation: file `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md:106`)
- A harness `covers` glob or a non-Go `packages()` (permanent: boundary: the closed prose list is the cover heuristic; growing either is a new record)
- Growing the prose-extension list (permanent: boundary: Decision names the five; execute does not add)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Default check on a large unscoped suite blows the round-trip saving | High on non-Go code writes | High | Prose skip; `--no-check`. Do not claim `{packages}` scoping outside Go. |
| Desktop write exits 2 because no harness | High if always-demand | High | Default runs only when `Load` has a command |
| Balance false-positives on prose train callers to ignore the line | High if unrestricted | High | Omit `Balance` on the closed prose list; still Med on braces-in-strings in code |
| `failed_check` still empty for MCP-only checkouts | Med | Med | Named in Consequences; ADR-044 |
| Landed F/N is read as "of those we checked" | Med | Med | Decision names that N includes `check_not_run` and unchecked `applied` |
| Existing scripts that parse stats miss a new line | Low | Low | Five names stay; derived line is extra |

## Rollback

Revert the branch. `--check` becomes opt-in again. Drop `--no-check`, `Balance`, and the stats derived line. Vocabulary on disk does not migrate.

## Follow-ups

- Execute on Accept. Do not start T1 while Proposed.
- Neighbour license on single-line addresses stays BACKLOG, unarmed.
