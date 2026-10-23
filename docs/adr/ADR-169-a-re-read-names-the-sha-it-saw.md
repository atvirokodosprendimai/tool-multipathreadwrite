# ADR-169: A re-read names the sha it saw, and an unchanged file is answered without its text

**Status:** Accepted
**Accepted:** 2026-10-23 by Zy — *"good, so i accdpt the next adr, use measures, benchmarking, implemenitng, verifying, cutting a new release"*, after the record was drafted to Zy's answers of the same day: "Repeatable flag", "Success, named".
**Amended:** 2026-10-23, before any code, from two challenges of the accepted draft (Codex at high effort; the repository's `adr-challenger`), every finding checked against the code: a walked guard that its walk did not serve is named, not refused (Decision 1); the answer is per spec and says what it withheld (Decision 2); `--max-chars` keeps ADR-165's withholding (Decision 2); the no-licence claim covers the unchanged answer only, and the collision reasoning no longer promises a refusal the shared ledger cannot give (Decision 3); the receipt key is `mcp_read unchanged` with its element fields (Decision 4); `--core` carries the guard only if its bound holds (Decision 6); missing the adoption bar deprecates, never removes (Rollback); the measured numbers are stated as estimates over captured reads.
**Amended:** 2026-10-23, during execution, from the `go-reviewer` pass on the engine diff and Zy's answer to it (*"Still a problem"*): a range a guarded spec would not have served is reported and counted as without the guard (Decision 2); N and B honour `--max-lines` (Decision 2); a PATH holding a comma cannot be guarded, as with `--exclude`, and the help says so (Decision 1); entries an index invites back as plain specs are named files, and a walk that matches nothing still refuses two guards on one file (Decision 5).
**Date:** 2026-10-23
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-014, ADR-017, ADR-023, ADR-024, ADR-025, ADR-031, ADR-032, ADR-093, ADR-111, ADR-117, ADR-122, ADR-141, ADR-146, ADR-150, ADR-165, ADR-168, docs/bench/context-study/README.md, docs/adr/BACKLOG.md
**Governs:** `internal/read/read.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/readreceipt.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `scripts/contract.sh`
**Enforced-by:** None — until T1 lands, which adds `internal/read/ifchanged169_test.go::TestAGuardedReadOfAnUnchangedFileServesNothing`
**Invalidates:** ADR-025 — the clause of its Decision reading "An answer that delivered none of what was asked for is an error": a spec a guard answered `unchanged` is delivered, so the served-read return passes `len(observed) == 0 && len(unchanged) == 0` (T2, which adds the notice and Amendment to ADR-025 as ADR-024 carries ADR-025's). ADR-146's `incomplete:` kind is unchanged: an unchanged spec is not a range that was not served.
**Served-path change:** `mrw read --if-changed PATH=SHA …` (MCP `mrw_read` `if_changed`) answers each spec of a named file whose whole content still has that sha with one line, `==> PATH  unchanged since sha SHA — N lines, B bytes not served`, instead of its lines, and serves it as before when it changed.

## Context

Zy, 2026-10-23, choosing among next steps after v1.73.0: *"plan to address all of these"*, then, asked how a delta-read idea should proceed, *"Measure first"*. The idea (memoryd tool-multipathreadwrite/40053273) was a session map that answers a re-read with a delta.

**What the ledger can carry.** The read ledger is shared by every caller of a checkout, records no session and stores no text (`internal/seen/seen.go`, `Observation{SHA, Spans, Written, Shown}`, format 6). So mrw cannot decide on its own that a caller already holds some lines — another caller's read licenses the same lines — and it cannot serve a line delta, which needs the old text. What it can answer exactly is a question the caller asks about the whole file: "is it still the bytes whose sha I saw?" Every served spec's header already prints that sha's first 8 hex digits (`internal/read/read.go:667`, the sha256 of the whole file).

**Measured 2026-10-23** over the agent transcripts on one Mac (`docs/bench/context-study/scan.py`, BACKLOG "Pre-registered for a whole-file re-read guard", PR #512; two runs, identical counts). The guard's numerator — one call's serve of one file in which every line was a within-epoch re-serve with no write to the file between — is 1,687,016 chars on Claude Code, 1.32% of all served chars there; 0.86% on Codex; 6.10% on Cursor IDE. The bar registered before the run was 1.0% on at least one host; it is met on Claude Code and Cursor. The same run's within-epoch re-serves of any kind are 4.03%, 2.41% and 9.87%; the rest of those were served in one call beside lines of the same file that were not re-serves.

**What those numbers are, and are not.** They are estimates over the reads each adapter could number: on Codex that is 42% of read-like commands, and the study says its Codex shares are bounds in neither direction. The scanner sees repeated lines, not whether the whole file was unchanged, and not whether the caller kept the sha. It counts every read a host made, its own Read tool included, while the guard saves only reads made through mrw by a caller that passes it — on Cursor most served text came through the host's own tools. The share of re-reads that are both through mrw and made by a caller holding the sha is **not measured**; T3 S3 measures what the guard actually answers after it ships.

## Existing Primitives Audit

- `read.Run` (`internal/read/read.go:594`) loads each spec's file whole and computes its sha256 before serving (`:639-689`, header `:667`): the guard compares against a value the read already has. Reuse.
- The `sha=` plan guard (`internal/plan/plan.go`) compares a caller's sha with the file's for a write. Same comparison, other direction; the read side takes a prefix as printed. Reuse the idea, not the code.
- `--stat` (ADR-117) lists each file's length and sha without serving lines. Kept; not replaced.
- `mcp.ReadReceipt` (`internal/mcp/readreceipt.go`) builds both the `mrw_read` receipt and the CLI `--json` receipt (`cmd/mrw/main.go:1286`); the new key is added there once.
- The ledger (`internal/seen`) is untouched: an unchanged answer contributes nothing to it.
- `docs/receipts.txt` (ADR-111) takes the new key, appended, and `TestNoShippedMCPReceiptFieldDisappears` holds it.

## Decision

**1. The flag.** `mrw read` takes `--if-changed PATH=SHA`, repeatable, one per file; PATH is split from SHA at the last `=`. `SHA` is a hex prefix of the file's sha256, 8 to 64 digits — the 8 the header prints are enough — compared case-insensitively. `PATH` is resolved as a spec's path is (ADR-007), so two spellings of one file are one file; two guards on one file are refused, exit 2. A guard on a file the read names explicitly — a positional spec, `--files-from`, stdin is not a file and takes none — but which no spec of the read serves is refused, exit 2, naming it. A guard on a file a walk (`--grep`, `--ast-grep`, ADR-122) did not reach is **not** refused: the file may have stopped matching precisely because it changed, so the read names it on one stderr line, `-- guard not used: PATH (this read did not serve it)`, and it adds no problem. `--if-changed` is refused with `--count`, `--stat`, `--quiet`, `--sub` and `--rewrite`, exit 2 — each serves no lines or exists to license. It combines with ranges, patterns, the walks, `--files-from`, the working set, `-C`, `-N`, `--max-cols`, `--exclude`, `--no-ignore`, `--max-lines`, `--max-chars` and `--json`, none of which changes what it judges. The flag splits its value on commas, as `--exclude` does, so a PATH holding a comma cannot be guarded; the help says so.

**2. The answer, per spec.** `Run` loads a file once per spec; the guard is judged at each load. A spec of a guarded file whose loaded bytes still have that sha prints one line in place of its header and lines: `==> PATH  unchanged since sha SHA — N lines, B bytes not served`, where N and B are what that spec would have served, `--max-lines` included (the caller's prefix as given). A spec whose file changed is served exactly as an unguarded one is, its header naming the new sha. So `--if-changed f=X f:1-3 f:9-12` prints two unchanged lines, or, if a writer lands between the two loads, one unchanged line and one served range. An unchanged spec adds no problem of its own: a read of only unchanged specs exits 0. A range the spec would not have served — a pattern that matches nothing, a start past the end — is reported and counted beside the unchanged line exactly as without the guard, so `no match:` and `incomplete:` (ADR-146) still come from the range (Zy, 2026-10-23, after the engine review: a read that returns nothing stays visible). `--max-chars` (ADR-165) treats the unchanged line as output like any other: if the budget withholds it, the spec is withheld and listed on stderr as ADR-165 lists any spec, the read exits 1 `incomplete:`, and the receipt does not list it as unchanged.

**3. What the unchanged answer licenses: nothing.** An unchanged spec records nothing in the ledger and revokes nothing. Lines an earlier read licensed stay licensed as long as the ledger's own observation still holds this sha; lines it did not license stay unlicensed. A guard that does not match changes nothing either: the spec is served and recorded as any read is, and a `mrw_read` `ack` promotes its checkpoints as before (ADR-031). The guard is a question about text, never about permission. Two limits are stated, not hidden: the ledger is shared, so if another caller reads or writes a changed version, the ledger's observation moves to it (`seen.go` merge), and a caller who was told "unchanged" by an 8-hex prefix that collided (about one in 2^32 for an accidental match; not a defence against a chosen collision) is not then protected by a refusal; and a file changed and changed back (X to Z to X) is answered unchanged while a write may still be refused as "changed since", which is safe. A longer prefix narrows the first.

**4. The receipt.** `--json` (ADR-165) and the `mrw_read` receipt gain `unchanged`: a list of `{"path", "sha", "lines", "bytes"}` — the sha as the caller gave it, `lines` and `bytes` as Decision 2's line prints them — absent when empty. Appended to `docs/receipts.txt` as `mcp_read unchanged` and its element fields (ADR-111), with the receipt-key test extended to read a list's elements.

**5. MCP.** `mrw_read` takes `if_changed`, an object of path to sha, declared in its input schema (`internal/mcp/mcp.go`, ADR-093), with Decision 1's rules; a refused guard is a tool error naming it. A JSON object with a key repeated decodes to its last value, so two guards on one file cannot be told apart there; this is said in the argument's description. `isError` is set only when nothing was served and nothing was answered unchanged (amends ADR-025, above). An unchanged spec gets no checkpoint, since no line is served (ADR-031). When matches are too large to serve (ADR-017), the index answer is not judged — it serves no text — and the caller sends `if_changed` with the specs it resends; a guard whose file a page resent with `grep` does not include is named, as a walked guard is in Decision 1; entries resent as plain `specs` are files the call names, so Decision 1's refusal applies to them. A walk that matches nothing still refuses two guards on one file. A page's `next_read` (ADR-014) is a plain spec; the guard is judged on the call that serves, and a caller resends it with a continuation if it wants it judged again.

**6. Taught where callers look.** `mrw instructions` and the skill `mrw skill` prints (ADR-168), README and AGENTS.md name the guard as the way to re-read a file you read earlier this session: `mrw read --if-changed f.go=8dc37337 f.go:40-60`. `mrw instructions --core` carries it only if ADR-141's 300-word bound holds with it — measured 299 words on 2026-10-23, so a core line that it replaces must be named in T3's pull request, or the core keeps its text and the full instructions carry the guard.

**What would make this decision wrong:** callers that never pass the guard, or pass it on reads that were going to change anyway. The ceiling above is an estimate of the most it could save; the data that could show it saving nothing exists only after it ships. T3 S3 registers, before the release, the cohort, the formula and the bar it must clear, and what follows if it does not. Valid for the three hosts measured on 2026-10-23 on one Mac.

## Alternatives Considered

- **Elide from the ledger:** mrw remembers what it served and answers a re-read of unchanged lines with "unchanged" on its own. Rejected because the ledger is shared and sessionless: lines another caller read, or this caller read before a compaction, would be withheld from a context that never held them.
- **A line delta:** serve only the lines that changed since the last read. Rejected because the ledger stores no text, and storing it would put file contents in mrw's state for every read — a new persistent store of source text outside the tree, for a saving the scan bounds below 10%.
- **A spec suffix** (`f.go@8dc37337:40-60`): one token per guarded spec. Rejected by Zy (2026-10-23) for the repeatable flag: a new address grammar every parser (`read`, `--files-from`, MCP `specs`) must learn, and `@` occurs in paths.
- **Exit 1 with a kind of its own** for an unchanged answer, keeping ADR-025 whole. Rejected by Zy (2026-10-23): every caller using the guard would have to read exit 1 as success.
- **Refuse every guard whose file the read did not serve.** The first draft's rule. Rejected after the `adr-challenger` review: under a walk, a file that stopped matching because it changed would refuse the whole read, exit 2, when the true answer is "changed".
- **Do nothing; `--stat` then read.** Two calls where one does. How often callers ask first was not measured.

## Component / Boundary Impact

- `internal/read` (engine; this record owns the change): the guard check in `Run` at each spec's load, the unchanged line, the per-spec counts.
- `cmd/mrw`: the flag, its refusals, the unused-guard line, the `--json` key through `mcp.ReadReceipt`.
- `internal/mcp`: the `if_changed` argument and its schema, the `isError` condition, the receipt key.
- `internal/guide`: the taught line.
No package changes ownership; the ledger (`internal/seen`) is not touched.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --if-changed PATH=SHA` | new flag | `cmd/mrw` | agents, scripts |
| `==> PATH  unchanged since sha SHA — N lines, B bytes not served` | new output line, per spec | `internal/read` | agents; parsers of mrw output (copy-study's `parse_mrw_output` counts numbered lines only, so it adds nothing) |
| `-- guard not used: PATH (this read did not serve it)` | new stderr line | `cmd/mrw` | agents |
| `--json` / `mrw_read` receipt `unchanged[]` `{path, sha, lines, bytes}` | new key, appended to `docs/receipts.txt` | `internal/mcp` (`ReadReceipt`) | adapters (ADR-165) |
| `mrw_read` `if_changed` | new argument in the declared input schema | `internal/mcp` | MCP hosts |
| `mrw_read` `isError` | false when anything was served or answered unchanged | `internal/mcp` | MCP hosts |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `read.Options.IfChanged` and the `unchanged` receipt list | T1 | T2, T3 | No — additive |
| `mrw_read` `if_changed` | T2 | T3 | No — additive |

## Implementation

`tasks/README.md`: T1 (the CLI guard), T2 (MCP, and ADR-025's amendment), T3 (taught, and the adoption bar). Execution order T1, T2, T3; one pull request.

## Consequences

- **Positive:** a re-read of a file that did not change costs one line instead of its text, on the reads a caller guards; an estimated ceiling of 1.32% of all served chars on Claude Code and 6.10% on Cursor, over captured reads, of which the mrw-and-sha share is unmeasured.
- **Negative:** one more flag on `read` and two new output lines; the saving depends on callers keeping a sha and passing it.
- **Neutral:** an MCP read whose only answers are unchanged is no longer an error (ADR-025 amended); exit codes and the ledger are otherwise unchanged.

## Out of Scope

- Eliding re-served lines from the ledger, or serving line deltas (permanent: boundary: the ledger is shared, sessionless and holds no text; this record chooses not to change that)
- A guard on `mrw write` beyond the existing `sha=` (permanent: boundary: a write already refuses a file whose sha moved under the ledger's observation, ADR-002)
- Teaching hosts' own Read tools to take a sha (permanent: boundary: mrw controls only its own surfaces)
- Standing-instruction files outside this repository — `~/.cursor/rules/mrw.mdc`, `~/.codex/AGENTS.md`, the user's `~/.claude/CLAUDE.md` (permanent: boundary: they are the owner's machine configuration, not this repository's; the memoryd `craft` `mrw` skill is refreshed from `mrw skill` at release, ADR-168 T4)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Callers never pass the guard, so nothing is saved | Med | Low | taught in the instructions and the skill (T3); the adoption bar is registered before release (T3 S3), and missing it deprecates the flag |
| An 8-hex prefix matches a changed file by accident | Low | Low | about one in 2^32 per guarded read; the caller reasons on stale text and a write may not be refused (Decision 3); a longer prefix lowers it |
| A caller guards with a sha from somewhere else (git, another tool) | Low | Low | the answer is still true about the bytes; nothing is licensed (Decision 3) |
| `--max-chars` withholds an unchanged line | Low | Low | ADR-165's rule applies unchanged: withheld, listed, exit 1, not in `unchanged` (T1 tests it) |

## Rollback

Before a release: revert the pull request; no persistent state is written. After a release the flag, the MCP argument and the receipt key are contracts (ADR-111, ADR-165): if T3 S3's bar is missed, a record deprecates them — the taught lines are removed and the flag keeps working — and removal waits for a major version.

## Follow-ups

- [ ] none at authoring
