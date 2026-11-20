# ADR-212: A moved symbolic link's receipt gives no content

**Status:** Accepted
**Accepted by:** Zy, 2026-11-20: *"accept all"*, to the question "write the lines_after record?", with the recommendation that a record state the meaning rather than restore a line count. Accepted before its challenge: the Codex xhigh challenge runs after acceptance, and History records what it changed.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-005, ADR-057, ADR-111, ADR-114, issue #712, #725
**Supersedes:** in part: ADR-111 Decision 1's "keeps … its meaning across releases" for `files[].sha_after` and `files[].lines_after` of one record only, the destination of a moved symbolic link, from the first release that contains #725 (`b1188333`). No tag contains it yet (`git tag --contains b1188333` is empty on 2026-11-20; the last release, `v1.93.0`, predates it), so the exception applies to that next release, which names it. Also ADR-057 Decision 7's "Record the dest after rename as a whole-file write", for a moved symbolic link (ADR-057 already carries a notice of #725).
**Governs:** `internal/apply/pathop.go`, `internal/writer/writer.go`, `cmd/mrw/main.go`, `internal/mcp/schema.go`, `internal/mcp/era_test.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `docs/receipts.txt`, `scripts/contract.sh`, `README.md`
**Enforced-by:** `internal/writer/link712_test.go::TestAMovedSymlinkCarriesNoContentAndLicensesNothing`
**Served-path change:** none to behaviour: this record states what #725 already does on `main`, which the next release will carry. The words change: `mrw_write`'s output schema describes `files.sha_after` and `files.lines_after` for a moved symbolic link, and the rule a consumer reads them by on both transports (Decision 3, T1); ADR-111 and `docs/receipts.txt`'s header name the exception.

## Context

**What #725 changed, without a record.** Moving a relative symbolic link to another directory can make it read another file. Before #725 the rename step filled the destination's row from the source snapshot it already held — the digest and line count of what the link read before the move — and the writer licensed that. #725 (`b1188333`, issue #712) stopped reporting those values: when the renamed entry is a link, the destination's record is set to `LinesTo` 0, `SHAAfter` "" and `NotProduced` (`internal/apply/pathop.go:535-545`), and nothing reads the target to fill them. What a caller sees:

| Surface | For a moved symbolic link's destination |
|---------|------------------------------------------|
| `--json` and `mrw_write` receipt (the same `FileResult`, `internal/apply/apply.go:220`) | `sha_after` absent (it is `omitempty`), `lines_after` 0, `written` true |
| human receipt | `sha -` (`shaOrNone`, `cmd/mrw/main.go:3498-3503`) |
| reads of the target | the receipt builder does not read the target for those fields; validation still reads through the source link (the load at `internal/apply/apply.go:743`), and the drift scan of a write with a check hashes what the link reads at its new place when that is inside the root (`internal/writer/drift.go:20-35`) |
| ledger | the move grants no licence: the writer records nothing for the destination and drops what an earlier read left at that entry under any spelling (`internal/writer/writer.go:114-121`). An observation of the link's target under the target's own path is not dropped, and apply's lookup by identity can still find it for a plan that names the link (`apply.go:1485` and the alias lookup after it), so an edit through the link needs a read only when no current observation of the target exists |
| dry run | no destination record at all: a dry run lists the source with `renamed_to` (`apply.go:948-967`) |

Contract §364 and `TestAMovedSymlinkCarriesNoContentAndLicensesNothing` pin it. ADR-057 carries a notice of the change; the README's rename paragraph says it (`README.md:315`).

**What the words still say.** `mrw_write`'s output schema describes the two keys as "The sha256 the file WOULD have after this plan" and "How many lines the file would hold after this plan" (`internal/mcp/schema.go:274`, `:276`). For a moved link the first is absent and the second is 0 while the link may read a file with lines. A caller reading the schema would take 0 for an empty file. `docs/receipts.txt` lists keys, not meanings (ADR-111's convention); its key list stays as it is, and its header comment, which states the contract, gains the exception.

**Why not restore the count.** Restoring `lines_after` means reading what the link leads to at commit, which is what #712 stopped: the plan's later path operations may not have settled, and the link may lead outside the root or into mrw's state (the comment at `pathop.go:535-541`, from the Codex review of #725).

**Audit of the class**: *the receipt records whose content mrw does not give.* An unlinked file (`sha_after` absent, `lines_after` 0, `removed` true, ADR-057); a moved symbolic link (this record); a dry run's rename destination (no record). Only the second lacked a stated meaning.

## Existing Primitives Audit

- `FileResult.NotProduced` already marks the case inside mrw; it is not a receipt field (`apply.go`).
- `sha_after`'s `omitempty` already gives "absent" a meaning for a removed file.
- `TestALegacyResultIsUnchangedByTheModernPath` (`internal/mcp/era_test.go`) holds the legacy transcript, whose `tools/list` carries these descriptions, and its comment lists every regeneration and why.

## Decision

1. **For the destination of a moved symbolic link, the receipt gives no content.** `sha_after` is absent, `lines_after` is 0 and says nothing about the lines the link leads to, `written` is true, and the human receipt prints `sha -`. The receipt builder does not read the target to fill them (Context gives what else reads it). The move licenses nothing of its own and drops stale records at that entry; an existing current observation of the target under its own path stands.
2. **ADR-111 is amended for that case only, from the first release that contains #725.** ADR-111 makes a change of a key's meaning wait for a major version. #725 changes one, for one record whose old value was false (the digest of a file the link no longer reads), and no release carries it yet; the next one, a minor release, will, and its release note names the exception. Zy accepted stating it rather than reverting or restoring the count. Every other record keeps the keys' meaning; no key is added, renamed or removed. ADR-111 and the header of `docs/receipts.txt` point at this record.
3. **The consumer's rule, on both transports.** On a row with `written` true and no `removed`, `lines_after` is a measured count only when `sha_after` is present; without `sha_after` the row's content is not given. The words are about how the receipt is built, not about what mrw reads: validation reads through the source link, and a check's drift scan can hash the target (Context). `mrw_write`'s output schema says it (T1): `files.sha_after` gains "Absent for a removed file and for a moved symbolic link: the receipt is built without reading what the link leads to."; `files.lines_after` gains "0 for a removed file and for a moved symbolic link, whose receipt is built without reading what the link leads to: on a written row it is a count only when sha_after is present." The legacy golden is regenerated for those two strings and nothing else. The CLI's `--json` has no per-key text; `docs/receipts.txt`'s header says the rule and the release it starts in.
4. **What would make this wrong.** A caller that needs the count. It reads the destination after the write. An edit through the link needs a read unless a current observation of the same file already licenses it, under the target's own path or under any other name that reaches it (another link, another spelling): apply finds that observation by identity when the plan names the link (`internal/apply/apply.go:1500`, `sameFileEntry`).

## Alternatives Considered

- **Restore `lines_after` by reading the target.** Rejected: it brings back the read #712 removed (Context).
- **Revert #725 to its old receipt.** Rejected: the old `sha_after` named a file the link no longer reads, and the writer licensed it.
- **Wait for a major version, as ADR-111 says.** Rejected: the behaviour is on main since #725 and will be in the next release (none contains it yet; v1.93.0 still reports the old values); reverting it to wait would bring back the false digest #712 removed, and keeping it without the words would leave the schema saying something false.
- **A new key, e.g. `content_unknown`.** Rejected: `sha_after`'s absence already says it, and a new key is a surface to keep for one case.

## Component / Boundary Impact

`internal/mcp` (`schema.go`'s two descriptions; the legacy golden and `era_test.go`'s list of regenerations). No engine package changes; `internal/apply` and `internal/writer` are as #725 left them.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_write` output schema: `files.sha_after`, `files.lines_after` | one sentence each, Decision 3 | T1 | MCP hosts and callers |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerated for those two strings | T1 | `TestALegacyResultIsUnchangedByTheModernPath` |
| `docs/receipts.txt` header, ADR-111 | a comment naming the exception and the rule; a notice | this record | readers of the receipt contract |
| contract §364 | extended in T1: the applied receipt's `lines_after` 0, `written` true and no `sha_after` in CLI `--json`, the human `sha -`, and a dry run's one source row with `renamed_to` and `written` false and no destination row; the MCP shape in T1's Go test | T1 | the contract gate |

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

- `docs/receipts.txt`'s key list (permanent: boundary: it lists keys and their receipts, never their meanings; ADR-111's convention. Its header comment is changed by this record: Wiring)
- The CLI's `--json` key documentation (permanent: fact: the CLI has no per-key description; its receipt keys are listed in docs/receipts.txt only; citation: file `docs/receipts.txt:1`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Editing `era_test.go` moves another record's test lock | Med | Low | T1 runs `adr-lint` on every record and relocks what it names |

