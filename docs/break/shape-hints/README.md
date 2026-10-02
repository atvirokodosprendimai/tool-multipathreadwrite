# Shape hints — measured against the pre-registered bar (ADR-119)

`stress.py` replays every modification hunk of `git diff -U0 k^ k`, over the non-merge history of each
repository under `~/GolandProjects` and `~/CursorProjects`, as the `replace` a caller would have sent, and
counts how often each hint fires. A committed hunk is taken to be a correct edit, so the rate is a
false-positive rate. The bar was registered in `docs/adr/BACKLOG.md` ("Pre-registered for ADR-119") before
any run: under 5% in every language bucket with 50 or more replaces, and the field fixtures caught.

```sh
python3 docs/break/shape-hints/stress.py [--seed N] [--cap N] [--max-commits N] [--mrw bin/mrw] DIR...
```

## Measured 2026-10-02, on this machine's 16 repositories (one worktree skipped)

**Run 1 — the registered heuristics** (seed 119, 400 replaces a repository, 3,475 replaces). Both missed:

| heuristic | fixtures | worst buckets of 50+ |
|-----------|----------|----------------------|
| closer, line right after the range | 2 of 3 (the markdown fence's survivor is the 4th line after) | 0.00% everywhere |
| indent, first or last line | 2 of 2 | `.py` 19.35%, extensionless 14.29%, `.js` 11.89%, `.php` 8.99% |

Zy chose "K4 closer, defer indent". The amended closer was registered in BACKLOG ("Amended after the first
measurement") before run 2, which used a fresh sample.

**Run 2 — the amended closer** (seed 2119, 1,200 replaces a repository, 6,000 commits sampled, 8,262 replaces).
`closer_k4_token`: the body's last non-blank line, trimmed, is a closer-shaped token and equals one of the
next 4 non-blank lines. Fixtures 3 of 3. False-positive rate per bucket:

| bucket | n | closer_k4_token | (registered indent, for the record) |
|--------|---|-----------------|-------------------------------------|
| `.php` | 1825 | 3.45% | 39.34% |
| `.md` | 1599 | 0.13% | — (prose) |
| `.py` | 1377 | 0.36% | 16.85% |
| `.ts` | 1070 | 1.96% | 3.93% |
| `.go` | 718 | 3.34% | 3.48% |
| `.js` | 534 | 2.81% | 6.93% |
| `.json` | 486 | 1.85% | 0.00% |
| extensionless | 177 | 0.00% | 18.64% |
| `.yml` | 146 | 0.00% | 13.01% |
| `.sh` | 123 | 2.44% | 4.07% |

Under 5% in every bucket of 50+ — but the review of #322 showed the corpus could not hold the commonest correct
shape. A `-U0` hunk never contains an unchanged closer, so a replace THROUGH its own closer (an inner `if {…}`
replaced through its `}`, the outer `}` right below) was never replayed, and `closer_k4_token` fires on it. The
window without the token test (`closer_k4_trim`) went over in `.php` (5.15%) and `.yml` (8.22%).

**Run 3 — the refined closer** (registered in BACKLOG, "Amended after review", before it ran; seed 3119, 1,200
replaces a repository, 6,000 commits). `closer_k4_token_ne` adds: the replaced range's last non-blank line,
trimmed, is not the body's. Each hunk whose next 4 non-blank original lines hold a closer-shaped line was replayed
a second time extended through it (`… thru` buckets). Fixtures 3 of 3.

| bucket | n | closer_k4_token (run 2's) | closer_k4_token_ne (ships) |
|--------|---|---------------------------|----------------------------|
| `.md` | 1650 | 0.12% | 0.12% |
| `.php` | 1458 | 3.50% | 3.36% |
| `.py` | 1338 | 0.37% | 0.37% |
| `.ts` | 1270 | 1.42% | 1.34% |
| `.go` | 738 | 1.90% | 1.90% |
| `.js` | 540 | 1.67% | 1.67% |
| `.json` | 453 | 0.44% | 0.44% |
| `.vue` | 76 | 5.26% | 3.95% |
| `.php thru` | 796 | 37.19% | 0.00% |
| `.ts thru` | 672 | 23.66% | 0.00% |
| `.go thru` | 485 | 55.46% | 0.00% |
| `.json thru` | 365 | 10.96% | 0.00% |
| `.js thru` | 230 | 32.61% | 0.00% |
| `.md thru` | 186 | 18.82% | 0.00% |
| `.py thru` | 150 | 11.33% | 0.00% |
| `.vue thru` | 54 | 35.19% | 0.00% |

The refined closer is under 5% in every bucket of 50+, on both replays: it ships. Its 0.00% on the `thru` replay
holds BY CONSTRUCTION — those replays end in the closer they replaced, which is exactly what the refinement
excludes — so that replay's value is the left column: it is what the unrefined closer would have cost.

**Binary cross-check**: see the end of this file. An earlier cross-check of run 2's definition (seed 2119, 3,475
replaces) agreed on all of them after a harness fix: it had JSON-quoted `anchor=`, so a line holding `ė` never
matched. The harness now passes the anchor raw, on multi-line replaces only.

## What this does not measure

- The field fixtures are reconstructions built to the offsets BACKLOG records, not the original files.
- Replayed history is committed edits, not the edits agents attempt; the fixtures stand for those.
- The corpus is one machine's repositories, dominated by PHP, Markdown, Python and TypeScript. YAML had 18
  replaces in run 1 and 146 in run 2, which is why the indent hint has no YAML-only measurement yet.
- Orphans above the body (a short address) are not looked for; see BACKLOG "From ADR-119".
- A correct replace that adds a new block ending in a closer just above a sibling's identical closer, within four
  non-blank lines, can still fire. The `-U0` replay contains that shape, and its rate is the one in the table.
- A known miss, not measured: a range that stops short on an INNER closer — `3-6` ending in `\t}` where the
  function's `}` at 7 was meant, or an HTML range ending in an inner `</div>` — carries no hint, because the
  comparison is trimmed and the range "already ended in" the same token. The balance row still reports the Go
  case (`{ +1 → +0`); nothing reports the HTML one. Comparing with indentation kept would catch both, at the cost
  of a hint on a correct replace that re-indents a block.

## Binary cross-check of the shipped closer

`stress.py --seed 3119 --cap 400 --mrw bin/mrw`, the binary built from the ADR-119 branch with the refined
closer: 4,686 replays (the `-U0` replays and their through-closer twins), and the binary's `closer` key agreed
with `closer_k4_token_ne` on every one.
