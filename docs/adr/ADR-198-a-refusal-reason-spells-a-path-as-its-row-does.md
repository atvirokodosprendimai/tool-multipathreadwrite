# ADR-198: A refusal reason spells a plan's path as its row does

**Status:** Accepted
**Accepted:** 2026-11-15 by Zy — asked *"Do you accept ADR-198 as written?"* on the draft rewritten after the two challenges, answered *"accept"*.
**Date:** 2026-11-15
**Amended:** 2026-11-15, before acceptance, from an adr-challenger pass and a Codex xhigh challenge (each finding checked against the code): the class audit gained fourteen members, the commonest refusal among them; the first draft's mechanism (keep each reason's format and arguments, format again in `Slashed`) is replaced by spelling the path where the reason is built, which reaches the reasons that are already a string or an error when a hunk fails; the `attrib` and `chmod` advice is kept in the platform's spelling; the "as typed" sites are told from the cleaned ones; contract §338 is stated as the guard it is.
**Amended:** 2026-11-15, after acceptance, from the go-reviewer pass on #643 and four Windows 11 sessions that probed a binary built from d6526b90 (decisions unchanged; claims corrected): `from=P: ERR` for a source `rooted` refuses is kept, because the error begins with the path as the platform spells it; the coverage claim names the sites with no test case; `attrib` was measured to accept `/`, so keeping its command's spelling is caution; `mrw skill --dir`'s own refusal and the raw error line are not converted and are deferred. From the Codex review of #643: a rooted path that reaches a converted site is returned as typed (`foldClashes` runs before an absolute destination is refused), and the link-to-nothing fixture makes the link's own parent, so it no longer skips on Windows for a missing directory.
**Amended:** 2026-11-15, after the merge: `mrw skill --dir`'s own refusal moves from deferred to a boundary. Zy, asked whether to leave that path as typed instead of converting it, answered *"agree"*.
**Amended:** 2026-11-15, accepted by Zy — asked *"Do you accept ADR-199, and the ADR-198 amendment (Decisions 6 and 7, including the ADR-132 sentence it replaces and the identity error as a boundary)?"*, answered *"accep"*: Decisions 6 and 7 and task T2, for the two deferrals of this record and one more member a Windows probe of `main` found (a refused rename destination, `d\nul\x leads to "nul"`). Drafted after an adr-challenger pass and a Codex xhigh challenge of a first version, which keyed Decision 7 on a result whose paths were already converted.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-091, ADR-132, ADR-197, ADR-167, ADR-111, ADR-069, issue #642, docs/adr/BACKLOG.md
**Supersedes:** ADR-132, in one clause of its Out of Scope: a plan's own path that mrw's words quote inside a reason is no longer left in the platform's spelling. The clause stands for text mrw did not format. With T2: ADR-132, the last sentence of its Decision 4, "Under `--json`, `error` is unchanged": the path a stopped commit's error begins with is spelled as the receipt spells it, which completes ADR-132 Decision 3 ("`write --json`, its refusals").
**Invalidates:** ADR-076 — one expectation of its test `TestARenameToADirectorySpellingIsRefused` (`internal/apply/paths076_test.go`): the name the refusal suggests is `d/a.txt` on every platform, where the test expected the platform's spelling. Zy, 2026-11-15, asked whether to update it: *"yes"*. ADR-076's rule is unchanged.
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/apply/fromcopy.go`, `internal/apply/replace.go`, `internal/apply/wirepath.go`, `internal/mcp/tools.go`, `cmd/mrw/main.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/reason198_test.go::TestAReasonSpellsAPlansPathAsItsRowDoes`
**Served-path change:** on Windows, a refused hunk's reason spells with `/` every root-relative path that mrw's own words put in it: the hunk's file, a rename's destination and a `from=` source, wherever in the reason they stand, and the `ALREADY WRITTEN` list. Before, only a path the reason began with and the one inside `` `mrw read X` `` advice were converted, so one line could read `FAIL sub/f.txt … no line of sub\f.txt holds it`. Kept as they are: a path quoted as the caller typed it, a path the system printed, an absolute path, and the path in `attrib -r` advice. On every other platform nothing changes. The exit code and every receipt key are unchanged.

## Context

Issue #642, measured 2026-11-15 on Windows 11 with `mrw` built from 09762a79, in Git Bash 5.2.37, pwsh 7.6.9 and Windows PowerShell 5.1 (12 writes, each exit 1, nothing written). With `sub\f.txt` read and a plan whose anchor is in no line:

```
FAIL sub/f.txt 2-3 replace (plan line 1): anchor "nowhere" not in line 2: "b"; no line of sub\f.txt holds it
```

Spelling the plan path `sub\f.txt` gives the same line.

**Why.** The engine keeps the platform's spelling in its path fields, because the ledger and the drift baseline key on them (`internal/writer/writer.go:106`, `drift.go:30`). A receipt is built from a copy that `apply.Slashed` converts (`internal/apply/wirepath.go:17`), for the CLI (`cmd/mrw/main.go:3590`) and for `mrw_write` (`internal/mcp/tools.go:1257`). `Slashed` converts the path fields, and in a reason two positions through `SlashIn` (`wirepath.go:57`): the hunk's path at the start of the text, and the path inside `` `mrw read P` `` advice, the backtick right after the path.

ADR-132 left every other position out as a permanent boundary: three Codex rounds on #347 each found a way a rule over the finished text half-rewrote a path the system had printed. That holds for a rule over finished text. It does not hold where mrw builds the reason: there mrw knows which argument is a plan's path. Nothing keys on a reason's text, and the engine already writes `/` into two reasons itself (`servedAs`, `apply.go:2776`; the `.git` refusal, `tree.go:61`).

**Audit of the class**: *every reason in which mrw's own words put a root-relative path anywhere but the start of the text, or at the start where `SlashIn` does not reach it.* Read on 09762a79 from `grep -nE '\b(fail|failK|refuseFile|abortStage|refuseStage)\(' internal/apply/*.go` without the test files, from every function those sites take a finished string from, and from the lines that set a reason directly (`apply.go:591, 604, 659, 743, 770, 974, 984, 1012`).

Converted here, the path being a cleaned root-relative one:
- The hunk's file: `SPAN of P has not been read` (`apply.go:1429`, the ordinary unread refusal); `Read P with mrw_read first` (`1394` after `forceRemedy`, `1235`, on `mrw_write`); `unlink/rename cannot mix with other hunks on P` (`1327`); `create: P already exists` (`1387`, `1579`); `sha=… given but P does not exist` (`1474`, `pathop.go:136`); the pattern refusals (`1627`, `1640`, `1652`, `1657`, `1690`, `1692`); `no line of P holds it` (`2686`, ADR-197); `P: its identity could not be read: P cannot be opened` (`replace.go:58`, the second `P`); `rename source P is not spelled that way` (`pathop.go:219`).
- A second file of the plan: `P names the same file as Q` (`apply.go:659`); `P may name the same file as Q` (`2899`, `foldClashes`, where a rename's leading name is its destination and so is not the hunk's path); `rename dest D …` (`pathop.go:170, 174, 178, 183, 189, 212, 216, 222, 226`, with `nests with D2` and `respells the directory DIR`), and the `such as D/NAME` example of `165`; the staging and commit refusals `rename dest D is under …`, `rename dest D: …` and `rename dest D appeared before commit` (`apply.go:1096, 1101`, `pathop.go:361`), which reach the reason through an `error`.
- A `from=` source, cleaned: `from=P: WHY` (`fromcopy.go:96`), the `loadSource` reasons that `mrw` words itself (`176-201`), and `sourceGate`'s `` Run `mrw read P:A-B` first `` and `lines SPAN of P have not been shown` (`224`, `234`).
- `ALREADY WRITTEN: P, Q` (`apply.go:1220`): mrw joins `FileResult.Path` values.

Kept:
- A path quoted as the caller typed it, to show the typing: the trailing-separator refusals (`apply.go:580`, `fromcopy.go:81`, and the first path of `pathop.go:165`), an absolute rename destination (`pathop.go:159`), and `from=X` where `X` is the header's own text (`fromcopy.go:72, 75`).
- `P is absolute … mrw looked for FULL` (`apply.go:1599`, `pathop.go:145`): `P` leads and `SlashIn` converts it today; `FULL` is absolute.
- The command in `clear the mark first: attrib -r P` or `chmod u+w P` (`apply.go:2995-2999`): a command for the platform's own shell. Measured 2026-11-15 on Windows 11 26200: `attrib -r d/ro.txt` works from cmd and from pwsh, so the backslash is kept as the spelling that shell's own tools print, not because `/` fails.
- `from=P: ERR` when `rooted.Resolve` refuses the source (`fromcopy.go:169`): the error begins with `P` as the platform spells it (`internal/rooted/rooted.go:185, 239`), and converting the first alone would put two spellings on one line. T1 kept it; Decision 6 converts both.
- An `error`'s own text, an absolute path, a link target; and a rooted path that reaches a converted site, which `asPlan` returns as it is (two rename destinations `\missing\Q.txt` and `\missing\q.txt` reach `foldClashes` before the absolute-path refusal).
- `; you were served A, B` (`servedAs`): already `/`.
- Reasons made in `internal/writer/flow.go:225, 229`: they begin with the path, which `SlashIn` converts.
- `mrw cannot read the file identity of P` (`internal/rooted/hardlink.go:38`): made in `rooted`, which the read side shares (Out of Scope, a boundary since the amendment that added T2).

Only the ADR-197 ending was measured on Windows. The rest is read from the code.

## Existing Primitives Audit

- `apply.Slash(p, sep)` (`wirepath.go:46`) is the conversion; `receiptSep` (`cmd/mrw/main.go:3587`) is the precedent for a separator a test can set.
- `Slashed` and `SlashIn` stay as they are, for the path fields and for reasons made outside `internal/apply`.

## Decision

1. **A reason spells a plan's path as its row does.** Where mrw's words put a cleaned root-relative path into a reason (the sites the audit lists as converted), it is spelled with `/`, at any position.
2. **The path is spelled where the reason is built.** Each such site formats its path through one function of `internal/apply`, which converts from a package separator that is the platform's and that a test can set. Nothing is done to a finished reason beyond what `SlashIn` does today, so no text mrw did not format is touched.
3. **The path fields are unchanged.** `HunkResult.Path`, `FileResult.Path` and every other path field `Apply` returns keep the platform's spelling, as do the ledger and the drift baseline. A reason and the error that carries it are built from the same text, so the exit-2 line still finds one in the other (`cmd/mrw/main.go:3611`).
4. **Kept in the platform's or the caller's spelling**: the sites the audit lists as kept.
5. **Nothing changes where the separator is `/`.** A file whose name holds a backslash is named as it is.
6. **A refusal `rooted` words about a path that is not the hunk's own begins with that path as the row spells one** (T2). Six of `rooted`'s refusals begin with the exact path `Resolve` was given, followed by a space or a colon: a name Windows does not keep (`P: Windows does not keep …`), a device name (`P leads to "nul" …`), an escape (`P resolves to ABS, which is outside the root`), mrw's state (`P is inside …`), a hard link to it and a name that cannot be compared (`internal/rooted/rooted.go:185, 239`, `hardlink.go:23, 30`). For the hunk's own path the receipt converts that start (`SlashIn`); a rename destination and a `from=` source are other paths. The four sites that put such an error into a reason (`pathop.go:195`, `apply.go:743, 770`, `fromcopy.go:169`) convert the start, by one rule: the text begins with the exact path they resolved, followed by a space or a colon. A refusal that begins otherwise is left whole: the root's own errors, a link that cannot be followed (it begins with an absolute path), `.git` (already `/`) and the identity error. Measured on Windows 11 at 0d49d6e8: `FAIL d/g.txt 0 rename (plan line 1): d\nul\x leads to "nul" …`. Nothing in `internal/rooted` changes.
7. **The error of a stopped write begins with its file as the row spells it, on the surfaces that carry a receipt** (T2). `Apply` returns `P: CAUSE (…)`, `P appeared before commit …` or `P changed after mrw read it …` (`apply.go:1021, 1166, 1172, 1177`, `pathop.go:439`) with `P` as the platform spells it. The plain exit-2 line converts that start (ADR-132 Decision 4). A CLI `--json` refusal's `error` and the stderr line beside it, and `mrw_write`'s `error` field and `error:` line, print it raw. Each converts the start by Decision 6's rule, against the paths of the result as `Apply` returned it, before the receipt's own conversion (`cmd/mrw/main.go:2305`, `internal/mcp/tools.go:1257`) replaces them. The error `Apply` returns is unchanged, and an error that begins with no path of the result (a directory named in a plan, `apply.go:697`, begins with one that has no hunk; a lock, a ledger failure) is printed as it is.

**What would make this decision wrong:** a caller on Windows that takes a path out of a reason's prose and hands it to a tool that refuses `/`. None is known: the one command mrw's own words print, `attrib -r`, was measured to accept both. The receipt's path fields have been `/` on `mrw_write` since ADR-091 and on the CLI since ADR-132.

## Alternatives Considered

- **A rule over the finished reason.** Rejected by ADR-132, with its evidence.
- **Keep each reason's format and arguments and format again in `Slashed`** (the first draft). Rejected after the challenges: sixteen of the audit's sites hand a finished string or an `error` to the place a hunk fails, so the marks would have to travel through `loadSource`, `sourceGate`, `foldClashes`, `identityRefusal` and three `error` values; and `Slashed` clones the hunks shallowly, so the kept arguments would be shared with the engine's result.
- **Leave it, and document it.** Rejected: ADR-132 Decision 3 says the receipt spells every root-relative path with `/`, and one line showing two spellings of one file reads as two files.

## Component / Boundary Impact

`internal/apply` (engine; `apply.go`, `pathop.go`, `fromcopy.go`, `replace.go`, `wirepath.go`, owned here and by issue #642, go-reviewer pass per `lifecycle.md` §6). `read`, `plan`, `seen`, `check`, `state`, `writer` and `rooted` stay byte-identical. T1 left `cmd/mrw` and `internal/mcp` byte-identical; T2 edits `cmd/mrw/main.go` and `internal/mcp/tools.go` for Decision 7. No receipt key is added (ADR-111): the words are in `hunks[].reason` and `error`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a refused hunk's `reason`, CLI and `mrw_write`, on Windows | a plan's path inside it is spelled with `/` | T1 | every caller on Windows |
| `scripts/contract.sh` | §338: on Linux a name holding a backslash is not rewritten in a reason; T2 adds a section for a refused rename destination | T1, T2 | CI Linux |
| a refused rename destination or `from=` source that `rooted` words, on Windows | its leading path is spelled with `/` | T2 | every caller on Windows |
| a stopped write's `error`: CLI `--json` and its stderr line, `mrw_write`'s field and `error:` line, on Windows | the leading path is spelled with `/` | T2 | callers that parse them |
| `README.md`, `AGENTS.md`, `docs/receipts.txt` | none: no sentence in them states how a reason spells a path (`grep -n -i "separator\|ADR-091" README.md AGENTS.md docs/receipts.txt`, 2026-11-15, no hit on the subject) | — | — |

## Inter-task Contracts

None — T2 consumes T1's `asPlan` and `reasonSep`.

## Implementation

`tasks/README.md`: T1, then T2. One pull request each.

## Consequences

- **Positive:** one spelling of a file on one line, on every surface that prints a reason.
- **Negative:** a reason `Apply` returns spells a path with `/` beside path fields that keep the platform's; a new refusal site that formats a bare path prints the platform's spelling, as today. The error `Apply` returns for a commit that stopped begins with the file as the platform spells it and now holds `/` paths after it; `mrw_write`'s `error:` line and a CLI `--json` refusal print it raw (the plain exit-2 line converts its start). T2 converts that start (Decision 7).
- **Neutral:** what cannot be shown here. `scripts/contract.sh` runs on Linux only, where the separator is `/`: §338 guards Decision 5 and cannot show the conversion. The conversion's proof is the Go test with `\` set as the separator, which runs on every platform, the same test in the Windows CI job, and a probe of the built binary on Windows before the merge.

## Out of Scope

- A path quoted as the caller typed it (permanent: boundary: the reason quotes the spelling it refuses or echoes the header, and converting it would misquote the caller; ADR-069)
- The path in `attrib -r` and `chmod u+w` advice (permanent: boundary: it is a command for the platform's own shell, printed as that shell's tools spell a path)
- A path the system printed inside a reason (permanent: boundary: ADR-132's reasoning stands for text mrw did not format)
- `mrw cannot read the file identity of P`, made in `internal/rooted` (permanent: boundary: the path stands inside the sentence, which Decision 6's start-of-text rule does not reach, and the same words are a read's refusal, whose spelling ADR-132 left to the platform)
- `mrw skill --dir`'s own refusal, `DIR\mrw\SKILL.md was not written by …`, built in `cmd/mrw` (permanent: boundary: the path is relative to the working directory and begins with the directory as the caller typed it, not a plan's path under a root; `dirReason` beside it, `cmd/mrw/skill168.go:276-283`, records three rewrites of that directory that got it wrong)
- A path inside the cause of a stopped write's error, after its start (permanent: boundary: the cause is the system's or already spelled by Decision 1)
- The landing error `mrw skill --dir` prints (`cmd/mrw/skill168.go:207, 222`) and the benchmark's score output (`internal/curve/score.go:133`) (permanent: boundary: neither is a write receipt; the skill's paths are relative to the directory the caller typed, as its own refusal is)
- The CLI `read` side's refusals (permanent: boundary: the read side is its own surface, as ADR-132 left it)
- Plan-parse refusals, which quote the header as written (permanent: boundary: no path has been resolved when a header is refused)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Something searches a reason for the platform's spelling of a path | Low | Med | read 2026-11-15: `exitTwoLine` compares a reason with the error built from it (`main.go:3611`), `mrw skill --dir` puts the directory before a reason and keeps the reason whole (`cmd/mrw/skill168.go:212, 281`), `mrw_write` appends to one (`internal/mcp/tools.go:2564`), `newerLedger` replaces one (`internal/writer/flow.go:225`); T1 checks each and stops if one needs the old spelling |
| A refusal site is missed | Med | Low | it prints what it prints today; T1 has a case for each converted site but these, which no fixture reached: `fromcopy.go:179, 185, 188, 195-199` (a stat or read error, a non-regular source, a source over the size limit) and `replace.go:58` (an identity that cannot be read). They share their format with a covered neighbour. On Windows `pathop.go:226` is not reached (a path under a regular file reports as not found), and the two link cases skip where links need a privilege |
| A case needs a filesystem the gate's platform lacks | Med | Low | the respelling cases need a filesystem that folds case and skip where it does not (Linux CI); macOS and the Windows CI job run them |

## Rollback

Revert T2, then T1 (T2 uses T1's `asPlan` and `reasonSep`): a path inside a reason, and the start of a stopped write's error, are spelled as the platform spells them.

## Follow-ups

- [ ] The Windows probe of the built binary before the merge (T1, Stop Condition)
- [ ] T2: a refused destination or source, and a stopped commit's error (Decisions 6 and 7)
