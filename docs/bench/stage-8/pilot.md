# Stage 8: the canary, the preflight and the uncounted smoke (ADR-193 T3)

Run 2026-11-04 on this Mac, bench built from `1973d056`, before any counted attempt. Nothing here is counted: the smoke's
stage is `pilot-8` and none of it sits in T4's directory.

cli: 2.1.317 (Claude Code)
mrw: v1.83.0 (3a97083), the release asset ADR-192 T3 checked against `SHA256SUMS.txt`
paragraph: In this session, make every edit to an existing file as a SEARCH/REPLACE block and apply it with mrw write --format=search_replace FILE. A block is the file's path on its own line, then <<<<<<< SEARCH, the file's current lines exactly as they are, =======, the lines that replace them, and >>>>>>> REPLACE; one FILE may hold many blocks for many files. SEARCH must match exactly once. End SEARCH and REPLACE with the same unchanged line, and the block needs no mrw read of the lines it quotes. Use a native @@ hunk only for an edit by line number, and mrw write --grep P --sub R for one change at many sites.
paragraph sha256: 44bb357923a27086530f350397cceb2de6f66f7a90bffa06093959cb9b99e817

The sha is the one both smoke runs' `run.json` record as `edit_paragraph_sha256`. The paragraph is ADR-193's as accepted:
round 1 met S4's bar, so it was not changed.

## S2: the canary

`canary-claude-sonnet-5-5.json` and `canary-claude-opus-5-5.json`: stage8, B and S, pass, no problems.

## S3: the preflight

A scratch git checkout holding `notes.txt` (alpha, beta, delta), Sonnet, the CLI above, mrw v1.83.0 on PATH, a fresh
`XDG_STATE_HOME`, S's tools (Bash, Glob), its appended prompt (`mrw instructions` and the paragraph) and bench's Bash hook
(`bench hook --log … --tree …`). The agent was told not to read the file. In one Bash call it wrote a SEARCH/REPLACE
document to `$TMPDIR/edit.txt` by heredoc and ran `mrw write --format=search_replace "$TMPDIR/edit.txt"`, with no
`mrw read`:

- `ok notes.txt 2 replace -1 +1`, `1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied`, exit 0; line 2 became `gamma`.
- The hook refused nothing. Its log holds one `read cat` line: `cat > "$TMPDIR/edit.txt"` counted as a read, not refused.
- `bench_edits.py count` on it: landed 1, refused 0, attributed 1, attributed_landed 1, decided 1, undecided 0.

Cost $0.07.

## S4: the smoke

B and S on `matplotlib__matplotlib-22719` and `astropy__astropy-13033` (the first 2 of `bench/swe-bench/selection.json`,
outside the pool), both models, one attempt each. Receipts and `run.json` in `smoke/`. `bench_edits.py count`:

| Attempt | landed | refused | attributed | attributed landed | solved | cost |
|---|---|---|---|---|---|---|
| astropy B Sonnet | 1 | 0 | 0 | 0 | yes | $0.041 |
| astropy S Sonnet | 1 | 0 | 1 | 1 | yes | $0.048 |
| matplotlib B Sonnet | 1 | 0 | 0 | 0 | yes | $0.081 |
| matplotlib S Sonnet | 1 | 0 | 1 | 1 | yes | $0.099 |
| astropy B Opus | 2 | 0 | 0 | 0 | yes | $0.159 |
| astropy S Opus | 2 | 1 | 1 | 1 | yes | $0.228 |
| matplotlib B Opus | 4 | 0 | 0 | 0 | yes | $0.191 |
| matplotlib S Opus | 2 | 0 | 2 | 2 | yes | $0.150 |

Solved is `bench analyze`'s rule. Every S attempt on both models made an attributed SEARCH/REPLACE write call, so S4's bar
was met on round 1. Total $1.00. Two instances say nothing about cost or solving; T4 is the counted check.
