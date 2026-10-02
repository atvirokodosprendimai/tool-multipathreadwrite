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

Under 5% in every bucket of 50+: the closer ships. The window without the token test (`closer_k4_trim`) went
over in `.php` (5.15%) and `.yml` (8.22%), which is what the token test is for.

**Binary cross-check** (seed 2119, 400 replaces a repository, 3,475 replaces, `--mrw bin/mrw` built from the
ADR-119 branch): the binary's `closer` key agreed with the computed one on all 3,475. The first attempt
reported one disagreement, which was the harness: it JSON-quoted `anchor=`, so a line holding `ė` never
matched and the hunk failed. The harness now passes the anchor raw, on multi-line replaces only.

## What this does not measure

- The field fixtures are reconstructions built to the offsets BACKLOG records, not the original files.
- Replayed history is committed edits, not the edits agents attempt; the fixtures stand for those.
- The corpus is one machine's repositories, dominated by PHP, Markdown, Python and TypeScript. YAML had 18
  replaces in run 1 and 146 in run 2, which is why the indent hint has no YAML-only measurement yet.
- Orphans above the body (a short address) are not looked for; see BACKLOG "From ADR-119".
