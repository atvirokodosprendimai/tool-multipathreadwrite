# ADR-040: The help a PATH caller trusts names how to quote a header option

**Status:** Accepted
**Accepted:** 2026-09-12 by M — *"good, accepted all"*
**Date:** 2026-09-12
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-037 (the binary teaches the format), ADR-035 (multi-line replace needs `anchor=`), ADR-015 (a refusal names the fix), ADR-001 (plan grammar), ADR-012 (MCP handshake), ADR-019 (Naming pick A — launch `--root` only), ADR-027 (`create` at 0), ADR-038 (torn `Load`), ADR-039 (host-cut under ceiling), docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `internal/guide/guide.go`, `internal/plan/plan.go`
**Enforced-by:** `cmd/mrw/writehelp_test.go::TestWriteHelpNamesHowToQuoteAHeaderOption`
**Invalidates:** none — checked. `write --help` names `anchor=` and omits quoting; that is a gap, not a false promise. ADR-035 still requires `anchor=` on a multi-line replace and is not this number. ADR-019 pick A is not reopened.
**Served-path change:** `mrw write --help` and `guide.CLI()` name how to quote a header option (double, single, or unquoted `anchor=` until the next `key=`), that `body=` is a line count (Python `str` is characters), that `lines=` is a range guard, and that global `-C` / `--root` name the checkout; unquoted and single-quoted `anchor=` parse; `mrw version` prints `versionString()` (keep `-v` / `--version`).
**Notes:** M, 2026-09-12: *"good, accepted all"* after the inventory of arming quotes for every 040 fork and every backlog-named leftover. That quote Accepts this Decision and every fork below. *"I NAME THEM ALL"* still only ranked the inventory.

## Context

**The class this record governs.** Every CLI document a PATH caller uses to learn how to write a
plan-header option. Enumerated 2026-09-12 with

```
git ls-files cmd/mrw/main.go internal/guide/guide.go internal/plan/plan.go
```

Three files. This record is authoritative over all three: `writeCmd` `Description` is `write --help`;
`guide.CLI()` is stdout of `mrw instructions`; `splitHeader` is the lexer a quoted or unquoted
`anchor=` actually meets. Members left out, and why:

- `AGENTS.md`, `README.md`, `.claude/skills/mrw/SKILL.md` — already show `anchor="func Apply"`. They
  are mirrors for callers who have this checkout or the palace. The field report is that a PATH
  caller trusted `--help` / `-v` over those mirrors.
- `internal/mcp/instructions.go` — `guide.Shared()` stays the five ADR-037 sentences unless Accept
  puts quoting there. Every MCP session pays `maxInstructionsChars` (4096). Quoting is a plan-text
  trap both surfaces share; teaching it on the handshake is a bound risk, not a free copy.

**Why this is a record.** On 2026-09-12 a consumer session in another project wrote a local
`# Hard: mrw CLI` because PATH `mrw write --help` did not carry the traps. Inbox
`wing_tool-multipathreadwrite` / `WHAT BREAKS WHEN AN AGENT WRITES AN MRW PLAN ANCHOR WITH SPACES?`
(filed 2026-09-12 from memory-runtime). PATH binary `mrw -v` → `dev (0b75313)` (mtime 8 Sep).
Re-measured on this tree at v1.12.0 (`origin/main` `4b70dd0`, 2026-09-12) **before
execute**. The table is the field-report hole. Decisions 2–5 are the post-Accept state:

| Claim | Still true here |
|---|---|
| Unquoted `anchor=func openTestStore` splits | `splitHeader` honours `"` only (`internal/plan/plan.go:456`). The next token is `option "openTestStore" is not key=value` (`internal/plan/plan.go:341`). Exit 2. Not a successful wrong write. |
| Single quotes do not parse | `anchor='func openTestStore'` → `option "openTestStore'" is not key=value`. |
| `write --help` names `anchor=` and not quoting | `cmd/mrw/main.go:717-731`. |
| `mrw instructions` exists and also omits quoting | here since v1.10.0; `guide.CLI()` (`internal/guide/guide.go:20-29`) names the pipe, exit 3, MSYS, glob. Not quoting. The field PATH binary predates the subcommand (`unknown command "instructions"`). This record must not pretend an old binary grows one. |
| No `mrw version` subcommand | `rootCommand().Commands` (`cmd/mrw/main.go:167`) has eight entries. `-v` / `--version` only. The field rule used `mrw -v`. |
| Python `str` body character-splits | caller, not us. `body=` is already a line count. |

