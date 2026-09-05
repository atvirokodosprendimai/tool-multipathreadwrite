# Reading 14, plan: a harder fixture for the strong client, through the bare tool result

**Written and committed before any trial ran.** Readings 3, 8 and 9 stand as collected; nothing
here changes them.

## Why a fourteenth reading

The note's limits say it: the strong client sat at the ceiling on both fixtures measured (readings
2 and 3), so its curve could not bend and said nothing about where it would. Reading 3's fixture
had four services — the target and three distractors — so the relation "the one whose retry budget
differs" is decided over four candidates. This reading makes the relation harder to hold across a
large window: thirteen services, the target and twelve distractors, at every size, through the
delivery that readings 8 and 9 showed to be flat for the weaker client on the four-service fixture.

## Cells

Forty-five new cells, generated with `curve generate -bytes {2000,20000,200000} -position
{early,middle,late} -distractors 12 -seed {1..5} -selector odd-retries`, window from line 1.
Served sizes measured at generation: about 2,009, 20,006–20,045 and 200,002–200,037 bytes; thirteen
`[service …]` blocks in every cell. Cells regenerate deterministically from those parameters.

## Client and arm

**Client:** a fresh Sonnet subagent per trial (`claude-sonnet-4-5`) — the client of readings 2 and 3.

**Arm: the bare tool-result arm of readings 8 and 9.** Bash and Write only; the listed commands
are `cat task.json` and then `sed -n 'A,Bp' served.txt` ranges: one range for the 2 KB and 20 KB
cells (72 and 385–395 rows), twelve or thirteen 300-row ranges plus a tail for the 200 KB cells
(3,662–3,678 rows). mrw's `N|` is the only number on any row. Compliance as reading 8
pre-registered it: the listed set run, merges of consecutive listed ranges tolerated, nothing else
run, no Read/Grep/Glob before the Write, memory-tool calls after the Write not the trial's, no spill.

## Predictions, recorded before collection

1. **The strong client stays at the ceiling on the harder fixture: at least 14 of 15 in every
   tier.** This is the author's expectation, stated so it can be wrong.
2. **If it bends, it bends with size**: the 200 KB tier below the 2 KB tier by at least three.
3. **Any miss is not at `target+2`** — this arm carries no second number — and a miss that IS a
   wrong-service miss (the right line of a distractor) is reported as such, separately from an
   off-by-N on the right service.
4. **Compliance at least 40 of 45.**
5. **Cost from 2 KB to 200 KB between 2.2× and 3.0×**, the band readings 2, 3, 4 and 9 all fell in.

## What this decides

At the ceiling: the strong client's flat curve is not an artefact of an easy fixture, and a
thirteen-way relation held across 200 KB costs it nothing. Bent: the first place the strong client's
addressing degrades with served size is found, and it is a property of the fixture's difficulty
and the window together — and the cap question reopens for that client, on this fixture, through
this delivery. Either answer is accepted (the criterion in `BACKLOG.md`: a flat curve is an answer).

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`.
