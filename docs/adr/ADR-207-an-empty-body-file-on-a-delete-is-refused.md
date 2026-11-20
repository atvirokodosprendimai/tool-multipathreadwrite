# ADR-207: An empty `body=@` file on a delete is refused

**Status:** Accepted
**Accepted by:** Zy, 2026-11-20: *"accepted the rest"*, to the question of accepting the open records ADR-207 to ADR-211 (#723, #720) as they stand.
**Drafted for:** Zy, 2026-11-20, from an independent critique of the same day (issue #700). Zy's instruction on the critique's findings, as relayed to the drafter by the session that commissioned these records, was *"address all, no choosing"*, so this record carries one decision. Challenged before acceptance (Codex at xhigh, 2026-11-20, on `2ded135b`: AMEND); every amendment is applied below or answered in History.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-006, ADR-008, ADR-009, ADR-027, ADR-057, ADR-060, ADR-108, ADR-111, ADR-132, ADR-176, ADR-195
**Supersedes:** in part: ADR-060 Decision 4's "An empty file is `body=0`", for a `delete` only. For `create` it stands; `replace` and the inserts keep refusing an empty body as they do today (at apply, exit 1).
**Governs:** `internal/plan/plan.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/plan/emptybody207_test.go::TestAnEmptyBodyFileOnADeleteIsRefused`
**Served-path change:** `@@ PATH ADDR delete body=@FILE` whose FILE holds zero bytes is refused before anything is written, exit 2, naming the plan line and FILE; until now it deleted the range with no expectation checked and exited 0. On `mrw write --json` the refusal is the existing top-level `error`; on `mrw_write` an `isError` answer. Every other `body=@` behaves as before.

## Context

**What happens today.** A `delete` may carry a body: the lines the caller expects to remove, checked against the file before anything is written (ADR-008). The check runs only when the body has lines (`internal/apply/apply.go:1971`). `body=@FILE` loads a body from a root-relative file after parsing; `LoadBodyFiles` sets the body to nil when the file holds zero bytes (`internal/plan/plan.go:1066-1069`). So the guard is skipped. Reproduced on v1.93.0 (`18ffa49`; its `internal/plan` and `internal/apply` equal the reviewed commit `2ded135b`), 2026-11-20, on a three-line file `a b c` read whole:

| Plan | Result |
|------|--------|
| `@@ f.txt 2 delete body=@wrong.txt` (`zzz`) | exit 1, `expected removal differs at line 2` |
| `@@ f.txt 2 delete body=@expected.txt` (zero bytes) | exit 0, line 2 removed, `ok … from "b" to "b"` |
| `@@ f.txt 2 delete body=0` | exit 0, line 2 removed |
| `@@ f.txt 2 delete body=@nope.txt` (no such file) | exit 2, `line 1: body=@nope.txt: stat …: no such file or directory` |

A caller who names a file means an expectation. The cause of an empty file is not known to mrw, and how often a caller writes one on purpose is not measured; but a guard that turns itself off when its input is empty fails exactly when a producer failed. ADR-195 refused empty `--create` input for that reason.

**An empty `body=@` on the other ops** (same probe):

| Op | Empty `body=@` | Literal `body=0` |
|----|----------------|------------------|
| `create` | an empty file, exit 0 (`body=@` sets the declared count, `plan.go:537`, so `CountedBody` holds) | the same |
| `replace` | refused at apply, exit 1, `replace with an empty body would delete 1` (`apply.go:1614`) | refused at parse, exit 2 (`plan.go:1007`) |
| `insert-after` | refused at apply, exit 1, `with an empty body would change nothing` (`apply.go:1606`) | refused at parse, exit 2 (`plan.go:993`) |
| `delete` | no expectation, exit 0 | no expectation, exit 0 |

So "an empty file is `body=0`" (`AGENTS.md:427-428`, `cmd/mrw/main.go:1758-1759`) is exact for `create` and `delete` only; on `replace` and the inserts both forms refuse, at different stages and exit codes, which the sentence does not say.

**Where the body is loaded.** `LoadBodyFiles` has three callers: `cmd/mrw/main.go:2238` (a failure is `refuse`, exit 2, counted `refused_parse`), `internal/mcp/tools.go:957` (an `isError` answer, counted `refused_parse`) and `internal/curve/score.go:102` (outcome `RefusedParse`). A refusal there reaches all three with no other change, and a piped plan becomes eligible to be kept for editing (ADR-176, within its size and state-write limits, `cmd/mrw/refused176.go`).

**Audit of the class, and its limits**: *the ways a delete gets an expectation.* An inline body under the header, counted (`body=N`) or not; `body=@FILE`; and none (no body, or `body=0`). `from=` is refused on a delete (`plan.go:556`). This record closes one way an intended expectation becomes none: an empty `body=@` file. It does not close the other: an *uncounted inline* body cut off before its first line arrives as a valid bodyless delete, which mrw cannot tell from one written without a body (a counted body that is cut off is already refused by its count, ADR-027). `unlink` is not in the class: it removes a path, takes no line address and has its own no-body meaning (ADR-057; `internal/apply/pathop.go:125`).

## Existing Primitives Audit

- `LoadBodyFiles` already refuses a body file that is missing, rooted, not regular or over the size limit, each naming the plan line and the file; the empty case joins that list.
- `Hunk.Op` is set at parse, so `LoadBodyFiles` knows the op without new state.
- `lines.Split` gives an empty text no lines and a single `\r\n` one empty line (`internal/lines/lines.go:21-30`), so "zero bytes" and "one blank line" stay distinct: the second is an expectation of one empty line.

## Decision

1. **On a `delete`, a `body=@FILE` that holds zero bytes is refused** in `LoadBodyFiles`, before validation: `line N: body=@FILE is empty (0 bytes), and a delete's body=@ is the lines it expects to remove, so nothing was deleted: give the file the lines you expect, or drop body=@ to delete without an expectation`. CLI exit 2, nothing written, counted `refused_parse`; `mrw write --json` carries it in the top-level `error`; `mrw_write` answers `isError` with the same sentence. A file holding one blank line is not empty and loads as today.
2. **`body=0` written on a delete keeps its meaning**: no expectation. So does a delete with no body.
3. **No other op changes.** An empty `body=@` on `create` is still an empty file; whether that should be refused stays where ADR-195 deferred it (`docs/adr/BACKLOG.md`, "`body=@path` naming an empty file"). On `replace`, `insert-after` and `insert-before` it is still refused by apply's empty-body checks, exit 1.
4. **The two sentences change in the same PR.** `AGENTS.md:427-428` and `mrw write --help` (`cmd/mrw/main.go:1758-1759`) say, in these words: `an empty file is body=0 for a create, is refused on a delete, and is an empty body, refused, on a replace or an insert`.

## Alternatives Considered

- **Treat an empty `body=@` on a delete as an expectation of zero lines**, so the length check fails it at apply (exit 1). Rejected: it reads as a mismatch with the file when the problem is the producer, and its refusal would quote a line back.
- **Refuse an empty `body=@` on every op.** Rejected: an empty create from a file is legitimate today and ADR-195 left it open; replace and insert already refuse.
- **Refuse an uncounted inline delete with no body.** Rejected: it is indistinguishable from the ordinary bodyless delete every caller writes.
- **Leave it and document it.** Rejected: the guard is off exactly when its input failed.

## Component / Boundary Impact

Engine package owned here: `internal/plan` (`LoadBodyFiles`). `cmd/mrw/main.go` changes only its help text. `internal/apply`, `internal/read`, `internal/seen`, `internal/check` and `internal/state` do not change.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` / `mrw_write`: a delete with a zero-byte `body=@` | refused, exit 2 / `isError`, naming the plan line and the file | T1 | every caller |
| `mrw stats` | the refusal counts as `refused_parse`, as every `LoadBodyFiles` refusal does | T1 | its readers |
| `AGENTS.md`, `mrw write --help` | the "empty file is `body=0`" sentence (Decision 4) | T1 | every caller |
| `scripts/contract.sh` §358 | the refusal beside a valid sibling hunk, with nothing written; a wrong non-empty `body=@` exits 1; a matching one lands; `body=0` and a bodyless delete still land; empty `body=@` on both inserts still exits 1 | T1 | the contract gate |

No receipt key is added or renamed (ADR-111).

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** a delete's expectation cannot vanish because the file holding it came out empty.
- **Negative:** a caller that used an empty `body=@` to mean "no expectation" is refused and must drop `body=@` or write `body=0`. No run-time cost: the check is on bytes already read.
- **Neutral:** the exit-code difference between an empty `body=@` and `body=0` on a replace or an insert stays, now stated.

## Out of Scope

- Moving the empty-body refusal of `replace` and the inserts from apply (exit 1) to load (exit 2) (permanent: boundary: both refuse with nothing written; an exit-code change needs its own record, and no caller has reported the difference)
- An uncounted inline delete body lost in transit (permanent: boundary: it parses as a bodyless delete, which mrw cannot tell from one written without a body; a counted body is already refused when short, ADR-027)
- An empty create from a `body=@` file (deferred: docs/adr/BACKLOG.md, "`body=@path` naming an empty file (deferred from ADR-195)")
- An empty `from=` span (permanent: fact: `from=` is refused on a delete, so a delete has no copied span to come out empty; citation: file `internal/plan/plan.go:556`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A script writes an empty expected file on purpose | Low (unmeasured) | Low | the refusal names both ways out; `body=0` and no body still delete |

## Rollback

Revert T1; no state is involved.

## History

- 2026-11-20, Codex at xhigh on `2ded135b`, AMEND, three points, all applied: the audit names the uncounted inline delete and `unlink`, and the empty create stays under ADR-195's deferral; ADR-060 Decision 4 is superseded for deletes only; §358 and the Go test pin the write boundary (a valid sibling, a wrong body, a matching body, zero bytes against one CRLF blank line, the CLI JSON and MCP shapes).

## Follow-ups

- [ ] None beyond T1.
