# ADR-212: A moved symbolic link's receipt gives no content

**Status:** Accepted
**Accepted by:** Zy, 2026-11-20: *"accept all"*, to the question "write the lines_after record?", with the recommendation that a record state the meaning rather than restore a line count. Accepted before its challenge: the Codex xhigh challenge runs after acceptance, and History records what it changed.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-005, ADR-057, ADR-111, ADR-114, issue #712, #725
**Supersedes:** in part: ADR-111 Decision 1's "keeps … its meaning across releases" for `files[].sha_after` and `files[].lines_after` of one record only, the destination of a moved symbolic link, from the release that carried #725 (`b1188333`); ADR-057 Decision 7's "Record the dest after rename as a whole-file write", for a moved symbolic link (ADR-057 already carries a notice of #725).
**Governs:** `internal/apply/pathop.go`, `internal/writer/writer.go`, `cmd/mrw/main.go`, `internal/mcp/schema.go`, `internal/mcp/era_test.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `README.md`
**Enforced-by:** `internal/writer/link712_test.go::TestAMovedSymlinkCarriesNoContentAndLicensesNothing`
**Served-path change:** none to behaviour: this record states what #725 already ships. The words change: `mrw_write`'s output schema describes `files.sha_after` and `files.lines_after` for a moved symbolic link (T1).

## Context

**What #725 changed, without a record.** Moving a relative symbolic link to another directory can make it read another file. Before #725 the destination's record gave the digest and line count of the file the link read before the move, and the writer licensed that. #725 (`b1188333`, issue #712) made the rename step stop reading the target: when the renamed entry is a link, the destination's record is set to `LinesTo` 0, `SHAAfter` "" and `NotProduced` (`internal/apply/pathop.go:535-545`). What a caller sees:

| Surface | For a moved symbolic link's destination |
|---------|------------------------------------------|
| `--json` and `mrw_write` receipt | `sha_after` absent (it is `omitempty`, `internal/apply/apply.go`), `lines_after` 0, `written` true |
| human receipt | `sha -` (`shaOrNone`, `cmd/mrw/main.go:3498-3503`) |
| ledger | no record for it; one an earlier read left at that path, under any spelling of the entry, is dropped (`internal/writer/writer.go:114-121`), so a write to it needs a read first |
| dry run | no destination record at all: a dry run lists the source with `renamed_to` (`apply.go:948-967`) |

Contract §364 and `TestAMovedSymlinkCarriesNoContentAndLicensesNothing` pin it. ADR-057 carries a notice of the change; the README's rename paragraph says it (`README.md:315`).

**What the words still say.** `mrw_write`'s output schema describes the two keys as "The sha256 the file WOULD have after this plan" and "How many lines the file would hold after this plan" (`internal/mcp/schema.go:274`, `:276`). For a moved link the first is absent and the second is 0 while the link may read a file with lines. A caller reading the schema would take 0 for an empty file. `docs/receipts.txt` lists keys only and has no place for a meaning (ADR-111's convention), so it does not change.

**Why not restore the count.** Restoring `lines_after` means reading what the link leads to at commit, which is what #712 stopped: the plan's later path operations may not have settled, and the link may lead outside the root or into mrw's state (the comment at `pathop.go:535-541`, from the Codex review of #725).

**Audit of the class**: *the receipt records whose content mrw does not give.* An unlinked file (`sha_after` absent, `lines_after` 0, `removed` true, ADR-057); a moved symbolic link (this record); a dry run's rename destination (no record). Only the second lacked a stated meaning.

## Existing Primitives Audit

- `FileResult.NotProduced` already marks the case inside mrw; it is not a receipt field (`apply.go`).
- `sha_after`'s `omitempty` already gives "absent" a meaning for a removed file.
- `TestALegacyResultIsUnchangedByTheModernPath` (`internal/mcp/era_test.go`) holds the legacy transcript, whose `tools/list` carries these descriptions, and its comment lists every regeneration and why.

## Decision

1. **For the destination of a moved symbolic link, the receipt gives no content.** `sha_after` is absent, `lines_after` is 0 and says nothing about the lines the link leads to, `written` is true, the human receipt prints `sha -`, and the destination licenses nothing. This is the keys' meaning for that one case, and mrw does not read the target to fill them.
2. **ADR-111 is amended for that case only.** ADR-111 makes a change of a key's meaning wait for a major version. #725 shipped one in a minor release, for one record whose old value was false (the digest of a file the link no longer reads). Zy accepted stating it rather than reverting or restoring the count. Every other record keeps the keys' meaning; no key is added, renamed or removed.
3. **The words.** `mrw_write`'s output schema says it (T1): `files.sha_after` gains "Absent for a removed file and for a moved symbolic link, whose target mrw does not read."; `files.lines_after` gains "0 for a removed file and for a moved symbolic link, whose target mrw does not read: there it says nothing about the lines the link leads to." The legacy golden is regenerated for those two strings and nothing else.
4. **What would make this wrong.** A caller that needs the count. It reads the destination after the write, which is what the ledger requires before an edit anyway.

## Alternatives Considered

- **Restore `lines_after` by reading the target.** Rejected: it brings back the read #712 removed (Context).
- **Revert #725 to its old receipt.** Rejected: the old `sha_after` named a file the link no longer reads, and the writer licensed it.
- **Wait for a major version, as ADR-111 says.** Rejected: the behaviour has shipped; waiting would leave the schema saying something false until then.
- **A new key, e.g. `content_unknown`.** Rejected: `sha_after`'s absence already says it, and a new key is a surface to keep for one case.

## Component / Boundary Impact

`internal/mcp` (`schema.go`'s two descriptions; the legacy golden and `era_test.go`'s list of regenerations). No engine package changes; `internal/apply` and `internal/writer` are as #725 left them.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_write` output schema: `files.sha_after`, `files.lines_after` | one sentence each for a removed file and a moved symbolic link | T1 | MCP hosts and callers |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerated for those two strings | T1 | `TestALegacyResultIsUnchangedByTheModernPath` |
| contract §364 | unchanged: it already drives the behaviour through the built binary | — | the contract gate |

No receipt key is added or renamed (ADR-111); the meaning of two keys is stated for one record.

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** the schema says what the receipt does; a caller does not take 0 for an empty file.
- **Negative:** ADR-111's promise has one stated exception, and a caller that counted on `lines_after` for a moved link must read the destination.
- **Neutral:** behaviour is unchanged since #725.

## Out of Scope

- `docs/receipts.txt` (permanent: boundary: it lists keys and their receipts, never their meanings; ADR-111's convention)
- The CLI's `--json` key documentation (permanent: fact: the CLI has no per-key description; its receipt keys are listed in docs/receipts.txt only; citation: file `docs/receipts.txt:1`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Editing `era_test.go` moves another record's test lock | Med | Low | T1 runs `adr-lint` on every record and relocks what it names |

## Rollback

Revert T1; the behaviour is #725's.

## History

- 2026-11-20: drafted after Zy's acceptance; the Codex xhigh challenge follows.

## Follow-ups

- [ ] None beyond T1.
