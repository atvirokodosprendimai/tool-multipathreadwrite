# copy-study — what would ADR-167's `from=` copy have saved?

Measured 2026-10-22 (Vilnius) over every Claude Code transcript on one Mac. `scan.py` replays each
session in order and asks, at every write: could this body have been sent as `from=PATH:ADDR`,
which copies lines the session had already been served, instead of typing them out?

**Answer: very little.** Under ADR-167 as accepted (a `from=` hunk's whole body is the copied range),
the copy would have saved about **21,700 output tokens across 83.8 million**: 0.026% of reported output,
0.23% of write-body text. 95% of that saving is in one project, from a session that assembled a
dataset out of rows it had read. The k = 3 split estimate (each run of three or more lines sent as its
own `from=` hunk beside typed hunks) is about 125,700 tokens (0.15%); at k = 1 it is 143,079.

**What was decided.** Zy chose to build ADR-167 anyway (2026-10-22): this corpus measures what agents did
when copying cost a full re-type, so it cannot see copies they avoided for that reason. This study is the
baseline; re-run `scan.py` over transcripts written after `from=` ships and compare the same columns.

## Corpus

| | count |
|---|---|
| transcripts scanned (`~/.claude/projects/**/*.jsonl` over 10 KB) | 2,188 |
| of them main sessions / subagent transcripts | 1,308 / 880 |
| project directories (many are scratch or bench dirs) | 1,156 |
| bytes read | 3.62 GB |
| transcripts with at least one write | 374 |
| write units (one Edit, one Write, one plan hunk, one `--create`) | 42,136 |
| `--sub`/`--rewrite` writes (no body, so not measured) | 193 |
| output tokens the API reported (`usage.output_tokens`, one per message id) | 83,788,516 |

Runtime: 20.3 s wall clock under `nice -n 15`, 126 MB peak RSS, one process. The corpus is live: this
run, after the review fixes, saw 7 more subagent transcripts than the first (2,181), so a before/after
difference mixes the fixes with new sessions.

## Headline (k = 3, exact bytes)

| | chars | ≈ tokens | share |
|---|---|---|---|
| write-body text | 38,268,239 | 9,567,060 | 11.4% of reported output |
| KEPT (re-typed lines from inside the edited region) | 1,469,186 | 367,296 | 3.84% of body |
| COPYABLE (any run of ≥ 3 lines already served) | 1,098,803 | 274,701 | 2.87% of body |
| **saving, whole-body `from=` (ADR-167 as accepted)** | 86,762 | **21,690** | 0.026% of output |
| saving, k = 3 split into adjacent hunks | 502,885 | 125,721 | 0.15% of output |

Whitespace-normalised (a re-indented copy, which `from=` cannot make because it copies bytes as they
are) barely moves these numbers: copyable 3.04%, whole-body 21,694 tokens, split 133,091 tokens.

### By k (exact / normalised)

| k | runs | copyable share | whole-body hunks | whole-body ≈ tokens | split ≈ tokens |
|---|---|---|---|---|---|
| 1 | 13,334 / 14,343 | 4.06% / 4.36% | 271 / 275 | 25,428 / 25,460 | 143,079 / 152,330 |
| 3 | 7,546 / 7,964 | 2.87% / 3.04% | 30 / 31 | 21,690 / 21,694 | 125,721 / 133,091 |
| 5 | 1,699 / 1,834 | 1.36% / 1.43% | 18 / 18 | 21,352 / 21,352 | 95,597 / 99,930 |
| 10 | 232 / 241 | 0.59% / 0.61% | 9 / 9 | 20,943 / 20,943 | 51,952 / 53,335 |

k counts the run's non-blank lines.

### By tool family (k = 3, exact)

| family | units | body ≈ tokens | KEPT | copyable | whole-body ≈ tokens | split ≈ tokens |
|---|---|---|---|---|---|---|
| mrw plan hunks | 39,401 | 7,534,748 | 4.71% | 3.20% | 21,690 | 109,316 |
| mrw `--create` | 410 | 336,674 | 0.17% | 3.13% | 0 | 3,728 |
| Write | 1,519 | 1,592,214 | 0.35% | 1.44% | 0 | 12,571 |
| Edit | 806 | 103,425 | 6.29% | 0.12% | 0 | 106 |

### Main vs subagent (k = 3, exact)

| | units | body ≈ tokens | KEPT | copyable | whole-body ≈ tokens | split ≈ tokens |
|---|---|---|---|---|---|---|
| main | 31,936 | 7,126,957 | 3.40% | 3.15% | 21,578 | 102,935 |
| subagent | 10,200 | 2,440,102 | 5.14% | 2.05% | 113 | 22,786 |

### Distribution

- 198 of the 374 transcripts with writes have at least one copyable run. Only 14 have a whole-body saving.
- The top decile of transcripts holds 100% of the whole-body saving and 83% of the split saving. For
  comparison, it holds 72% of all write-body text.
- One project (P2, 294 transcripts) holds 20,653 of the 21,690 whole-body tokens.
- Run lengths (k = 3, exact): 3–4 lines 5,847; 5–9 lines 1,467; 10–19 lines 204; 20–49 lines 23; 50 or more 5.
- Moves: 186 of 7,546 runs (69,806 chars, 6.4%) copy a source that the same write, or the next one,
  deletes or replaces. This is approximate, because line numbers go stale after edits.
- Per project (labels by write volume, no names): see `by_project_top10` in `results.json`.

## Read redundancy (exploratory)

The question here is about INPUT, not output: how much of what reads served had the session already
been served unchanged? That is what a "delta re-read" (serve only what changed since you last read
it) could save. A served line is redundant when the session had already been served the same text at
the same file and line number, and wrote nothing to that file in between: any write unit naming the
file empties its pool. Results are under `read_redundancy` in `results.json`.

| | transcripts with numbered reads | served chars | ≈ tokens | redundant chars | ≈ tokens | share |
|---|---|---|---|---|---|---|
| all | 892 | 124,614,135 | 31,153,534 | 6,579,729 | 1,644,932 | 5.28% |
| main | 147 | 49,038,886 | 12,259,722 | 3,761,258 | 940,314 | 7.67% |
| subagent | 745 | 75,575,249 | 18,893,812 | 2,818,471 | 704,618 | 3.73% |

The top decile of those transcripts holds 75% of the redundant chars. Limits:
- "Served" counts every numbered line in a tool result, once per time it was served. It does not
  count the same text staying in context across later turns; that is a cache question, not a re-read.
- Writes the scanner does not see (`sed -i`, a formatter, `git checkout`, a `--sub` write) do not
  empty the pool. A line re-served with the same text after such a write still counts as redundant,
  and that is still content the session already held.
- An mrw read's `==>` header and `@@` lines, and the Read tool's line-number gutter, are not counted.

## Method

Each transcript is one session with its own pool. Subagent transcripts (`*/subagents/*.jsonl`) are
scanned and counted separately, and do not inherit the parent's pool.

1. **SEEN**, the lines a session was served with a line number, as the model saw them (the
   tool_result text, not `toolUseResult.stdout`):
   - `mrw read` output in a Bash result whose command names mrw: `==> PATH` headers and `N| text` lines;
   - `mrw_read` MCP results;
   - Read tool results (`N<TAB>` or `N→`), with the path from `toolUseResult.file.filePath`. A Read of
     saved mrw output is parsed as mrw output.
2. **WRITES**:
   - every heredoc in a Bash command that holds an `@@ PATH ADDR OP` header (plans are often written
     to a file in one call and applied in a later one);
   - inline plans when `mrw write` has no heredoc;
   - `mrw write --create P`, from a heredoc or from `< F` where F was written by an earlier `cat > F <<EOF`;
   - the `mrw_write` MCP `plan`;
   - Edit, MultiEdit and Write. A Write whose content is a plan counts as a plan.

   A `tool_use` id repeated in a transcript counts once. 10% of the raw units were such repeats.
3. **Same file**: two paths are the same file when they are equal or one is a `/`-suffix of the other.
4. **KEPT**:
   - Edit: the `new_string` lines that also occur in `old_string`, as a multiset.
   - mrw `replace` with a numeric address: body lines equal to that file's served lines inside the range.
   - Write, or an mrw `create` of a file already read: body lines equal to any served line of that file.
   - The 708 pattern-addressed replace hunks have no range, so they get no KEPT.

   Narrower hunks are the fix for KEPT lines, not copying.
5. **COPYABLE**: a greedy maximal run of consecutive non-KEPT body lines that occurs as consecutive
   lines of one served file, seen before the write, and outside the region being edited.
   - A run is seeded only at a line with ≥ 8 non-whitespace chars, and tries at most 200 pool
     positions per seed.
   - It extends both ways across short lines, and is trimmed of blank edge lines.
   - It must carry ≥ 40 non-whitespace chars.
   - A seed must still hold the text it was indexed under: when a re-read shows a line changed, its
     old text leaves the index, so no run starts on a source line that is no longer there.
   - A body's terminal newline ends its last line; it does not add an empty one. So a copied file
     ending in `\n` is still one whole-body run.
6. **SAVING**:
   - Whole-body: a hunk whose entire body is one run saves its chars minus 45, the assumed size of a
     `from=path:N-M` header.
   - Split: each run saves its chars minus 45 if it sits at a body edge, or minus 90 if it sits in the
     middle (one extra header to resume typed lines). Negative savings count as 0.
   - Tokens ≈ chars / 4. This is an approximation; no tokenizer was used.

## Limits

- Write bodies are not added to the pool. mrw licenses a file it just wrote, but the caller holds no
  line numbers for it. This undercounts copies of text the session wrote itself.
- Line numbers in the pool are the ones served. A file edited after it was read keeps its old numbers,
  so the source address and the move detection are approximate.
- Plan extraction is textual. A plan built by a script (awk, a loop) or written in a way the heredoc
  pattern misses is not seen, and neither is a body sent through `--sub`/`--rewrite` (193 writes).
  Plain `cat > file <<EOF` writes that are not mrw plans are not counted as writes.
- Bash output the host truncated or saved to a file is counted only as far as the model was shown it.
- k counts lines, so short boilerplate (`if err != nil {` … `}`) can form a 3-line run. The 40-char
  floor and the header cost keep most of it out of the saving.
- Left out:
  - Codex sessions: `~/.codex/sessions` held 4 files, not the ≈2,000 expected, in a different format
    (JS `exec` tool calls).
  - Cursor.
  - Sessions on other machines.

## Reproduce

```sh
python3 -I docs/bench/copy-study/test_scan.py          # self-test: PASS, exit 0
nice -n 15 timeout 1800 python3 -I docs/bench/copy-study/scan.py --out docs/bench/copy-study/results.json
```

`results.json` is written by `scan.py`. It and this file carry aggregates only: no paths, no file
contents, no project names, no session ids. `--examples` prints five sampled runs to stderr for a
human check, and is never published.

The self-test plants, in a synthetic transcript:

- a 6-line copy inside an Edit, between KEPT and new lines;
- a 4-line whole-body copy;
- a trivial `}` / `)` run, and a run under 40 chars;
- a heredoc plan with KEPT lines in its range and a re-indented copy that only the normalised
  variant counts;
- a repeated `tool_use`;
- an mrw `create` of a file already read;
- a Write whose content ends in `\n` and is a whole-body copy (red before the terminal-newline fix);
- a re-read in which one source line changed, followed by a copy that starts on its old text (red
  before the stale-seed fix);
- one file read twice with no write between (redundant) and one re-read after a write (not redundant).

Breaking the forward extension of the matcher (`n + (b - i)` → `n + (b - i) + 1`) turns 7 of its
14 checks red.