M said *"build a spec for this"* and named the smallest close as teach quoting on `write --help`,
and/or accept unquoted `anchor=` until end of options. They asked for a record, not execute.

**What is already this repo's contract and is not 040.** Named so a later turn cannot treat them
as noise or silently fold them into this Decision:

| Already shipped | Where |
|---|---|
| Wrap-tail / read on past the addressed range | `AGENTS.md:262` |
| `raw=true` (escape for a body line that looks like a header) | ADR-015; `internal/plan/plan.go` |
| `create` at 0 (`@@ new.txt 0 create body=0`) | ADR-027; `AGENTS.md:185` |
| Insert vs neighbour-replace (`insert-after` / `insert-before` are not `replace`) | `AGENTS.md:182` |
| ADR-035 `anchor=` on a multi-line replace | `AGENTS.md:197` |
| Original-file addresses | ADR-001; `AGENTS.md:195` |
| Never `write \| head` (exit code through a pipe) | `AGENTS.md:314` |
| `body=` is a line count | Decision 1 teaches it; ADR-027 uses `body=0` |

The spec is the **gap the local rule had to carry**, not a re-opening of that table.

## Existing Primitives Audit

- **`writeCmd` `Description` (`cmd/mrw/main.go:715`):** the document a PATH caller actually reads.
  **Extended** with the quoting / `body=` / `lines=` sentences. Not replaced.
- **`guide.CLI()` (ADR-037):** Shared plus CLI-only traps. **Extended** with the same sentences, so
  a v1.10.0+ binary-only installer who runs `mrw instructions` is not taught two stories. `Shared()`
  is **not** extended (4096 / ADR-037).
- **`splitHeader` (`internal/plan/plan.go:398`):** already keeps double-quoted fields together, and
  already says so in its comment. **Reshaped** for unquoted consume-to-next-key and single quotes.
- **`option %q is not key=value` (`internal/plan/plan.go:341`):** the refusal the field report hit.
  **Named** so a leftover after a finished quoted `anchor=` points at double quotes.
- **ADR-015:** a refusal names the fix. **Same class**, different mistake. This record does not
  rewrite ADR-015.
- **`-v` / `--version` / `versionString()`:** already work. A `version` subcommand would be a new
  `Commands` row, not a new stamper.
- **A syntax-aware engine, installing mrw, forcing MCP-off, upgrading other machines' PATH:**
  audited and **NOT taken.**

## Decision

**The PATH binary's `write --help` is the document a consumer trusts over the central skill.**
That document currently names `anchor=` and does not mention quoting. This record closes that gap.
It does not re-decide ADR-035, wrap-tail, `body=` as line count, the pipe trap, `raw=true`, or
original-file addresses.

**1. Teach (the floor).** `write --help` names these, so a caller who never opens AGENTS.md
can still author a legal header:

- a value with spaces can be double-quoted (`anchor="func openTestStore"`), single-quoted,
  or — for `anchor=` only — left unquoted until the next `key=`
- `body=` is a line count, not a character count (Python `str` splits characters)
- `lines=` is a guard on the addressed range and is not `body=`
- the checkout is named by global `-C` / `--root` before the subcommand; after `read`, `-C`
  is context lines

The same sentences go in `guide.CLI()`. They do **not** go in `guide.Shared()` (4096 / ADR-037).
A PATH binary that predates `instructions` is not this record's to upgrade.

When a header token is not `key=value` and the previous token started with `anchor=`, the refusal
names double quotes rather than only `option %q is not key=value`.

**2. Parse — accepted.** `joinUnquotedAnchorOptions` treats an unquoted `anchor=` as
consuming until the next `key=` token, so `anchor=func openTestStore` parses. Other keys
(`sha=`, `lines=`, `body=`, `raw=`) do not gain the rule. A quoted `anchor=` is complete
even when the value has no space (`anchor="class=" leftover`); a leftover after it is
still usage and names double quotes.

