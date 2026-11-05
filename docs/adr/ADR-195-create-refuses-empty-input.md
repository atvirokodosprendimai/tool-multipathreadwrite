# ADR-195: `mrw write --create` refuses empty standard input, as a plan refuses an empty create

**Status:** Accepted
**Accepted:** 2026-11-04 by Zy — answered *"acept if this is beneficial"* to the two drafts, then *"go"*: it closes a silent 0-byte create (issue #606) at the cost of one exit code, with no `--empty` flag.
**Date:** 2026-11-04
**Amended:** 2026-11-04, before acceptance, from an adr-challenger pass and a Codex xhigh challenge (each finding checked against the code): a fifth member of the class (an empty SEARCH/REPLACE create), the refusal comes before the exists check, the suggested plan quotes the path, one mutant replaced by one that can be killed, and the fence asserts the new words in both documents.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-027, ADR-139, ADR-142, ADR-009, docs/adr/BACKLOG.md
**Supersedes:** ADR-139, in one clause of Decision 3: "empty input makes an empty file"
**Governs:** `internal/ingest/create.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `README.md`, `AGENTS.md`
**Enforced-by:** `internal/ingest/create195_test.go::TestCreateRefusesEmptyInputAndNamesTheEmptyFilePlan`
**Invalidates:** ADR-139 — Decision 3's clause "empty input makes an empty file", and the empty case of its Enforced-by test `TestCompileCreateMakesACreatePlanOfStdin`, which moves to a refusal in T1
**Served-path change:** `mrw write --create PATH` whose standard input holds no bytes exits 2, creates nothing, and names the plan that makes an empty file (`@@ "PATH" 0 create body=0`, the path quoted as `CompileCreate` quotes it); the refusal comes before the apply, so an empty `--create` of a PATH that exists, or one inside `.git`, is this exit 2 rather than the apply's exit 1; content of one byte or more, a lone newline included, is created as now.

## Context

ADR-027 decided that an empty file is created on purpose or not at all: a plan's `create` with no body is refused unless it says `body=0`, because a lost body reads exactly like one never written. ADR-139 then added `mrw write --create PATH`, which makes a file from standard input "as the `create` plan of those lines would", and its Decision 3 said "empty input makes an empty file". That clause is the one place where an empty create is not asked for. `adr-context internal/ingest/create.go` names both records as governing.

Issue #606, observed 2026-11-03 on v1.83.0-dev+94d125a8 (macOS) and again on v1.84.0 (2026-11-04):

```sh
: | mrw write --create empty.txt                # exit 0, "created empty.txt 0L -> 0L", a 0-byte file
printf '\n' | mrw write --create nl.txt         # exit 0, a 1-line file (1 byte)
printf '@@ b.txt 0 create\n' | mrw write -      # refused: "create with an empty body: say body=0 ..."
```

How it was noticed: in benchmark run 9 a Python producer piped into `mrw write --create PATH` crashed before it printed. The write landed a 0-byte file with a clean receipt, and the crash showed only in the producer's stderr. That is the failure the design exists to prevent (AGENTS.md): a write that went wrong and looks fine.

**What "empty" is.** `CompileCreate` gets the raw bytes, after `CreateContent` (ADR-142) has dropped a PowerShell pipe's final CRLF. `CreateContent` never turns non-empty input into zero bytes: `TrimPipedCRLF` drops a CRLF only after text holding an LF. So zero bytes means nothing at all arrived. A lone newline is one empty line, which a producer printed on purpose.

**Audit of the class**: *every input that makes a file from content mrw did not see typed in the plan header.* Enumerated by reading the lines `mrw read --grep 'create' --exclude '*_test.go' internal/ingest internal/plan` serves (39 lines in 5 files on 2026-11-04) beside the plan grammar (`AGENTS.md` §2): five members.
- A native `create` with no body: refused unless `body=0` (ADR-027). Unchanged.
- `body=@path` naming an empty file: "empty file ≡ `body=0`" (AGENTS.md §2). The caller named a file in the root that it can inspect, and the header says where the body comes from. Left out here (Out of Scope).
- An apply_patch `*** Add File:` with no `+` lines: an empty create. The document carries the header and the end marker its producer wrote, so a lost body is not silent there in the way a lost pipe is. Left out (Out of Scope).
- A `--format=search_replace` block with an empty SEARCH and an empty REPLACE, on a path that does not exist: `compileSearchHunk` emits a create with no body (`internal/ingest/searchreplace.go:100-109`), and the shared emitter writes `body=0`; measured 2026-11-04 on v1.84.0, exit 0 and a 0-byte file. Like apply_patch, the document carries the producer's markers around the empty side. Left out (Out of Scope).
- `--create PATH` with empty standard input: decided here.

**A precedent for the refusal.** `: | mrw write -`, an empty plan on standard input, is already refused: `plan is empty: no @@ headers found`, exit 2 (measured 2026-11-04 on v1.84.0). `--create` with nothing on standard input is the one write that takes no bytes for a file.

## Existing Primitives Audit

- `ingest.CompileCreate` (`internal/ingest/create.go:106`) already refuses content a plan cannot carry (`lines.Unsplittable`, a bare CR, a line over 16 MiB) with an error that `cmd/mrw/main.go` turns into a content refusal: exit 2, not counted by ADR-009's parse tally (`cmd/mrw/main.go:2185`). The empty refusal is one more of these. No new flag, exit path or receipt field.
- ADR-027's refusal text (`internal/plan/plan.go:936`, `refusal.CreateEmptyBody`) is the precedent for the wording: it names `body=0`.

## Decision

1. **`CompileCreate` refuses content of zero bytes**, before anything is parsed or applied: `--create PATH: standard input was empty, which reads exactly like a producer that failed before it printed; to make an empty file, send the plan @@ "PATH" 0 create body=0 to mrw write -`, with PATH quoted and escaped exactly as `CompileCreate` writes a header (`create.go:131-134`), so a path with a space or a quote is a plan that parses. `mrw write` exits 2 and creates nothing. The check runs in `CompileCreate`, before the apply, so it comes before the apply's own refusals: an empty `--create` of a PATH that exists is exit 2, not 1. Like ADR-139's other content refusals it is not counted by the parse tally. `--dry-run` refuses the same way.
2. **Content of one byte or more is unchanged**: a lone newline makes a 1-line file, and every ADR-139 and ADR-142 rule holds.
3. **No flag asks for an empty file through `--create`.** The plan `@@ PATH 0 create body=0` already says it in one line, and ADR-027 made it the one way to ask.
4. **Nothing else changes**: native plans, `body=0`, `body=@path`, apply_patch, `mrw_write`.

**What would make this decision wrong:** callers who rely on `: | mrw write --create PATH` for an empty file in a script. No measurement of that exists: ADR-009's tally does not record a create's size, and no transcript search was made. The cost to such a caller is one refused call whose message names the replacement. This changes the exit code of that one input from 0 to 2, so the release notes say so.

## Alternatives Considered

- **`--empty`, an opt-in beside `--create`.** It would mirror `body=0` as a flag. Rejected: it is a second spelling of what the plan already says, one more flag in the contract, and `--empty` beside non-empty input would need a refusal of its own. Zy may prefer it; it can be added later without changing this refusal.
- **Refuse only when standard input is a pipe.** Rejected: a redirect from a file a producer failed to write (`< out.txt`) loses its body the same way.
- **Keep it, and print a warning.** Rejected: a warning beside exit 0 is the shape ADR-003 and ADR-072 refuse; the receipt would still say the create succeeded.

## Component / Boundary Impact

`internal/ingest`, and `internal/guide` for the `--create` sentence of `mrw instructions`. `cmd/mrw` is unchanged: it already reports `CompileCreate`'s error as a content refusal. `apply`, `plan`, `seen`, `read`, `check`, `state` and `writer` stay byte-identical.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write --create PATH` with empty standard input | exit 0 → exit 2, nothing created, the message names `body=0` | T1 | CLI callers |
| `scripts/contract.sh` | §336 | T1 | CI Linux |
| `README.md`, `AGENTS.md`, `mrw instructions` | the `--create` sentences say empty input is refused and name the `body=0` plan | T1 | every agent |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** a producer that dies before printing no longer leaves a 0-byte file under a clean receipt.
- **Negative:** one input changes exit code (0 → 2). A caller that meant an empty file spends one call.
- **Neutral:** a producer that dies after printing part of its output still lands what it printed; mrw cannot see the producer's exit status (Out of Scope).

## Out of Scope

- `body=@path` naming an empty file, the same question for a body file (deferred: docs/adr/BACKLOG.md)
- An apply_patch Add File with no lines (permanent: boundary: the document carries its producer's header and end marker, so the body's absence is written in it, not lost in a pipe)
- A SEARCH/REPLACE block with both sides empty on a missing path (permanent: boundary: the block's three markers are written around the empty side, so its absence is written in the document, as for apply_patch)
- A producer that fails after printing part of its content (permanent: boundary: mrw reads its standard input, not the exit status of the process writing it; `set -o pipefail` in the caller's shell is what sees that)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A script relied on empty stdin making an empty file | Low | Low | the refusal names the one-line plan; the release notes name the exit-code change |
| A pipe's trimming turns real content into zero bytes | Low | Med | `TrimPipedCRLF` needs an LF before the CRLF it drops; T1's test holds a lone CRLF and a lone LF to "created" |

## Rollback

Revert T1: empty standard input makes an empty file again.

## Follow-ups

- [ ] `body=@path` naming an empty file (BACKLOG.md, From ADR-195)
