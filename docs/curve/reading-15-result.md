# Reading 15, result: the second family found every target and wrote mrw's grammar eleven times in forty-five

**Collected 2026-09-05 under `reading-15-plan.md`, committed before any trial ran.** Forty-five
trials, `gpt-5.6-sol` at high reasoning effort through `codex-cli 0.153.0`, the prompt delivery,
on reading 4's forty-five cells (byte-identical to readings 8–13's copies).

## Compliance: 11 of 45 under the pre-registered rule; the other 34 are void and reported

The rule voided a trial whose final message holds no parseable plan. Thirty-four final messages
hold no `@@ <path> <line> <op>` header that `curve score` accepts: every one of them is a plan in a
format the client chose — six in Codex's own `*** Begin Patch` grammar, seven under a
`*** Begin Plan` / `*** Begin MRW Plan` banner, seven as a JSON object with `path`, `sha` and
`hunks`, thirteen under a prose header (`file: services.conf`, `path: services.conf`,
`replace services.conf:725`), one echoing mrw's own `==> services.conf  sha …` read header as if it
were a plan. Every process exited 0; none ran a command in its empty directory.

The prompt was the cell's instruction and the served text, and the instruction says "author one
mrw write plan" without showing what one looks like. Every Claude client in readings 2–14 was shown
the two-line shape in its prompt. This client was not, and thirty-four times it supplied its own.
**That is a finding about ADR-012's question — a surface that demands a format must teach it —
not about addressing**, and it is what reading 16 corrects: the same arm with the two-line shape
in the prompt.

## The curve, over the eleven parsed plans

| Served bytes | parsed | hits among parsed | void (foreign format) |
|---|---|---|---|
| 2,000 | 8 | 8 | 7 |
| 20,000 | 2 | 2 | 13 |
| 200,000 | 1 | 1 | 14 |

Eleven of eleven parsed plans addressed the target. The fall in the parsed count with size — 8, 2, 1
— is an observation about which format the client reached for after more served text, not a rate
this reading can put an interval on. Prediction 1 (at least 14 of 15 per tier) is **not decidable**:
the tiers have 8, 2 and 1 scorable trials. Prediction 3 (compliance at least 42 of 45) **fails**:
11 of 45.

## What the thirty-four void trials named

Read after collection and not scored: each of the thirty-four foreign-format messages names the
target's line number in its text — 34 of 34. A number in a message is not an applied plan, and
`curve score` applies plans, so this is reported beside the table and not in it. Taken with the
eleven that parsed, no trial in forty-five pointed at a wrong line; forty-five of forty-five found
the target, and thirty-four could not tell mrw so. Prediction 2 (no miss at `target+2`) holds
trivially: there was no miss.

## Cost

Not measured, as the plan said.

## What this decides

Nothing about served size for this family; the reading voided itself on format. What it measured
instead is the cost of a format that is demanded and not taught: a strong client from a second
family, shown mrw's served text with no example of a plan, wrote one mrw could apply eleven times in
forty-five, and wrote its own patch grammar or a JSON object the rest of the time. ADR-012 put the
teaching into the MCP surface's descriptions and instructions; this prompt carried neither, and
the result is what a caller who has only the served text does.

## Provenance

Forty-five scores in `docs/curve/reading-15-scores/` (thirty-four `refused_parse`);
`reading-15-tally.json` is computed over them. The final messages are not committed; the format
counts above are from them.
