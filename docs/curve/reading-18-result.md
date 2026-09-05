# Reading 18, result: the arm cannot be measured at 200 KB, because the host truncates mrw's page

**Collected 2026-09-05 under `reading-18-plan.md`, committed before any trial ran. Five of fifteen
trials, all five non-compliant by the plan's own in-band rule. The reading is VOID at this size, and
the reason it is void is the finding.**

The remaining ten were not run: they would fail compliance for the same structural reason, which is
not a property of the client.

## What the five trials reported

Every trial wrote both files in the required order (`coverage.json` strictly newer than
`result.json`, checked on mtime), used no forbidden tool, sent `next_read` back once, and named the
file's true last line. Every one failed on one clause: `service_block_lines`.

| cell | true block lines | blocks the client reported seeing | addressed | target |
|---|---|---|---|---|
| `200000-early-1` | 723, 1448, 2172, 2896 | **2896** | 2898 | 725 |
| `200000-early-2` | 724, 1450, 2176, 2902 | **2902** | 2904 | 726 |
| `200000-early-3` | 724, 1449, 2174, 2899 | **2899** | 2901 | 726 |
| `200000-early-4` | 724, 1450, 2176, 2901 | **2901** | 2903 | 726 |
| `200000-early-5` | 728, 1457, 2186, 2915 | **2915** | 2917 | 730 |

The clients were not guessing and were not careless. Each reported seeing exactly one service block,
addressed the `timeout` line two rows under it, and got the file's last line right. **The report is
accurate**: one block is all they were shown.

## Why: mrw's page is truncated again before it reaches the model

A diagnostic client, scored nowhere, made one bare-path `mrw_read` against `200000-early-1` and was
asked only what it received.

mrw did its part correctly. It served the first page and said so, in its own words
(`internal/mcp/tools.go:539`):

```
-- PARTIAL: lines 1-2727 of 3619. 892 line(s) remain.
-- Send specs ["tmp/curve/200000-early-1/tree/services.conf:2728-"] to continue, or a narrower range of your own.
-- Stopping here means you have part of this file, not the file.
```

What reached the model was not that page. It was lines 1–90, then a marker reading
`[141140 characters truncated]`, then lines 2644–2727 — 58,860 of the page's 200,000
characters, head and tail, middle discarded. The phrase "characters truncated" appears **nowhere in
this repository's source**; the truncation is the host's, applied to a tool result mrw had already
capped at its own limit.

The consequence for the task is exact. In this cell the four service blocks sit at 723, 1448, 2172
and 2896. Page one spans lines 1–2727 and so contains the first three — all three inside the
discarded middle. The client asked for `next_read`, received lines 2728–3619, and there met the fourth block,
the only one it ever saw. It answered from it. **Three of the four candidates were never delivered,
and nothing in the conversation said which ones were missing.**

## The part that is a defect, not a measurement

`internal/mcp/tools.go:535` records the page in the read-before-modify ledger, with this reason:

> The page WAS shown, so it is recorded — and only the span it served, which is what keeps a page
> from licensing lines the caller never saw.

On this host the page was **not** shown, and the ledger records it anyway. After the trials above,
`mrw seen` claims **`lines 1-3619`** — the whole file — for that path.

**This was tested, not inferred.** A plan replacing line 1500, which sits inside the discarded
middle and which no client was ever shown, was handed to `mrw write`. It applied: `"status": "ok"`,
`"applied": true`, exit 0, and line 1500 of the fixture then held the probe text.

**ADR-002's guarantee is inverted on this delivery path** — mrw edited a line its caller had not
seen, because the ledger was written from what the server sent rather than from what the host
delivered. It is the same class of defect as issue #109, reached by a different route: ADR-023
removed an envelope that replaced the served text; this is the served text surviving mrw and being
cut afterwards, with the ledger already written.

`mcp.MaxResultChars` is 200,000. One observation, on one host and one build, puts what actually
reached the model at 200,000 − 141,140 = 58,860 characters of that page. That is arithmetic from a
single result, not a measured boundary, and no boundary is characterised here. What matters for the
defect is not where the limit sits but that mrw cannot observe it: from inside the server a
truncated result is indistinguishable from a delivered one.

## What this does and does not establish

**Establishes:** at 200 KB, through `mrw_read` on Claude Code 2.1.261, a client following mrw's own
paging protocol faithfully is shown a fraction of the file and is not told which part is missing, and
mrw's ledger then licenses writes to the rest. The five trials' addressing behaviour is fully
explained by what they were shown, so they say nothing about the client and nothing about served
size.

**Does not establish:** any correct-address rate for the MCP delivery arm. ADR-020's open item is
still open. Nor does it establish where the host's limit lies: 58,860 characters is arithmetic from
one result, not a measured boundary.

**Not affected:** readings 1 to 17. None used this delivery. The flat curve of readings 8 and 9
stands, and so does everything the corpus says about served size.

## What replaces it

`reading-19-plan.md`: the same arm at **2,000 and 20,000 bytes**, where mrw's own render is far below
both its cap and the host's truncation point, so the whole served text reaches the model in one
result with no paging. That is the size range where this arm can be measured at all, and readings 9
and 13 have committed numbers on exactly those cells for two other deliveries to compare against.

The 200 KB question is not a curve question any more; it is a defect, and it belongs in a record
rather than in a reading.

## Provenance

The five scores and five compliance reports are not committed: they are not measurements of the
variable the plan named. The cells are the fifteen verified in `reading-18-plan.md`. The diagnostic's
own answers are in the transcript of one Haiku client and are quoted above rather than committed.