**3. `mrw version` — accepted.** A `version` subcommand prints `versionString()`. `-v` /
`--version` stay. Extra arguments are usage (exit 2). A PATH binary that predates the
subcommand still answers `-v`.

**4. Single quotes — accepted.** `splitHeader` accepts `anchor='func openTestStore'`. A
single-quoted **path** stays literal (`'docs/adr/x.md'` is that name). Help teaches
double, single, and unquoted-until-next-key.

**5. `-C` vs `--root` — accepted: teach both global flags. Do not reopen 019 B/C.**
Help names global `-C DIR` and `--root DIR` before the subcommand, and says that after
`read`, `-C` is context lines. Pick A stands. B/C and `roots/list` stay closed.

*"good, accepted all"* arms every fork above. `"I NAME THEM ALL"` only ranked the inventory.

**Go/no-go, checked during execution. If any fails, the task is withdrawn rather than shipped:**

- **`maxInstructionsChars` is still 4096.** Shared() is not the place the four sentences land
  unless Accept quoted that and the bound still holds.
- **ADR-035 is unmodified.** A multi-line replace still needs `anchor=`. Teach and parse change
  how a value is spelled, not whether it is required.
- **A PATH binary that predates `instructions` is not claimed to have grown it.**
- **Teach-only does not change apply, read, the ledger, or the MCP tool set.** A parse Accept
  may change `splitHeader` only.

## Alternatives Considered

- **Leave teaching in AGENTS.md and the central skill.** Today's state, and the one the field
  report measured as insufficient: the consumer trusted PATH `--help` / `-v` over the skill.
  Rejected as the standing defect.
- **Point `write --help` at `mrw instructions`.** Free on this tree; the field PATH binary does
  not have the command. A reference to a subcommand the reader cannot run is the defect ADR-012
  rejected.
- **Teach-only, no refusal-message change.** Smaller. Rejected as the floor's other half: the
  caller who already wrote the bad header never re-reads `--help`; they read the refusal.
- **Parse unquoted `anchor=` without teaching.** Rejected as a silent pick of fork 2. The floor
  still teaches; *"good, accepted all"* then armed the parse.
- **Accept single quotes as well as double, silently.** Rejected as a silent pick of fork 4.
  The floor taught that they did not parse; *"good, accepted all"* then armed the parse.
- **Add `mrw version` in the same breath as teach.** Rejected as a silent pick of fork 3. The
  field rule already has `-v`; *"good, accepted all"* then armed the subcommand.
- **Put the four sentences in `guide.Shared()`.** Rejected: every MCP session pays 4096, and
  ADR-037 forbade raising it. Stays out (ADR-045).
- **Upgrade other machines' PATH / install mrw as a side quest.** Rejected: consumer rule, not a
  contract. Version skew (PATH binary vs skill) is named in Out of Scope, not absorbed here.
- **Teach `-C .` as the checkout pointer, or reopen 019 B/C.** Rejected as a silent pick of
  fork 5 / a reopening of an Accepted naming. Pick A stands. The help-wording conflict is
  named, not resolved.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `cmd/mrw` `writeCmd` | Still the CLI write surface | Yes — `Description` is `write --help` |
| `internal/guide` | Still the pamphlet | Yes — `CLI()` gains the four sentences |
| `internal/plan` `splitHeader` / `parseHeader` | Still the header lexer | Teach-only: refusal text. Parse fork: token rule. Nothing else. |
| `internal/apply`, `internal/read`, `internal/seen` | Unchanged | Untouched |

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| CLI `write --help` | names quoting, `body=` vs `lines=`, global `-C` / `--root` | `cmd/mrw` | any caller with the binary |
| CLI `mrw instructions` | `guide.CLI()` carries the same sentences | `internal/guide` | any caller whose binary has the command |
| plan header | unquoted and single-quoted `anchor=` parse; leftover after a quoted `anchor=` names double quotes | `internal/plan` | any author of a header |
| contract.sh | §79 help, §80 parse, §81 `version` | `scripts/contract.sh` | CI, `adr-verify` |
| CLI `mrw version` | prints `versionString()`; `-v` / `--version` stay | `cmd/mrw` | any caller; `#73` gate requires AGENTS.md to name it |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `write --help` quoting sentences (T1) | T1 | none | No — additive |
| `guide.CLI()` quoting sentences (T1) | T1 | none | No — additive |
| unquoted `anchor=` consume-to-next-key (T2) | T2 | none | No — today's string is exit 2, not a successful write |
| `mrw version` subcommand (T3) | T3 | none | No — additive; old binaries stay without it |
| single-quoted `anchor=` parses (T4) | T4 | none | No — today's string is exit 2, not a successful write |

