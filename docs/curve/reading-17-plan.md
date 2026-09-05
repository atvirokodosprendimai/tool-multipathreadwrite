# Reading 17, plan: reading 14's arm, with the reply channel named before any trial

**Written and committed before any trial ran.** Reading 14's result stands as collected:
evidence-limited, because thirty of its forty-five trials replied through a harness channel its
rule had not named.

## Cells

Reading 14's forty-five cells, byte-identical (verified before the first trial): `curve generate
-bytes {2000,20000,200000} -position {early,middle,late} -distractors 12 -seed {1..5} -selector
odd-retries`, window from line 1. As generated they serve 71–73, 395–403 and 3,644–3,670 rows.

## Client and arm

**Client:** a fresh Sonnet subagent per trial (`claude-sonnet-4-5`).

**Arm: the bare tool-result arm.** Bash and Write only; the listed commands are `cat task.json`
and then `sed -n 'A,Bp' served.txt` ranges — one range for the 2 KB and 20 KB cells, twelve
300-row ranges and a tail for the 200 KB cells (fourteen commands with the `cat`). mrw's `N|` is
the only number on any row.

**Compliance, with the harness's channels named.** The listed set run, in order, nothing else in
Bash; merges of consecutive listed ranges tolerated; no Read, Grep or Glob before the Write; no
spill. After the Write, the harness's own protocol is not the trial's and does not void it:
`ToolSearch`, `mcp__agentsmemory__*` calls, and **`SendMessage` — the reply channel this client
uses to deliver its one-line answer**. Any of those before the Write, or any other tool at any
time, voids the trial.

## Predictions, recorded before collection

1. **At least 14 of 15 in every tier.** The author's expectation, stated so it can be wrong.
2. **If it bends, it bends with size**: the 200 KB tier below the 2 KB tier by at least three.
3. **Any miss is not at `target+2`**, and is classified as wrong-service or off-by-N on the right
   service.
4. **Compliance at least 40 of 45.**
5. **Cost from 2 KB to 200 KB between 2.2× and 3.0×.**

## What this decides

What reading 14 was meant to: at the ceiling, the strong client's flat curve is not an artefact of
an easy fixture; bent, the first place it degrades with served size is found, on this fixture
through this delivery. Either is accepted.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`.