## Rollback

Revert T1; the behaviour is #725's.

## History

- 2026-11-20: drafted after Zy's acceptance; the Codex xhigh challenge follows.
- 2026-11-20, Codex xhigh challenge after acceptance, at `a549b331`: AMEND, four points, each checked against the code and applied. (1) No tag contains `b1188333` (`git tag --contains` empty; `v1.93.0` is the last release), so the exception applies from the next release; the consumer's rule is stated for both transports, and ADR-111 and `docs/receipts.txt`'s header point here. (2) The claims are narrowed: validation reads through the source link (`apply.go:743`) and the drift scan hashes an in-root target (`drift.go:20-35`); an observation of the target under its own path still licenses an edit through the link (`apply.go:1485`). (3) The applied and dry-run wire shapes are pinned in T1 and §364. (4) T1's fence unsets `MRW_UPDATE_LEGACY_GOLDEN`, checks that exactly the two descriptions changed, protects `internal/writer`, and its sentence mutants fail by the test's named message.
- 2026-11-20, Codex review of #758 (record and T1): five points, each checked and fixed. (1) The schema sentences, Decision 3 and `docs/receipts.txt`'s header now speak of how the receipt is built, not of what mrw reads. (2) Context says #725 stopped reporting values the rename took from its source snapshot, and Decision 4 says when a read is needed (`apply.go:1500`). (3) The header names the release the exception starts in, and the Out of Scope entry covers the key list only. (4) The MCP test requires `written` present and false on the dry run, fails on a second row for one path, and type-checks before it asserts. (5) §364 requires an int `lines_after`, counts rows before indexing, sets `MRW_RECEIPT=full` and matches the human line exactly.

## Follow-ups

- [ ] None beyond T1.