## Implementation

See `docs/adr/ADR-040-the-help-a-path-caller-trusts-names-how-to-quote-a-header-option/tasks/README.md`.

## Consequences

- **Positive:** a PATH caller who reads `write --help` can quote `anchor=` without a local hard
  rule. The refusal they hit names the same fix. `instructions` on a current binary agrees.
- **Negative:** a fourth copy of some sentences (help / CLI() / AGENTS.md / skill). Drift is
  asserted by §79 and the CLI() test, not eliminated. Old PATH binaries stay silent until
  reinstalled.
- **Neutral:** *"good, accepted all"* armed every fork. `"I NAME THEM ALL"` only ranked the
  inventory. Engine leftovers live as ADR-041–050.

## Out of Scope

Named 2026-09-12 because M said *"I NAME THEM ALL"*. Disposition on every row. The table in
`docs/adr/BACKLOG.md` is the same list. This Decision absorbs none of C–E.

**A — already shipped, not this Decision**

- Wrap-tail / read on past the addressed range (permanent: fact: already this repo's contract; citation: file `AGENTS.md:262`)
- `raw=true` (permanent: fact: already the header-in-body escape; citation: file `docs/adr/ADR-015-a-refusal-names-the-fix-for-the-two-mistakes-the-syntax-invites.md:67`)
- `create` at 0 (`body=0`) (permanent: fact: ADR-027 shipped the deliberate empty file; citation: file `docs/adr/ADR-027-an-empty-file-is-created-on-purpose-or-not-at-all.md:12`)
- Insert vs neighbour-replace (permanent: fact: `insert-after` / `insert-before` are ops, not a `replace` of the neighbour; citation: file `AGENTS.md:182`)
- ADR-035 `anchor=` on a multi-line replace (permanent: fact: still required; this record does not reopen it; citation: file `AGENTS.md:197`)
- Original-file addresses (permanent: fact: ADR-001; citation: file `AGENTS.md:195`)
- Never `write | head` (permanent: fact: a pipe swallows the write's exit; citation: file `AGENTS.md:314`)
- `body=` is a line count (permanent: fact: ADR-027's `body=0` and Decision 1 teach the same number; citation: file `docs/adr/ADR-027-an-empty-file-is-created-on-purpose-or-not-at-all.md:45`)

**B — named forks, now accepted by *"good, accepted all"***.

- Teach-only (Decision 1) — the floor; still the help text (permanent: boundary: already the Decision; not Out of Scope work)
- Parse unquoted `anchor=` until the next `key=` (permanent: boundary: Decision 2; T2)
- `mrw version` subcommand (permanent: boundary: Decision 3; T3)
- Single quotes parse (permanent: boundary: Decision 4; T4)
- `-C` vs `--root` (permanent: boundary: Decision 5; help names both global flags; 019 pick A stands; do not reopen 019 B/C or `roots/list`)
- Extending consume-to-next-key to `sha=`, `lines=`, `body=`, `raw=` (permanent: boundary: the parse fork is `anchor=` only)

**C — dangling product this repo owns. Named, not implemented.**

- PATH binary vs skill **version skew** (deferred: docs/adr/ADR-041-path-binary-and-skill-version-skew-is-named-not-healed.md)
- Pretending a pre-v1.10.0 binary grew `mrw instructions` (permanent: fact: `instructions` exists here since v1.10.0; the field PATH binary at `0b75313` predates it; citation: file `docs/adr/ADR-037-the-binary-teaches-the-format-it-demands.md:94`)
- `mrw check` silent in-root fallback (deferred: docs/adr/ADR-042-check-in-root-fallback-stays-silent-until-its-own-record-ships.md)
- Torn `Load` / atomic save (deferred: docs/adr/ADR-043-torn-load-is-measured-before-anyone-locks-it.md)
- MCP cargo: `check` / `iter` / `seen` / `stats` (deferred: docs/adr/ADR-044-mcp-cargo-stays-two-tools.md)
- Generate AGENTS.md from `guide.Shared()` (deferred: docs/adr/ADR-045-agents-md-is-not-generated-from-shared.md)
- Host-cut under the advertised ceiling (deferred: docs/adr/ADR-046-host-cut-under-the-ceiling-is-measured-not-assumed.md)
- Python `str` body character-split (permanent: boundary: the caller builds the plan; Decision 1 teaches `body=` is a line count in this help; ADR-047 records the leftover)
- Putting the four sentences in `guide.Shared()` / the MCP handshake (deferred: docs/adr/ADR-045-agents-md-is-not-generated-from-shared.md)
- Raising `maxInstructionsChars` / 4096 (permanent: boundary: ADR-037's go/no-go; quoting stays out of Shared() unless Accept quotes that and the bound still holds)

**D — engine dreams. Parked, not 040.**

- Syntax awareness / making mrw parse or refuse target-language structure (deferred: docs/adr/ADR-048-mrw-models-no-target-syntax.md)
- Streaming or memory-bounded apply (deferred: docs/adr/ADR-049-streaming-apply-waits-for-a-size-that-hurts.md)
- Windows `%LOCALAPPDATA%` state path (deferred: docs/adr/ADR-050-windows-state-stays-xdg-until-windows-is-exercised.md)

**E — must not become mrw work.**

- Playtrix T4 / that paste (external: wing_playtrix: wing_playtrix/inbox)
- Other wings' inboxes, including quality-harness (external: wing_quality-harness: wing_quality-harness/inbox)
- Reopening ADR-019 pick B/C or `roots/list` (permanent: fact: Accepted Naming pick A is launch --root only; citation: file `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md:106`)
- A Canvas file (external: outside-repo: not in this checkout)
- Installing mrw as a side quest (permanent: boundary: the consumer rule; if mrw is missing, say so)
- A `keep/` gitignore convention (external: consumer-checkout: gitignore hygiene is not this binary)
- "Use CLI not MCP" as *this* binary's contract (permanent: boundary: consumer harness rule; ADR-010 already chose two MCP tools)

**F — housekeeping (BACKLOG text, not this Decision).**

- Stale BACKLOG receipts for ADR-029 alias ledger and `shellArgs` quoting (permanent: fact: both shipped 2026-09-09; the open reading was the unstruck receipt, closed 2026-09-12; citation: file `docs/adr/BACKLOG.md:312`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Help text lands and `CLI()` does not, so `instructions` and `--help` disagree | Med | Med | T1 fence requires both; deleting either sentence fails |
| Shared() is edited "to keep them in sync" and overflows 4096 | Med | High | Go/no-go; Stop Condition if the bound is the proposed fix |
| Parse Accept silently applies to every key | Low | High | Decision 2 names `anchor=` only; T2 fixture is that key |
| A skill teaches `mrw version` after T3 and old PATH binaries still fail | Med | Low | Decision 3 states it; T3 docs keep `-v` |
| Clearer refusal matches any trailing token, not an `anchor=` split | Med | Med | T1 fixture is `anchor=func openTestStore`; a `replace openTestStore` usage error must stay the generic message |
| `"Accept"` without the fork list is read as teach-only after this quote | Low | Med | Notes name *"good, accepted all"* as every fork |
| `"I NAME THEM ALL"` is read as Accept or as arming cargo / syntax | Med | High | Notes: that quote only ranked the inventory; 041–050 are record-only |

## Rollback

Revert the commit. `write --help` and `CLI()` lose the four sentences. The refusal returns to
`option %q is not key=value`. Additive; no state, no plan-format change unless the parse fork
had shipped — that fork's rollback is "unquoted spaced `anchor=` is again exit 2", which is
today's behaviour. A caller who scripted `mrw version` (only if that fork shipped) gets
"unknown command" at exit 2.

## Follow-ups

- [ ] After teach has shipped, decide whether AGENTS.md's plan example should keep being the
      only place a checkout-holding caller sees quoting, or whether it should point at
      `mrw write --help`.
- [ ] The inventory M ranked on 2026-09-12 lives in `docs/adr/BACKLOG.md`. Do not treat an
      unnamed chat item as work. The next quote that arms work is listed beside each row there.
